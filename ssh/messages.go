package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
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

type DirectTcpipOpenRequest struct {
	HostToConnect       string
	PortToConnect       uint32
	OriginatorIPAddress string
	OriginatorPort      uint32
}

type TcpIpForwardRequest struct {
	AddressToBind    string
	PortNumberToBind uint32
}

func (r DirectTcpipOpenRequest) OriginatorAddr() (net.Addr, error) {
	rAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", r.OriginatorIPAddress, r.OriginatorPort))
	if err != nil {
		return nil, fmt.Errorf("Invalid source address %s:%d %s\n", r.OriginatorIPAddress, r.OriginatorPort, err)
	} else {
		return rAddr, nil
	}
}

func (r DirectTcpipOpenRequest) HostAddr() (net.Addr, error) {
	lAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", r.HostToConnect, r.PortToConnect))
	if err != nil {
		return nil, fmt.Errorf("Invalid destination address %s:%d %s\n", r.HostToConnect, r.PortToConnect, err)
	} else {
		return lAddr, nil
	}
}
