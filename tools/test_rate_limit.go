package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

const addr = "localhost:7223"

var tlsConfig = &tls.Config{
	InsecureSkipVerify: true, // #nosec G402 skip cert verification
}

func connect(n int, label string) {
	fmt.Println("----", label)
	conns := make([]net.Conn, 0, n)
	for i := 0; i < n; i++ {
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("connect %d failed: %v", i, err)
			continue
		}
		conns = append(conns, conn)
	}
	time.Sleep(300 * time.Millisecond)
	for _, c := range conns {
		_ = c.Close()
	}
	time.Sleep(200 * time.Millisecond)
}

func wait() {
	time.Sleep(1100 * time.Millisecond) // ensure we roll over fixed 1s window
}

func main() {
	// 1. Single connection, then disconnect, should not rate limit
	connect(1, "single connection")
	wait()

	// 2. Two connections (under threshold), should show info
	connect(2, "two connections")
	wait()
	connect(2, "two connections again")
	wait()

	// 3. Hit 5 times to trigger suspicious threshold once
	connect(5, "5 suspicious connections")
	wait()
	connect(5, "5 more suspicious")
	wait()

	// 4. Hit 15 times to trigger malicious
	connect(15, "15 malicious connections")
	wait()
	connect(15, "15 more malicious connections")
}
