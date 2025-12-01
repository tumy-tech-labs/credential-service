package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bradtumy/credential-service/internal/version"
)

// HealthResponse is used for health and readiness checks.
type HealthResponse struct {
	Status     string `json:"status"`
	APIVersion string `json:"api_version"`
}

// RegisterHealthRoutes attaches health and readiness endpoints to the mux.
func RegisterHealthRoutes(mux *http.ServeMux, readinessCheck func(context.Context) error) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeHealthOK(w)
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if readinessCheck != nil {
			if err := readinessCheck(r.Context()); err != nil {
				WriteAPIError(w, http.StatusServiceUnavailable, "not_ready", err.Error())
				return
			}
		}

		writeHealthOK(w)
	})
}

func writeHealthOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok", APIVersion: version.APIVersion})
}
