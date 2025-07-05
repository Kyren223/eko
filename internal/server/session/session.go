package session

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"sync"
	"time"

	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

const WriteQueueSize = 10

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

	writeQueue chan packet.Packet
	writerWg   *sync.WaitGroup
	writeMu    sync.RWMutex

	issuedTime time.Time
	challenge  []byte

	pubKey ed25519.PublicKey
	id     snowflake.ID

	mu sync.Mutex
}

func NewSession(
	manager SessionManager,
	addr *net.TCPAddr, cancel context.CancelFunc,
	writerWg *sync.WaitGroup,
) *Session {
	assert.NotNil(addr, "tcp address should be valid")
	assert.NotNil(manager, "session manager should be valid")
	session := &Session{
		manager:    manager,
		addr:       addr,
		cancel:     cancel,
		writeQueue: make(chan packet.Packet, WriteQueueSize),
		writerWg:   writerWg,
		issuedTime: time.Time{},
		challenge:  make([]byte, 32),
		pubKey:     ed25519.PublicKey{},
		id:         snowflake.InvalidID,
		mu:         sync.Mutex{},
	}
	return session
}

func (s *Session) Addr() *net.TCPAddr {
	assert.NotNil(s.addr, "tcp address should be valid")
	return s.addr
}

func (s *Session) IsAuthenticated() bool {
	return s.id != snowflake.InvalidID
}

func (s *Session) ID() snowflake.ID {
	assert.Assert(s.IsAuthenticated(), "use of ID in an unauthenticated session", "addr", s.addr)
	return s.id
}

func (s *Session) PubKey() ed25519.PublicKey {
	assert.Assert(s.IsAuthenticated(), "use of PubKey in an unauthenticated session", "addr", s.addr)
	return s.pubKey
}

func (s *Session) Promote(userId snowflake.ID, pubKey ed25519.PublicKey) {
	s.id = userId
	s.pubKey = pubKey
}

func (s *Session) Manager() SessionManager {
	return s.manager
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

func (s *Session) Write(ctx context.Context, pkt packet.Packet) bool {
	s.writerWg.Add(1)
	defer s.writerWg.Done()

	s.writeMu.RLock()
	defer s.writeMu.RUnlock()

	select {
	case s.writeQueue <- pkt:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Session) Read() <-chan packet.Packet {
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
