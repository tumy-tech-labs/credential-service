package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
)

func TestIssuerIntegration(t *testing.T) {
	store := keystore.NewMemoryKeyStore()
	cfg := config.LoadIssuerConfigFromEnv()
	mux := http.NewServeMux()
	httpx.RegisterIssuerRoutes(mux, store, cfg, nil)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	reqBody := httpx.IssueRequest{
		SubjectDID: "did:jwk:subject-123",
		TTLSeconds: int64((10 * time.Minute).Seconds()),
		Claims:     map[string]interface{}{"scope": "test"},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(ts.URL+"/v1/credentials/issue", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to call issuer endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var issueResp httpx.IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if issueResp.Credential == "" {
		t.Fatalf("expected credential token")
	}
}
