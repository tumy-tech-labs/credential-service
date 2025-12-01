# Architecture

The credential-service repository now follows an idiomatic Go layout that separates binaries from shared libraries.

## Binaries
- `cmd/issuer`: starts the credential issuer HTTP service and worker.
- `cmd/verifier`: starts the credential verifier HTTP service.

## Internal Modules
- `internal/domain`: core verifiable credential types, issuance helpers, and multi-tenant context primitives.
- `internal/httpx`: HTTP helpers for error responses and middleware (request/tenant IDs).
- `internal/logging`: structured logging setup shared by binaries.
- `internal/metrics`: stubs for verifier metrics instrumentation.
- `internal/config`: lightweight environment-based configuration loader.
- `internal/keystore`: abstraction for retrieving signing keys for tenants.
- `internal/issuer`: issuer-specific routing, queueing, and issuance orchestration.
- `internal/verifier`: verification handlers that wrap domain validation.
- `internal/storage`: storage helpers including the Postgres-backed trust registry.

## Data Flow (simplified)
1. **Issue**: HTTP request hits `cmd/issuer` → router (`internal/issuer`) → credential built/signature attached (`internal/domain`) → stored (PostgreSQL) and queued (RabbitMQ).
2. **Store/Queue**: Credentials persisted via `pgx` (if configured) and issuance jobs emitted to RabbitMQ for background processing.
3. **Verify**: Presentation posted to `cmd/verifier` → handler (`internal/verifier`) → validation rules in `internal/domain`.
4. **Future**: Trust registry and delegation rules will extend `internal/domain` and share middleware/config utilities.

## Trust Registry Storage
- A Postgres-backed trust registry is available via `internal/storage.PGTrustRegistry`.
- Deployments using Postgres must create the `trusted_issuers` table (see `migrations/0001_create_trusted_issuers.sql`).

## Delegation and Agent Identity
- Credentials can be delegated from one DID to another, enabling agents to act with constrained authority.
- Each delegation step must shrink scope and shorten time-to-live; children cannot outlive or outrange parents.
- Verification enforces a maximum delegation depth to cap chain length.
- Verifier responses surface both the active subject and who they are acting on behalf of.
- This makes agent actions auditable while preserving least-privilege semantics.
