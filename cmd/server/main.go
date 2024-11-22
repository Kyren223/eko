package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyren223/eko/internal/server"
	"github.com/kyren223/eko/internal/server/api"
)

const port = 7223

func main() {
	stdout := flag.Bool("stdout", false, "enable logging to stdout")
	flag.Parse()

	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer logFile.Close()

	if *stdout {
		log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	} else {
		log.SetOutput(logFile)
	}

	api.ConnectToDatabase()
	defer api.CloseDatabase()

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
