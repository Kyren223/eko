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

package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/internal/client/config"
	"github.com/kyren223/eko/internal/client/ui"
	"github.com/kyren223/eko/internal/packet"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
)

var (
	framer  packet.PacketFramer
	conn    net.Conn
	writeMu sync.Mutex
	closed  = false
)

type (
	AuthenticationEstablished snowflake.ID
	ConnectionEstablished     struct{}
	ConnectionFailed          error
	ConnectionLost            error
	ConnectionClosed          struct{}
)

func Connect(timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := connect(ctx)
		if err != nil {
			return ConnectionFailed(err)
		}
		return ConnectionEstablished{}
	}
}

func connect(ctx context.Context) error {
	assert.Assert(conn == nil, "cannot connect, connection is active")
	closed = false

	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	go func() {
		framer = packet.NewFramer()

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(embeds.ServerCertificate) {
			log.Fatalln("failed to append server certificate")
		}

		tlsConfig := &tls.Config{
			RootCAs:    certPool,
			ServerName: config.ReadConfig().ServerName,
			MinVersion: tls.VersionTLS12,
			// This is fine, it's always false by default
			// The user may change the config, the name should be clear enough
			// that this is insecure (valid use cases are for testing purposes)
			InsecureSkipVerify: config.ReadConfig().InsecureDebugMode, // #nosec 402
		}

		address := config.ReadConfig().ServerName
		if config.ReadConfig().InsecureDebugMode {
			address = "localhost"
		}

		connection, err := tls.Dial("tcp4", address+":7223", tlsConfig)
		if err != nil {
			errChan <- err
			return
		}
		log.Println("established connection with the server")

		connChan <- connection
	}()

	select {
	case connection := <-connChan:
		conn = connection
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	go readForever(conn)
	go handlePacketStream()

	return nil
}

func readForever(conn net.Conn) {
	buffer := make([]byte, 512)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			onDisconnect(err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = framer.Push(ctx, buffer[:n])
		if ctx.Err() != nil {
			cancel()
			onDisconnect(errors.New("new packet blocked for more than a second, closing connection"))
			return
		}
		cancel()
		assert.NoError(err, "packets from server should always be correctly formatted")
	}
}

func handlePacketStream() {
	for {
		pkt, ok := <-framer.Out
		if !ok {
			break
		}

		payload, err := pkt.DecodedPayload()
		assert.NoError(err, "server should always provide a decodeable packet")

		// log.Printf("received streamed packet %v: %v\n", payload.Type(), payload)
		ui.Program.Send(payload)
	}
}

func Disconnect() {
	if conn != nil {
		closed = true
		_ = conn.Close()
	}
}

func onDisconnect(err error) {
	writeMu.Lock()
	defer writeMu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Close()
	conn = nil
	close(framer.Out)
	if closed {
		log.Println("connection closed")
		ui.Program.Send(ConnectionClosed{})
	} else {
		log.Println("connection lost:", err)
		ui.Program.Send(ConnectionLost(err))
	}
}

type RequestSentMsg struct {
	request packet.Payload
	err     error
}

func Send(request packet.Payload) tea.Cmd {
	return func() tea.Msg {
		err := send(request)
		if err != nil {
			log.Println("request send error:", err)
		} else {
			// log.Println("request sent successfully:", request)
		}
		return RequestSentMsg{
			request: request,
			err:     err,
		}
	}
}

func SendAsync(request packet.Payload) <-chan error {
	ch := make(chan error, 1)
	go func() {
		err := send(request)
		if err != nil {
			log.Println("async request send error:", err)
		}
		ch <- err
	}()
	return ch
}

func send(request packet.Payload) error {
	pkt := packet.NewPacket(packet.NewMsgPackEncoder(request))

	writeMu.Lock()
	if conn == nil {
		writeMu.Unlock()
		return errors.New("connection is closed")
	}
	_, err := pkt.Into(conn)
	writeMu.Unlock()

	if err != nil {
		return err
	}

	return nil
}
