package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

//go:embed server.crt
var certPEM []byte

func Run() {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		log.Fatalln("failed to append server certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs: certPool,
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

func processRequest(request string, tlsConfig *tls.Config) error {
	conn, err := tls.Dial("tcp4", ":7223", tlsConfig)
	if err != nil {
		return fmt.Errorf("error establishing connection with server: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second))
	log.Println("established connection with server:", conn.RemoteAddr().String())

	_, err = conn.Write([]byte(request))
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("error receiving response: %v", err)
	}

	log.Println("server response:", string(buffer[:n]))
	return nil
}
