package server

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kyren223/eko/certs"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/internal/server/ctxkeys"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	nodeId    int64 = 0
	tlsConfig *tls.Config
)

func init() {
	path, ok := os.LookupEnv("EKO_SERVER_CERT_FILE")
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

func (s *server) AddSession(session *session.Session, userId snowflake.ID, pubKey ed25519.PublicKey) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	session.Promote(userId, pubKey)

	if sess, ok := s.sessions[session.ID()]; ok {
		EvictSession(sess) // last connection wins
		slog.Info("closed due to new connection from another location",
			ctxkeys.IpAddr.String(), sess.Addr(),
			ctxkeys.UserID.String(), sess.ID(),
			ctxkeys.EvictedBy.String(), session.Addr(),
		)
		slog.Info("this session evicted another session",
			ctxkeys.IpAddr.String(), session.Addr(),
			ctxkeys.UserID.String(), session.ID(),
			ctxkeys.Evicted.String(), sess.Addr(),
		)
	}

	s.sessions[session.ID()] = session
}

func EvictSession(sess *session.Session) {
	timeout := 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	payload := &packet.Error{
		Error: "new connection from another location, closing this one",
	}
	sess.Write(ctx, api.WrapPayload(payload))

	sess.Close()
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
func (s *server) Run() {
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

	slog.Info("server started accepting new connections", "port", s.Port)
	var wg sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				slog.Error("failed accepting new connection", "error", err)
			}

			if s.ctx.Err() != nil {
				slog.Info("server context done", "error", s.ctx.Err())
				break
			}
			continue // Ignore and skip (don't connect)
		}
		wg.Add(1)
		go func() {
			s.handleConnection(conn)
			wg.Done()
		}()
	}
	slog.Info("server stopped accepting new connections", "port", s.Port)

	slog.Info("waiting for all active connections to close...")
	wg.Wait()
	slog.Info("completed server shutdown")
}

func (server *server) handleConnection(conn net.Conn) {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	assert.Assert(ok, "getting tcp address should never fail as we are using tcp connections")

	ctx, cancel := context.WithCancel(server.ctx)
	defer cancel()

	ctx = context.WithValue(ctx, ctxkeys.IpAddr, addr)

	slog.InfoContext(ctx, "connection accepted")
	defer slog.InfoContext(ctx, "connection closed")
	defer conn.Close()

	var writerWg sync.WaitGroup
	done := make(chan struct{})
	framer := packet.NewFramer()

	sess := session.NewSession(server, addr, cancel, &writerWg)
	go func() {
		<-ctx.Done()
		// Remove session after cancellation
		if sess.IsAuthenticated() {
			sameAddress := addr.String() == server.Session(sess.ID()).Addr().String()
			// false if the user signed in from a different connection
			if sameAddress {
				server.RemoveSession(sess.ID())
			}
		}
	}()

	// Writer
	go func() {
		defer close(done)
		defer conn.Close() // To unblock reader
		writeQueue := sess.Read()

		for packet := range writeQueue {
			if _, err := packet.Into(conn); err != nil {
				// TODO: probably should add this to prevent the
				// "use of closed connection" error, as it's intended to happen
				// and once it happens we can just return
				// if !errors.Is(err, net.ErrClosed) {
				// 	log.Println(addr, err)
				// }
				slog.ErrorContext(ctx, "error sending packet", "error", err, "packet", packet)
				return
			}
			slog.InfoContext(ctx, "packet sent", "packet", packet)
		}
	}()

	// Writer closer
	go func() {
		writerWg.Wait()
		sess.CloseWriteQueue() // causes writer to return
	}()

	// Processor
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		localCtx := context.WithoutCancel(ctx)
		// Local context to not be effected by parent cancellation
		// will still have a time limit upper bound, from timeout()

		for request := range framer.Out {
			processPacket(localCtx, sess, request)
		}
	}()

	// NOTE: IMPROTANT LEGAL STUFF
	// Sending this first thing, before client sends us any data
	sendTosInfo(ctx, sess)

	// Reader
	buffer := make([]byte, 512)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.InfoContext(ctx, "closed gracefully")
			} else {
				slog.ErrorContext(ctx, "failed reading from buffer", "error", err)
			}
			break
		}

		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "reader context done", "error", ctx.Err())
			break
		}
		if err != nil {
			writerWg.Add(1)
			sess.Write(ctx, api.WrapPayload(&packet.Error{Error: err.Error()}))
			writerWg.Done()
			slog.WarnContext(ctx, "received malformed packet", "error", err)
			break
		}
	}
	close(framer.Out) // stop processing

	<-done
}

