package ssh

import (
	"golang.org/x/crypto/ssh"
)

type Session struct {
	ssh *ssh.Session
}

func (s Session) Shell() error {
	if err := s.ssh.Shell(); err != nil {
		return err
	}
	return s.ssh.Wait()
}

func (s Session) Exec(cmd string) (int, error) {
	if err := s.ssh.Run(cmd); err != nil {
		return 126, err
	}
	if err := s.ssh.Wait(); err != nil {
		//
	}
	return 0, nil
}

func (s Session) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return s.ssh.SendRequest(name, wantReply, payload)
}
