// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/internal/server/ctxkeys"
	"github.com/kyren223/eko/internal/server/metrics"
	"github.com/kyren223/eko/internal/server/session"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
	"github.com/prometheus/client_golang/prometheus"
)

var nodeId int64 = 0

const CertFile = "EKO_SERVER_CERT_FILE"

const (
	ReadCheckCancelledInterval       = 1 * time.Second
	RateLimitWindowSize              = 1 * time.Second
	RateLimitCountThresholdSus       = 3
	RateLimitCountThresholdMalicious = 10
)

func getTLSConfig() *tls.Config {
	path, ok := os.LookupEnv(CertFile)
	if !ok {
		// DEV MODE ONLY, DUMMY CERT
		cert, err := generateDummyCert()
		if err != nil {
			slog.Error("failed to generate dummy cert", "error", err)
			assert.Abort("see logs")
		}

		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}
	keyPEM, err := os.ReadFile(path) // #nosec 304
	if err != nil {
		slog.Error("failed to read certificate key", "path", path)
		assert.Abort("see logs")
	}

	cert, err := tls.X509KeyPair(embeds.ServerCertificate, keyPEM)
	if err != nil {
		slog.Error("error loading certificate", "error", err)
		assert.Abort("see logs")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
}

func generateDummyCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))

	template := x509.Certificate{
		SerialNumber: serial,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Subject:      pkix.Name{CommonName: "localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}

type server struct {
	ctx      context.Context
	node     *snowflake.Node
	sessions map[snowflake.ID]*session.Session
	sessMu   sync.RWMutex
	Port     uint16
	ipConns  map[uint32]struct {
		start time.Time
		count uint8
	}
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
		sessMu:   sync.RWMutex{},
		ipConns: map[uint32]struct {
			start time.Time
			count uint8
		}{},
	}
}

func (s *server) AddSession(session *session.Session, userId snowflake.ID, pubKey ed25519.PublicKey) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	session.Promote(userId, pubKey)

	if sess, ok := s.sessions[session.ID()]; ok {
		EvictSession(sess) // last connection wins
		metrics.UsersActive.Dec()
		slog.Info("closed due to new connection from another location",
			ctxkeys.IpAddr.String(), sess.Addr(),
			ctxkeys.UserID.String(), sess.ID(),
			"evicted_by", session.Addr(),
		)
		slog.Info("this session evicted another session",
			ctxkeys.IpAddr.String(), session.Addr(),
			ctxkeys.UserID.String(), session.ID(),
			"evicted", sess.Addr(),
		)
	}

	s.sessions[session.ID()] = session
	metrics.UsersActive.Inc()
}

func EvictSession(sess *session.Session) {
	timeout := 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	payload := &packet.Error{
		Error: "new connection from another location, closing this one",
	}
	sess.Write(ctx, payload)

	sess.Close()
}

