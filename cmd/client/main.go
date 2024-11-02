package main

import (
	"log"
	"os"

	"github.com/kyren223/eko/internal/client"
)

func main() {
	logFile, err := os.OpenFile("logs/client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	client.Run()
}
