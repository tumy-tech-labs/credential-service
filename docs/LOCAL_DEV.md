# Local Development with Docker Compose

This repository includes a containerized development environment that runs the issuer, verifier, sample API, and sample client services together with Postgres.

## Prerequisites
- Docker and Docker Compose installed

## Environment Setup
- Copy `.env.example` to `.env` if you want to override defaults locally; the compose file already includes sensible defaults for local use.

## Starting the Stack
- From the repository root, start everything with: `docker compose up --build`

## What Gets Started
- **Postgres** on `localhost:5432` with database `credentialsvc`
- **Issuer** on `localhost:8080`
- **Verifier** (gateway authorize endpoint) on `localhost:8081`
- **Sample API service** on `localhost:8082`
- **Sample client service** runs once on startup, issues a credential, obtains a synthetic JWT from the gateway, and calls the sample API. Logs show each step and any errors.

## How to Test
1. Issue a credential directly from the issuer:
   - `curl -X POST http://localhost:8080/v1/credentials/issue -H 'Content-Type: application/json' -d '{"subject_did":"did:example:test","ttl_seconds":300,"claims":{"scope":"read:orders","aud":"sample-api"}}'`
2. Call the gateway authorize endpoint (replace `<CREDENTIAL>` with the credential from the issuer response):
   - `curl -X POST http://localhost:8081/v1/gateway/authorize -H 'Content-Type: application/json' -d '{"credential":"<CREDENTIAL>","expected_audience":"sample-api","want_synthetic_jwt":true}'`
3. Hit the sample API with the synthetic JWT returned by the gateway (replace `<TOKEN>`):
   - `curl -H 'Authorization: Bearer <TOKEN>' http://localhost:8082/orders`

## Tear Down
- Stop the stack with `docker compose down`. If you want to remove the Postgres volume, include `-v` when tearing down.

## Releasing a New Version
- Update the `CHANGELOG.md` with the changes for the release.
- Build binaries with `make release VERSION=v0.1.0` (replace with your version).
- Create a git tag: `git tag v0.1.0`.
- Push tags to origin: `git push --tags`.
