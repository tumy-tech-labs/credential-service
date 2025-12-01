package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type issueResponse struct {
	Credential string `json:"credential"`
}

type gatewayAuthzResponse struct {
	Allowed      bool                   `json:"allowed"`
	Subject      string                 `json:"subject"`
	Claims       map[string]interface{} `json:"claims"`
	SyntheticJWT string                 `json:"synthetic_jwt"`
	Reason       string                 `json:"reason"`
}

func main() {
	issuerURL := envOrDefault(
		[]string{"ISSUER_BASE_URL", "ISSUER_URL"},
		"http://localhost:8080",
	)
	gatewayURL := envOrDefault(
		[]string{"GATEWAY_AUTHORIZE_URL", "GATEWAY_URL"},
		"http://localhost:8081",
	)
	apiURL := envOrDefault(
		[]string{"API_BASE_URL", "API_URL"},
		"http://localhost:8082",
	)

	credential, err := issueCredential(issuerURL)
	if err != nil {
		log.Fatalf("issue credential: %v", err)
	}
	log.Printf("issued credential: %s", credential)

	decision, err := authorizeWithGateway(gatewayURL, credential)
	if err != nil {
		log.Fatalf("authorize: %v", err)
	}

	if !decision.Allowed {
		log.Fatalf("gateway denied request: %s", decision.Reason)
	}

	log.Printf("gateway authorized subject %s with claims %v", decision.Subject, decision.Claims)

	if decision.SyntheticJWT == "" {
		log.Fatalf("gateway did not return a synthetic jwt")
	}

	if err := callAPI(apiURL, decision.SyntheticJWT); err != nil {
		log.Fatalf("call api: %v", err)
	}

	log.Printf("client-service completed successfully")
}

func issueCredential(baseURL string) (string, error) {
	payload := map[string]interface{}{
		"subject_did": "did:example:client",
		"ttl_seconds": 300,
		"claims": map[string]interface{}{
			"scope": "read:orders",
			"aud":   "sample-api",
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/v1/credentials/issue", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(data))
	}

	var parsed issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Credential, nil
}

func authorizeWithGateway(baseURL, credential string) (gatewayAuthzResponse, error) {
	payload := map[string]interface{}{
		"credential":         credential,
		"expected_audience":  "sample-api",
		"want_synthetic_jwt": true,
		"resource":           "orders",
		"action":             "read",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/v1/gateway/authorize", "application/json", bytes.NewReader(body))
	if err != nil {
		return gatewayAuthzResponse{}, err
	}
	defer resp.Body.Close()

	var parsed gatewayAuthzResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return gatewayAuthzResponse{}, err
	}
	return parsed, nil
}

func callAPI(baseURL, token string) error {
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned %d: %s", resp.StatusCode, string(data))
	}

	log.Printf("api response: %s", string(data))
	return nil
}

func envOrDefault(keys []string, fallback string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return fallback
}
