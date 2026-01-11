package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/version"
)

func TestIssueHandlerSuccess(t *testing.T) {
	// Initialize logging for tests
	logging.Init("info")
	logging.AuditLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	mux := http.NewServeMux()
	RegisterIssuerRoutes(mux, store, cfg, nil)

	body := IssueRequest{
		SubjectDID: "did:jwk:subject",
		TTLSeconds: 600,
		Claims:     map[string]interface{}{"role": "agent"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/credentials/issue", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var resp IssueResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Credential == "" {
		t.Fatalf("expected credential in response")
	}

	if resp.APIVersion != version.APIVersion {
		t.Fatalf("expected api version %s, got %s", version.APIVersion, resp.APIVersion)
	}

	if !strings.Contains(resp.Credential, ".") {
		t.Fatalf("expected jwt-like token")
	}
}

func TestIssueHandlerMissingSubject(t *testing.T) {
	// Initialize logging for tests
	logging.Init("info")
	logging.AuditLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	mux := http.NewServeMux()
	RegisterIssuerRoutes(mux, store, cfg, nil)

	body := IssueRequest{TTLSeconds: int64((10 * time.Minute).Seconds())}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/credentials/issue", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var errResp APIError
	if err := json.NewDecoder(recorder.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error == "" || errResp.APIVersion != version.APIVersion {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestDelegateHandlerSuccess(t *testing.T) {
	// Initialize logging for tests
	logging.Init("info")
	logging.AuditLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	mux := http.NewServeMux()
	RegisterIssuerRoutes(mux, store, cfg, nil)

	signer, err := store.GetSigningKey(cfg.DefaultTenantID)
	if err != nil {
		t.Fatalf("get signing key: %v", err)
	}
	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		t.Fatalf("did from key: %v", err)
	}

	parentToken, err := domain.IssueBasicCredential(issuerDID, "did:jwk:parent", signer, 10*time.Minute, map[string]interface{}{"scope": []string{"read", "write"}})
	if err != nil {
		t.Fatalf("issue parent: %v", err)
	}

	body := DelegateRequest{
		ParentCredential: parentToken,
		DelegateDID:      "did:jwk:agent",
		Scope:            []string{"read"},
		TTLSeconds:       int64((5 * time.Minute).Seconds()),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/credentials/delegate", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var resp IssueResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Credential == "" {
		t.Fatalf("expected delegated credential")
	}

	if resp.APIVersion != version.APIVersion {
		t.Fatalf("expected api version %s, got %s", version.APIVersion, resp.APIVersion)
	}
}

func TestDelegateHandlerScopeExpansion(t *testing.T) {
	// Initialize logging for tests
	logging.Init("info")
	logging.AuditLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	mux := http.NewServeMux()
	RegisterIssuerRoutes(mux, store, cfg, nil)

	signer, _ := store.GetSigningKey(cfg.DefaultTenantID)
	issuerDID, _ := domain.DIDFromPublicKey(signer.Public())
	parentToken, _ := domain.IssueBasicCredential(issuerDID, "did:jwk:parent", signer, 10*time.Minute, map[string]interface{}{"scope": []string{"read"}})

	body := DelegateRequest{
		ParentCredential: parentToken,
		DelegateDID:      "did:jwk:agent",
		Scope:            []string{"read", "write"},
		TTLSeconds:       int64((5 * time.Minute).Seconds()),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/credentials/delegate", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestDelegateHandlerTTLExpansion(t *testing.T) {
	// Initialize logging for tests
	logging.Init("info")
	logging.AuditLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	mux := http.NewServeMux()
	RegisterIssuerRoutes(mux, store, cfg, nil)

	signer, _ := store.GetSigningKey(cfg.DefaultTenantID)
	issuerDID, _ := domain.DIDFromPublicKey(signer.Public())
	parentToken, _ := domain.IssueBasicCredential(issuerDID, "did:jwk:parent", signer, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}})

	body := DelegateRequest{
		ParentCredential: parentToken,
		DelegateDID:      "did:jwk:agent",
		Scope:            []string{"read"},
		TTLSeconds:       int64((10 * time.Minute).Seconds()),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/credentials/delegate", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}
