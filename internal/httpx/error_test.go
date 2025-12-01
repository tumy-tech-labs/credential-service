package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bradtumy/credential-service/internal/version"
)

func TestWriteAPIError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteAPIError(rr, http.StatusBadRequest, "bad_request", "invalid payload")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp APIError
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "bad_request" || resp.Description != "invalid payload" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if resp.APIVersion != version.APIVersion {
		t.Fatalf("expected api version %s, got %s", version.APIVersion, resp.APIVersion)
	}
}
