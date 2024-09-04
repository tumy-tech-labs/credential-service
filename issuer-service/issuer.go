package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

func initDB() {
	var err error
	db, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	} else {
		log.Println("Connected to database successfully")
	}
}

func createCredential(w http.ResponseWriter, r *http.Request) {
	// Sample data for the credential
	credentialID := uuid.New().String()
	name := r.URL.Query().Get("name")
	email := r.URL.Query().Get("email")
	phone := r.URL.Query().Get("phone")
	issueDate := time.Now().UTC().Format(time.RFC3339)
	expirationDate := time.Now().Add(365 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Construct the VC as per W3C VC specification
	vc := map[string]interface{}{
		"@context": "https://www.w3.org/2018/credentials/v1",
		"type":     []string{"VerifiableCredential", "EmploymentCredential"},
		"id":       credentialID,
		"issuer":   "did:example:issuer", // Replace with actual DID
		"credentialSubject": map[string]string{
			"id":    "did:example:subject", // Replace with actual DID
			"name":  name,
			"email": email,
			"phone": phone,
		},
		"issuanceDate":   issueDate,
		"expirationDate": expirationDate,
	}

	// Sign the VC (stubbed logic)
	// In production, you would sign this with the issuer's private key
	signedVC := vc // Here you'd apply signing logic

	// Respond with the signed VC
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(signedVC)
	if err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to create VC", http.StatusInternalServerError)
		return
	}

	log.Printf("VC created successfully: %s", credentialID)
}

func main() {
	_ = godotenv.Load()
	initDB()

	http.HandleFunc("/credentials", createCredential)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("VC Issuance Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
