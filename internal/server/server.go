package server

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

//go:embed server.crt
var certPEM []byte

//go:embed server.key
var keyPEM []byte

var (
	nodeId    int64 = 0
	tlsConfig *tls.Config

	unsupportedEncodingErrorPacket packet.Packet
	unsupportedTypeErrorPacket     packet.Packet
)

var ErrClosedNilListener error = errors.New("server: close on nil listener")

func init() {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalln("error loading certificate:", err)
	}

	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	message := packet.ErrorMessage{Error: packet.PacketUnsupportedEncoding.Error()}
	encoder, err := packet.NewMsgPackEncoder(&message)
	assert.NoError(err, "constant packets should not error")
	unsupportedEncodingErrorPacket = packet.NewPacket(encoder)

	message = packet.ErrorMessage{Error: packet.PacketUnsupportedType.Error()}
	encoder, err = packet.NewMsgPackEncoder(&message)
	assert.NoError(err, "constant packets should not error")
	unsupportedTypeErrorPacket = packet.NewPacket(encoder)
}

type server struct {
	node *snowflake.Node
	port uint16
}

func NewServer(port uint16) server {
	assert.Assert(nodeId <= snowflake.NodeMax, "maximum amount of servers reached %v", snowflake.NodeMax)
	node := snowflake.NewNode(nodeId)
	nodeId++

	return server{
		node: node,
		port: port,
	}
}

func (s *server) ListenAndServe(ctx context.Context) {
	listener, err := tls.Listen("tcp4", ":"+strconv.Itoa(int(s.port)), tlsConfig)
	if err != nil {
		log.Fatalf("error starting server: %s", err)
	}
	defer listener.Close()
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	log.Printf("started listening on port %v...\n", s.port)
	var wg sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Println("error accepting connection:", err)
			}
			break
		}
		wg.Add(1)
		go func() {
			handleConnection(ctx, conn)
			wg.Done()
		}()
	}
	log.Printf("stopped listening on port %v\n", s.port)

	log.Println("waiting for all active connections to close...")
	wg.Wait()
	log.Println("server shutdown complete")
}

func handleConnection(ctx context.Context, conn net.Conn) {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	assert.Assert(ok, "getting tcp address should never fail as we are using tcp connections")

	writeQueue := make(chan packet.Packet, 10)
	session := newSession(addr, writeQueue)
	nonce := session.Challenge()

	ctx = newContext(ctx, session)
	framer := packet.NewFramer(ctx)

	log.Println(addr, "accepted")

	defer func() {
		conn.Close()
		log.Println(addr, "disconnected")
	}()

	go func() {
		var mu sync.Mutex
		for {
			packet, ok := <-writeQueue
			if !ok {
				break
			}
			mu.Lock()
			packet.Into(conn)
			mu.Unlock()
		}
	}()

	go func() {
		for {
			request, ok := <-framer.Out
			if !ok {
				break
			}
			response := processPacket(ctx, request)
			writeQueue <- response
		}
	}()

	buffer := make([]byte, 512)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println(addr, "read error:", err)
			}
			break
		}

		err = framer.Push(ctx, buffer[:n])
		if err != nil {
			// Wrap err and send to client then break
		}
	}
}

func _handleConnection(ctx context.Context, conn net.Conn) {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	assert.Assert(ok, "getting tcp address should be valid")

	log.Println(addr, "accepted")
	defer log.Println(addr, "disconnected")
	defer conn.Close()

	// ctx = newContext(ctx, addr)
	// // TODO: consider adding timeout/deadline to ctx?

	out, outErr := packet.RunFramer(ctx, conn)
outer:
	for {
		select {
		case packet, ok := <-out:
			if !ok {
				break outer
			}
			log.Printf("client %v: request packet: %v\n", conn.RemoteAddr().String(), packet)
			responsePacket, err := handlePacket(packet)
			log.Printf("client %v: response packet: %v\n", conn.RemoteAddr().String(), responsePacket)
			if err != nil {
				log.Printf("client %v: error processing request: %v\n", conn.RemoteAddr().String(), err)
				break outer
			}
			_, err = responsePacket.Into(conn)
			if err != nil {
				log.Printf("client %v: error writing packet: %v\n", conn.RemoteAddr().String(), err)
				break outer
			}

		case err := <-outErr:
			if err == packet.PacketUnsupportedEncoding {
				_, err := unsupportedEncodingErrorPacket.Into(conn)
				log.Printf("client %v: error writing unsupported encoding packet: %v\n", conn.RemoteAddr().String(), err)
			} else if err == packet.PacketUnsupportedType {
				_, err := unsupportedTypeErrorPacket.Into(conn)
				log.Printf("client %v: error writing unsupported type packet: %v\n", conn.RemoteAddr().String(), err)
			} else if err != nil {
				log.Printf("client %v: internal error: %v\n", conn.RemoteAddr().String(), err)
			}
			break outer

		case <-ctx.Done():
			log.Printf("client %v: %v\n", conn.RemoteAddr().String(), ctx.Err())
			break outer
		}
	}
}

type Session struct {
	Addr       *net.TCPAddr
	WriteQueue <-chan packet.Packet

	mu         sync.Mutex
	challenge   []byte
	issuedTime time.Time
}

func newSession(addr *net.TCPAddr, writeQueue <-chan packet.Packet) *Session {
	session := &Session{
		Addr:       addr,
		WriteQueue: writeQueue,
		challenge:  make([]byte, 32), // Recommended nonce size
	}
	session.Challenge() // Make sure an initial nonce is generated
	return session
}

func (s *Session) Challenge() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.issuedTime) > time.Minute {
		s.issuedTime = time.Now()
		_, err := rand.Read(s.challenge)
		assert.NoError(err, "random should always produce a value")
	}
	return s.challenge
}

type key struct{}

var sessKey key

func newContext(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessKey, sess)
}

func FromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessKey).(*Session)
	return sess, ok
}

// TODO: Move everything below this to somewhere else

func handlePacket(pkt packet.Packet) (packet.Packet, error) {
	var response packet.TypedMessage
	switch pkt.Type() {
	case packet.PacketEko:
		var request packet.EkoMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		response = &packet.EkoMessage{Message: "Eko \"" + request.Message + "\""}
	case packet.PacketSendMessage:
		var request packet.SendMessageMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		content := strings.TrimSpace(request.Content)
		if content == "" {
			response = &packet.ErrorMessage{Error: "content must not be blank"}
			break
		}

		message := data.Message{
			Id:          node.Generate(),
			SenderId:    node.Generate(),
			FrequencyId: node.Generate(),
			NetworkId:   node.Generate(),
			Contents:    content,
		}
		messages = append(messages, message)

		response = &packet.EkoMessage{Message: "Eko OK"}
	case packet.PacketGetMessages:
		var request packet.GetMessagesMessage
		if err := pkt.DecodePayload(&request); err != nil {
			return packet.Packet{}, fmt.Errorf("decode error: %v", err)
		}

		response = &packet.MessagesMessage{Messages: messages}
	default:
		return packet.Packet{}, errors.New("TODO: not implemented yet")
	}

	assert.NotNil(response, "response must always be set")
	encoder, err := packet.NewMsgPackEncoder(response)
	if err != nil {
		return packet.Packet{}, fmt.Errorf("encode error: %v", err)
	}
	return packet.NewPacket(encoder), nil
}

var (
	node     = snowflake.NewNode(1)
	messages []data.Message
)
