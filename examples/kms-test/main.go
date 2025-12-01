package main

import (
	"fmt"
	"log"

	"github.com/bradtumy/credential-service/internal/keystore"
)

func main() {
	fmt.Println("Testing production keystore functionality...")

	// Test memory backend
	store, err := keystore.NewProductionKeyStore(keystore.ProductionConfig{
		Backend: keystore.BackendMemory,
	})
	if err != nil {
		log.Fatalf("Failed to create memory keystore: %v", err)
	}

	signer, err := store.GetSigningKey("test-tenant")
	if err != nil {
		log.Fatalf("Failed to get signing key: %v", err)
	}

	fmt.Printf("✅ Memory keystore: Created signer with key type %T\n", signer.Public())

	// Test auto-detection
	autoStore, err := keystore.NewProductionKeyStoreFromEnv()
	if err != nil {
		log.Fatalf("Failed to create auto keystore: %v", err)
	}

	fmt.Printf("✅ Auto-detection: Backend selected: %s\n", autoStore.GetBackend())
	fmt.Println("🎉 Production keystore tests passed!")
}