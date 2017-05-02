package ssh

import (
	"github.com/JeanSebTr/SshBrain/actor"
	"github.com/JeanSebTr/SshBrain/domain"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"time"
)

type Node struct {
	a            *actor.Actor
	c            *SshConnection
	t            time.Time
	log          *log.Logger
	activeClient *ssh.Client
}

func NewNode(c *SshConnection) *Node {
	return &Node{
		a:   actor.NewActor(),
		c:   c,
		t:   time.Now(),
		log: c.log,
	}
}

func (n *Node) Id() string {
	return n.c.User()
}

func (n *Node) Address() string {
	return n.c.RemoteAddr()
}

func (n *Node) LastUpdate() time.Time {
	return n.t
}

func (n *Node) NewSession(ch domain.Channel, pty *domain.PtyRequest) (sess domain.Session, err error) {
	sess, err = nil, nil
	n.a.Run(func() {
		var client *ssh.Client
		var session *ssh.Session
		client, err = n.getSshClient()
		if err != nil {
			return
		}

		session, err = client.NewSession()
		if err != nil {
			log.Println("client.NewSession: ", err)
			return
		}

		if pty != nil {
			session.SendRequest("pty-req", false, ssh.Marshal(pty))
		}

		session.Stdin = ch
		session.Stderr = ch.Stderr()
		session.Stdout = ch

		sess = Session{session}
	})
	return
}

func (n *Node) Dial(addr string, port uint32) (net.Conn, error) {
	return n.c.Dial(addr, port)
}

func (n *Node) getSshClient() (*ssh.Client, error) {
	if n.activeClient != nil {
		return n.activeClient, nil
	}

	nConn, err := n.c.Dial("127.0.0.1", 22)
	if err != nil {
		log.Println("c.Dial: ", err)
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (string, error) {
				return os.Getenv("COMMON_PASS"), nil
			}),
		},
	}
	sConn, chans, reqs, err := ssh.NewClientConn(nConn, n.c.RemoteAddr(), config)
	if err != nil {
		defer nConn.Close()
		log.Println("ssh.NewClientConn: ", err)
		return nil, err
	}

	go func() {
		err := sConn.Wait()
		n.log.Printf("Reverse connection closed: %s\r\n", err)
		n.a.Run(func() {
			n.activeClient = nil
		})
	}()

	n.activeClient = ssh.NewClient(sConn, chans, reqs)
	return n.activeClient, nil
}
