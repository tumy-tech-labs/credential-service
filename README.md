# Credential Service

Credential-aware gateway + SDKs for issuing, delegating, verifying, and authorizing VC-JWT/SD-JWTs. Ships with a sample API protected by synthetic JWTs so legacy `Authorization: Bearer` clients keep working.

## 60-second quickstart
Copy/paste to bring up the local stack (issuer, verifier/gateway, Postgres, sample API):

```bash
git clone https://github.com/bradtumy/credential-service.git
cd credential-service
docker compose up -d --build issuer verifier s2s-api postgres
```

Health checks:

```bash
curl -sf http://localhost:8080/healthz
curl -sf http://localhost:8081/healthz
curl -i http://localhost:8082/orders  # should 401 without a token
```

## 5-minute killer demo (human → agent → gateway → synthetic JWT → API)
Run the scripted flow end-to-end:

```bash
./scripts/demo_killer_5min.sh
```

What it does:
1. Issues a parent (human) VC with broad scope.
2. Registers the issuer in the trust registry.
3. Spawns a short-lived, scope-narrowed agent VC.
4. Calls `/v1/gateway/authorize` with the chain to mint a synthetic JWT.
5. Calls the sample `/orders` API with the synthetic JWT (allow).
6. Replays the gateway call for `action=write` to show `policy_denied`.

If you want to see the raw responses, the script prints the authorize and deny payloads. Bring the stack down with `docker compose down`.

## How it works (short)
- **Issuer (:8080)**: Issues VC-JWT and SD-JWT credentials and handles delegation with TTL/scope narrowing.
- **Verifier/Gateway (:8081)**: Verifies credential chains, checks trusted issuers, enforces policy (`resource` + `action`), and mints short-lived synthetic JWTs for legacy APIs.
- **Sample API (:8082)**: Decodes synthetic JWTs on `/orders` to show backward-compatible bearer auth.
- **Trust Registry + Policy**: Tenant-scoped stores (memory or Postgres) seeded for local dev; default policy allows `read` on `orders`.

## When NOT to use VCs
- You just need OAuth2/OIDC session tokens or CIAM logins—this stack is for constrained delegation, not user authentication flows.
- You cannot operate any trust registry or policy engine; the gateway enforces issuer trust and policies.
- You require wallet coupling or DID ideology; we intentionally hide wallet setup and keep DID use pragmatic.
- You need long-lived bearer tokens without rotation—short TTLs and delegation limits are core to the product.

## SDKs & examples
- **Go SDK**: module `github.com/bradtumy/credential-service/sdk/go` (source: `sdk/go`) with low-level and high-level helpers.
  - Import: `import sdk "github.com/bradtumy/credential-service/sdk/go"`
  - Quick local client: `client := sdk.NewMagicLocalClient()`
  - See `examples/go/killer-demo` for a minimal end-to-end call chain.
- **Node SDK**: source at `sdk/node` — see [`sdk/node`](sdk/node/README.md) for usage and examples.

## Advanced topics (opt-in)
- [Gateway authorize API](docs/GATEWAY.md) — request/response shapes and error codes.
- [Policy model](docs/POLICY_ENGINE.md) — resource/action rules and ABAC hooks.
- [DID support](docs/DID.md) — `did:jwk` and `did:web` resolvers.
- [Architecture](docs/ARCHITECTURE.md) — component layout.
- [Testing](docs/TESTING.md) — guidance for running unit/integration suites.

## Development
- Go 1.22; run `go test ./...` from repo root.
- Docker Compose powers deterministic local dev; Postgres is optional for in-memory mode but required for trust registry persistence.
- Linting/tests live alongside services; contributions welcome via PR.
