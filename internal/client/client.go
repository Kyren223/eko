package client

import (
	"net"

	"github.com/kyren223/eko/internal/utils/log"
)

func Run() {
	conn, err := net.Dial("tcp", ":7223")
	if err != nil {
		log.Error("Unable to establish connection with server: %v", err)
		return
	}
	defer conn.Close()
	log.Info("Established connection to server: %v", conn.RemoteAddr().String())

	message := "Hello, server!"
	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Error("Unable to send message: %v", err)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Error("Unable to read response: %v", err)
		return
	}

	log.Info("Received from server: %v", string(buffer[:n]))
}
