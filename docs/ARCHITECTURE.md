# Architecture

The repository now follows an idiomatic Go layout that separates binaries, service wiring, and shared libraries.

## Repository layout

```
cmd/                 # CLI entrypoints
  issuer/            # main for the issuer service
  verifier/          # main for the verifier + gateway service
  keygen/            # helper for generating DIDs/keys
services/            # service wiring and HTTP server setup
  issuer/            # builds the issuer HTTP server
  verifier/          # builds the verifier + gateway HTTP server
internal/            # shared libraries (not importable by SDKs)
  config/            # environment-driven configuration
  crypto/            # key handling, signers, and key rotation
  domain/            # core VC, delegation, and trust registry types
  httpserver/        # HTTP handlers, middleware, and route registration
  keystore/          # pluggable keystore backends
  metrics/, policy/, ratelimit/, storage/, tenant/ ...
sdk/go               # Go SDK (module path: github.com/bradtumy/credential-service/sdk/go)
sdk/node             # Node SDK
```

## Services
- **Issuer (`cmd/issuer` + `services/issuer`)**: issues VC-JWT and SD-JWT credentials, handles bootstrap flows, and writes audit logs when configured.
- **Verifier & Gateway (`cmd/verifier` + `services/verifier`)**: verifies credentials, enforces policies, mints synthetic JWTs, exposes admin routes, and publishes Prometheus metrics.
- **Sample API (`samples/` binaries)**: demonstrates protecting APIs with the gateway-minted JWTs; Docker Compose keeps behavior the same.

## Shared code
- Core credential, delegation, and DID helpers live in `internal/domain`.
- HTTP handler and middleware utilities live in `internal/httpserver` and are reused by both services.
- Cross-cutting concerns such as configuration (`internal/config`), keystore selection (`internal/keystore`), metrics (`internal/metrics`), and persistence (`internal/storage`) remain centralized for reuse.

## Trust and delegation
- Trust registries default to in-memory stores but can be backed by Postgres via `internal/storage.PGTrustRegistry`.
- Verification enforces delegation depth and trusted issuer checks before resolving DIDs.
- Agents/delegations maintain least-privilege by shrinking scope and TTL at each hop.
