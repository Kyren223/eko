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

package session

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/internal/server/ctxkeys"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/rate"
	"github.com/kyren223/eko/pkg/snowflake"
)

const (
	WriteQueueSize = 10
	NonceSize      = 32

	DefaultRate        = 0.1 // ms per second
	DefaultLimit       = 3   // ms burst
	AuthenticatedRate  = 1   // ms per second
	AuthenticatedLimit = 20  // ms burst
)

type SessionManager interface {
	AddSession(session *Session, userId snowflake.ID, pubKey ed25519.PublicKey)
	RemoveSession(id snowflake.ID)
	Session(id snowflake.ID) *Session
	UseSessions(f func(map[snowflake.ID]*Session))

	Node() *snowflake.Node
}

type Session struct {
	manager SessionManager
	addr    *net.TCPAddr
	cancel  context.CancelFunc

	writeQueue chan packet.Payload
	writerWg   *sync.WaitGroup
	writeMu    sync.RWMutex

	issuedTime  time.Time
	challenge   []byte
	challengeMu sync.Mutex

	isTosAccepted bool
	pubKey        ed25519.PublicKey
	id            snowflake.ID
	rl            rate.Limiter
	start         time.Time
	analytics     *packet.DeviceAnalytics

	mu sync.RWMutex
}

func NewSession(
	manager SessionManager,
	addr *net.TCPAddr, cancel context.CancelFunc,
	writerWg *sync.WaitGroup,
) *Session {
	assert.NotNil(addr, "tcp address should be valid")
	assert.NotNil(manager, "session manager should be valid")
	session := &Session{
		manager:       manager,
		addr:          addr,
		cancel:        cancel,
		writeQueue:    make(chan packet.Payload, WriteQueueSize),
		writerWg:      writerWg,
		writeMu:       sync.RWMutex{},
		issuedTime:    time.Time{},
		challenge:     make([]byte, NonceSize),
		pubKey:        ed25519.PublicKey{},
		id:            snowflake.InvalidID,
		challengeMu:   sync.Mutex{},
		isTosAccepted: false,
		rl:            rate.NewLimiter(DefaultRate, DefaultLimit),
		start:         time.Time{},
		analytics:     nil,
		mu:            sync.RWMutex{},
	}
	return session
}

func (s *Session) Addr() *net.TCPAddr {
	assert.NotNil(s.addr, "tcp address should be valid")
	return s.addr
}

func (s *Session) RateLimiter() *rate.Limiter {
	return &s.rl
}

func (s *Session) IsTosAccepted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isTosAccepted
}

func (s *Session) ReceivedTosAcceptance() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isTosAccepted = true
}

func (s *Session) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id != snowflake.InvalidID
}

func (s *Session) ID() snowflake.ID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assert.Assert(s.IsAuthenticated(), "use of ID in an unauthenticated session", "addr", s.addr)
	return s.id
}

func (s *Session) PubKey() ed25519.PublicKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assert.Assert(s.IsAuthenticated(), "use of PubKey in an unauthenticated session", "addr", s.addr)
	return s.pubKey
}

func (s *Session) Promote(userId snowflake.ID, pubKey ed25519.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.id = userId
	s.pubKey = pubKey
	s.rl.SetLimit(AuthenticatedLimit)
	s.rl.SetRate(AuthenticatedRate)
	s.start = time.Now().UTC()
}

func (s *Session) Duration() time.Duration {
	assert.Assert(s.IsAuthenticated(), "tried accessing session duration in an authenticated session")
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.start)
}

func (s *Session) Analytics() *packet.DeviceAnalytics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.analytics
}

func (s *Session) SetAnalytics(analytics *packet.DeviceAnalytics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.analytics = analytics
}

func (s *Session) Manager() SessionManager {
	return s.manager
}

func (s *Session) Challenge() []byte {
	s.challengeMu.Lock()
	defer s.challengeMu.Unlock()
	if time.Since(s.issuedTime) > time.Minute {
		s.issuedTime = time.Now()
		_, err := rand.Read(s.challenge)
		assert.NoError(err, "random should always produce a value")
	}
	return s.challenge
}

func (s *Session) Write(ctx context.Context, payload packet.Payload) bool {
	s.writerWg.Add(1)
	defer s.writerWg.Done()

	s.writeMu.RLock()
	defer s.writeMu.RUnlock()

	select {
	case s.writeQueue <- payload:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Session) Read() <-chan packet.Payload {
	s.writeMu.RLock()
	defer s.writeMu.RUnlock()
	return s.writeQueue
}

func (s *Session) CloseWriteQueue() {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	close(s.writeQueue)
	s.writeQueue = nil
}

func (s *Session) Close() {
	s.cancel()
}

func (s *Session) LogValue() slog.Value {
	if s.IsAuthenticated() {
		return slog.GroupValue(
			slog.Any(ctxkeys.IpAddr.String(), s.Addr()),
			slog.Any(ctxkeys.UserID.String(), s.ID()),
			slog.Any("public_key", s.PubKey()),
		)
	} else {
		return slog.GroupValue(
			slog.Any(ctxkeys.IpAddr.String(), s.Addr()),
		)
	}
}
