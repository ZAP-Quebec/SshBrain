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
	serverKey   string
	admins      = []string{
		// zap_rsa (jstremblay)
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC4roEPEt5d+GFJU7znMZJNaAB+iLOeiCwmN20YTwxBxCE8PcoxQXkeyx1HE64wsIzCrHXUz3cFUeqUP6ChmMe5KQ+NyOGKMHgmIGXjKUtP4w/dPEmu/h9IaOOTu7s8BWxltSYA8BdM+tswyheT8qrClPgp8QG+zBhgmUy3+l30CooCO6OlvYHs9z2KnnjWEgm2RA/SjZT/C/62z1eti549nFoV2qCBKeAASFV/WWOYg4OUKzvm2DVrNjNqfXNADBydPoxcdTIYbG/TybnnokcyCUrK61Wk6XjKZuixW0q7h52DoOpuw6ksDVbUG7GgnrMypENDZ0P/GWb+Dei2wDFL",
	}
)

func init() {
	flag.StringVar(&serverKey, "key", "", "SSH key to use for the server")
	flag.StringVar(&httpAddress, "http", "0.0.0.0:80", "TCP address for the Web server to listen")
	flag.StringVar(&sshAddress, "ssh", "0.0.0.0:22", "TCP address for the SSH server to listen")
}

func main() {
	flag.Parse()

	log.Println("Addresses", httpAddress, sshAddress)

	server := ssh.NewServer(serverKey, admins)

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
