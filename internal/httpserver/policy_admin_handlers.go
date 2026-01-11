package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/version"
)

type PolicyRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Effect      string         `json:"effect"`
	Actions     []string       `json:"actions"`
	Resources   []string       `json:"resources"`
	Subjects    []string       `json:"subjects"`
	Conditions  map[string]any `json:"conditions,omitempty"`
	Priority    *int           `json:"priority,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
}

type PolicyResponse struct {
	Policy     policy.Policy `json:"policy"`
	APIVersion string        `json:"api_version"`
}

type PoliciesResponse struct {
	Policies   []policy.Policy `json:"policies"`
	APIVersion string          `json:"api_version"`
}

// RegisterPolicyAdminRoutes wires admin policy CRUD handlers.
func RegisterPolicyAdminRoutes(mux *http.ServeMux, store policy.Store, defaultTenantID string) {
	mux.HandleFunc("/v1/admin/policies", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/admin/policies" {
			WriteAPIError(w, http.StatusNotFound, "not_found", "not found")
			return
		}

		switch r.Method {
		case http.MethodPost:
			handleCreatePolicy(w, r, store, defaultTenantID)
		case http.MethodGet:
			handleListPolicies(w, r, store, defaultTenantID)
		default:
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})

	mux.HandleFunc("/v1/admin/policies/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/v1/admin/policies/")
		if idStr == "" {
			WriteAPIError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			WriteAPIError(w, http.StatusBadRequest, "invalid_request", "invalid policy id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleGetPolicy(w, r, store, defaultTenantID, id)
		case http.MethodPut:
			handleUpdatePolicy(w, r, store, defaultTenantID, id)
		case http.MethodDelete:
			handleDeletePolicy(w, r, store, defaultTenantID, id)
		default:
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func handleCreatePolicy(w http.ResponseWriter, r *http.Request, store policy.Store, defaultTenantID string) {
	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	p, err := buildPolicyFromRequest(req, TenantIDFromContext(r.Context()), defaultTenantID)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := store.CreatePolicy(r.Context(), &p); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "policy_error", err.Error())
		return
	}

	writePolicyResponse(w, p)
}

func handleListPolicies(w http.ResponseWriter, r *http.Request, store policy.Store, defaultTenantID string) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	policies, err := store.ListPoliciesForTenant(r.Context(), tenantID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "policy_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PoliciesResponse{Policies: policies, APIVersion: version.APIVersion})
}

func handleGetPolicy(w http.ResponseWriter, r *http.Request, store policy.Store, defaultTenantID string, id int64) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	p, err := store.GetPolicy(r.Context(), tenantID, id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == policy.ErrPolicyNotFound {
			status = http.StatusNotFound
			WriteAPIError(w, status, "not_found", "policy not found")
			return
		}
		WriteAPIError(w, status, "policy_error", err.Error())
		return
	}

	writePolicyResponse(w, p)
}

func handleUpdatePolicy(w http.ResponseWriter, r *http.Request, store policy.Store, defaultTenantID string, id int64) {
	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	p, err := buildPolicyFromRequest(req, TenantIDFromContext(r.Context()), defaultTenantID)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	p.ID = id

	if err := store.UpdatePolicy(r.Context(), &p); err != nil {
		status := http.StatusInternalServerError
		if err == policy.ErrPolicyNotFound {
			status = http.StatusNotFound
			WriteAPIError(w, status, "not_found", "policy not found")
			return
		}
		WriteAPIError(w, status, "policy_error", err.Error())
		return
	}

	writePolicyResponse(w, p)
}

func handleDeletePolicy(w http.ResponseWriter, r *http.Request, store policy.Store, defaultTenantID string, id int64) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	if err := store.DeletePolicy(r.Context(), tenantID, id); err != nil {
		status := http.StatusInternalServerError
		if err == policy.ErrPolicyNotFound {
			status = http.StatusNotFound
			WriteAPIError(w, status, "not_found", "policy not found")
			return
		}
		WriteAPIError(w, status, "policy_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func buildPolicyFromRequest(req PolicyRequest, tenantID, defaultTenant string) (policy.Policy, error) {
	if tenantID == "" {
		tenantID = defaultTenant
	}
	if req.Name == "" {
		return policy.Policy{}, errField("name is required")
	}
	eff := strings.ToLower(req.Effect)
	if eff != string(policy.EffectAllow) && eff != string(policy.EffectDeny) {
		return policy.Policy{}, errField("effect must be allow or deny")
	}
	if len(req.Actions) == 0 || len(req.Resources) == 0 || len(req.Subjects) == 0 {
		return policy.Policy{}, errField("actions, resources, and subjects are required")
	}

	priority := 100
	if req.Priority != nil {
		priority = *req.Priority
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	return policy.Policy{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Effect:      policy.Effect(eff),
		Actions:     req.Actions,
		Resources:   req.Resources,
		Subjects:    req.Subjects,
		Conditions:  req.Conditions,
		Priority:    priority,
		Enabled:     enabled,
	}, nil
}

func writePolicyResponse(w http.ResponseWriter, p policy.Policy) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PolicyResponse{Policy: p, APIVersion: version.APIVersion})
}

func errField(msg string) error {
	return &policyError{msg: msg}
}

type policyError struct {
	msg string
}

func (e *policyError) Error() string { return e.msg }
