package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/vmihailenco/msgpack/v5"
)

//go:embed server.crt
var certPEM []byte

var tlsConfig *tls.Config

func Run() {
	logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		log.Fatalln("failed to append server certificate")
	}

	tlsConfig = &tls.Config{
		RootCAs:    certPool,
		ServerName: "localhost",
	}

	log.Println("client started, waiting for user input...")
	startUI()

	// for {
	// 	fmt.Print("> ")
	// 	input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	// 	input = strings.TrimSpace(input)
	// 	if input == ":q" || input == "exit" || input == "quit" {
	// 		break
	// 	}
	// 	if input == "" {
	// 		continue
	// 	}
	// 	err := processRequest(input)
	// 	if err != nil {
	// 		log.Println(err)
	// 		fmt.Println(err)
	// 	}
	// }
}

func sendMessage(message string) error {
	request := packet.SendMessageMessage{Content: message}
	var response packet.EkoMessage
	if err := SendAndReceive(&request, &response); err != nil {
		return err
	}
	assert.Assert(response.Message == "Eko OK", "server should return an OK status")
	return nil
}

func getMessages() ([]string, error) {
	request := packet.GetMessagesMessage{}
	var response packet.MessagesMessage
	if err := SendAndReceive(&request, &response); err != nil {
		return nil, err
	}
	var messages []string
	for _, message := range response.Messages {
		messages = append(messages, message.Contents)
	}
	return messages, nil
}

func SendAndReceive(request packet.TypedMessage, response packet.TypedMessage) error {
	conn, err := tls.Dial("tcp4", ":7223", tlsConfig)
	if err != nil {
		return fmt.Errorf("error establishing connection with server: %v", err)
	}
	defer conn.Close()
	log.Println("established connection with server:", conn.RemoteAddr().String())

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
			if responsePacket.Type() != packet.PacketError {
				return fmt.Errorf("error decoding response: %v", err)
			}
			var errorResponse packet.ErrorMessage
			if err := responsePacket.DecodePayload(&errorResponse); err != nil {
				return fmt.Errorf("error decoding error packet: %w", err)
			}
			return fmt.Errorf("server error: %v", errorResponse.Error)
		}

	case err := <-outErr:
		return fmt.Errorf("error receiving response: %v", err)
	}

	return nil
}
