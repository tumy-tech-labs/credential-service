package main

// BlockchainAdapter defines the interface for a blockchain adapter
type BlockchainAdapter interface {
	AnchorDID(did string) (string, error)
	AnchorCredential(credential string) (string, error) // Changed to return (string, error)
	Close() error
}
