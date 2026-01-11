package issuer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpserver"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/storage"
	"github.com/bradtumy/credential-service/internal/tenant"
)

// Server bundles the HTTP server for the issuer service.
type Server struct {
	srv *http.Server
}

// NewServer wires dependencies and returns a ready-to-run issuer HTTP server.
func NewServer(ctx context.Context, cfg config.IssuerConfig) (*Server, error) {
	logging.Init(cfg.LogLevel)

	var auditStore storage.AuditStore
	if cfg.AuditDB_DSN != "" {
		db, err := sql.Open("postgres", cfg.AuditDB_DSN)
		if err != nil {
			return nil, err
		}
		auditStore = storage.NewPGAuditStore(db)
		log.Printf("Using Postgres audit store")
	}

	var store keystore.KeyStore
	if os.Getenv("USE_PRODUCTION_KEYSTORE") == "true" {
		productionStore, err := keystore.NewProductionKeyStoreFromEnv()
		if err != nil {
			return nil, err
		}
		store = productionStore
		log.Printf("Using production keystore with backend: %s", productionStore.GetBackend())
	} else {
		fileStore, err := keystore.NewFileBackedKeyStoreFromEnv()
		if err != nil {
			log.Printf("Using memory keystore (development mode, file store unavailable: %v)", err)
			store = keystore.NewMemoryKeyStore()
		} else {
			store = fileStore
			log.Printf("Using file-backed keystore at %s", fileStore.Path())
		}
	}

	tenantStore := tenant.NewMemoryStore()
	_ = tenantStore.UpsertTenant(ctx, domain.Tenant{ID: cfg.DefaultTenantID, Name: "default", Enabled: true})
	resolver := tenant.Resolver{Mode: tenant.Mode(cfg.TenancyMode), DefaultTenantID: cfg.DefaultTenantID, Store: tenantStore}

	mux := http.NewServeMux()
	httpserver.RegisterIssuerRoutes(mux, store, cfg, auditStore)
	httpserver.RegisterBootstrapRoutes(mux, store, cfg, nil)
	httpserver.RegisterHealthRoutes(mux, nil)

	handler := httpserver.StandardMiddlewareChain()(httpserver.TenantMiddleware(resolver, mux))

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: handler,
	}

	return &Server{srv: srv}, nil
}

// ListenAndServe starts the issuer HTTP server.
func (s *Server) ListenAndServe() error {
	err := s.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
