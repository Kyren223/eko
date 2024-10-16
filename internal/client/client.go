package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kyren223/eko/internal/packet"
)

//go:embed server.crt
var certPEM []byte

func Run() {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		log.Fatalln("failed to append server certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: "localhost",
	}

	log.Println("client started, waiting for user input...")
	for {
		fmt.Print("> ")
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		input = strings.TrimSpace(input)
		if input == ":q" || input == "exit" || input == "quit" {
			break
		}
		err := processRequest(input, tlsConfig)
		if err != nil {
			log.Println(err)
		}
	}
}

func processRequest(input string, tlsConfig *tls.Config) error {
	conn, err := tls.Dial("tcp4", ":7223", tlsConfig)
	if err != nil {
		return fmt.Errorf("error establishing connection with server: %v", err)
	}
	defer conn.Close()

	log.Println("established connection with server:", conn.RemoteAddr().String())

	if input == "SHOW" {
		request := packet.GetMessagesMessage{}
		var response packet.MessagesMessage
		if err := SendAndReceive(conn, &request, &response); err != nil {
			return err
		}
		log.Println("server response:", response.Messages)
	} else {
		request := packet.SendMessageMessage{Content: input}
		var response packet.EkoMessage
		if err := SendAndReceive(conn, &request, &response); err != nil {
			return err
		}
		log.Println("server response:", response.Message)
	}

	return nil
}

func SendAndReceive(conn net.Conn, request packet.TypedMessage, response packet.TypedMessage) error {
	encoder, err := packet.NewMsgPackEncoder(request)
	if err != nil {
		return fmt.Errorf("error encoding request: %v", err)
	}
	requestPacket := packet.NewPacket(encoder)
	err = requestPacket.Into(conn)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	log.Println("sent request to server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, outErr := packet.RunFramer(ctx, conn)

	select {
	case responsePacket := <-out:
		if err := responsePacket.DecodePayload(response); err != nil {
			return fmt.Errorf("error decoding response: %v", err)
		}

	case err := <-outErr:
		return fmt.Errorf("error receiving response: %v", err)
	}

	return nil
}
