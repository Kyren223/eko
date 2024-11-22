package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

//go:embed certs/server.crt
var certPEM []byte

//go:embed certs/server.key
var keyPEM []byte

var (
	nodeId    int64 = 0
	tlsConfig *tls.Config
)

func init() {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalln("error loading certificate:", err)
	}

	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

type server struct {
	node     *snowflake.Node
	Port     uint16
	sessions map[snowflake.ID]*session.Session
	sessMu   sync.RWMutex
}

// Creates a new server on the given port.
// Will generate a unique node ID automatically, will crash if there are no available IDs.
func NewServer(port uint16) server {
	assert.Assert(nodeId <= snowflake.NodeMax, "maximum amount of servers reached")
	node := snowflake.NewNode(nodeId)
	nodeId++

	return server{
		node:     node,
		Port:     port,
		sessions: map[snowflake.ID]*session.Session{},
	}
}

func (s *server) AddSession(session *session.Session) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.sessions[session.ID()] = session
}

func (s *server) RemoveSession(id snowflake.ID) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, id)
}

func (s *server) Session(id snowflake.ID) (*session.Session, bool) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

func (s *server) Node() *snowflake.Node {
	return s.node
}

// Starts listening and accepting clients on the server's port.
//
// The given context is used for cancellation,
// note that the server will wait for all active connections to close before
// returning, this is a blocking operation.
func (s *server) ListenAndServe(ctx context.Context) {
	listener, err := tls.Listen("tcp4", ":"+strconv.Itoa(int(s.Port)), tlsConfig)
	if err != nil {
		log.Fatalf("error starting server: %s", err)
	}

	assert.AddFlush(listener)
	defer listener.Close()
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	log.Printf("started listening on port %v...\n", s.Port)
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
			handleConnection(ctx, conn, s)
			wg.Done()
		}()
	}
	log.Printf("stopped listening on port %v\n", s.Port)

	log.Println("waiting for all active connections to close...")
	wg.Wait()
	log.Println("server shutdown complete")
}

func handleConnection(ctx context.Context, conn net.Conn, server *server) {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	assert.Assert(ok, "getting tcp address should never fail as we are using tcp connections")

	log.Println(addr, "accepted")

	nonce := [32]byte{}
	_, err := rand.Read(nonce[:])
	assert.NoError(err, "random should always produce a value")
	pubKey, err := handleAuth(conn, nonce[:])
	if err != nil {
		log.Println(addr, err)
		conn.Close()
		log.Println(addr, "disconnected")
		return
	}

	user, err := api.CreateOrGetUser(ctx, server.Node(), pubKey)
	if err != nil {
		log.Println(addr, "user creation/fetching error:", err)
		conn.Close()
		log.Println(addr, "disconnected")
		return
	}
	sess := session.NewSession(server, addr, user.ID, pubKey)
	server.AddSession(sess)
	ctx = session.NewContext(ctx, sess)
	framer := packet.NewFramer()

	// Write ID back, it's useful for the client to know, and signals successful authentication
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(user.ID))
	_, err = conn.Write(id[:])
	if err != nil {
		log.Println(addr, "failed to write user id")
		conn.Close()
		log.Println(addr, "disconnected")
		return
	}

	defer func() {
		conn.Close()
		close(framer.Out)
		server.RemoveSession(sess.ID())
		close(sess.WriteQueue)
		sess.WriteQueue = nil
		log.Println(addr, "disconnected")
	}()

	go func() {
		for {
			packet, ok := <-sess.WriteQueue
			if !ok {
				break
			}
			if packet.Type().IsPush() {
				log.Println(addr, "streaming packet:", packet)
			}
			if _, err := packet.Into(conn); err != nil {
				log.Println(addr, err)
				break
			}
		}
	}()

	go func() {
		for {
			request, ok := <-framer.Out
			if !ok {
				break
			}
			response := processPacket(ctx, request)
			if sess.WriteQueue == nil {
				break
			}
			sess.WriteQueue <- response
		}
	}()

	buffer := make([]byte, 512)
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Second))
		assert.NoError(err, "setting read deadline should not error")
		n, err := conn.Read(buffer)
		deadlineExceeded := errors.Is(err, os.ErrDeadlineExceeded)
		if err != nil && !deadlineExceeded {
			if !errors.Is(err, io.EOF) {
				log.Println(addr, "read error:", err)
			}
			break
		}

		if ctx.Err() != nil {
			log.Println(addr, ctx.Err())
			break
		}

		err = framer.Push(ctx, buffer[:n])
		if err != nil {
			if ctx.Err() != nil {
				log.Println(addr, ctx.Err())
			} else {
				payload := packet.ErrorMessage{Error: err.Error()}
				pkt := packet.NewPacket(packet.NewMsgPackEncoder(&payload))
				sess.WriteQueue <- pkt
			}
			break
		}
	}
}