func (s *server) RemoveSession(id snowflake.ID) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, id)
	metrics.UsersActive.Dec()
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
	slog.Info("starting eko-server...")

	listener, err := tls.Listen("tcp4", ":"+strconv.Itoa(int(s.Port)), getTLSConfig())
	if err != nil {
		slog.Error("error starting server", "error", err)
		assert.Abort("see logs")
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

		ip := binary.BigEndian.Uint32(conn.RemoteAddr().(*net.TCPAddr).IP.To4())
		if s.isRateLimited(ip) {
			_ = conn.Close()
			continue
		}

		wg.Add(1)
		go func() {
			metrics.ConnectionsEstablished.Inc()
			metrics.ConnectionsActive.Inc()
			s.handleConnection(conn)
			metrics.ConnectionsActive.Dec()
			metrics.ConnectionsClosed.Inc()
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
	writeDone := make(chan struct{})
	framer := packet.NewFramer()

	sess := session.NewSession(server, addr, cancel, &writerWg)
	go func() {
		<-ctx.Done()
		if sess.IsAuthenticated() {
			server.handleSessionMetrics(ctx, sess)

			// Remove session after cancellation
			sameAddress := addr.String() == server.Session(sess.ID()).Addr().String()
			// false if the user signed in from a different connection
			if sameAddress {
				server.RemoveSession(sess.ID())
			}
		}
	}()

	// Writer
	go func() {
		defer close(writeDone)
		defer conn.Close() // To unblock reader
		writeQueue := sess.Read()

		for payload := range writeQueue {
			packet := packet.NewPacket(packet.NewMsgPackEncoder(payload))
			if _, err := packet.Into(conn); err != nil {
				if errors.Is(err, syscall.EPIPE) {
					slog.InfoContext(ctx, "client disconnected while sending packet", "error", err, "packet", packet.LogValue(), "payload", payload)
				} else {
					slog.ErrorContext(ctx, "error sending packet", "error", err, "packet", packet.LogValue(), "payload", payload)
				}
				return
			}
			slog.InfoContext(ctx, "packet sent", "packet", packet.LogValue(), "payload", payload)
		}
		slog.InfoContext(ctx, "writer done")
	}()

	// Writer closer
	go func() {
		writerWg.Wait()
		sess.CloseWriteQueue() // causes writer to return
		slog.InfoContext(ctx, "closing writer...")
	}()

	// Processor
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		localCtx := context.WithoutCancel(ctx)
		// Local context to not be effected by parent cancellation
		// will still have a time limit upper bound, from timeout()

		for request := range framer.Out {
			metrics.RequestsInProgress.WithLabelValues(request.Type().String()).Inc()
			start := time.Now().UTC()
			success := processPacket(localCtx, sess, request)
			duration := time.Since(start)
			metrics.RequestsInProgress.WithLabelValues(request.Type().String()).Dec()

			labels := prometheus.Labels{
				"request_type": request.Type().String(),
				"dropped":      strconv.FormatBool(!success),
			}

			metrics.RequestProcessingDuration.With(labels).Observe(float64(duration.Seconds()))
			if success {
				slog.InfoContext(ctx, "processed request", "request_type", request.Type().String(), "duration", duration.String(), "duration_ns", duration.Nanoseconds())
			} else {
				slog.InfoContext(ctx, "dropped request", "request_type", request.Type().String(), "duration", duration.String(), "duration_ns", duration.Nanoseconds())
			}
		}
		slog.InfoContext(ctx, "processor done")
	}()

	// NOTE: IMPROTANT LEGAL STUFF
	// Sending this first thing, before client sends us any data
	sendTosInfo(ctx, sess)

	// Reader
	buffer := make([]byte, 512)
	for {
		err := conn.SetReadDeadline(time.Now().Add(ReadCheckCancelledInterval))
		if err != nil {
			slog.ErrorContext(ctx, "failed setting read deadline", "error", err)
			break
		}

		n, err := conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.InfoContext(ctx, "closing gracefully")
				break
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if ctx.Err() != nil {
					slog.InfoContext(ctx, "reader context done", "error", ctx.Err())
				} // else continue normally
			} else {
				slog.ErrorContext(ctx, "failed reading from conn", "error", err)
				break
			}
		}

		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "reader context done", "error", ctx.Err())
			break
		}
		if err != nil {
			writerWg.Add(1)
			sess.Write(ctx, &packet.Error{Error: err.Error()})
			writerWg.Done()
			slog.WarnContext(ctx, "received malformed packet", "error", err)
			break
		}
	}
	close(framer.Out) // stop processing
	slog.InfoContext(ctx, "reader done, closed framer")

	<-writeDone
}

func processPacket(ctx context.Context, sess *session.Session, pkt packet.Packet) bool {
	tokens := TokensPerRequest(pkt.Type())
	if !sess.RateLimiter().Take(tokens) {
		_ = sess.Write(ctx, &api.ErrRateLimited)
		return false // Rate limit was hit
	}

	var response packet.Payload

	request, err := pkt.DecodedPayload()
	if err != nil {
		response = &packet.Error{Error: "malformed payload"}
	} else {
		response = processRequest(ctx, sess, request)
	}

	// Nil is ok if responses were handled manually using sess.Write()
	if response != nil {
		ok := sess.Write(ctx, response)
		assert.Assert(ok, "context is never done and write will panic if queue is closed")
	}

	return true
}

func processRequest(ctx context.Context, sess *session.Session, request packet.Payload) packet.Payload {
	slog.InfoContext(ctx, "processing request",
		"request", request, "request_type", request.Type().String(),
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

	case *packet.DeviceAnalytics:
		response = timeout(5*time.Millisecond, api.DeviceAnalytics, ctx, sess, request)

	default:
		response = &packet.Error{Error: fmt.Sprintf(
			"use of disallowed packet type %v", request.Type().String(),
		)}
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

	// TODO: add a timeout for this (even tho it should be super fast)
	api.SetLastUserActivity(ctx, sess)

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
			"request", request, "request_type", request.Type().String(),
		)
		return &packet.Error{Error: "request timeout"}
	}
}

