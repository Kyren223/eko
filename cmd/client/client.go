package main

import (
	"os"

	"github.com/kyren223/eko/internal/utils/log"
)

func main() {
	log.SetDefault(log.NewLogger("Client", os.Stdout, true))
	log.Info("Eko 'Hello, World!'")
}
