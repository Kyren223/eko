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

type SessionManager interface {
	AddSession(session *Session)
	RemoveSession(id snowflake.ID)
	Session(id snowflake.ID) (session *Session, ok bool)

	Node() *snowflake.Node
}

type Session struct {
	// Channel to directly write packets to the cient.
	// Can be nil in cases where the connection is not available.
	WriteQueue chan packet.Packet
	PubKey     ed25519.PublicKey

	manager SessionManager
	addr    *net.TCPAddr
	id      snowflake.ID

	mu         sync.Mutex
	challenge  []byte
	issuedTime time.Time
}

func NewSession(manager SessionManager, addr *net.TCPAddr, id snowflake.ID, pubKey ed25519.PublicKey) *Session {
	session := &Session{
		manager:    manager,
		addr:       addr,
		PubKey:     pubKey,
		WriteQueue: make(chan packet.Packet, 10),
		challenge:  make([]byte, 32), // Recommended nonce size
	}
	session.Challenge() // Make sure an initial nonce is generated
	return session
}

func (s *Session) Addr() *net.TCPAddr {
	return s.addr
}

func (s *Session) ID() snowflake.ID {
	return s.id
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

type key struct{}

var sessKey key

func NewContext(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessKey, sess)
}

func FromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessKey).(*Session)
	return sess, ok
}
