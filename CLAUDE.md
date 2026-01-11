# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Core Commands

### Building and Development
- `make build-all` - Build all services (issuer, verifier, keygen)
- `make keygen` - Build just the keygen CLI tool for generating DIDs
- `make idctl` - Build the idctl CLI tool
- `go run ./cmd/issuer` - Run issuer locally (requires env vars)
- `go run ./cmd/verifier` - Run verifier locally (requires env vars)

### Testing
- `make test` or `make test-unit` - Run unit tests for internal/ and cmd/
- `make test-coverage` - Generate coverage report (outputs coverage.html)
- `make test-integration` - Run integration tests in test/integration/
- `make test-e2e` - Run end-to-end tests with e2e build tag
- `go test ./internal/domain/... -v` - Run tests for a specific package
- `make ci` - Run full CI suite (lint, test-unit, test-integration, test-e2e, sec)

### Docker Environment
- `make dev-up` or `docker compose up --build -d` - Start all services (issuer:8080, verifier:8081, postgres:5432, s2s-api:8082)
- `make dev-down` or `docker compose down` - Stop services
- `make dev-reset` or `docker compose down -v && docker compose up -d --build` - Reset environment with fresh volumes

### Code Quality
- `make lint` - Run golangci-lint
- `make sec` - Run gosec security scanner
- `make clean` - Remove build artifacts (bin/, coverage files)

### Single Test Execution
```bash
# Run a specific test function
go test ./internal/domain -run TestCredentialIssuance -v

# Run tests with race detector
go test ./internal/... -race -count=1
```

## Architecture Overview

This is a Go-based W3C Verifiable Credentials service with three main components:

### Service Architecture
```
cmd/                      # Binary entrypoints
  issuer/                # Issues VCs (JWT and SD-JWT formats)
  verifier/              # Verifies VCs, evaluates policies, mints synthetic JWTs
  keygen/                # DID generation utility

services/                 # Service initialization and wiring
  issuer/                # HTTP server setup for issuer
  verifier/              # HTTP server setup for verifier + gateway

internal/                 # Shared libraries (not SDK-importable)
  domain/                # Core VC/DID logic, delegation, trust registry
  httpserver/            # HTTP handlers, middleware, route registration
  crypto/                # Key management, signers (local, Vault, KMS)
  keystore/              # Pluggable keystore backends
  policy/                # RBAC/ABAC policy engine
  storage/               # Postgres adapters for trust, policies, tenants
  config/                # Environment-driven configuration
  cache/, ratelimit/, metrics/, tenant/, logging/, trust/, jwtutil/

sdk/                      # Client SDKs
  go/                    # Go SDK (module path: github.com/bradtumy/credential-service/sdk/go)
  node/                  # Node.js SDK (@id2me/credential-sdk)
  python/                # Python SDK (credential-service-sdk)
```

### Component Responsibilities

**Issuer Service (port 8080)**
- Issues W3C VC-JWT compliant credentials with `typ: "vc+jwt"`
- Supports SD-JWT (Selective Disclosure JWT) format
- Handles credential delegation with scope/TTL constraints
- Bootstrap endpoint for one-time admin credential creation
- Audit logging (structured logs + optional Postgres persistence)
- DID generation endpoint (`POST /v1/keys/generate`)

**Verifier & Gateway Service (port 8081)**
- Verifies credential signatures via DID resolution (did:jwk, did:web)
- Enforces tenant-scoped trust registries (lists of trusted issuer DIDs)
- Evaluates authorization policies (hybrid RBAC/ABAC)
- Mints synthetic JWTs for legacy API compatibility
- Admin endpoints for managing trust registry and policies
- Health/readiness probes, optional Prometheus metrics

**Sample API Service (port 8082)**
- Demonstrates protected API using synthetic JWTs
- Located in samples/ directory

### Key Architectural Patterns

**DID-Based Trust Model**
- No pre-shared keys required between issuer and verifier
- Verifiers resolve DIDs at runtime to fetch public keys
- Trust registries maintain per-tenant lists of trusted issuer DIDs
- Supports did:jwk (self-contained) and did:web (DNS-hosted)

**Credential Delegation**
- Root credentials issued by trusted issuers
- Agents can receive delegated credentials with constrained scope/TTL
- Delegation depth tracked and validated during verification
- Least-privilege enforcement: child credentials must narrow permissions

