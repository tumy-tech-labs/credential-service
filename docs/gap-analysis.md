# Credential Service Gap Analysis

## 1. Executive Summary
The repository already ships a multi-service Go stack for issuing JWT-encoded Verifiable Credentials, verifying delegation chains, and fronting existing APIs with a gateway that can mint synthetic JWTs, but the implementation is uneven: core flows exist, yet production-grade identity safeguards, DID/key lifecycle tooling, SDK ergonomics, and DX polish are incomplete or inconsistent. Routing, key custody, policy enforcement, and observability are present in outline, but require refactoring toward clearer interfaces, hardened defaults, modern Go patterns, and cohesive developer tooling to realize the intended developer-first credential toolkit and agent identity gateway vision.

## 2. What Currently Exists (Feature Inventory)
- **Services:** Issuer (`cmd/issuer`), Verifier/Gateway (`cmd/verifier`), and sample key generation tool (`cmd/keygen`) exist with HTTP routing and basic tenant middleware wiring in place for the issuer entrypoint. 【F:cmd/issuer/main.go†L21-L66】
- **Documentation:** A comprehensive README outlines goals, deployment topologies (centralized/distributed), quick-start workflow, and enumerates features including VC issuance, trust registry, policy engine, DID resolution, and SDKs. 【F:README.md†L24-L149】
- **Domain + Infrastructure Packages:** Internal packages cover issuer logic, policy evaluation, trust registry, caching, rate limiting, keystore abstraction (memory/production), logging, metrics, and configuration helpers. 【F:internal/issuer/service.go†L1-L82】【F:internal/logging】【F:internal/policy】【F:internal/keystore】
- **Storage & Audit Hooks:** Postgres audit store initialization for the issuer is wired when `AuditDB_DSN` is provided. 【F:cmd/issuer/main.go†L25-L33】
- **Tenant Handling:** In-memory tenant store with default tenant seeding and middleware for tenant resolution. 【F:cmd/issuer/main.go†L49-L60】
- **Examples & Docker:** Docker Compose manifests, Makefile targets, and quick-start scripts are present for local runs. 【F:README.md†L150-L193】【F:Makefile】【F:docker-compose.yml】

## 3. Gap Analysis (What’s Missing)
- **DID/Key Generation Tools:** Only a basic `keygen` binary exists; no dedicated DID method helpers, key rotation tooling, KMS/Vault-backed DID document publication, or automated thumbprint/kid validation pipelines.
- **VC Issuance & Verification Hardening:** Issuer flow signs credentials after fetching Vault keys and resolver data but lacks configurable scope/TTL enforcement, audience binding, SD-JWT disclosure handling, and structured error responses; verifier/gateway posture not clearly guarded by policy defaults or denial-safe evaluation.
- **VC Inspection Tools:** No CLI or HTTP utilities to inspect, decode, or lint VC/JWT structure and delegation chains for developers.
- **Agent Credential Helpers & Delegation Chains:** README mentions delegation but there is no explicit delegation depth enforcement, on-behalf-of chain tracing, or reusable helper APIs/SDK ergonomics for agents.
- **Gateway Route Structure:** Routes live in `httpserver` with mux usage; no typed router grouping, versioned subrouters, or clear contract for gateway authorization/decision schemas.
- **CLI Tools:** Keygen exists; missing cohesive CLI with subcommands for DID generation, issuance, verification, trust-registry management, policy debugging, and gateway auth testing.
- **SDK Readiness:** Go/Node SDKs are referenced but not surfaced with versioning, generated models, or consistent error types; no samples showing SDK usage for agents.
- **Containerization + Local Dev:** Compose exists but lacks one-liner data seeding, migration orchestration, hot-reload dev mode, and pre-configured TLS/secure defaults for local testing.
- **Professional Folder Layout:** Multiple service roots (issuer, verifier, etc.) and internal packages exist but lack consolidated `pkg`/`internal` layering per bounded context; command folders are thin shells with business logic scattered in `internal/httpserver` without clean separation of handlers/services.
- **Secure Defaults:** Memory keystore is default with opt-in production keystore; tenant store defaults to in-memory; no default rate limits, CSRF/JWT audience enforcement, or strict time skew checks.
- **High-Quality DX:** Sparse tests, no lint/format pre-commit hooks, limited structured logging usage (standard `log` calls), no observability defaults, and limited docs for SDK/API schemas.

