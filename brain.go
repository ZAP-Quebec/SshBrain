package main

import (
	"fmt"
	"github.com/Unknwon/com"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func handleServerConn(keyID string, chans <-chan ssh.NewChannel) {
	for newChan := range chans {

		fmt.Printf("New Channel [%s]\n", newChan.ChannelType())
		go handleChannel(newChan)
	}
}

func handleChannel(newChan ssh.NewChannel) {
	if newChan.ChannelType() == "direct-tcpip" {
		reqData := &DirectTcpipOpenRequest{}
		if err := ssh.Unmarshal(newChan.ExtraData(), reqData); err != nil {
			fmt.Printf("Error parsing direct-tcpip channel: %s\n", err)
		} else {
			fmt.Printf("direct-tcpip channel: %v\n", reqData)
		}
		return
	}
	if newChan.ChannelType() != "session" {
		newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	ch, reqs, err := newChan.Accept()
	if err != nil {
		// handle error
		return
	}

	go func(in <-chan *ssh.Request) {
		defer ch.Close()
		for req := range in {

			fmt.Printf("Req : %s >>> %s\n", req.Type, string(req.Payload))

			//ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
		}
	}(reqs)
}

func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}

func main() {

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			fmt.Printf("Server: %s User: %s\n", string(conn.ClientVersion()), conn.User())
			fmt.Printf("PubKey: %s\n", strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
			return &ssh.Permissions{Extensions: map[string]string{"key-id": ""}}, nil
		},
	}

	// config.Ciphers = []string{
	// 	"aes128-ctr",
	// 	"aes256-ctr",
	// 	"aes128-cbc",
	// 	"aes256-cbc",
	// 	"twofish256-cbc",
	// 	"twofish-cbc",
	// 	"twofish128-cbc",
	// 	"3des-ctr",
	// 	"3des-cbc",
	// }

	// config.KeyExchanges = []string{
	// 	"diffie-hellman-group14-sha1",
	// 	"diffie-hellman-group1-sha1",
	// }

	//fmt.Println(config.KeyExchanges)

	keyPath := "/Users/jeansebtr/ZapQuebec/SshBrain/id_rsa"
	if keyExists, _ := exists(keyPath); !keyExists {
		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)
		_, stderr, err := com.ExecCmd("ssh-keygen", "-f", keyPath, "-t", "rsa", "-N", "")
		if err != nil {
			panic(fmt.Sprintf("Fail to generate private key: %v - %s", err, stderr))
		}
		fmt.Printf("New private key is generateed: %s\n", keyPath)
	}

	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic("Fail to load private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Fail to parse private key")
	}
	config.AddHostKey(private)

	listener, err := net.Listen("tcp", "0.0.0.0:2222")
	if err != nil {
		panic(err)
	}

	for {
		// Once a ServerConfig has been configured, connections can be accepted.
		conn, err := listener.Accept()
		fmt.Println("Got connection!")
		if err != nil {
			// handle error
			fmt.Println("Accept error!", err)
			continue
		}

		go handleClientConn(conn, config)
	}
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

func handleClientConn(conn net.Conn, config *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming net.Conn.
	sConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		// handle error
		fmt.Println("handshake error!", err)
		return
	}

	// The incoming Request channel must be serviced.
	go handleMainRequest(sConn, reqs)
	go handleServerConn(sConn.Permissions.Extensions["key-id"], chans)
}

func handleMainRequest(sConn *ssh.ServerConn, reqs <-chan *ssh.Request) {
	for req := range reqs {

		fmt.Printf("MainReq(%s) : %s >>> %v\n", req.WantReply, req.Type, req.Payload)

		req.Reply(true, nil)
		//ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})

		if req.Type == "tcpip-forward" {

			reqData := &TcpIpForwardRequest{}
			if err := ssh.Unmarshal(req.Payload, reqData); err != nil {
				fmt.Printf("Error parsing tcpip-forward request: %s\n", err)
			} else {
				fmt.Printf("tcpip-forward %v\n", reqData)
			}

			// msg := ssh.Marshal(DirectTcpipOpenRequest{
			// 	// initialWindowSize:   64 * (1 << 15),
			// 	// maximumPacketSize:   1 << 15,
			// 	HostToConnect:       "10.10.0.230",
			// 	PortToConnect:       2222,
			// 	OriginatorIPAddress: "127.0.0.1",
			// 	OriginatorPort:      22,
			// })

			// fmt.Printf("Msg: %v\n", msg)
			// if _, _, err := sConn.OpenChannel("forwarded-tcpip", msg); err == nil {
			// 	//handleChannel(chann)
			// 	fmt.Println("Opened direct-tcpip!! :D")
			// } else {
			// 	fmt.Println("direct-tcpip: ", err)
			// }
		}
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
