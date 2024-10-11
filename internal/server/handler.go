package server

import (
	"fmt"
	"log"
	"net"
	"sync"
)

func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	log.Println("accepted client:", conn.RemoteAddr().String())
	defer log.Println("disconnected client:", conn.RemoteAddr().String())
	defer conn.Close()
	defer wg.Done()

	buffer := make([]byte, 1024)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		log.Println("failed reading request:", err)
		return
	}
	request := string(buffer[:bytesRead])
	log.Printf("Read %v bytes: %v\n", bytesRead, request)

	response := []byte(fmt.Sprintf("Eko \"%v\"", request))
	bytesWritten, err := conn.Write(response)
	if err != nil {
		log.Println("failed writing response:", err)
		return
	}
	log.Printf("written %v bytes\n", bytesWritten)
}