## 4. Recommended Refactors
- **Adopt Structured Logging + Context Propagation:** Replace `log.Printf`/`Fatalf` in services with a logger interface (zap/zerolog) threaded via context to handlers for consistent request correlation and auditability. 【F:cmd/issuer/main.go†L21-L66】【F:internal/issuer/service.go†L41-L82】
- **Clarify Handler vs Service Layers:** Move business logic out of HTTP handlers in `internal/issuer`/`internal/httpserver` into services with interfaces to enable testing and future transports.
- **Normalize Router Layout:** Use versioned subrouters (e.g., `/v1`) with typed request/response structs and OpenAPI generation, consolidating gateway/issuer routes under explicit packages rather than ad-hoc mux registration.
- **Configuration Hardening:** Centralize config validation (required env vars, sane defaults), explicit TLS/hostname settings, and per-service observability toggles; avoid implicit defaults like memory keystore without warning flags. 【F:cmd/issuer/main.go†L35-L47】
- **Key Management Abstraction:** Standardize keystore interface to include key metadata, rotation hooks, and DID document publication; decouple Vault/KMS clients from issuer logic via interfaces.
- **Policy Engine Modernization:** Encapsulate policy evaluation behind interfaces with deterministic deny-by-default semantics and structured error types; add policy compilation/validation step at startup.
- **Testing Infrastructure:** Add unit tests for handler/service boundaries with mocks for keystore, policy, and trust registries; integrate go-test workflow in CI and coverage targets.

## 5. Recommended New Features
1. **Developer CLI (`credctl`)** for DID/key generation, VC issuance/verification, trust-registry operations, and gateway authorize calls to streamline manual testing.
2. **VC Inspection & Linting Tools** (CLI + HTTP) to decode JWT/SD-JWT, validate required claims, delegation depth, audience, and expiry to aid debugging.
3. **Agent Delegation Framework** with chain enforcement, on-behalf-of tracing, and SDK helpers for automatic re-delegation respecting scope/TTL.
4. **Gateway Policy-as-Code** with declarative policy files (YAML/Rego) and hot-reload, plus synthetic JWT template customization.
5. **Observability & Security Defaults**: structured logging, metrics, tracing hooks, default rate limits, TLS-ready local configs, and secure cookie/session management for any web surface.
6. **SDK Publishing Pipeline** with versioned clients (Go/Node) generated from OpenAPI, examples, and semver tagging.

## 6. Recommended Folder Structure
```
cmd/
  issuer/main.go
  verifier/main.go
  gateway/main.go
  credctl/main.go
internal/
  config/
  http/            # transport adapters (handlers)
  issuer/
    service.go
  verifier/
    service.go
  gateway/
    service.go
  trust/
  policy/
  crypto/
  keystore/
  storage/
  metrics/
  middleware/
pkg/
  sdk/             # shared types & client helpers
  vc/              # VC/DID utilities usable by SDKs/CLIs
api/
  openapi/
  protobuf/
examples/
  ...
deploy/
  docker/
  helm/
```
This clarifies transport vs domain logic, isolates reusable packages, and keeps command entrypoints thin.

## 7. DX Improvements
- **Pre-commit Tooling:** Add `golangci-lint`, `gofumpt`, `goimports`, `markdownlint`, and `hadolint` with a `make lint` target and CI enforcement.
- **Makefile Enhancements:** Add `make dev` (hot reload via `air`), `make test` with coverage, `make docs` to build OpenAPI/SDKs, and `make migrate` wrappers.
- **Sample Collections:** Ship HTTPie/cURL collections and SDK code snippets for issuer, verifier, and gateway flows.
- **Error/Response Schemas:** Document and implement consistent JSON error envelopes with machine-friendly codes; generate OpenAPI and client types from them.
- **Local Secrets Management:** Provide `docker compose` profiles for Vault/KMS emulators with seeded data, plus `.env.example` for quick onboarding.
- **Observability Defaults:** Enable request/response logging (sans secrets), Prometheus metrics endpoints, and optional tracing exporters with sane defaults.

