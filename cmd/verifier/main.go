package main

import (
	"context"
	"log"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/services/verifier"
)

func main() {
	cfg := config.LoadVerifierConfigFromEnv()

	srv, err := verifier.NewServer(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to build verifier server: %v", err)
	}

	log.Printf("Verifier service running on port %s...", cfg.HTTPPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
