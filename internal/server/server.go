package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
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

	"github.com/kyren223/eko/certs"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	nodeId    int64 = 0
	tlsConfig *tls.Config
)

func init() {
	path, ok := os.LookupEnv("SERVER_CERT_KEY_FILE")
	if !ok {
		path = "certs/server.key"
	}
	keyPEM, err := os.ReadFile(path) // #nosec 304
	if err != nil {
		log.Fatalln("failed to read certificate key from", path)
	}

	cert, err := tls.X509KeyPair(certs.CertPEM, keyPEM)
	if err != nil {
		log.Fatalln("error loading certificate:", err)
	}

	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
}

type server struct {
	ctx      context.Context
	node     *snowflake.Node
	sessions map[snowflake.ID]*session.Session
	sessMu   sync.RWMutex
	Port     uint16
}

// Creates a new server on the given port.
// Will generate a unique node ID automatically, will crash if there are no available IDs.
func NewServer(ctx context.Context, port uint16) server {
	assert.Assert(nodeId <= snowflake.NodeMax, "maximum amount of servers reached")
	node := snowflake.NewNode(nodeId)
	nodeId++

	return server{
		ctx:      ctx,
		node:     node,
		sessions: map[snowflake.ID]*session.Session{},
		Port:     port,
	}
}

func (s *server) AddSession(session *session.Session) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	if sess, ok := s.sessions[session.ID()]; ok {
		sess.Close()
	}
	s.sessions[session.ID()] = session
}

func (s *server) RemoveSession(id snowflake.ID) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, id)
}

func (s *server) Session(id snowflake.ID) *session.Session {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	session := s.sessions[id]
	return session
}

func (s *server) UseSessions(f func(map[snowflake.ID]*session.Session)) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	f(s.sessions)
}

func (s *server) Node() *snowflake.Node {
	return s.node
}

// Run starts listening and accepting clients,
// blocking until it gets terminated by cancelling the context.
func (s *server) Run() error {
	listener, err := tls.Listen("tcp4", ":"+strconv.Itoa(int(s.Port)), tlsConfig)
	if err != nil {
		log.Fatalf("error starting server: %s", err)
	}

	assert.AddFlush(listener)
	defer listener.Close()
	go func() {
		<-s.ctx.Done()
		_ = listener.Close()
	}()

	log.Println("started listening on port", s.Port)
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
			s.handleConnection(conn)
			wg.Done()
		}()
	}
	log.Println("stopped listening on port", s.Port)

	log.Println("waiting for all active connections to close...")
	wg.Wait()
	log.Println("server shutdown complete")
	return nil
}

func (server *server) handleConnection(conn net.Conn) {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	assert.Assert(ok, "getting tcp address should never fail as we are using tcp connections")

	log.Println(addr, "accepted")

	initialCtx, initialCancel := context.WithTimeout(server.ctx, 5*time.Second)
	deadline, _ := initialCtx.Deadline()
	err := conn.SetDeadline(deadline)
	assert.NoError(err, "setting read deadline should not error")

	err = conn.SetDeadline(time.Time{})
	assert.NoError(err, "unsetting read deadline should not error")

	pubKey, err := handleAuth(conn)
	if err != nil {
		initialCancel()
		log.Println(addr, err)
		_ = conn.Close()
		log.Println(addr, "disconnected")
		return
	}

	user, err := api.CreateOrGetUser(initialCtx, server.Node(), pubKey)
	if err != nil {
		initialCancel()
		log.Println(addr, "user creation/fetching error:", err)
		_ = conn.Close()
		log.Println(addr, "disconnected")
		return
	}
	ctx, cancel := context.WithCancel(server.ctx)
	defer cancel()
	sess := session.NewSession(server, addr, cancel, user.ID, pubKey)
	server.AddSession(sess)
	framer := packet.NewFramer()

	// Write ID back, it's useful for the client to know, and signals successful authentication
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(user.ID)) // #nosec G115 -- sign bit is always 0 in snowflake IDs
	_, err = conn.Write(id[:])
	if err != nil {
		initialCancel()
		log.Println(addr, "failed to write user id")
		_ = conn.Close()
		log.Println(addr, "disconnected")
		return
	}

	initialCancel()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	defer func() {
		_ = conn.Close()
		sameAddress := addr.String() == server.Session(sess.ID()).Addr().String()
		if sameAddress {
			server.RemoveSession(sess.ID())
		}
		log.Println(addr, "disconnected")
	}()

	go func() {
		for {
			packet, ok := sess.Read(ctx)
			if !ok {
				return
			}
			log.Println(addr, "sending packet:", packet)
			if _, err := packet.Into(conn); err != nil {
				log.Println(addr, err)
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case request, ok := <-framer.Out:
				if !ok {
					return
				}

				response := processPacket(ctx, sess, request)
				if ok := sess.Write(ctx, response); !ok {
					return
				}
			}
		}
	}()

	// Send initial packets
	payload := api.GetUserData(ctx, sess, &packet.GetUserData{})
	dataPacket := packet.NewPacket(packet.NewMsgPackEncoder(payload))
	sess.Write(ctx, dataPacket)

	payload, err = api.GetNetworksInfo(ctx, sess)
	if err != nil {
		return // closes the connection
	}
	infoPacket := packet.NewPacket(packet.NewMsgPackEncoder(payload))
	sess.Write(ctx, infoPacket)

	// Infinite read loop
	buffer := make([]byte, 512)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println(addr, err)
			}
			break
		}

		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			log.Println(addr, ctx.Err())
			break
		}
		if err != nil {
			payload := packet.Error{Error: err.Error()}
			pkt := packet.NewPacket(packet.NewMsgPackEncoder(&payload))
			sess.Write(ctx, pkt)
			break
		}
	}
}

