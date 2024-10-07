package main

import (
	"os"
	"testing"
)

func TestSavePrivateKeyToVault(t *testing.T) {
	//mockClient := &mockVaultClient{}
	os.Setenv("VAULT_ADDR", "http://127.0.0.1:8200")
	os.Setenv("VAULT_TOKEN", "root")

	err := savePrivateKeyToVault("did:key:z6M", "mockPrivateKey")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestInitDB(t *testing.T) {
	// Example test for database initialization
	// Set environment variables for testing
	initDB()

	// Optionally, add additional tests or assertions here
	if db == nil {
		t.Error("Expected db to be initialized, got nil")
	}

	// Cleanup: Close the database connection
	defer db.Close()
}

// Define a mockVaultClient function
