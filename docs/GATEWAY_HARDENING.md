# Gateway Hardening

## Overview
The gateway authorization endpoint (`/v1/gateway/authorize`) is the primary enforcement point for service-to-service access. It verifies delegated credentials, evaluates policy, and optionally issues synthetic JWTs for downstream compatibility.

## Input Validation
Requests are validated for presence and reasonable length of `credential` (or `credentials`), `resource`, and `action`. Empty or oversized fields are rejected with a standardized `invalid_request` API error, preserving API versioning in responses.

## Synthetic JWT
When `want_synthetic_jwt` is true, the gateway issues a short-lived token signed by the verifier. Downstream services can validate these tokens using `internal/jwtutil` helpers, which enforce issuer, audience, algorithm allow-lists, and JWKS-based signature verification.

## Caching
Authorization decisions can be cached (optionally in Redis) to reduce latency for repeated requests. Keys include tenant, credential hash, resource, action, and whether a synthetic JWT was requested. TTLs are bounded by credential expiry and a short max window to balance performance with revocation responsiveness.

## Rate Limiting
The gateway exposes a pluggable limiter interface. By default a noop limiter is used; enabling `RATELIMIT_ENABLED` allows swapping in a real backend (for example, Redis) to mitigate brute-force or abuse patterns.

## Observability
Metrics track gateway requests, allows/denies, and cache hit rates. Structured JSON logs capture tenant, subject, acting-on-behalf-of data, delegation depth, cache status, and latency without leaking credential contents.

## Future Enhancements
- Enforce mTLS between gateway and downstream services
- Integrate IP reputation and anomaly signals
- Place a WAF in front of the gateway for coarse filtering
- Add redis-backed rate limiting tokens with configurable policies

