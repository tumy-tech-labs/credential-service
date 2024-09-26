package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx"
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

// Define the structure for the DID Document
type PublicKey struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Controller      string `json:"controller"`
	PublicKeyBase58 string `json:"publicKeyBase58"`
}

type DIDDocument struct {
	Context        string      `json:"@context"`
	ID             string      `json:"id"`
	CreatedAt      string      `json:"createdAt"`
	PublicKey      []PublicKey `json:"publicKey"`
	OrganizationID string      `json:"organization_id"`
}

// Resolve the DID Document
func resolveDID(ctx context.Context, did string) (DIDDocument, error) {
	var didDocument DIDDocument
	var organizationID string

	// Query the database for the DID document
	query := "SELECT document, organization_id FROM dids WHERE did = $1"
	err := db.QueryRow(ctx, query, did).Scan(&didDocument, &organizationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DIDDocument{}, fmt.Errorf("DID not found: %s", did)
		}
		return DIDDocument{}, err
	}

	// Set the organization ID
	didDocument.OrganizationID = organizationID
	// Update the created time if needed (optional)
	didDocument.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	return didDocument, nil
}
