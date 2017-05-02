package ssh

import (
	"fmt"
	"github.com/JeanSebTr/SshBrain/actor"
	"github.com/JeanSebTr/SshBrain/domain"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"strings"
)

type SshServer struct {
	a        *actor.Actor
	config   *ssh.ServerConfig
	clients  map[string]*Node
	services map[uint32]func(*SshConnection, net.Conn)
}

type ConnectionFactory func() (net.Conn, error)
type ServiceCallback func(*SshConnection, net.Conn)

func NewServer(keyPath string, adminKeys []string) *SshServer {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {

			log.Printf("Server: %s User: %s\n", string(conn.ClientVersion()), conn.User())
			pubkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			log.Printf("PubKey: %s\n", pubkey)

			if conn.User() == "root" && !isAdminKey(pubkey, adminKeys) {
				return nil, fmt.Errorf("Not authorized")
			}

			return &ssh.Permissions{Extensions: map[string]string{
				"key-id": pubkey,
			}}, nil
		},
	}

	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic("Fail to load private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Fail to parse private key")
	}
	config.AddHostKey(private)

	server := &SshServer{
		a:        actor.NewActor(),
		config:   config,
		clients:  make(map[string]*Node),
		services: make(map[uint32]func(*SshConnection, net.Conn)),
	}

	return server
}

func isAdminKey(key string, adminKeys []string) bool {
	for _, adminKey := range adminKeys {
		if adminKey == key {
			return true
		}
	}
	return false
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

	if client.User() != "root" {
		s.a.Post(func() {
			mac := strings.ToUpper(client.User())
			s.clients[mac] = NewNode(client)
		})
	}

	go client.handleConnection()
}

func (s *SshServer) handleServiceRequest(port uint32, client *SshConnection, builder ConnectionFactory) error {
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

func (s *SshServer) Count() int {
	return len(s.clients)
}

func (s *SshServer) GetAll() []domain.Node {
	nodes := make([]domain.Node, 0, len(s.clients))

	for _, node := range s.clients {
		nodes = append(nodes, node)
	}

	return nodes
}

func (s *SshServer) GetById(id string) (domain.Node, error) {
	mac := strings.ToUpper(id)
	if node, ok := s.clients[mac]; ok {
		return node, nil
	} else {
		return nil, nil
	}
}
