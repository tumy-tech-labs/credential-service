package main

import (
	"context"
	"crypto"
	"database/sql"
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
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/metrics"
	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/ratelimit"
	"github.com/bradtumy/credential-service/internal/storage"
	"github.com/bradtumy/credential-service/internal/tenant"
)

func main() {
	cfg := config.LoadVerifierConfigFromEnv()
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
			log.Fatalf("connect to database: %v", err)
		}
		if cfg.UseDBTrustRegistry {
			trustRegistry = storage.NewPGTrustRegistry(db)
		}
		tenantStore = tenant.NewPGTenantStore(db)
		policyStore = policy.NewPGStore(db)
	}

	if err := tenantStore.UpsertTenant(context.Background(), domain.Tenant{ID: cfg.DefaultTenantID, Name: "default", Enabled: true}); err != nil {
		log.Fatalf("seed default tenant: %v", err)
	}

	// Seed a default policy to allow the demo to work
	defaultPolicy := &policy.Policy{
		TenantID:    cfg.DefaultTenantID,
		Name:        "demo-read-orders",
		Description: "Allow reading orders for demo",
		Effect:      policy.EffectAllow,
		Actions:     []string{"read"},
		Resources:   []string{"orders"},
		Subjects:    []string{"any"}, // Allow any subject
		Priority:    100,
		Enabled:     true,
	}
	if err := policyStore.CreatePolicy(context.Background(), defaultPolicy); err != nil {
		log.Printf("warning: failed to seed default policy (may already exist): %v", err)
	} else {
		log.Printf("successfully created default policy: %s", defaultPolicy.Name)
	}

	tenantResolver := tenant.Resolver{Mode: tenancyMode, DefaultTenantID: cfg.DefaultTenantID, Store: tenantStore}

	// Create DID resolver for distributed public key resolution
	didResolver, err := domain.NewCompositeResolver(
		domain.NewJWKResolver(), // Support did:jwk method
		domain.NewWebResolver(), // Support did:web method
	)
	if err != nil {
		log.Fatalf("create DID resolver: %v", err)
	}

	// Create public key resolver that uses DID resolution and trust registry
	resolver := func(issuer string) (crypto.PublicKey, error) {
		// First check if issuer is trusted
		trusted, err := trustRegistry.IsTrustedIssuer(context.Background(), cfg.DefaultTenantID, issuer)
		if err != nil {
			return nil, fmt.Errorf("trust registry check failed: %w", err)
		}
		if !trusted {
			return nil, domain.ErrUntrustedIssuer
		}
		
		// Resolve public key from DID
		return didResolver.ResolvePublicKey(context.Background(), issuer)
	}

	// Initialize keystore - use production keystore if configured
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
	
	// For backward compatibility, add a default trusted issuer (can be removed later)
	signer, err := store.GetSigningKey(cfg.DefaultTenantID)
	if err != nil {
		log.Fatalf("get signing key: %v", err)
	}

	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		log.Fatalf("derive issuer did: %v", err)
	}

	if err := trustRegistry.AddTrustedIssuer(context.Background(), cfg.DefaultTenantID, issuerDID); err != nil {
		log.Fatalf("seed trust registry: %v", err)
	}

	readiness := func(ctx context.Context) error {
		if db != nil {
			return db.PingContext(ctx)
		}
		return nil
	}

	// Initialize metrics factory from environment
	metricsFactory := metrics.NewFactoryFromEnv()
	gatewayMetrics, err := metricsFactory.CreateGatewayMetrics()
	if err != nil {
		log.Fatalf("create gateway metrics: %v", err)
	}
	verifierMetrics, err := metricsFactory.CreateVerifierMetrics()
	if err != nil {
		log.Fatalf("create verifier metrics: %v", err)
	}

	log.Printf("Metrics enabled: %s (namespace: %s)", metricsFactory.GetMetricsType(), metricsFactory.GetNamespace())

	mux := http.NewServeMux()
	httpx.RegisterVerifierRoutes(mux, resolver, trustRegistry, cfg.DefaultTenantID, verifierMetrics, time.Now)
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
	httpx.RegisterGatewayRoutes(mux, &httpx.GatewayConfig{
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
	
	// Register trust registry management routes (development mode - no auth)
	httpx.RegisterTrustRegistryRoutes(mux, trustRegistry, cfg.DefaultTenantID)
	
	// Create protected admin routes
	adminMux := http.NewServeMux()
	httpx.RegisterPolicyAdminRoutes(adminMux, policyStore, cfg.DefaultTenantID)
	
	// Apply admin authentication middleware to all /v1/admin routes
	// Admin middleware also uses DID resolution for public key verification
	adminMiddleware := httpx.AdminAuthMiddleware(resolver, trustRegistry, cfg.DefaultTenantID)
	mux.Handle("/v1/admin/", adminMiddleware(adminMux))
	
	// Expose Prometheus metrics endpoint if enabled
	if metricsFactory.GetMetricsType() == metrics.PrometheusMetrics {
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("Prometheus metrics exposed at /metrics")
	}
	
	httpx.RegisterHealthRoutes(mux, readiness)

	// Use the new standard middleware chain with comprehensive audit logging
	handler := httpx.StandardMiddlewareChain()(httpx.TenantMiddleware(tenantResolver, mux))

	log.Printf("Verifier service running on port %s...", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, handler))
}
