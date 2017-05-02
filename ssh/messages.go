package ssh

import (
	"fmt"
	"net"
	"strings"
)

type Arguments []string

func (args Arguments) Where(filter func(string) bool) Arguments {
	res := make(Arguments, 0)
	for _, arg := range args {
		if filter(arg) {
			res = append(res, arg)
		}
	}
	return res
}

func (args Arguments) First(filter func(string) bool) (string, error) {
	res := args.Where(filter)
	if len(res) == 0 {
		return "", fmt.Errorf("No argument matching filter")
	}
	return res[0], nil
}

func (args Arguments) Single(filter func(string) bool) (string, error) {
	res := args.Where(filter)
	if len(res) != 1 {
		return "", fmt.Errorf("None or more than one argument matching filter")
	}
	return res[0], nil
}

func (args Arguments) String() string {
	return strings.Join(args, " ")
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
