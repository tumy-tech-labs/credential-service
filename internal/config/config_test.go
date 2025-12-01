package config

import "testing"

func TestLoadIssuerConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("ISSUER_HTTP_PORT", "")
	t.Setenv("DEFAULT_TENANT_ID", "")
	t.Setenv("ISSUER_LOG_LEVEL", "")

	cfg := LoadIssuerConfigFromEnv()

	if cfg.HTTPPort != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.HTTPPort)
	}
	if cfg.DefaultTenantID != "default-tenant" {
		t.Fatalf("expected default tenant id, got %s", cfg.DefaultTenantID)
	}
	if cfg.TenancyMode != "single" {
		t.Fatalf("expected default tenancy mode single, got %s", cfg.TenancyMode)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.LogLevel)
	}
}

func TestLoadIssuerConfigFromEnvOverrides(t *testing.T) {
	t.Setenv("ISSUER_HTTP_PORT", "9090")
	t.Setenv("DEFAULT_TENANT_ID", "tenant-123")
	t.Setenv("ISSUER_LOG_LEVEL", "debug")
	t.Setenv("TENANCY_MODE", "multi")

	cfg := LoadIssuerConfigFromEnv()

	if cfg.HTTPPort != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.HTTPPort)
	}
	if cfg.DefaultTenantID != "tenant-123" {
		t.Fatalf("expected tenant-123, got %s", cfg.DefaultTenantID)
	}
	if cfg.TenancyMode != "multi" {
		t.Fatalf("expected tenancy mode override, got %s", cfg.TenancyMode)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected debug log level, got %s", cfg.LogLevel)
	}
}

func TestLoadVerifierConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("VERIFIER_HTTP_PORT", "")
	t.Setenv("DEFAULT_TENANT_ID", "")
	t.Setenv("VERIFIER_DB_DSN", "")
	t.Setenv("VERIFIER_USE_DB_TRUST_REGISTRY", "")
	t.Setenv("VERIFIER_LOG_LEVEL", "")
	t.Setenv("GATEWAY_CACHE_ENABLED", "")
	t.Setenv("RATELIMIT_ENABLED", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")

	cfg := LoadVerifierConfigFromEnv()

	if cfg.HTTPPort != "8081" {
		t.Fatalf("expected default verifier port 8081, got %s", cfg.HTTPPort)
	}
	if cfg.DefaultTenantID != "default-tenant" {
		t.Fatalf("expected default tenant id, got %s", cfg.DefaultTenantID)
	}
	if cfg.TenancyMode != "single" {
		t.Fatalf("expected default tenancy mode single, got %s", cfg.TenancyMode)
	}
	if cfg.DB_DSN != "" {
		t.Fatalf("expected empty DB DSN, got %s", cfg.DB_DSN)
	}
	if cfg.UseDBTrustRegistry {
		t.Fatalf("expected DB trust registry disabled by default")
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.LogLevel)
	}
	if cfg.GatewayCache {
		t.Fatalf("expected gateway cache disabled by default")
	}
	if cfg.RateLimitEnabled {
		t.Fatalf("expected rate limiting disabled by default")
	}
	if cfg.RedisAddr != "" || cfg.RedisPassword != "" || cfg.RedisDB != 0 {
		t.Fatalf("expected redis settings to be empty by default")
	}
}

func TestLoadVerifierConfigFromEnvOverrides(t *testing.T) {
	t.Setenv("VERIFIER_HTTP_PORT", "7070")
	t.Setenv("DEFAULT_TENANT_ID", "tenant-verifier")
	t.Setenv("VERIFIER_DB_DSN", "postgres://example")
	t.Setenv("VERIFIER_USE_DB_TRUST_REGISTRY", "true")
	t.Setenv("VERIFIER_LOG_LEVEL", "warn")
	t.Setenv("TENANCY_MODE", "multi")
	t.Setenv("GATEWAY_CACHE_ENABLED", "true")
	t.Setenv("RATELIMIT_ENABLED", "true")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("REDIS_DB", "2")

	cfg := LoadVerifierConfigFromEnv()

	if cfg.HTTPPort != "7070" {
		t.Fatalf("expected verifier port 7070, got %s", cfg.HTTPPort)
	}
	if cfg.DefaultTenantID != "tenant-verifier" {
		t.Fatalf("expected tenant-verifier, got %s", cfg.DefaultTenantID)
	}
	if cfg.TenancyMode != "multi" {
		t.Fatalf("expected multi tenancy mode, got %s", cfg.TenancyMode)
	}
	if cfg.DB_DSN != "postgres://example" {
		t.Fatalf("expected DSN override, got %s", cfg.DB_DSN)
	}
	if !cfg.UseDBTrustRegistry {
		t.Fatalf("expected DB trust registry enabled")
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("expected warn log level, got %s", cfg.LogLevel)
	}
	if !cfg.GatewayCache {
		t.Fatalf("expected gateway cache enabled")
	}
	if !cfg.RateLimitEnabled {
		t.Fatalf("expected rate limiting enabled")
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "secret" || cfg.RedisDB != 2 {
		t.Fatalf("expected redis settings to propagate, got %+v", cfg)
	}
}
