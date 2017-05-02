package domain

import (
	"net"
	"time"
)

type Node interface {
	Id() string
	Address() string
	LastUpdate() time.Time
	NewSession(ch Channel, pty *PtyRequest) (Session, error)
	Dial(addr string, port uint32) (net.Conn, error)
}
