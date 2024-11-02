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
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
)

//go:embed server.crt
var certPEM []byte

var (
	tlsConfig *tls.Config

	asyncResponses []chan packet.Payload
	responsesMu    sync.Mutex

	connection net.Conn
	connMu     sync.Mutex
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

func Connect(ctx context.Context, program *tea.Program, privKey ed25519.PrivateKey) {
	conn, err := tls.Dial("tcp4", ":7223", tlsConfig)
	if err != nil {
		assert.NoError(err, "TODO handle error")
	}
	log.Println("established connection with server")

	if err := handleAuth(ctx, conn, privKey); err != nil {
		assert.NoError(err, "TODO handle error")
	}
	log.Println("successfully authenticated with server")

	framer := packet.NewFramer()

	go func() {
		connection = conn
		handleConnection(ctx, conn, framer)
		close(framer.Out)
		conn.Close()
		connection = nil
	}()

	go handlePacketStream(framer, program)
}

func handleAuth(ctx context.Context, conn net.Conn, privKey ed25519.PrivateKey) error {
	const nonceSize = 32
	challengeRequest := make([]byte, 1+nonceSize)

	err := conn.SetDeadline(time.Now().Add(10 * time.Second))
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

func handleConnection(ctx context.Context, conn net.Conn, framer packet.PacketFramer) {
	buffer := make([]byte, 512)
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Second))
		assert.NoError(err, "setting a read deadline should not error")
		n, err := conn.Read(buffer)
		deadlineExceeded := errors.Is(err, os.ErrDeadlineExceeded)
		if err != nil && !deadlineExceeded {
			if !errors.Is(err, io.EOF) {
				log.Println("read error:", err)
			}
			break
		}

		if ctx.Err() != nil {
			log.Println("context error:", ctx.Err())
			break
		}

		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			log.Println("context error:", ctx.Err())
			break
		}
		assert.NoError(err, "packets from server should always be correct")
	}
}

func handlePacketStream(framer packet.PacketFramer, program *tea.Program) {
	for {
		pkt, ok := <-framer.Out
		if !ok {
			break
		}

		payload, err := pkt.DecodedPayload()
		assert.NoError(err, "server should always provide a decodeable packet")

		if pkt.Type().IsPush() {
			log.Println("received streamed packet:", payload)
			program.Send(payload)
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

func conn() net.Conn {
	return connection
}

func Send(request packet.Payload) <-chan packet.Payload {
	responseChan := make(chan packet.Payload)
	go func() {
		pkt := packet.NewPacket(packet.NewMsgPackEncoder(request))

		conn := conn()
		if conn == nil {
			log.Println("request send error:", "connection is closed")
			close(responseChan)
			return
		}

		connMu.Lock()
		errDeadline := conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		assert.NoError(errDeadline, "setting a write deadline should not error")
		_, err := pkt.Into(conn)
		errDeadline = conn.SetWriteDeadline(time.Time{})
		assert.NoError(errDeadline, "setting a write deadline should not error")
		connMu.Unlock()
		if err != nil {
			log.Println("request send error:", err)
			close(responseChan)
			return
		}

		responsesMu.Lock()
		asyncResponses = append(asyncResponses, responseChan)
		responsesMu.Unlock()

		time.Sleep(5 * time.Second)
		responsesMu.Lock()
		index := -1
		for i, ch := range asyncResponses {
			if ch == responseChan {
				index = i
			}
		}
		if index != -1 {
			copy(asyncResponses[index:], asyncResponses[index+1:])
			asyncResponses = asyncResponses[:len(asyncResponses)-1]
			close(responseChan)
		}
		responsesMu.Unlock()
	}()
	return responseChan
}
