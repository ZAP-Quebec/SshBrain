package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"sync"
)

type SshError int

const (
	ErrParsing SshError = iota
	ErrAddr
	ErrUnauthorized
)

type SshConnection struct {
	m        sync.Mutex
	server   *SshServer
	conn     *ssh.ServerConn
	chans    <-chan ssh.NewChannel
	reqs     <-chan *ssh.Request
	openAddr map[string]bool
	log      *log.Logger
	lPort    uint32
}

func NewConnection(server *SshServer, conn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) *SshConnection {
	return &SshConnection{
		server:   server,
		conn:     conn,
		chans:    chans,
		reqs:     reqs,
		openAddr: make(map[string]bool),
		log:      log.New(os.Stderr, conn.RemoteAddr().String()+"\t", log.LstdFlags|log.LUTC|log.Lshortfile),
		lPort:    32768,
	}
}

func (s *SshConnection) User() string {
	return s.conn.User()
}

func (s *SshConnection) PublicKey() string {
	if key, exists := s.conn.Permissions.Extensions["key-id"]; exists {
		return key
	} else {
		return "NONE"
	}
}

func (s *SshConnection) Dial(address string, port uint32) (net.Conn, error) {
	s.m.Lock()
	defer s.m.Unlock()

	req := &DirectTcpipOpenRequest{
		OriginatorIPAddress: "127.0.0.1",
		OriginatorPort:      s.lPort,
		HostToConnect:       address,
		PortToConnect:       port,
	}
	s.lPort++

	msg := ssh.Marshal(req)

	lAddr, err := req.OriginatorAddr()
	if err != nil {
		return nil, err
	}
	rAddr, err := req.HostAddr()
	if err != nil {
		return nil, err
	}

	if channel, reqs, err := s.conn.OpenChannel("forwarded-tcpip", msg); err == nil {
		conn := NewTcpChannel(rAddr, lAddr, channel, reqs, s.log)
		go conn.logAndRejectRequests()
		return conn, nil
	} else {
		return nil, err
	}
}

func (s *SshConnection) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *SshConnection) ListOpenAddr() []string {
	s.m.Lock()
	defer s.m.Unlock()
	addrs := make([]string, len(s.openAddr))

	i := 0
	for k := range s.openAddr {
		addrs[i] = k
		i++
	}
	return addrs
}

func (s *SshConnection) handleConnection() {
	addr := s.RemoteAddr()
	defer s.log.Printf("Deconnected from %s\n", addr)
	for {
		select {
		case req, ok := <-s.reqs:
			if !ok {
				return
			}
			s.handleMainRequest(req)
		case channel, ok := <-s.chans:
			if !ok {
				return
			}
			s.handleNewChannel(channel)
		}
	}
}

func (s *SshConnection) handleMainRequest(req *ssh.Request) {
	s.log.Printf("Request: %s(%v) %v\n", req.Type, req.Payload, req.WantReply)

	if req.Type == "tcpip-forward" {
		data := &TcpIpForwardRequest{}
		if err := ssh.Unmarshal(req.Payload, data); err != nil {
			s.log.Printf("Error parsing tcpip-forward request: %s\n", err)
			req.Reply(false, []byte("Error parsing tcpip-forward request"))
		} else {
			s.addTcpIpForward(*data)
			req.Reply(true, nil)
		}
	} else if req.Type == "cancel-tcpip-forward" {
		data := &TcpIpForwardRequest{}
		if err := ssh.Unmarshal(req.Payload, data); err != nil {
			s.log.Printf("Error parsing cancel-tcpip-forward request: %s\n", err)
			req.Reply(false, []byte("Error parsing cancel-tcpip-forward request"))
		} else {
			s.removeTcpIpForward(*data)
			req.Reply(true, nil)
		}
	}
}

func (s *SshConnection) handleNewChannel(newChan ssh.NewChannel) {
	channelType := newChan.ChannelType()
	if channelType == "direct-tcpip" {
		reqData := &DirectTcpipOpenRequest{}
		if err := ssh.Unmarshal(newChan.ExtraData(), reqData); err != nil {
			s.rejectNewChannelWithError(newChan, ssh.ConnectionFailed, ErrParsing, err)
		} else {
			if err := s.createServiceConnection(newChan, reqData); err != nil {
				s.rejectNewChannelWithError(newChan, ssh.ConnectionFailed, ErrAddr, err)
			}
		}
	} else if channelType == "session" {
		if s.conn.User() != "root" {
			s.rejectNewChannelWithError(newChan, ssh.Prohibited, ErrUnauthorized, fmt.Errorf("Session refused for user %s", s.conn.User()))
		} else {
			go s.startSession(newChan)
		}
	} else {
		s.log.Printf("Received channel request `%s`", channelType)
		s.rejectNewChannelWithError(newChan, ssh.Prohibited, ErrUnauthorized, fmt.Errorf("Unknown channel type %s", channelType))
	}
}

