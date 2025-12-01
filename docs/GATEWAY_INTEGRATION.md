# Gateway & Legacy Integration

Gateways or reverse proxies (Envoy, NGINX, API gateways) can call the verifier service to authorize requests that present verifiable credentials.

## How it works

- Gateways POST to `/v1/gateway/authorize` with the credential (or delegation chain) and optional expected audience.
- The platform verifies the VC/delegation chain using the trust registry and issuer keys.
- The endpoint returns an allow/deny decision with subject, acting-on-behalf-of, delegation depth, and the claims extracted from the credential.
- When requested, the verifier can mint a synthetic JWT so downstream services can keep using `Authorization: Bearer <token>` without understanding VCs.

This approach lets existing applications continue to rely on JWT-style headers while enforcing decentralized identity at the edge.