func sendTosInfo(ctx context.Context, sess *session.Session) bool {
	tos := embeds.TermsOfService.Load().(string)
	privacy := embeds.PrivacyPolicy.Load().(string)
	hash := embeds.TosPrivacyHash.Load().(string)

	payload := &packet.TosInfo{
		Tos:           tos,
		PrivacyPolicy: privacy,
		Hash:          hash,
	}
	return sess.Write(ctx, payload)
}

func TokensPerRequest(requestType packet.PacketType) float64 {
	// 1 token means 1 token per second, which is equivalent  to 1ms
	// The idea is that for 1000 users, each user has 1ms of server time
	// This is the baseline but requests may take less/more

	switch requestType {

	case packet.PacketAcceptTos:
		return 0.15
	case packet.PacketGetNonce:
		return 0.1
	case packet.PacketAuthenticate:
		return 1.5
	case packet.PacketDeviceAnalytics:
		return 0.2 // arbitrary

	// TODO: once I get more data for these, add them
	case packet.PacketBlockUser:
	case packet.PacketCreateFrequency:
	case packet.PacketCreateNetwork:
	case packet.PacketDeleteFrequency:
	case packet.PacketDeleteMessage:
	case packet.PacketDeleteNetwork:
	case packet.PacketEditMessage:
	case packet.PacketGetBannedMembers:
	case packet.PacketGetUserData:
	case packet.PacketGetUsers:
	case packet.PacketRequestMessages:
	case packet.PacketSendMessage:
	case packet.PacketSetLastReadMessages:
	case packet.PacketSetMember:
	case packet.PacketSetUserData:
	case packet.PacketSwapFrequencies:
	case packet.PacketTransferNetwork:
	case packet.PacketTrustUser:
	case packet.PacketUpdateFrequency:
	case packet.PacketUpdateNetwork:

	}

	return 1
}

func (s *server) isRateLimited(ip uint32) bool {
	if entry, ok := s.ipConns[ip]; ok {
		outsideWindow := time.Since(entry.start) > RateLimitWindowSize
		notMalicious := entry.count < RateLimitCountThresholdMalicious
		if outsideWindow && notMalicious {
			entry.start = time.Now().UTC()
			entry.count = 1

			s.ipConns[ip] = entry
			return false
		}

		ipStr := formatIPv4(ip)

		if entry.count < RateLimitCountThresholdSus {
			slog.Info("connection activity", "ip", ipStr, "count", entry.count)
			entry.count++
			s.ipConns[ip] = entry
			return false
		} else if entry.count < RateLimitCountThresholdMalicious {
			metrics.ConnectionsRateLimited.WithLabelValues("suspicious").Inc()
			if entry.count == RateLimitCountThresholdSus {
				slog.Warn("suspicious connection activity", "ip", ipStr, "count", entry.count)
				// Only log the first one
			}
			entry.count++
			s.ipConns[ip] = entry
			return true
		} else {
			metrics.ConnectionsRateLimited.WithLabelValues("malicious").Inc()
			if entry.count == RateLimitCountThresholdMalicious {
				slog.Warn("potential malicious connection behavior", "ip", ipStr, "count", entry.count)
				// Only log the first one
				entry.count++
				s.ipConns[ip] = entry
				// Update so it doesn't spam
			}
			// Don't bother to update counts, save resources
			return true
		}
	}

	s.ipConns[ip] = struct {
		start time.Time
		count uint8
	}{
		start: time.Now().UTC(),
		count: 1,
	}

	return false
}

func formatIPv4(ip uint32) string {
	var b [15]byte // max len for "255.255.255.255"
	n := strconv.AppendUint(b[:0], uint64(ip>>24), 10)
	n = append(n, '.')
	n = strconv.AppendUint(n, uint64((ip>>16)&0xFF), 10)
	n = append(n, '.')
	n = strconv.AppendUint(n, uint64((ip>>8)&0xFF), 10)
	n = append(n, '.')
	n = strconv.AppendUint(n, uint64(ip&0xFF), 10)
	return string(n)
}

func (s *server) handleSessionMetrics(ctx context.Context, sess *session.Session) {
	duration := sess.Duration()
	analytics := sess.Analytics()
	if api.IsValidAnalytics(ctx, analytics) {
		metrics.SessionDuration.WithLabelValues(
			analytics.OS, analytics.Arch, analytics.Term, analytics.Colorterm,
		).Observe(duration.Seconds())
		slog.DebugContext(ctx, "observed session duration", "session_duration", duration.Seconds())
	} else {
		metrics.SessionDuration.WithLabelValues(
			"", "", "", "",
		).Observe(duration.Seconds())
		slog.DebugContext(ctx, "observed session duration (empty)", "session_duration", duration.Seconds())
	}
}
