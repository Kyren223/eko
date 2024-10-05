package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kyren223/eko/internal/utils/log"
)

func Run() {
	log.SetLevel(log.LevelDebug)
	log.Info("Client started, waiting for user input...")
	for {
		fmt.Print("> ")
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		input = strings.TrimSpace(input)
		if input == ":q" || input == "exit" || input == "quit" {
			break
		}
		log.Debug("Input: %v", input)
		err := processRequest(input)
		if err != nil {
			log.Error("%v", err)
		}
	}
}

func processRequest(request string) error {
	conn, err := net.Dial("tcp", ":7223")
	if err != nil {
		return fmt.Errorf("Unable to establish connection with server: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second))
	log.Info("Established connection to server: %v", conn.RemoteAddr().String())

	_, err = conn.Write([]byte(request))
	if err != nil {
		return fmt.Errorf("Unable to send request: %v", err)
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("Unable to receive response: %v", err)
	}

	log.Info("Response from server: %v", string(buffer[:n]))
	return nil
}