**Multi-Tenancy**
- Single-tenant mode (default): Uses DEFAULT_TENANT_ID automatically
- Multi-tenant mode: Requires explicit X-Tenant-ID header
- Trust registries, policies, and keys scoped per tenant
- Tenant middleware resolves tenant early in request pipeline

**Policy Engine (internal/policy/)**
- Hybrid RBAC/ABAC authorization model
- Policies define: effect (allow/deny), actions, resources, subjects, conditions
- Priority-based evaluation (deny overrides allow)
- Subject matching supports DIDs, role:name patterns, or "any"
- Conditions support scope_contains and claim_equals

**Gateway Pattern (internal/httpserver/gateway_handlers.go)**
- `/v1/gateway/authorize` endpoint for external gateways (Envoy, NGINX)
- Verifies credential chains, evaluates policies, returns allow/deny decisions
- Optional synthetic JWT minting for legacy downstream services
- Returns decision with subject, acting-on-behalf-of, claims, policy_id

## Critical Implementation Details

### Credential Format
Credentials follow W3C VC-JWT specification:
- JWT header: `{"alg":"EdDSA","typ":"vc+jwt"}`
- JWT payload has standard claims (iss, sub, iat, exp) at top level
- VC structure nested under "vc" claim per W3C spec
- kid computed using RFC 7638 JWK Thumbprint

### DID Resolution (internal/domain/did_resolver.go)
- Built-in resolvers for did:jwk and did:web
- Resolution happens at verification time (no caching by default)
- Verifier validates signature against resolved public key
- Trust registry check happens after DID resolution

### Trust Registry Flow
1. Extract issuer DID from credential
2. Resolve tenant from X-Tenant-ID or default
3. Query trust registry for tenant + issuer DID pair
4. Deny if issuer not trusted for this tenant
5. Proceed to signature verification and policy evaluation

### SD-JWT Support (internal/domain/vc.go)
- Issuer salts individual claims and generates disclosures
- Format: `format: "sd-jwt"` in issue request
- Response includes credential + disclosures array
- Verifier reconstructs claims from provided disclosures
- Selective disclosure allows revealing subset of claims

### Configuration (internal/config/config.go)
Services configured entirely via environment variables:
- ISSUER_HTTP_PORT, VERIFIER_HTTP_PORT
- DEFAULT_TENANT_ID (required for single-tenant mode)
- VERIFIER_DB_DSN (Postgres connection string)
- VERIFIER_USE_DB_TRUST_REGISTRY (true/false)
- ISSUER_LOG_LEVEL, VERIFIER_LOG_LEVEL (debug/info/warn/error)

### Signing Key Management
- Default: ephemeral Ed25519 keys generated at startup
- Production: Vault or Google Cloud KMS integration available
- Key rotation supported (internal/crypto/rotation.go)
- Signers abstracted via crypto.Signer interface

## Common Development Patterns

### Adding a New HTTP Handler
1. Define handler in `internal/httpserver/` (e.g., `foo_handlers.go`)
2. Add route registration in `services/{issuer|verifier}/`
3. Wire dependencies through Service struct
4. Add tests in `internal/httpserver/foo_handlers_test.go`
5. Use tenant middleware for multi-tenant endpoints

### Modifying Policy Evaluation
1. Policy types defined in `internal/policy/engine.go`
2. Evaluation logic in `EvaluatePolicy()` function
3. Admin API handlers in `internal/httpserver/policy_admin_handlers.go`
4. Postgres storage in `internal/storage/pg_policy.go`
5. Update tests in `internal/policy/` and `internal/httpserver/`

### Adding DID Method Support
1. Implement Resolver interface in `internal/domain/did_resolver.go`
2. Add resolver to registry in initialization code
3. Update DID parsing logic if needed
4. Add integration tests in `internal/domain/did_resolver_test.go`

### Working with Storage Layer
- Abstractions defined in `internal/storage/`
- Postgres implementations use jackc/pgx/v4
- In-memory implementations for testing
- Migration scripts in `db/init.sql`
- Connection pooling handled by pgxpool

## Testing Patterns

### Unit Tests
- Mock dependencies using interfaces (storage, trust registry, etc.)
- Use testify/assert and testify/mock
- Table-driven tests preferred
- Example: `internal/domain/vc_test.go`

### Integration Tests
- Located in `test/integration/`
- May use in-memory stores or test databases
- Focus on handler-level integration

