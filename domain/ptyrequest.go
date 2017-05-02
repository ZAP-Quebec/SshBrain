package domain

import (
	"golang.org/x/crypto/ssh"
)

type PtyRequest struct {
	TermEnv    string        // TERM environment variable value (e.g., vt100)
	CharWidth  uint32        // terminal width, characters (e.g., 80)
	CharHeight uint32        // terminal height, rows (e.g., 24)
	PxWidth    uint32        // terminal width, pixels (e.g., 640)
	PxHeight   uint32        // terminal height, pixels (e.g., 480)
	TermModes  TerminalModes // encoded terminal modes
}

type TerminalModes string

func (m TerminalModes) Parse() ssh.TerminalModes {
	return make(map[uint8]uint32)
}