func handleAuth(conn net.Conn) (ed25519.PublicKey, error) {
	nonce := [32]byte{}
	_, err := rand.Read(nonce[:])
	assert.NoError(err, "random should always produce a value")

	challengePacket := make([]byte, len(nonce)+1)
	challengePacket[0] = packet.VERSION
	copy(challengePacket[1:], nonce[:])

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

	if ok := ed25519.Verify(pubKey, nonce[:], signature); !ok {
		return nil, errors.New("signature verification failed")
	}

	return pubKey, nil
}

func processPacket(ctx context.Context, sess *session.Session, pkt packet.Packet) packet.Packet {
	var response packet.Payload

	request, err := pkt.DecodedPayload()
	if err != nil {
		response = &packet.Error{Error: "malformed payload"}
	} else {
		response = processRequest(ctx, sess, request)
	}

	assert.NotNil(response, "response must always be assigned to")
	log.Println(sess.Addr(), "sending", response.Type(), "response:", response)
	return packet.NewPacket(packet.NewMsgPackEncoder(response))
}

func processRequest(ctx context.Context, sess *session.Session, request packet.Payload) packet.Payload {
	log.Println(sess.Addr(), "processing", request.Type(), "request:", request)

	// TODO: add a way to measure the time each request/response took and log it
	// Potentially even separate time for code vs DB operations
	var response packet.Payload
	switch request := request.(type) {

	case *packet.SetUserData:
		response = timeout(5*time.Millisecond, api.SetUserData, ctx, sess, request)
	case *packet.GetUserData:
		response = timeout(5*time.Millisecond, api.GetUserData, ctx, sess, request)

	case *packet.CreateNetwork:
		response = timeout(10*time.Millisecond, api.CreateNetwork, ctx, sess, request)
	case *packet.UpdateNetwork:
		response = timeout(5*time.Millisecond, api.UpdateNetwork, ctx, sess, request)
	case *packet.DeleteNetwork:
		response = timeout(500*time.Millisecond, api.DeleteNetwork, ctx, sess, request)

	case *packet.CreateFrequency:
		response = timeout(5*time.Millisecond, api.CreateFrequency, ctx, sess, request)
	case *packet.UpdateFrequency:
		response = timeout(5*time.Millisecond, api.UpdateFrequency, ctx, sess, request)
	case *packet.DeleteFrequency:
		response = timeout(200*time.Millisecond, api.DeleteFrequency, ctx, sess, request)
	case *packet.SwapFrequencies:
		response = timeout(5*time.Millisecond, api.SwapFrequencies, ctx, sess, request)

	case *packet.SendMessage:
		response = timeout(20*time.Millisecond, api.SendMessage, ctx, sess, request)
	case *packet.RequestMessages:
		response = timeout(50*time.Millisecond, api.RequestMessages, ctx, sess, request)

	case *packet.SetMember:
		response = timeout(50*time.Millisecond, api.SetMember, ctx, sess, request)

	default:
		response = &packet.Error{Error: "use of disallowed packet type for request"}
	}

	if response, ok := response.(*packet.Error); ok {
		response.PktType = request.Type()
	}

	return response
}

func timeout[T packet.Payload](
	timeoutDuration time.Duration,
	apiRequest func(context.Context, *session.Session, T) packet.Payload,
	ctx context.Context, sess *session.Session, request T,
) packet.Payload {
	// TODO: Remove the channel and just wait directly?
	responseChan := make(chan packet.Payload)
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	go func() {
		responseChan <- apiRequest(ctx, sess, request)
	}()

	select {
	case response := <-responseChan:
		return response
	case <-ctx.Done():
		log.Println(sess.Addr(), "timeout of", request.Type(), "request")
		return &packet.Error{Error: "request timeout"}
	}
}
