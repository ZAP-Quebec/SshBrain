package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
)

type SshServer struct {
	m              sync.Mutex
	config         *ssh.ServerConfig
	clients        map[string]string
	clientsUpdates chan *SshConnection
	services       map[uint32]func(*SshConnection, net.Conn)
}

type ConnectionFactory func() (net.Conn, error)
type ServiceCallback func(*SshConnection, net.Conn)

func NewServer(key string) *SshServer {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {

			log.Printf("Server: %s User: %s\n", string(conn.ClientVersion()), conn.User())
			pubkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			log.Printf("PubKey: %s\n", pubkey)
			return &ssh.Permissions{Extensions: map[string]string{
				"key-id": pubkey,
			}}, nil
		},
	}

	privateBytes, err := ioutil.ReadFile(key)
	if err != nil {
		panic("Fail to load private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Fail to parse private key")
	}
	config.AddHostKey(private)

	server := &SshServer{
		config:         config,
		clients:        make(map[string]string),
		clientsUpdates: make(chan *SshConnection, 10),
		services:       make(map[uint32]func(*SshConnection, net.Conn)),
	}

	go server.manageServerStates()
	return server
}

func (s *SshServer) Listen(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	for {
		// Once a ServerConfig has been configured, connections can be accepted.
		conn, err := listener.Accept()
		if err != nil {
			// handle error
			log.Println("Accept error!", err)
			continue
		}

		log.Println("Got connection! ", conn.RemoteAddr())
		go s.handleClient(conn)
	}
}

func (s *SshServer) ExposeService(port uint32, handler func(*SshConnection, net.Conn)) {
	s.m.Lock()
	defer s.m.Unlock()

	if _, exists := s.services[port]; exists {
		panic(fmt.Errorf("Port %d is already exposed", port))
	}
	s.services[port] = handler
}

func (s *SshServer) handleClient(conn net.Conn) {
	sConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		// handle error
		log.Println("handshake error!", err)
		return
	}

	client := NewConnection(s, sConn, chans, reqs)

	s.clientsUpdates <- client

	go client.handleConnection()
}

func (s *SshServer) handleServiceRequest(port uint32, client *SshConnection, builder ConnectionFactory) error {
	s.m.Lock()
	defer s.m.Unlock()

	var (
		handler ServiceCallback
		exists  bool
	)
	if handler, exists = s.services[port]; !exists {
		return fmt.Errorf("No service on port %d", port)
	}

	conn, err := builder()
	if err != nil {
		return err
	}
	go handler(client, conn)
	return nil
}

func (s *SshServer) manageServerStates() {
	for _ = range s.clientsUpdates {
		//
	}
}
