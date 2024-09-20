package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid" // Import UUID package
	"github.com/jackc/pgx/v4/pgxpool"
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

func closeDB() {
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
	}
}

// CredentialPresentation represents the structure of a credential presentation
type CredentialPresentation struct {
	ID               string                 `json:"id"` // Unique identifier for the presentation
	CredentialID     string                 `json:"credentialId"`
	HolderDID        string                 `json:"holderDID"`
	PresentationData map[string]interface{} `json:"presentationData"`
	ProcessingID     string                 `json:"processingId"` // New field for processing ID
}

// handlePresentation handles incoming credential presentations
func handlePresentation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var presentation CredentialPresentation
	if err := json.NewDecoder(r.Body).Decode(&presentation); err != nil {
		http.Error(w, "Invalid presentation format", http.StatusBadRequest)
		return
	}

	// Generate a unique processing ID
	presentation.ProcessingID = uuid.New().String()

	// Process the presentation (e.g., verify the credentials)
	err := processPresentation(presentation)
	if err != nil {
		log.Printf("Processing Error %s", err)
		http.Error(w, "Failed to process presentation", http.StatusInternalServerError)
		return
	}

	// Save the presentation to the database
	if err := savePresentationToDB(presentation); err != nil {
		log.Printf("Failed to save presentation: %v", err)
		http.Error(w, "Failed to save presentation", http.StatusInternalServerError)
		return
	}

	// Respond with the processing ID
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "processingId": presentation.ProcessingID})
}

// savePresentationToDB saves the presentation to the database
func savePresentationToDB(presentation CredentialPresentation) error {
	query := `
        INSERT INTO presentations (credential_id, holder_did, presentation_data, processing_id)
        VALUES ($1, $2, $3, $4)
    `
	_, err := db.Exec(context.Background(), query, presentation.CredentialID, presentation.HolderDID, presentation.PresentationData, presentation.ProcessingID)
	return err
}

// processPresentation verifies the provided credential presentation
func processPresentation(presentation CredentialPresentation) error {
	log.Printf("Processing presentation for credential ID %s", presentation.CredentialID)

	// Example verification (replace with real logic)
	if presentation.CredentialID == "" || presentation.HolderDID == "" {
		return errors.New("invalid presentation data")
	}

	// Verification logic goes here...

	return nil
}

// getPresentation retrieves a specific presentation by ID
func getPresentation(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/presentations/"):]

	presentation, err := fetchPresentationByProcessingID(id) // You'd need to implement this function
	if err != nil {
		log.Printf("Failed to retrieve presentation: %v", err)
		http.Error(w, "Presentation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presentation)
}

// fetchPresentationByID retrieves a presentation by its ID from the database
func fetchPresentationByProcessingID(processingID string) (CredentialPresentation, error) {
	var presentation CredentialPresentation

	query := `
        SELECT credential_id, holder_did, presentation_data, processing_id
        FROM presentations
        WHERE processing_id = $1
    `

	row := db.QueryRow(context.Background(), query, processingID)

	err := row.Scan(&presentation.CredentialID, &presentation.HolderDID, &presentation.PresentationData, &presentation.ProcessingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return CredentialPresentation{}, fmt.Errorf("presentation not found")
		}
		return CredentialPresentation{}, fmt.Errorf("error retrieving presentation: %v", err)
	}

	return presentation, nil
}

func main() {
	initDB()
	defer closeDB() // Ensure the database is closed when the service shuts down

	http.HandleFunc("/presentations", handlePresentation)
	http.HandleFunc("/presentations/", getPresentation)

	port := "8080"
	log.Printf("Presentation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
