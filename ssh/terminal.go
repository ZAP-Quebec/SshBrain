package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"strings"
)

type TerminalSession struct {
	ssh.Channel
	server  *SshServer
	reqs    <-chan *ssh.Request
	term    *terminal.Terminal
	ptyInfo *PtyRequest
}

type Cmd struct {
	desc string
	cb   func(*TerminalSession, []string)
}

var commands map[string]Cmd

func init() {
	commands = map[string]Cmd{
		"help": Cmd{"This help text", func(t *TerminalSession, _ []string) {
			for name, cmd := range commands {
				fmt.Fprintf(t, "%s\t%s\r\n", name, cmd.desc)
			}
		}},
		"devices": Cmd{"List connected devices", func(t *TerminalSession, _ []string) {
			fmt.Fprintln(t, "Addr\tServices\r")
			for _, client := range t.server.ListClients() {
				fmt.Fprintf(t, "%s\t%v\r\n", client.RemoteAddr(), client.ListOpenAddr())
			}
		}},
		"connect": Cmd{"Establish a SSH connection to a device", func(t *TerminalSession, args []string) {
			log.Printf("Trying to connect to %v\n", args)
			if len(args) < 1 {
				fmt.Fprintln(t, "Missing client ID\r")
				return
			}

			for _, client := range t.server.ListClients() {
				if client.RemoteAddr() == args[0] {
					connected, err := t.connectToClient(client)
					if !connected && err != nil {
						fmt.Fprintf(t, "Could not connect to %s : %s\r\n", client.RemoteAddr(), err)
					} else if err != nil {
						fmt.Fprintf(t, "Connection error with %s : %s\r\n", client.RemoteAddr(), err)
					} else {
						fmt.Fprintf(t, "Connection closed from %s\r\n", args[0])
					}
					return
				}
			}
			fmt.Fprintf(t, "Could not find client %s\r\n", args[0])
		}},
	}
}

func NewTerminal(server *SshServer, channel ssh.Channel, reqs <-chan *ssh.Request) *TerminalSession {
	t := &TerminalSession{
		Channel: channel,
		server:  server,
		reqs:    reqs,
	}
	t.term = terminal.NewTerminal(t, "root@brain > ")

	t.term.AutoCompleteCallback = t.autoCompleteCallback

	return t
}

func (t *TerminalSession) Start() {
	for req := range t.reqs {
		ok, payload := t.handleChannelRequest(req)
		if req.WantReply {
			req.Reply(ok, payload)
		}
	}
}

func (t *TerminalSession) runShell() {
	for {
		line, err := t.term.ReadLine()
		if err == io.EOF {
			t.Channel.Close()
			return
		} else if err != nil {
			fmt.Fprintf(t, "Error reading cmd %s\r\n", err)
		} else {
			t.execCommand(line, true)
		}
	}
}

func (t *TerminalSession) autoCompleteCallback(line string, pos int, key rune) (string, int, bool) {
	if key == 9 {
		//
	}
	return line, pos, false
}

func (t *TerminalSession) handleChannelRequest(req *ssh.Request) (bool, []byte) {
	if req.Type == "pty-req" {
		ptyInfo := &PtyRequest{}
		if err := ssh.Unmarshal(req.Payload, ptyInfo); err != nil {
			log.Printf("Error parsing pty-req(%v) %s", req.Payload, err)
			return false, []byte("Parsing error")
		} else {
			t.ptyInfo = ptyInfo
			t.term.SetSize(int(t.ptyInfo.CharWidth), int(t.ptyInfo.CharWidth))
			return true, nil
		}
	} else if req.Type == "shell" {
		go t.runShell()
		return true, nil
	} else if req.Type == "exec" {
		var cmd struct {
			Line string
		}
		if err := ssh.Unmarshal(req.Payload, &cmd); err != nil {
			log.Printf("Error parsing exec(%v) %s", req.Payload, err)
			return false, []byte("Parsing error")
		} else {
			go t.execCommand(cmd.Line, false)
			return true, nil
		}
	}
	log.Printf("Unknown req %s(%v) %v\r\n", req.Type, req.Payload, req.WantReply)
	return false, []byte("Unknown request type")
}

func (t *TerminalSession) execCommand(line string, isPrompt bool) {
	args := strings.Split(line, " ")
	if cmd, exists := commands[args[0]]; exists {
		cmd.cb(t, args[1:])
	}

	if !isPrompt {
		t.Channel.Close()
	}
}

func (t *TerminalSession) connectToClient(c *SshConnection) (connected bool, err error) {
	nConn, err := c.Dial("127.0.0.1", 22)
	if err != nil {
		log.Println("c.Dial: ", err)
		return false, err
	}
	defer nConn.Close()

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (string, error) {
				return t.term.ReadPassword("password: ")
			}),
		},
	}
	sConn, chans, reqs, err := ssh.NewClientConn(nConn, c.RemoteAddr(), config)
	if err != nil {
		log.Println("ssh.NewClientConn: ", err)
		return false, err
	}
	defer sConn.Close()

	client := ssh.NewClient(sConn, chans, reqs)
	session, err := client.NewSession()
	if err != nil {
		log.Println("client.NewSession: ", err)
		return true, err
	}
	defer session.Close()

	session.Stdin = t.Channel
	session.Stderr = t.Channel.Stderr()
	session.Stdout = t.Channel

	if t.ptyInfo != nil {
		if err := session.RequestPty(t.ptyInfo.TermEnv, int(t.ptyInfo.CharHeight), int(t.ptyInfo.CharWidth), t.ptyInfo.TermModes.Parse()); err != nil {
			log.Println("session.RequestPty: ", err)
			return true, err
		}
	}

	if err := session.Shell(); err != nil {
		log.Println("session.Shell: ", err)
		return true, err
	}

	if err := session.Wait(); err != nil {
		log.Println("session.Wait: ", err)
		return true, err
	}

	return true, nil
}
