package main

var credentialStore []VerifiableCredential

// StoreCredential adds a credential to the in-memory store
func StoreCredential(vc VerifiableCredential) {
	credentialStore = append(credentialStore, vc)
}

// GetStoredCredentials returns all stored credentials
func GetStoredCredentials() []VerifiableCredential {
	return credentialStore
}
