package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyren223/eko/internal/server"
)

const port = 7223

func main() {
	server := server.NewServer(port)

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		signal := <-signalChan
		log.Println("signal:", signal.String())
		cancel()
	}()

	server.ListenAndServe(ctx)
}
