package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kyren223/eko/internal/server"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/pkg/assert"
)

const port = 7223

func main() {
	stdout := flag.Bool("stdout", false, "enable logging to stdout")
	flag.Parse()

	logDir := "logs"
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		log.Fatalln(err)
	}
	logPath := fmt.Sprintf("eko-server-%s.log", time.Now().Format("2006-01-02_15-04-05"))
	logPath = filepath.Join(logDir, logPath)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) // #nosec G304
	if err != nil {
		log.Fatalln(err)
	}
	defer logFile.Close()
	assert.AddFlush(logFile)

	if *stdout {
		log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	} else {
		log.SetOutput(logFile)
	}

	api.ConnectToDatabase()
	assert.AddFlush(api.DB())
	defer api.DB().Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		signal := <-signalChan
		log.Println("signal:", signal.String())
		cancel()
	}()

	server := server.NewServer(ctx, port)
	if err := server.Run(); err != nil {
		log.Println(err)
	}
}
