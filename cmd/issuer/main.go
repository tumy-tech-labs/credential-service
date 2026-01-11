package main

import (
	"context"
	"log"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/services/issuer"
)

func main() {
	cfg := config.LoadIssuerConfigFromEnv()

	srv, err := issuer.NewServer(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to build issuer server: %v", err)
	}

	log.Printf("Issuer service running on port %s", cfg.HTTPPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
