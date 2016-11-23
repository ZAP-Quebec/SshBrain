package ssh

import (
	"fmt"
	"net"
)

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
