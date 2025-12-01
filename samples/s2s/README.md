# Service-to-Service Sample

This sample demonstrates how an issuer, gateway authorization endpoint, and a legacy-style API can work together. The flow turns a verifiable credential into a synthetic JWT that a typical service can accept without understanding VCs.

## Components

- **client-service**: requests a VC from the issuer, calls the gateway authorize endpoint to obtain a decision + synthetic JWT, then calls the API service with `Authorization: Bearer <token>`.
- **api-service**: represents a legacy API. It accepts the synthetic JWT, decodes it, logs the claims, and responds with a simple JSON payload.

## Running the sample

1. Start the issuer and verifier services (from the repository root):

   ```bash
   go run ./cmd/issuer &
   go run ./cmd/verifier &
   ```

2. Start the API service:

   ```bash
   go run ./samples/s2s/api-service
   ```

3. Start the client service (it will call the issuer, gateway, then API):

   ```bash
   go run ./samples/s2s/client-service
   ```

Environment variables:

- `ISSUER_URL` (default `http://localhost:8080`)
- `GATEWAY_URL` (default `http://localhost:8081`)
- `API_URL` (default `http://localhost:8082`)

## Request flow

1. Client requests a verifiable credential from the issuer with desired claims/scopes.
2. Client presents the credential to `/v1/gateway/authorize` on the verifier service.
3. Gateway endpoint verifies the VC (and any delegation), returns an allow/deny decision and optionally a synthetic JWT.
4. Client calls the API with `Authorization: Bearer <synthetic_jwt>`.
5. The API trusts the JWT and logs claims without needing to parse verifiable credentials.
