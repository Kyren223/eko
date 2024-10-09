package server

import (
	"crypto/tls"
	_ "embed"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/kyren223/eko/internal/utils/log"
)

const port = 7223

//go:embed server.crt
var certPEM []byte

//go:embed server.key
var keyPEM []byte

func Start() {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatal("Error loading certificate: %s", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := tls.Listen("tcp", ":"+strconv.Itoa(port), tlsConfig)
	if err != nil {
		log.Fatal("Error starting listener: %s", err)
	}
	defer listener.Close()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go handleInterrupt(listener, signalChan)

	var wg sync.WaitGroup
	listen(listener, &wg)
	wg.Wait()
}

func handleInterrupt(listener net.Listener, stopChan <-chan os.Signal) {
	<-stopChan
	log.Info("Interrupt Signal")
	log.Info("Closing listener from receiving new connections")
	listener.Close()
}

func listen(listener net.Listener, wg *sync.WaitGroup) {
	log.Info("Started listening on port %v...", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warn("Failed to accept connection: %v", err)
			break
		}
		wg.Add(1)
		go handleConnection(conn, wg)
	}
	log.Info("Stopped listening on port %v...", port)
}
