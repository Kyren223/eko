package main

import (
	"os"

	"github.com/kyren223/eko/internal/server"
	"github.com/kyren223/eko/internal/utils/log"
)

func main() {
	log.SetDefault(log.NewLogger("Server", os.Stdout, true))
	server.Start()
}
