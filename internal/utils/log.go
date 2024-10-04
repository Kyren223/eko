package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"time"
)

func SetupLogger(name string) {
	logger := slog.New(CustomHandler{os.Stdout, name, nil})
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel()
}

type CustomHandler struct {
	writer io.Writer
	group   string
	attrs  []slog.Attr
}

func (h CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	// timestamp := time.Now().Format(time.RFC3339Nano)
	timestamp := time.Now().Format(time.TimeOnly)
	severity := r.Level.String()
	_, err := fmt.Fprintf(h.writer, "[%s] [%s/%s]: %s\n", timestamp, h.group, severity, r.Message)
	return err
}

func (h CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	slog.Handler
	return true
}

func (h CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{h.writer, h.group, append(h.attrs, attrs...)}
}

func (h CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{h.writer, name, h.attrs}
}
