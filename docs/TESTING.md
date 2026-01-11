# Testing

## Prerequisites
- Go toolchain (version from `go.mod`).
- Docker available in the PATH for running the docker-compose demo stack.
- Postgres and Redis are provided by Docker when using `docker compose`; no manual setup is required for the in-process tests.

## Unit tests
Run the core unit suites under `internal/...` and `cmd/...`:

```bash
make test-unit
```

## Integration tests
HTTP handler and domain-level integration scenarios live in `test/integration`. Execute them with:

```bash
make test-integration
```

## End-to-end (E2E) tests
End-to-end tests live in `tests/e2e` and exercise Issuer → Trust Registry → Gateway/Verifier → Sample API flows. They start in-process services by default but can point at already running local services via the `E2E_ISSUER_URL`, `E2E_VERIFIER_URL`, and `E2E_API_URL` environment variables. Run them with the `e2e` build tag:

```bash
make test-e2e
# or
go test ./tests/e2e -tags=e2e -count=1
```

## Full suite and CI
`make test` runs unit tests. `make ci` runs lint, unit, integration, e2e, and security checks.

## Plain Go tooling
Standard `go test ./...` works from the repository root with the module layout in `go.work`.
