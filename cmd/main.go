package main

import (
	"log"
	"os"

	"github.com/kibetnathan/minjibot/internal/entrypoint"
)

func main() {
	if err := entrypoint.Run(true); err != nil {
		log.Fatalf("Failed to initialize API: %v", err)
		os.Exit(1)
	}
}
