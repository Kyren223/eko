package server

import (
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/kyren223/eko/internal/utils/log"
)

const PORT int = 7223

func Start() {
	server, err := NewServer(PORT)
	if err != nil {
		log.Error("Unable to start server: %v", err)
		return
	}

	var wg sync.WaitGroup
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go handleInterrupt(server, stopChan, &wg)

	server.Listen()
	wg.Wait()
}

func handleInterrupt(server *Server, stopChan <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	<-stopChan
	log.Info("Interrupt Occurred")
	log.Info("Shutting down server...")
	server.Close()
	log.Info("Waiting for all connections to close")
	server.Wait()
	log.Info("Server has been shutdown")
}

type Server struct {
	listener net.Listener
	wg       sync.WaitGroup
}

func NewServer(port int) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	log.Info("Created server on port %v", port)
	return &Server{listener, sync.WaitGroup{}}, nil
}

func (s *Server) Listen() {
	log.Info("Server started listening... %v", s.listener.Addr().String())
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			break
		}
		s.wg.Add(1)
		go handleClient(conn, &s.wg)
	}
}

// Stop stops the server. The blocked Listen call will be unlocked
func (s *Server) Close() {
	s.listener.Close()
}

// Wait blocks until all active connections to the server are done
func (s *Server) Wait() {
	s.wg.Wait()
}
