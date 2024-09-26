package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/jackc/pgx/v4/pgxpool"
)

func initDB() {
	var err error
	db, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	} else {
		log.Println("Connected to database successfully")
	}
}

func getVaultClient() (*api.Client, error) {
	config := api.DefaultConfig()
	err := config.ReadEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to read Vault environment: %w", err)
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	return client, nil
}
