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
		log:      log.New(os.Stderr, conn.RemoteAddr().String(), log.LstdFlags|log.LUTC|log.Llongfile),
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

func (s *SshConnection) Dial(address string) (net.Conn, error) {
	s.m.Lock()
	defer s.m.Unlock()

	req := &DirectTcpipOpenRequest{
		OriginatorIPAddress: "127.0.0.1",
		OriginatorPort:      s.lPort,
	}
	s.lPort++

	if _, err := fmt.Sscanf(address, "%s:%d", &req.HostToConnect, &req.PortToConnect); err != nil {
		return nil, err
	}

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
	//ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})

	// reqData := &TcpIpForwardRequest{}
	// if err := ssh.Unmarshal(req.Payload, reqData); err != nil {
	// 	log.Printf("Error parsing tcpip-forward request: %s\n", err)
	// } else {
	// 	log.Printf("tcpip-forward %v\n", reqData)
	// }

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
