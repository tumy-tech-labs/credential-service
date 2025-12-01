package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/bradtumy/credential-service/internal/domain"
)

type response struct {
	OK      bool                   `json:"ok"`
	Subject string                 `json:"subject"`
	Claims  map[string]interface{} `json:"claims"`
}

func main() {
	port := envOrDefault("API_HTTP_PORT", "8082")

	mux := http.NewServeMux()
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"missing_token"}`))
			return
		}

		token := strings.TrimPrefix(authz, "Bearer ")

		payload, err := domain.DecodeSyntheticJWT(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}

		subject, _ := payload["sub"].(string)

		log.Printf("api-service received request from subject %s with claims %v", subject, payload)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{OK: true, Subject: subject, Claims: payload})
	})

	log.Printf("api-service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
