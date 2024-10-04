package main

import (
	"os"

	"github.com/kyren223/eko/internal/utils/log"
)

func main() {
	log.SetDefault(log.NewLogger("Server", os.Stdout, true))
	log.Debug("Eko 'Hello, World!'")
	log.Info("Eko 'Hello, World!'")
	log.Warn("Eko 'Hello, World!'")
	log.Error("Eko 'Hello, World!'")
}
