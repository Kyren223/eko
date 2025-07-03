package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kyren223/eko/internal/server"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/pkg/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

const port = 7223

var prod = true

func main() {
	prodFlag := flag.Bool("prod", true, "true for production mode, false for dev mode")
	flag.Parse()
	prod = !(*prodFlag)

	setupLogging()

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

func setupLogging() {
	logDir := os.Getenv("EKO_SERVER_LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		log.Fatalln(err)
	}

	rotator := &lumberjack.Logger{
		Filename: filepath.Join(logDir, "server.log"),
		MaxSize:  1,  // megabytes TODO: switch this to a more reasonable size (100?)
		MaxAge:   28, // days
	}

	level := slog.LevelDebug
	if prod {
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(rotator, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level) // TODO: remove me after fully migrating to slog
}
