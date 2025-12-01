package crypto

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// VaultSigner implements Signer backed by HashiCorp Vault transit engine.
type VaultSigner struct {
	client      *http.Client
	vaultAddr   string
	vaultToken  string
	keyName     string
	mountPath   string
	keyVersion  int
	publicKey   crypto.PublicKey
	initialized bool
}

// NewVaultSignerFromEnv creates a Vault-backed signer from environment variables.
// Required environment variables:
// - VAULT_ADDR: Vault server address
// - VAULT_TOKEN: Authentication token
// - VAULT_KEY_NAME: Name of the key in Vault transit engine
func NewVaultSignerFromEnv() (*VaultSigner, error) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		return nil, fmt.Errorf("VAULT_ADDR environment variable is required")
	}

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is required")
	}

	keyName := os.Getenv("VAULT_KEY_NAME")
	if keyName == "" {
		return nil, fmt.Errorf("VAULT_KEY_NAME environment variable is required")
	}

	mountPath := os.Getenv("VAULT_MOUNT_PATH")
	if mountPath == "" {
		mountPath = "transit" // Default mount path
	}

	signer := &VaultSigner{
		client:      &http.Client{},
		vaultAddr:   strings.TrimSuffix(vaultAddr, "/"),
		vaultToken:  vaultToken,
		keyName:     keyName,
		mountPath:   mountPath,
		initialized: false,
	}

	// Initialize the signer by creating or loading the key
	if err := signer.initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize Vault signer: %w", err)
	}

	return signer, nil
}

// initialize sets up the Vault signer by creating or retrieving the key.
func (v *VaultSigner) initialize(ctx context.Context) error {
	if v.initialized {
		return nil
	}

	// Try to get existing key info first
	keyInfo, err := v.getKeyInfo(ctx)
	if err != nil {
		// If key doesn't exist, create it
		if strings.Contains(err.Error(), "no key found") || strings.Contains(err.Error(), "404") {
			if err := v.createKey(ctx); err != nil {
				return fmt.Errorf("failed to create Vault key: %w", err)
			}
			// Get key info after creation
			keyInfo, err = v.getKeyInfo(ctx)
			if err != nil {
				return fmt.Errorf("failed to get key info after creation: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get key info: %w", err)
		}
	}

	// Extract public key from key info
	publicKey, keyVersion, err := v.extractPublicKey(keyInfo)
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	v.publicKey = publicKey
	v.keyVersion = keyVersion
	v.initialized = true

	return nil
}

// createKey creates a new Ed25519 key in Vault transit engine.
func (v *VaultSigner) createKey(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/%s/keys/%s", v.vaultAddr, v.mountPath, v.keyName)

	payload := map[string]interface{}{
		"type": "ed25519",
	}

	return v.makeVaultRequest(ctx, "POST", url, payload, nil)
}

// getKeyInfo retrieves key information from Vault.
func (v *VaultSigner) getKeyInfo(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/%s/keys/%s", v.vaultAddr, v.mountPath, v.keyName)

	var response map[string]interface{}
	if err := v.makeVaultRequest(ctx, "GET", url, nil, &response); err != nil {
		return nil, err
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return data, nil
}

// extractPublicKey extracts the public key from Vault key info response.
func (v *VaultSigner) extractPublicKey(keyInfo map[string]interface{}) (crypto.PublicKey, int, error) {
	keys, ok := keyInfo["keys"].(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid key data structure")
	}

	// Get the latest key version
	var publicKeyB64 string
	for _, keyData := range keys {
		if keyInfo, ok := keyData.(map[string]interface{}); ok {
			if pubKey, exists := keyInfo["public_key"].(string); exists {
				publicKeyB64 = pubKey
				break
			}
		}
	}

	if publicKeyB64 == "" {
		return nil, 0, fmt.Errorf("no public key found in key data")
	}

	// Decode the base64 public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode public key: %w", err)
	}

	// For Ed25519, the public key should be 32 bytes
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, 0, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)
	
	return publicKey, 1, nil // Default to version 1 for simplicity
}

// Public returns the public key.
func (v *VaultSigner) Public() crypto.PublicKey {
	if !v.initialized {
		return nil
	}
	return v.publicKey
}

// Sign signs the given payload using Vault transit engine.
func (v *VaultSigner) Sign(rand io.Reader, payload []byte, opts crypto.SignerOpts) ([]byte, error) {
	if !v.initialized {
		return nil, fmt.Errorf("vault signer not initialized")
	}

	// Encode payload as base64 for Vault
	input := base64.StdEncoding.EncodeToString(payload)

	url := fmt.Sprintf("%s/v1/%s/sign/%s", v.vaultAddr, v.mountPath, v.keyName)

	requestPayload := map[string]interface{}{
		"input": input,
	}

	var response map[string]interface{}
	if err := v.makeVaultRequest(context.Background(), "POST", url, requestPayload, &response); err != nil {
		return nil, fmt.Errorf("vault sign request failed: %w", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid sign response format")
	}

	signature, ok := data["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("no signature in response")
	}

	// Vault returns signature in format "vault:v1:base64-signature"
	// Extract just the base64 signature part
	parts := strings.Split(signature, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid signature format")
	}

	// Decode the signature
	signatureBytes, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	return signatureBytes, nil
}

// makeVaultRequest makes an HTTP request to Vault API.
func (v *VaultSigner) makeVaultRequest(ctx context.Context, method, url string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonPayload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.vaultToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}