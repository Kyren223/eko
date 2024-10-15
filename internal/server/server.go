package server

import (
	"crypto/tls"
	_ "embed"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

const port = 7223

//go:embed server.crt
var certPEM []byte

//go:embed server.key
var keyPEM []byte

func Start() {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalln("error loading certificate:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	prepareConstPackets()

	listener, err := tls.Listen("tcp4", ":"+strconv.Itoa(port), tlsConfig)
	if err != nil {
		log.Fatalf("error starting server: %s", err)
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
	signal := <-stopChan
	log.Println("signal:", signal.String())
	listener.Close()
}

func listen(listener net.Listener, wg *sync.WaitGroup) {
	log.Printf("started listening on port %v...\n", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Println("error accepting connection:", err)
			}
			break
		}
		wg.Add(1)
		go handleConnection(conn, wg)
	}
	log.Printf("stopped listening on port %v...\n", port)
}

var unsupportedEncodingErrorPacket packet.Packet
var unsupportedTypeErrorPacket packet.Packet

func prepareConstPackets() {
	message := packet.ErrorMessage{Error: packet.PacketUnsupportedEncoding.Error()}
	encoder, err := packet.NewMsgPackEncoder(&message)
	assert.NoError(err, "constant packets should not error")
	unsupportedEncodingErrorPacket = packet.NewPacket(encoder)

	message = packet.ErrorMessage{Error: packet.PacketUnsupportedType.Error()}
	encoder, err = packet.NewMsgPackEncoder(&message)
	assert.NoError(err, "constant packets should not error")
	unsupportedTypeErrorPacket = packet.NewPacket(encoder)
}
