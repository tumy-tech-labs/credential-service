package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/jackc/pgx/v4/pgxpool"
)

var dbClosed bool = true // Track if the pool is closed

func initDB() (*pgxpool.Pool, error) {
	var err error
	db, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	} else {
		log.Println("Connected to database successfully")
	}
	dbClosed = false // Set to false when connected
	return db, nil   // Return the connection pool and no error
}

func closeDB() {
	if db != nil {
		db.Close()
		dbClosed = true // Mark as closed
		log.Println("Database connection closed")
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
