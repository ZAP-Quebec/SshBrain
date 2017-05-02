package ssh

import (
	"fmt"
	"github.com/JeanSebTr/SshBrain/domain"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"os"
	"strings"
)

type TerminalSession struct {
	ssh.Channel
	server  *SshServer
	reqs    <-chan *ssh.Request
	term    *terminal.Terminal
	ptyInfo *domain.PtyRequest
}

func NewTerminal(server *SshServer, channel ssh.Channel, reqs <-chan *ssh.Request) *TerminalSession {
	t := &TerminalSession{
		Channel: channel,
		server:  server,
		reqs:    reqs,
	}
	t.term = terminal.NewTerminal(t.Channel, "root@brain > ")

	t.term.AutoCompleteCallback = t.autoCompleteCallback

	go func() {
		for req := range reqs {
			t.ptyInfo = &domain.PtyRequest{}
			ssh.Unmarshal(req.Payload, t.ptyInfo)
			t.term.SetSize(int(t.ptyInfo.CharWidth), int(t.ptyInfo.CharHeight))
		}
	}()

	return t
}

func (t *TerminalSession) Start() {
	for {
		line, err := t.term.ReadLine()
		if err == io.EOF {
			t.Channel.Close()
			return
		} else if err != nil {
			log.Printf("Error reading cmd %s\r\n", err)
			fmt.Fprintf(t.term, "Error reading cmd %s\r\n", err)
		} else if strings.Trim(line, " \t") != "" {
			commands.Exec(CmdContext{
				Channel: t.Channel,
				Log:     log.New(os.Stderr, "Terminal", log.LstdFlags|log.Lshortfile),
				Manager: t.server,
				Pty:     t.ptyInfo,
			}, line)
		}
	}
}

func (t *TerminalSession) autoCompleteCallback(line string, pos int, key rune) (string, int, bool) {
	if key == 9 {
		//
	}
	return line, pos, false
}
