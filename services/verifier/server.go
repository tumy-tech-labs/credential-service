package verifier

import (
	"context"
	"crypto"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bradtumy/credential-service/internal/cache"
	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpserver"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/metrics"
	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/ratelimit"
	"github.com/bradtumy/credential-service/internal/storage"
	"github.com/bradtumy/credential-service/internal/tenant"
)

// Server bundles the HTTP server for the verifier and gateway endpoints.
type Server struct {
	srv *http.Server
}

// NewServer wires dependencies and returns a ready-to-run verifier HTTP server.
func NewServer(ctx context.Context, cfg config.VerifierConfig) (*Server, error) {
	logging.Init(cfg.LogLevel)

	var (
		trustRegistry domain.TrustRegistry = domain.NewMemoryTrustRegistry()
		db            *sql.DB
		policyStore   policy.Store = policy.NewMemoryStore()
	)

	tenancyMode := tenant.Mode(cfg.TenancyMode)
	var tenantStore tenant.Store = tenant.NewMemoryStore()

	if cfg.DB_DSN != "" {
		var err error
		db, err = sql.Open("postgres", cfg.DB_DSN)
		if err != nil {
			return nil, fmt.Errorf("connect to database: %w", err)
		}
		if cfg.UseDBTrustRegistry {
			trustRegistry = storage.NewPGTrustRegistry(db)
		}
		tenantStore = tenant.NewPGTenantStore(db)
		policyStore = policy.NewPGStore(db)
	}

	if err := tenantStore.UpsertTenant(ctx, domain.Tenant{ID: cfg.DefaultTenantID, Name: "default", Enabled: true}); err != nil {
		return nil, fmt.Errorf("seed default tenant: %w", err)
	}

	defaultPolicy := &policy.Policy{
		TenantID:    cfg.DefaultTenantID,
		Name:        "demo-read-orders",
		Description: "Allow reading orders for demo",
		Effect:      policy.EffectAllow,
		Actions:     []string{"read"},
		Resources:   []string{"orders"},
		Subjects:    []string{"any"},
		Priority:    100,
		Enabled:     true,
	}
	if err := policyStore.CreatePolicy(ctx, defaultPolicy); err != nil {
		log.Printf("warning: failed to seed default policy (may already exist): %v", err)
	} else {
		log.Printf("successfully created default policy: %s", defaultPolicy.Name)
	}

	tenantResolver := tenant.Resolver{Mode: tenancyMode, DefaultTenantID: cfg.DefaultTenantID, Store: tenantStore}

	didResolver, err := domain.NewCompositeResolver(
		domain.NewJWKResolver(),
		domain.NewWebResolver(),
	)
	if err != nil {
		return nil, fmt.Errorf("create DID resolver: %w", err)
	}

	resolver := func(issuer string) (crypto.PublicKey, error) {
		trusted, err := trustRegistry.IsTrustedIssuer(ctx, cfg.DefaultTenantID, issuer)
		if err != nil {
			return nil, fmt.Errorf("trust registry check failed: %w", err)
		}
		if !trusted {
			return nil, domain.ErrUntrustedIssuer
		}

		return didResolver.ResolvePublicKey(ctx, issuer)
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

	signer, err := store.GetSigningKey(cfg.DefaultTenantID)
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		return nil, fmt.Errorf("derive issuer did: %w", err)
	}

	if err := trustRegistry.AddTrustedIssuer(ctx, cfg.DefaultTenantID, issuerDID); err != nil {
		return nil, fmt.Errorf("seed trust registry: %w", err)
	}

	readiness := func(ctx context.Context) error {
		if db != nil {
			return db.PingContext(ctx)
		}
		return nil
	}

	metricsFactory := metrics.NewFactoryFromEnv()
	gatewayMetrics, err := metricsFactory.CreateGatewayMetrics()
	if err != nil {
		return nil, fmt.Errorf("create gateway metrics: %w", err)
	}
	verifierMetrics, err := metricsFactory.CreateVerifierMetrics()
	if err != nil {
		return nil, fmt.Errorf("create verifier metrics: %w", err)
	}

	log.Printf("Metrics enabled: %s (namespace: %s)", metricsFactory.GetMetricsType(), metricsFactory.GetNamespace())

	mux := http.NewServeMux()
	httpserver.RegisterVerifierRoutes(mux, resolver, trustRegistry, cfg.DefaultTenantID, verifierMetrics, time.Now)

	var decisionCache cache.DecisionCache = cache.NoopDecisionCache{}
	if cfg.GatewayCache && cfg.RedisAddr != "" {
		redisCache, err := cache.NewRedisDecisionCacheFromEnv()
		if err != nil {
			log.Printf("gateway cache disabled: %v", err)
		} else {
			decisionCache = redisCache
		}
	}

	var limiter ratelimit.Limiter = ratelimit.NoopLimiter{}
	if cfg.RateLimitEnabled {
		if redisLimiter, err := ratelimit.NewRedisLimiterFromEnv(); err != nil {
			log.Printf("rate limiting enabled but redis setup failed: %v, using noop limiter", err)
		} else {
			limiter = redisLimiter
			log.Printf("rate limiting enabled with Redis backend")
		}
	}

	policyEngine := policy.NewEngine(policyStore)
	httpserver.RegisterGatewayRoutes(mux, &httpserver.GatewayConfig{
		Resolver:        resolver,
		Registry:        trustRegistry,
		PolicyEngine:    policyEngine,
		DefaultTenantID: cfg.DefaultTenantID,
		SigningKey:      signer,
		JWTIssuer:       issuerDID,
		DecisionCache:   decisionCache,
		Limiter:         limiter,
		Metrics:         gatewayMetrics,
		Now:             time.Now,
	})

	httpserver.RegisterTrustRegistryRoutes(mux, trustRegistry, cfg.DefaultTenantID)

	adminMux := http.NewServeMux()
	httpserver.RegisterPolicyAdminRoutes(adminMux, policyStore, cfg.DefaultTenantID)

	adminMiddleware := httpserver.AdminAuthMiddleware(resolver, trustRegistry, cfg.DefaultTenantID)
	mux.Handle("/v1/admin/", adminMiddleware(adminMux))

	if metricsFactory.GetMetricsType() == metrics.PrometheusMetrics {
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("Prometheus metrics exposed at /metrics")
	}

	httpserver.RegisterHealthRoutes(mux, readiness)

	handler := httpserver.StandardMiddlewareChain()(httpserver.TenantMiddleware(tenantResolver, mux))

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: handler,
	}

	return &Server{srv: srv}, nil
}

// ListenAndServe starts the verifier HTTP server.
func (s *Server) ListenAndServe() error {
	err := s.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
