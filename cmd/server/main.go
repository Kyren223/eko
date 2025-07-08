package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kyren223/eko/embeds"
	"github.com/kyren223/eko/internal/server"
	"github.com/kyren223/eko/internal/server/api"
	"github.com/kyren223/eko/internal/server/ctxkeys"
	"github.com/kyren223/eko/internal/webserver"
	"github.com/kyren223/eko/pkg/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	port          = 7223
	TosEnvVar     = "EKO_SERVER_TOS_FILE"
	PrivacyEnvVar = "EKO_SERVER_PRIVACY_FILE"
	LogDirEnvVar  = "EKO_SERVER_LOG_DIR"
)

var prod = true

func main() {
	prodFlag := flag.Bool("prod", true, "true for production mode, false for dev mode")
	flag.Parse()
	prod = *prodFlag

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupLogging()

	slog.Info("mode", "prod", prod)

	handleReload()
	handleShutdown(cancel)

	if ok := reloadTosAndPrivacy(); !ok {
		return
	}

	api.ConnectToDatabase()
	assert.AddFlush(api.DB())
	defer api.DB().Close()

	go webserver.ServeEkoWebsite()

	server := server.NewServer(ctx, port)
	server.Run() // blocks

	slog.Info("exited gracefully")
}

func handleReload() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	go func() {
		for range c {
			slog.Info("SIGHUP received, reloading...")
			reloadTosAndPrivacy()
			slog.Info("reload completed")
		}
	}()

	slog.Info("reload handler ready")
}

func handleShutdown(cancel context.CancelFunc) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signalChan
		slog.Info("shutdown signal received", "signal", signal)
		cancel()
	}()

	slog.Info("shutdown handler ready")
}

func setupLogging() {
	logDir := os.Getenv(LogDirEnvVar)
	if logDir == "" {
		logDir = "logs"
	}
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		log.Fatalln(err)
	}

	rotator := &lumberjack.Logger{
		Filename: filepath.Join(logDir, "server.log"),
		MaxSize:  100, // megabytes
		MaxAge:   7,   // days (see Privacy Section 3: Log Retention)
	}

	level := slog.LevelDebug
	if prod {
		level = slog.LevelInfo
	}
	baseHandler := slog.NewJSONHandler(rotator, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
	handler := ctxkeys.WrapLogHandler(baseHandler)

	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level) // TODO: remove me after fully migrating to slog

	slog.Info("logging handler ready")

	go func() {
		for {
			now := time.Now().UTC() // UTC Time (see Privacy Section 3)
			next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
			time.Sleep(time.Until(next)) // sleep until next midnight

			err := rotator.Rotate()
			if err != nil {
				slog.Error("log file rotation failed", "error", err)
			} else {
				slog.Info("log file rotated")
			}
		}
	}()
}

func reloadTosAndPrivacy() bool {
	if embeds.TosPrivacyHash.Load() == nil {
		if prod {
			embeds.TosPrivacyHash.Store("")
			embeds.TermsOfService.Store("")
			embeds.PrivacyPolicy.Store("")
		} else {
			// Set stub values for development
			embeds.TermsOfService.Store(embeds.StubTos)
			embeds.PrivacyPolicy.Store(embeds.StubPrivacy)
			tosPrivacy := []byte(string(embeds.StubTos) + string(embeds.StubPrivacy))
			stubHash := fmt.Sprintf("%x", sha256.Sum256(tosPrivacy))
			embeds.TosPrivacyHash.Store(stubHash)
			return true
		}
	}

	tosFile := os.Getenv(TosEnvVar)
	privacyFile := os.Getenv(PrivacyEnvVar)
	if tosFile == "" || privacyFile == "" {
		if prod {
			slog.Error("TOS or Privacy env vars are undefined", TosEnvVar, tosFile, PrivacyEnvVar, privacyFile)
			return false
		}
	}
	tos, err := os.ReadFile(tosFile) // #nosec G304
	if err != nil {
		slog.Error("error reading TOS file", "error", err)
		return false
	}
	privacy, err := os.ReadFile(privacyFile) // #nosec G304
	if err != nil {
		slog.Error("error reading Privacy file", "error", err)
		return false
	}

	tosPrivacy := []byte(string(tos) + string(privacy))
	hash := fmt.Sprintf("%x", sha256.Sum256(tosPrivacy))

	oldHash := embeds.TosPrivacyHash.Load().(string)
	if oldHash == hash {
		slog.Info("updated nothing, tos+privacy hash remained the same", "hash", hash)
		return true
	}

	embeds.TermsOfService.Store(string(tos))
	embeds.PrivacyPolicy.Store(string(privacy))
	embeds.TosPrivacyHash.Store(hash)

	if oldHash == "" {
		slog.Info("loaded Terms of Service and Privacy Policy", "hash", hash)
	} else {
		slog.Info("updated tos+privacy, hash changed", "hash", hash, "old_hash", hash)
	}

	return true
}
