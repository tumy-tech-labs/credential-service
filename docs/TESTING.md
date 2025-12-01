# Testing

- **Unit tests**: run `make test-unit` to execute packages under `internal/...` and `cmd/...`.
- **Integration tests**: place integration scenarios under `test/integration`; run `make test-integration` (currently placeholders skip/fail safe).
- **Full suite**: `make test` runs unit tests; `make ci` executes lint, unit, integration, e2e placeholder, and security checks.
- **Go tooling**: standard `go test ./...` works from the repository root with the new module layout.
