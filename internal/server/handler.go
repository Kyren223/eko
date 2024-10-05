package server

import (
	"net"
	"sync"

	"github.com/kyren223/eko/internal/utils/log"
)

func handleClient(conn net.Conn, wg *sync.WaitGroup) {
	log.Info("Handling client... %v", conn.RemoteAddr().String())
	defer log.Info("Disconnecting client: %v", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	var request []byte
	n, err := conn.Read(request)
	if err != nil {
		log.Error("Failed reading: %v", err)
		return
	}
	log.Info("Read %v bytes: %v", n, string(request))

	response := []byte("Server response")
	n, err = conn.Write(response)
	if err != nil {
		log.Error("Failed writing response: %v", err)
		return
	}
	log.Info("Responded successfully")
}