func handleAuth(conn net.Conn, nonce []byte) (ed25519.PublicKey, error) {
	err := conn.SetDeadline(time.Now().Add(time.Second * 5))
	assert.NoError(err, "setting read deadline should not error")

	defer func() {
		err := conn.SetDeadline(time.Time{})
		assert.NoError(err, "unsetting read deadline should not error")
	}()

	challengePacket := make([]byte, len(nonce)+1)
	challengePacket[0] = packet.VERSION
	copy(challengePacket[1:], nonce)

	_, err = conn.Write(challengePacket)
	if err != nil {
		return nil, fmt.Errorf("error writing challenge: %w", err)
	}

	challengeResponsePacket := make([]byte, ed25519.PublicKeySize+ed25519.SignatureSize+1)
	bytesRead := 0
	for bytesRead < len(challengeResponsePacket) {
		n, err := conn.Read(challengeResponsePacket[bytesRead:])
		if err != nil {
			return nil, fmt.Errorf("error reading challenge response: %w", err)
		}
		bytesRead += n
	}

	if challengeResponsePacket[0] != packet.VERSION {
		return nil, fmt.Errorf("incompatible version: %v", challengeResponsePacket[0])
	}

	pubKey := ed25519.PublicKey(challengeResponsePacket[1 : 1+ed25519.PublicKeySize])
	signature := ed25519.PrivateKey(challengeResponsePacket[1+ed25519.PublicKeySize:])

	if ok := ed25519.Verify(pubKey, nonce, signature); !ok {
		return nil, errors.New("signature verification failed")
	}

	return pubKey, nil
}

func processPacket(ctx context.Context, pkt packet.Packet) packet.Packet {
	session, ok := session.FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")

	var response packet.Payload

	request, err := pkt.DecodedPayload()
	if err != nil {
		response = &packet.ErrorMessage{Error: "malformed payload"}
	} else {
		response = processRequest(ctx, request)
	}

	assert.NotNil(response, "response must always be assigned to")
	log.Println(session.Addr(), "sending", response.Type(), "response:", response)
	return packet.NewPacket(packet.NewMsgPackEncoder(response))
}

func processRequest(ctx context.Context, request packet.Payload) packet.Payload {
	session, ok := session.FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")
	log.Println(session.Addr(), "processing", request.Type(), "request:", request)

	// TODO: add a way to measure the time each request/response took and log it
	// Potentially even separate time for code vs DB operations
	switch request := request.(type) {
	case *packet.SendMessage:
		return timeout(20*time.Millisecond, api.SendMessage, ctx, request)
	case *packet.GetMessagesRange:
		return timeout(50*time.Millisecond, api.GetMessages, ctx, request)
	case *packet.GetUserByID:
		return timeout(50*time.Millisecond, api.GetUserById, ctx, request)
	default:
		return &packet.ErrorMessage{Error: "use of disallowed packet type for request"}
	}
}

func timeout[T packet.Payload](
	timeoutDuration time.Duration,
	apiRequest func(context.Context, T) packet.Payload,
	ctx context.Context, request T,
) packet.Payload {
	responseChan := make(chan packet.Payload)
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	go func() {
		responseChan <- apiRequest(ctx, request)
	}()

	select {
	case response := <-responseChan:
		return response
	case <-ctx.Done():
		sess, ok := session.FromContext(ctx)
		assert.Assert(ok, "session should exist")
		log.Println(sess.Addr(), "timeout of", request.Type(), "request")
		// TODO: consider if we want to say it's a timeout or be vague to mitigate DOS attacks
		return &packet.ErrorMessage{Error: "internal server error"}
	}
}
