package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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
	unsupportedEncodingErrorPacket = packet.NewPacket(packet.NewMsgPackEncoder(&message))

	message = packet.ErrorMessage{Error: packet.PacketUnsupportedType.Error()}
	unsupportedTypeErrorPacket = packet.NewPacket(packet.NewMsgPackEncoder(&message))
}

type server struct {
	Node     *snowflake.Node
	Port     uint16
	sessions map[snowflake.ID]*Session
	sessMu   sync.RWMutex
}

func NewServer(port uint16) server {
	assert.Assert(nodeId <= snowflake.NodeMax, "maximum amount of servers reached")
	node := snowflake.NewNode(nodeId)
	nodeId++

	return server{
		Node:     node,
		Port:     port,
		sessions: map[snowflake.ID]*Session{},
	}
}

func (s *server) AddSession(session *Session) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.sessions[session.ID] = session
}

func (s *server) RemoveSession(id snowflake.ID) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, id)
}

func (s *server) Session(id snowflake.ID) (*Session, bool) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

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

	// TODO: replace this with DB query for id
	id := server.Node.Generate()
	session := newSession(server, addr, id, pubKey)
	server.AddSession(session)
	ctx = newContext(ctx, session)
	framer := packet.NewFramer(ctx)

	defer func() {
		conn.Close()
		close(framer.Out)
		server.RemoveSession(session.ID)
		close(session.WriteQueue)
		log.Println(addr, "disconnected")
	}()

	go func() {
		for {
			packet, ok := <-session.WriteQueue
			if !ok {
				break
			}
			if _, err := packet.Into(conn); err != nil {
				log.Println(addr, err)
				break
			}
		}
		session.WriteQueue = nil
	}()

	go func() {
		for {
			request, ok := <-framer.Out
			if !ok {
				break
			}
			response := processPacket(ctx, request)
			if session.WriteQueue == nil {
				break
			}
			session.WriteQueue <- response
		}
	}()

	buffer := make([]byte, 512)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Second))
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
				// TODO: Wrap err and send to client then break
			}
			break
		}
	}
}

func handleAuth(conn net.Conn, nonce []byte) (ed25519.PublicKey, error) {
	conn.SetDeadline(time.Now().Add(time.Second * 5))

	challengePacket := make([]byte, len(nonce)+1)
	challengePacket[0] = packet.VERSION
	copy(challengePacket[1:], nonce)

	_, err := conn.Write(challengePacket)
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

	conn.SetDeadline(time.Time{})
	return pubKey, nil
}

type Session struct {
	Server     *server
	Addr       *net.TCPAddr
	WriteQueue chan packet.Packet
	ID         snowflake.ID
	PubKey     ed25519.PublicKey

	mu         sync.Mutex
	challenge  []byte
	issuedTime time.Time
}

func newSession(server *server, addr *net.TCPAddr, id snowflake.ID, pubKey ed25519.PublicKey) *Session {
	session := &Session{
		Server:     server,
		Addr:       addr,
		PubKey:     pubKey,
		WriteQueue: make(chan packet.Packet, 10),
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

func processPacket(ctx context.Context, pkt packet.Packet) packet.Packet {
	session, ok := FromContext(ctx)
	assert.Assert(ok, "context in process packet should always have a session")

	var response packet.TypedMessage
	switch pkt.Type() {
	case packet.PacketSendMessage:
		var request packet.SendMessage
		if err := pkt.DecodePayload(&request); err != nil {
			log.Println("decode error:", err)
			response = &packet.ErrorMessage{Error: "malformed payload"}
			break
		}

		content := strings.TrimSpace(request.Content)
		if content == "" {
			response = &packet.ErrorMessage{Error: "message content must not be blank"}
			break
		}

		node := session.Server.Node
		message := data.Message{
			Id:          node.Generate(),
			SenderId:    session.ID,
			FrequencyId: node.Generate(), // TODO: replace with actual ID
			NetworkId:   node.Generate(), // TODO: replace with actual ID
			Contents:    content,
		}
		messages = append(messages, message)

		response = packet.NewOkMessage()
	default:
		response = &packet.ErrorMessage{Error: "use of unsupported packet type"}
	}

	assert.NotNil(response, "response must always be assigned to")
	return packet.NewPacket(packet.NewMsgPackEncoder(response))
}

var messages []data.Message
