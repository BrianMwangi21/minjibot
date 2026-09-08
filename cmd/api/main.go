package main

import (
	"log"
	"os"

	"github.com/kibetnathan/minjibot/internal/entrypoint"
)

func main() {
	if err := entrypoint.Run(false); err != nil {
		log.Printf("Failed to initialize application: %v", err)
		os.Exit(1)
	}
}
