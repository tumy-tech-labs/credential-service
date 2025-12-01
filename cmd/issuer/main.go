package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/storage"
	"github.com/bradtumy/credential-service/internal/tenant"
)

func main() {
	cfg := config.LoadIssuerConfigFromEnv()
	logging.Init(cfg.LogLevel)

	var auditStore storage.AuditStore
	if cfg.AuditDB_DSN != "" {
		db, err := sql.Open("postgres", cfg.AuditDB_DSN)
		if err != nil {
			log.Fatalf("Failed to connect to audit database: %v", err)
		}
		auditStore = storage.NewPGAuditStore(db)
		log.Printf("Using Postgres audit store")
	}

	// Use production keystore with automatic backend detection
	var store keystore.KeyStore
	if os.Getenv("USE_PRODUCTION_KEYSTORE") == "true" {
		productionStore, err := keystore.NewProductionKeyStoreFromEnv()
		if err != nil {
			log.Fatalf("Failed to initialize production keystore: %v", err)
		}
		store = productionStore
		log.Printf("Using production keystore with backend: %s", productionStore.GetBackend())
	} else {
		store = keystore.NewMemoryKeyStore()
		log.Printf("Using memory keystore (development mode)")
	}

	tenantStore := tenant.NewMemoryStore()
	_ = tenantStore.UpsertTenant(context.Background(), domain.Tenant{ID: cfg.DefaultTenantID, Name: "default", Enabled: true})

	resolver := tenant.Resolver{Mode: tenant.Mode(cfg.TenancyMode), DefaultTenantID: cfg.DefaultTenantID, Store: tenantStore}

	mux := http.NewServeMux()
	httpx.RegisterIssuerRoutes(mux, store, cfg, auditStore)
	httpx.RegisterBootstrapRoutes(mux, store, cfg, nil) // Add bootstrap endpoint
	httpx.RegisterHealthRoutes(mux, nil)

	// Use the new standard middleware chain with comprehensive audit logging
	handler := httpx.StandardMiddlewareChain()(httpx.TenantMiddleware(resolver, mux))

	log.Printf("Issuer service running on port %s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
