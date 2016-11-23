package ssh

import (
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"time"
)

type TcpChannel struct {
	ssh.Channel
	reqs  <-chan *ssh.Request
	log   *log.Logger
	rAddr net.Addr
	lAddr net.Addr
}

func NewTcpChannel(rAddr, lAddr net.Addr, c ssh.Channel, reqs <-chan *ssh.Request, l *log.Logger) *TcpChannel {
	return &TcpChannel{
		Channel: c,
		reqs:    reqs,
		log:     l,
		rAddr:   rAddr,
		lAddr:   lAddr,
	}
}

func (c *TcpChannel) LocalAddr() net.Addr {
	return c.lAddr
}

func (c *TcpChannel) RemoteAddr() net.Addr {
	return c.rAddr
}

func (c *TcpChannel) logAndRejectRequests() {
	for req := range c.reqs {
		c.log.Printf("Rejecting request %s(%v) %v", req.Type, req.Payload, req.WantReply)
		if req.WantReply {
			req.Reply(false, []byte("Rejecting all request on TCP Channel"))
		}
	}
}

func (c *TcpChannel) SetDeadline(t time.Time) error {
	c.log.Println("TcpChannel.SetDeadline not implemented", t)
	return nil
}

func (c *TcpChannel) SetReadDeadline(t time.Time) error {
	c.log.Println("TcpChannel.SetReadDeadline not implemented", t)
	return nil
}

func (c *TcpChannel) SetWriteDeadline(t time.Time) error {
	c.log.Println("TcpChannel.SetWriteDeadline not implemented", t)
	return nil
}
