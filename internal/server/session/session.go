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
	Session(id snowflake.ID) *Session
	UseSessions(f func(map[snowflake.ID]*Session))

	Node() *snowflake.Node
}

type Session struct {
	manager    SessionManager
	addr       *net.TCPAddr
	cancel     context.CancelFunc
	writeQueue chan packet.Packet

	issuedTime time.Time
	challenge  []byte

	PubKey ed25519.PublicKey
	id     snowflake.ID

	mu sync.Mutex
}

func NewSession(
	manager SessionManager,
	addr *net.TCPAddr, cancel context.CancelFunc,
	id snowflake.ID, pubKey ed25519.PublicKey,
) *Session {
	session := &Session{
		manager:    manager,
		addr:       addr,
		cancel:     cancel,
		writeQueue: make(chan packet.Packet, 10),
		issuedTime: time.Time{},
		challenge:  make([]byte, 32),
		PubKey:     pubKey,
		id:         id,
		mu:         sync.Mutex{},
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

func (s *Session) Write(ctx context.Context, pkt packet.Packet) bool {
	select {
	case s.writeQueue <- pkt:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Session) Read(ctx context.Context) (packet.Packet, bool) {
	select {
	case pkt := <-s.writeQueue:
		return pkt, true
	case <-ctx.Done():
		return packet.Packet{}, false
	}
}

func (s *Session) Close() {
	timeout := 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	payload := &packet.Error{
		Error:   "new connection from another location, closing this one",
		PktType: packet.PacketError,
	}
	pkt := packet.NewPacket(packet.NewMsgPackEncoder(payload))
	s.Write(ctx, pkt)
	cancel()

	s.cancel()
}
