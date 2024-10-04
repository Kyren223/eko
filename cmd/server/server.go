package main

import (
	"log/slog"

	"github.com/kyren223/eko/internal/utils"
)

func main() {
	utils.SetupLogger("Server")
	slog.Debug("Eko 'Hello, World!'")
	slog.Info("Eko 'Hello, World!'")
	slog.Warn("Eko 'Hello, World!'")
	slog.Error("Eko 'Hello, World!'")
}
