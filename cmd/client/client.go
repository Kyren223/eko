package main

import (
	"log/slog"

	"github.com/kyren223/eko/internal/utils"
)

func main() {
	utils.SetupLogger("Client")
	slog.Info("Eko 'Hello, World!'")
}
