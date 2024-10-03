package main

import (
	"log"
)

// Factory function to create a new blockchain adapter
func NewBlockchainAdapter() BlockchainAdapter {
	// Create an instance of HyperledgerAdapter
	adapter, err := NewHyperledgerAdapter()
	if err != nil {
		log.Printf("Factory: failed to create Hyperledger adapter: %v", err)
		return nil
	}

	return adapter
}