func (s *SshConnection) createServiceConnection(newChan ssh.NewChannel, reqData *DirectTcpipOpenRequest) error {
	rAddr, err := reqData.OriginatorAddr()
	if err != nil {
		return err
	}
	lAddr, err := reqData.HostAddr()
	if err != nil {
		return err
	}

	s.server.handleServiceRequest(reqData.PortToConnect, s, func() (net.Conn, error) {
		channel, reqs, err := newChan.Accept()
		if err != nil {
			return nil, err
		}

		conn := NewTcpChannel(rAddr, lAddr, channel, reqs, s.log)
		go conn.logAndRejectRequests()

		return conn, nil
	})
	return nil
}

func (s *SshConnection) startSession(newChan ssh.NewChannel) {
	s.log.Println("Starting session")
	channel, reqs, err := newChan.Accept()
	if err != nil {
		s.log.Println("Error accepting session: ", err)
		return
	}
	defer channel.Close()

	var isShell bool
	var cmd struct {
		Line string
	}
	savedReqs := make([]*ssh.Request, 0)
ReqLoop:
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			savedReqs = append(savedReqs, req)
		case "shell", "exec":
			if req.Type == "shell" {
				isShell = true
			} else if err := ssh.Unmarshal(req.Payload, &cmd); err != nil {
				s.log.Printf("Error parsing command: %s\r\n", err)
				if req.WantReply {
					req.Reply(false, []byte(err.Error()))
				}
				return
			}
			defer func() {
				log.Println("fucking defer!!!!")
				if req.WantReply {
					req.Reply(true, nil)
				}
			}()
			break ReqLoop
		default:
			s.log.Printf("Unknown req %s(%v) %v\r\n", req.Type, req.Payload, req.WantReply)
			if req.WantReply {
				req.Reply(false, []byte("Unknown request type"))
			}
		}
	}

	// TODO close newReqs
	newReqs := make(chan *ssh.Request)

	go func() {
		for _, req := range savedReqs {
			newReqs <- req
		}

		for req := range reqs {
			newReqs <- req
		}
	}()

	if isShell {
		s.startShell(channel, newReqs)
	} else {
		var exit struct {
			Code int
		}
		exit.Code = s.execCmd(channel, newReqs, cmd.Line)
		channel.SendRequest("exit-status", false, ssh.Marshal(exit))
	}
}

func (s *SshConnection) startShell(channel ssh.Channel, reqs <-chan *ssh.Request) {
	NewTerminal(s.server, channel, reqs).Start()
}

func (s *SshConnection) execCmd(channel ssh.Channel, reqs <-chan *ssh.Request, cmd string) int {
	ctx := CmdContext{
		channel,
		s.log,
		s.server,
		nil,
	}
	return commands.Exec(ctx, cmd)
}

func (s *SshConnection) rejectNewChannelWithError(newChan ssh.NewChannel, reason ssh.RejectionReason, code SshError, err error) {
	channelType := newChan.ChannelType()
	switch code {
	case ErrParsing:
		s.log.Printf("Error parsing %s channel: %s\n", channelType, err)
		err = newChan.Reject(reason, "Error parsing request")
	default:
		s.log.Println(err)
		err = newChan.Reject(reason, "Error creating channel")
	}

	if err != nil {
		s.log.Printf("Error rejecting `%s` channel: %s\n", channelType, err)
	}
}

func (s *SshConnection) addTcpIpForward(info TcpIpForwardRequest) {
	s.m.Lock()
	defer s.m.Unlock()

	key := fmt.Sprintf("%s:%d", info.AddressToBind, info.PortNumberToBind)
	s.openAddr[key] = true
}

func (s *SshConnection) removeTcpIpForward(info TcpIpForwardRequest) {
	s.m.Lock()
	defer s.m.Unlock()

	key := fmt.Sprintf("%s:%d", info.AddressToBind, info.PortNumberToBind)
	delete(s.openAddr, key)
}
