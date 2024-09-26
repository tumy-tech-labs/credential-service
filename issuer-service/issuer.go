package main

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

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
