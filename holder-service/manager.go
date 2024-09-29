package main

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

var credentialStore []VerifiableCredential

// StoreCredential adds a credential to the in-memory store
func StoreCredential(vc VerifiableCredential) {
	credentialStore = append(credentialStore, vc)
}

// GetStoredCredentials returns all stored credentials
func GetStoredCredentials() []VerifiableCredential {
	return credentialStore
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