func processPacket(ctx context.Context, sess *session.Session, pkt packet.Packet) {
	var response packet.Payload

	request, err := pkt.DecodedPayload()
	if err != nil {
		response = &packet.Error{Error: "malformed payload"}
	} else {
		response = processRequest(ctx, sess, request)
	}

	// Nil is ok if responses were handled manually using sess.Write()
	if response != nil {
		ok := sess.Write(ctx, api.WrapPayload(response))
		assert.Assert(ok, "context is never done and write will panic if queue is closed")
	}
}

func processRequest(ctx context.Context, sess *session.Session, request packet.Payload) packet.Payload {
	slog.InfoContext(ctx, "processing request",
		ctxkeys.PayloadType.String(),
		request.Type(), ctxkeys.Payload.String(), request,
	)

	if !sess.IsTosAccepted() {
		if acceptTos, ok := request.(*packet.AcceptTos); ok && acceptTos.IAgreeToTheTermsOfServiceAndPrivacyPolicy {
			sess.ReceivedTosAcceptance()
			slog.InfoContext(ctx, "terms of service accepted, continuing...")
			return &api.ErrSuccess
		}

		slog.InfoContext(ctx, "refused terms of service, refusing service...")
		sess.Close() // Refuse to receive any more requests
		return &packet.Error{Error: "Terms of Service not accepted, refusing service"}
	}

	assert.Assert(sess.IsTosAccepted(), "justified paranoia") // Just in case

	// TODO: add a way to measure the time each request/response took and log it
	// Potentially even separate time for code vs DB operations

	var response packet.Payload

	if sess.IsAuthenticated() {
		// Authentication only requests, others will be handled without auth even if authenticated
		authCtx := ctxkeys.WithValue(ctx, ctxkeys.UserID, sess.ID())
		response = processAuthenticatedRequests(authCtx, sess, request)
	}

	if response != nil {
		return response
	}

	switch request := request.(type) {

	case *packet.GetNonce:
		response = timeout(5*time.Millisecond, api.GetNonce, ctx, sess, request)

	case *packet.Authenticate:
		response = timeout(5*time.Millisecond, api.Authenticate, ctx, sess, request)

	default:
		response = &packet.Error{Error: "use of disallowed packet type for request"}
	}

	return response
}

func processAuthenticatedRequests(ctx context.Context, sess *session.Session, request packet.Payload) packet.Payload {
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
	case *packet.EditMessage:
		response = timeout(5*time.Millisecond, api.EditMessage, ctx, sess, request)
	case *packet.DeleteMessage:
		response = timeout(5*time.Millisecond, api.DeleteMessage, ctx, sess, request)
	case *packet.RequestMessages:
		response = timeout(50*time.Millisecond, api.RequestMessages, ctx, sess, request)

	case *packet.GetBannedMembers:
		response = timeout(10*time.Millisecond, api.GetBannedMembers, ctx, sess, request)
	case *packet.SetMember:
		response = timeout(50*time.Millisecond, api.SetMember, ctx, sess, request)

	case *packet.TrustUser:
		response = timeout(10*time.Millisecond, api.TrustUser, ctx, sess, request)

	case *packet.SetLastReadMessages:
		response = timeout(50*time.Millisecond, api.SetLastReadMessages, ctx, sess, request)

	case *packet.BlockUser:
		response = timeout(10*time.Millisecond, api.BlockUser, ctx, sess, request)

	case *packet.GetUsers:
		response = timeout(10*time.Millisecond, api.GetUsers, ctx, sess, request)

	default:
		response = nil
	}

	return response
}

func timeout[T packet.Payload](
	timeoutDuration time.Duration,
	apiRequest func(context.Context, *session.Session, T) packet.Payload,
	ctx context.Context, sess *session.Session, request T,
) packet.Payload {
	// TODO: Remove the channel and just wait directly?
	// No - We need to use a channel so timeout works properly
	responseChan := make(chan packet.Payload)

	// TODO: Check if this is now fixed after the rewrite:
	// currently just ignoring the given context
	// this fixes the issue where the client disconnects so the server
	// doesn't bother and cancels the request
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration) // no longer ignoring
	defer cancel()

	go func() {
		responseChan <- apiRequest(ctx, sess, request)
	}()

	select {
	case response := <-responseChan:
		return response
	case <-ctx.Done():
		slog.WarnContext(ctx, "request timeout",
			ctxkeys.Payload.String(), request,
			ctxkeys.PayloadType.String(), request.Type(),
		)
		return &packet.Error{Error: "request timeout"}
	}
}

func sendTosInfo(ctx context.Context, sess *session.Session) bool {
	// FIXME: ===== REPLACE WITH ACTUAL TOS =====
	// TODO: Get TOS and privacy policy
	tos := "WIP Terms of Service"
	privacy := "WIP Privacy Policy"
	date := "2025-07-03"

	payload := &packet.TosInfo{
		Tos:           tos,
		PrivacyPolicy: privacy,
		Date:          date,
	}
	return sess.Write(ctx, api.WrapPayload(payload))
}
