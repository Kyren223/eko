package server

import (
	"net"
	"sync"

	"github.com/kyren223/eko/internal/utils/log"
)

func handleClient(conn net.Conn, wg *sync.WaitGroup) {
	log.Info("Accepted client: %v", conn.RemoteAddr().String())
	defer log.Info("Disconnecting client: %v", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Error("Failed reading: %v", err)
		return
	}
	log.Info("Read %v bytes: %v", n, string(buffer[:n]))

	response := []byte("Server response")
	n, err = conn.Write(response)
	if err != nil {
		log.Error("Failed writing response: %v", err)
		return
	}
	log.Info("Written %v bytes", n)
}

