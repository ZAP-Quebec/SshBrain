package main

import (
	"flag"
	"github.com/JeanSebTr/SshBrain/ssh"
	"io"
	"log"
	"net"
)

var (
	httpAddress string
	sshAddress  string
)

func init() {
	flag.StringVar(&httpAddress, "http", "0.0.0.0:80", "TCP address for the Web server to listen")
	flag.StringVar(&sshAddress, "ssh", "0.0.0.0:22", "TCP address for the SSH server to listen")
}

func main() {
	flag.Parse()

	log.Println("Addresses", httpAddress, sshAddress)

	server := ssh.NewServer("/Users/jeansebtr/ZapQuebec/SshBrain/id_rsa")

	server.ExposeService(7, func(client *ssh.SshConnection, conn net.Conn) {
		log.Printf("[%s] Connection to echo service from %s\n", client.RemoteAddr(), conn.RemoteAddr().String())
		if _, err := io.Copy(conn, conn); err != nil {
			log.Printf("[%s] Error on echo service connection: %s\n", client.RemoteAddr(), err)
		} else {
			log.Printf("[%s] Connection to echo service closed.\n", client.RemoteAddr())
		}
	})

	server.Listen(sshAddress)
}