## 8. Security & Identity-Specific Improvements
- **DID Handling:** Add DID document publication/validation (did:web hosting, did:jwk thumbprint verification), caching with freshness/expiry, and trust-on-first-use controls.
- **Key Storage:** Default to HSM/KMS or Vault with envelope encryption; enforce key rotation intervals and audit logging for key access; remove silent fallback to in-memory keys outside explicit dev mode. 【F:cmd/issuer/main.go†L35-L47】
- **VC Issuance:** Enforce minimum/maximum TTL, audience binding, nonce/jti uniqueness, and delegated scope narrowing; add SD-JWT disclosure support and detached payload safeguards. 【F:internal/issuer/service.go†L70-L120】
- **VC Verification:** Implement clock-skew limits, delegation depth caps, revocation/CRL checks, issuer trust registry validation, and deterministic deny-by-default policy evaluation.
- **Gateway Authorization:** Normalize resource/action naming, support multiple audiences, include policy decision reason codes, and sign synthetic JWTs with separate keys/TTLs per client.
- **Audit & Compliance:** Expand audit logs with structured fields (tenant, DID, subject, decision, policy ID), tamper-evident storage, and privacy controls for selective claim redaction.

## 9. Suggested PR List
- **PR 1: Add Structured Logging & Config Validation** – Introduce logger interface, replace stdlib logging in services, and validate required env vars with fail-fast startup.
- **PR 2: Developer CLI (`credctl`)** – Provide subcommands for DID/keygen, VC issue/verify, trust registry, and gateway authorize; reuse shared `pkg/vc` utilities.
- **PR 3: VC Inspection & Delegation Linter** – CLI/HTTP endpoints to decode JWT/SD-JWT, enforce scope/TTL narrowing, and report delegation depth with reasons.
- **PR 4: Gateway Routing & OpenAPI** – Refactor HTTP handlers into versioned routers, add OpenAPI spec and generated Go/Node SDKs, and unify error schema.
- **PR 5: Security Hardening** – Enforce issuer audience/TTL limits, delegation depth caps, clock-skew checks, and default deny policy evaluation with tests.
- **PR 6: Container & Dev UX** – Compose profiles for dev/prod with Vault/KMS emulators, `make dev` hot reload, migrations automation, and seeded demo data.
- **PR 7: Test & Lint Pipeline** – Add CI for unit tests, coverage thresholds, linting, and artifact upload for coverage/linters.

## 10. Codex-Ready Next Steps
1. Create `cmd/credctl` scaffold with Cobra-based CLI wiring and shared config loader.
2. Introduce `pkg/vc` for DID/key utilities, JWT parsing, thumbprint computation, and delegation validation; refactor issuer/verifier to consume it.
3. Replace stdlib logging with structured logger interface and propagate context IDs through middleware.
4. Add config validation and secure defaults (explicit dev flag for in-memory keystore/tenant store, required audiences, TTL caps).
5. Draft OpenAPI spec for issuer/verifier/gateway endpoints; generate Go/Node SDKs into `sdk/` with examples.
6. Add VC inspection endpoint/CLI command that decodes JWT, validates claims, and reports delegation depth and trust decisions.
7. Wire CI with `make lint test`, golangci-lint config, and coverage gates; add unit tests for issuer/gateway handlers using mocks.
8. Extend Docker Compose with Vault/KMS emulator profile, migrations job, seeded tenants/policies, and TLS termination for local testing.
9. Document agent delegation model (scope/TTL narrowing rules, depth cap) and implement enforcement in issuance and verification flows.
10. Add audit log schema and sink (Postgres/OTLP) with structured fields for tenant, issuer, subject, decision, and policy ID.
