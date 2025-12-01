package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/version"
)

func TestPolicyAdminCRUD(t *testing.T) {
	store := policy.NewMemoryStore()
	mux := http.NewServeMux()
	RegisterPolicyAdminRoutes(mux, store, "tenant-1")

	createBody := PolicyRequest{
		Name:      "allow-read",
		Effect:    string(policy.EffectAllow),
		Actions:   []string{"read"},
		Resources: []string{"orders/*"},
		Subjects:  []string{"any"},
	}
	payload, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/policies", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var createResp PolicyResponse
	_ = json.NewDecoder(rr.Body).Decode(&createResp)
	if createResp.Policy.ID == 0 || createResp.APIVersion != version.APIVersion {
		t.Fatalf("unexpected create response: %+v", createResp)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/admin/policies", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRR.Code)
	}
	var listResp PoliciesResponse
	_ = json.NewDecoder(listRR.Body).Decode(&listResp)
	if len(listResp.Policies) != 1 {
		t.Fatalf("expected one policy, got %d", len(listResp.Policies))
	}

	policyID := createResp.Policy.ID
	updateBody := createBody
	updateBody.Name = "updated"
	updatePayload, _ := json.Marshal(updateBody)
	updateReq := httptest.NewRequest(http.MethodPut, "/v1/admin/policies/"+strconv.FormatInt(policyID, 10), bytes.NewReader(updatePayload))
	updateRR := httptest.NewRecorder()
	mux.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", updateRR.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/admin/policies/"+strconv.FormatInt(policyID, 10), nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d", getRR.Code)
	}
	var getResp PolicyResponse
	_ = json.NewDecoder(getRR.Body).Decode(&getResp)
	if getResp.Policy.Name != "updated" {
		t.Fatalf("expected updated name, got %s", getResp.Policy.Name)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/admin/policies/"+strconv.FormatInt(policyID, 10), nil)
	delRR := httptest.NewRecorder()
	mux.ServeHTTP(delRR, delReq)
	if delRR.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", delRR.Code)
	}
}

func TestPolicyAdminValidation(t *testing.T) {
	store := policy.NewMemoryStore()
	mux := http.NewServeMux()
	RegisterPolicyAdminRoutes(mux, store, "tenant-1")

	badBody := PolicyRequest{}
	payload, _ := json.Marshal(badBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/policies", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
