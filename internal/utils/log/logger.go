package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	name   string
	Level
	writer io.Writer
	mu sync.Mutex
	color bool
}

func NewLogger(name string, writer io.Writer, ansi bool) *Logger {
	color := ansi && (writer == os.Stdout || writer == os.Stderr)
	return &Logger{name, LevelInfo, writer, sync.Mutex{}, color}
}

func (l *Logger) Log(level Level, message string, a ...any) error {
	if l.Level > level {
		return nil
	}

	timestamp := time.Now().Format(time.TimeOnly)
	severity := level.String()
	formattedMessage := fmt.Sprintf(message, a...)

	// TODO: add support for ANSI coloring
	l.mu.Lock()
	_, err := fmt.Fprintf(l.writer, "[%s] [%s/%s]: %s\n", timestamp, l.name, severity, formattedMessage)
	l.mu.Unlock()
	return err
}

func (l *Logger) Debug(message string, a ...any) error {
	return l.Log(LevelDebug, message, a...)
}

func (l *Logger) Info(message string, a ...any) error {
	return l.Log(LevelInfo, message, a...)
}

func (l *Logger) Warn(message string, a ...any) error {
	return l.Log(LevelWarn, message, a...)
}

func (l *Logger) Error(message string, a ...any) error {
	return l.Log(LevelError, message, a...)
}

func (l *Logger) Fatal(message string, a ...any) error {
	err := l.Log(LevelError, message, a...)
	if err != nil {
		return err
	}
	os.Exit(1)
	return nil
}

