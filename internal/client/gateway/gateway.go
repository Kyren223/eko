package gateway

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

//go:embed server.crt
var certPEM []byte

var (
	tlsConfig *tls.Config

	asyncResponses []chan packet.Payload
	responsesMu    sync.Mutex

	framer  packet.PacketFramer
	conn    net.Conn
	writeMu sync.Mutex
)

type (
	ConnectionEstablished struct{}
	ConnectionFailed      error
	ConnectionLost        error
	ConnectionClosed      struct{}
)

func init() {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		log.Fatalln("failed to append server certificate")
	}

	tlsConfig = &tls.Config{
		RootCAs:    certPool,
		ServerName: "localhost",
	}
}

func Connect(ctx context.Context, privKey ed25519.PrivateKey) tea.Cmd {
	return func() tea.Msg {
		err := connect(ctx, privKey)
		if err != nil {
			return ConnectionFailed(err)
		}
		return ConnectionEstablished{}
	}
}

func connect(ctx context.Context, privKey ed25519.PrivateKey) error {
	assert.Assert(conn == nil, "cannot connect, connection is active")

	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	go func() {
		framer = packet.NewFramer()
		connection, err := tls.Dial("tcp4", ":7223", tlsConfig)
		if err != nil {
			errChan <- err
			return
		}
		log.Println("established connection with server")

		if err := handleAuth(ctx, privKey); err != nil {
			errChan <- err
			return
		}
		log.Println("successfully authenticated with server")
		connChan <- connection
	}()

	select {
	case connection := <-connChan:
		conn = connection
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	go readForever()
	go handlePacketStream()

	return nil
}

func handleAuth(ctx context.Context, privKey ed25519.PrivateKey) error {
	const nonceSize = 32
	challengeRequest := make([]byte, 1+nonceSize)

	deadline, _ := ctx.Deadline()
	err := conn.SetDeadline(deadline)
	assert.NoError(err, "setting deadline should not error")
	defer func() {
		err := conn.SetDeadline(time.Time{})
		assert.NoError(err, "unsetting deadline should not error")
	}()

	bytesRead := 0
	for bytesRead < 1+nonceSize {
		n, err := conn.Read(challengeRequest[bytesRead:])
		if err != nil {
			return err
		}
		bytesRead += n
	}

	assert.Assert(challengeRequest[0] == packet.VERSION, "client should always have the same version as the server")

	challengeResponse := make([]byte, 1+ed25519.PublicKeySize+ed25519.SignatureSize)
	challengeResponse[0] = packet.VERSION
	copy(challengeResponse[1:1+ed25519.PublicKeySize], privKey.Public().(ed25519.PublicKey))
	signedNonce := ed25519.Sign(privKey, challengeRequest[1:])
	n := copy(challengeResponse[1+ed25519.PublicKeySize:], signedNonce)
	assert.Assert(n == ed25519.SignatureSize, "copy should've copied the entire signature exactly")

	_, err = conn.Write(challengeResponse)
	if err != nil {
		return err
	}

	return nil
}

func readForever() {
	buffer := make([]byte, 512)
	for conn != nil {
		n, err := conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			onDisconnect(err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			cancel()
			onDisconnect(errors.New("framer blocked for more than a second, closing connection"))
			return
		}
		cancel()
		assert.NoError(err, "packets from server should always be correctly formatted")
	}
}

func handlePacketStream() {
	for {
		pkt, ok := <-framer.Out
		if !ok {
			break
		}

		payload, err := pkt.DecodedPayload()
		assert.NoError(err, "server should always provide a decodeable packet")

		if pkt.Type().IsPush() {
			log.Println("received streamed packet:", payload)
			ui.Program.Send(payload)
			continue
		}

		responsesMu.Lock()
		assert.Assert(len(asyncResponses) != 0, "there must always be at least 1 response waiting")
		responseChan := asyncResponses[0]
		copy(asyncResponses, asyncResponses[1:])
		asyncResponses = asyncResponses[:len(asyncResponses)-1]
		responsesMu.Unlock()

		go func() {
			responseChan <- payload
		}()
	}
}

func Disconnect() {
	assert.Assert(conn != nil, "cannot disconnect, connection is inactive")
	conn.Close()
}

func onDisconnect(err error) {
	conn.Close()
	conn = nil
	close(framer.Out)
	responsesMu.Lock()
	for _, responseChan := range asyncResponses {
		close(responseChan)
	}
	asyncResponses = nil
	responsesMu.Unlock()
	if err != nil {
		log.Println("connection lost:", err)
		ui.Program.Send(ConnectionLost(err))
	} else {
		log.Println("connection closed")
		ui.Program.Send(ConnectionClosed{})
	}
}

func Send(request packet.Payload) <-chan packet.Payload {
	responseChan := make(chan packet.Payload, 1)
	go func() {
		pkt := packet.NewPacket(packet.NewMsgPackEncoder(request))

		responsesMu.Lock()
		asyncResponses = append(asyncResponses, responseChan)
		responsesMu.Unlock()

		if conn == nil {
			log.Println("request send error:", "connection is closed")
			return
		}

		writeMu.Lock()
		_, err := pkt.Into(conn)
		writeMu.Unlock()
		if err != nil {
			log.Println("request send error:", err)
			return
		}
	}()
	return responseChan
}