### E2E Tests
- Located in `tests/e2e`
- Require `-tags=e2e` build tag
- Can use in-process services or point to running instances via env vars:
  - E2E_ISSUER_URL
  - E2E_VERIFIER_URL
  - E2E_API_URL

### Test Data Generation
- Use `make keygen && ./bin/keygen -did-only` for test DIDs
- Ed25519 and ES256 algorithms supported
- Private keys can be saved with `-output` flag

## Quick Development Workflow

### Local Development (with Docker)
```bash
# Start services
docker compose up --build -d

# Generate test DID
make keygen
ALICE_DID=$(./bin/keygen -did-only)

# Issue credential
VC=$(curl -s -X POST http://localhost:8080/v1/credentials/issue \
  -H "Content-Type: application/json" \
  -d '{"subject_did":"'$ALICE_DID'","ttl_seconds":600,"claims":{"aud":"sample-api","scope":"read:orders"}}' \
  | jq -r .credential)

# Extract issuer DID and add to trust registry
ISSUER_DID=$(docker logs credential-service-issuer-1 2>&1 | grep "issuer_did" | tail -1 | sed 's/.*issuer_did=\([^ ]*\).*/\1/')
curl -X POST http://localhost:8081/v1/trust/issuers \
  -H "Content-Type: application/json" \
  -d '{"issuer_did":"'$ISSUER_DID'"}'

# Authorize and get synthetic JWT
curl -s -X POST http://localhost:8081/v1/gateway/authorize \
  -H "Content-Type: application/json" \
  -d '{"credential":"'$VC'","expected_audience":"sample-api","want_synthetic_jwt":true,"resource":"orders","action":"read"}' | jq
```

### Local Development (without Docker)
```bash
# Terminal 1: Start Postgres
docker run -d -p 5432:5432 -e POSTGRES_USER=credentialsvc -e POSTGRES_PASSWORD=credentialsvc -e POSTGRES_DB=credentialsvc postgres:16

# Terminal 2: Start issuer
export ISSUER_HTTP_PORT=8080
export DEFAULT_TENANT_ID=default-tenant
export ISSUER_LOG_LEVEL=debug
go run ./cmd/issuer

# Terminal 3: Start verifier
export VERIFIER_HTTP_PORT=8081
export DEFAULT_TENANT_ID=default-tenant
export VERIFIER_LOG_LEVEL=debug
export VERIFIER_DB_DSN=postgres://credentialsvc:credentialsvc@localhost:5432/credentialsvc?sslmode=disable
export VERIFIER_USE_DB_TRUST_REGISTRY=true
go run ./cmd/verifier
```

## Important Conventions

### Error Handling
- Domain errors defined in `internal/httpserver/errors.go`
- HTTP errors use standardized JSON format with error code
- Audit logging happens before error responses
- Don't expose sensitive details in error messages

### Logging
- Structured logging using standard library log package
- Format: `key=value` pairs for parsability
- Log levels: debug, info, warn, error
- Audit events logged with `event_type` key

### Code Organization
- Keep handlers thin, business logic in domain/ or service layer
- Use dependency injection via struct fields
- Avoid global state except for CLI tools
- Interfaces defined close to usage, not in separate files

### Database Migrations
- Manual migrations in `db/init.sql`
- Schema includes: tenants, tenant_trusted_issuers, policies, audit_log
- Always use parameterized queries
- Connection pooling via pgxpool

### Security Considerations
- Never log private keys or credentials
- Validate all external inputs
- Use constant-time comparisons for cryptographic values
- Trust registry checks mandatory before credential acceptance
- Delegation depth limits enforced (default max 5)

## Module and Import Structure

- Main module: `github.com/bradtumy/credential-service`
- SDK modules:
  - Go: `github.com/bradtumy/credential-service/sdk/go`
  - Node.js: `@id2me/credential-sdk` (published to npm)
  - Python: `credential-service-sdk` (PyPI package name)
- Go version: 1.22+
- Python version: 3.8+
- Node.js version: 16+
- Internal packages not importable outside this repo
- SDK code must not import internal/ packages

## Standards Compliance

- W3C Verifiable Credentials Data Model 1.1
- W3C VC-JWT specification
- RFC 7638 (JWK Thumbprint)
- RFC 7519 (JWT)
- SD-JWT specification (IETF draft)
- DID Core specification
