package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/kyren223/eko/internal/utils/log"
)

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	log.Info("Accepted client: %v", conn.RemoteAddr().String())
	defer log.Info("Disconnecting client: %v", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	buffer := make([]byte, 1024)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		log.Error("Failed reading: %v", err)
		return
	}
	request := string(buffer[:bytesRead])
	log.Info("Read %v bytes: %v", bytesRead, request)

	response := []byte(fmt.Sprintf("Eko \"%v\"", request))
	bytesWritten, err := conn.Write(response)
	if err != nil {
		log.Error("Failed writing response: %v", err)
		return
	}
	log.Info("Written %v bytes", bytesWritten)
}

