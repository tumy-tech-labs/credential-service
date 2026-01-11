# Changelog

All notable changes to this repository are documented in this file.

## Unreleased - 2026-01-11

- chore(sdk): consolidate Go SDK module path to `github.com/bradtumy/credential-service/sdk/go`
  - Updated `sdk/go/go.mod` module path and README references.
  - Updated examples and example `go.mod` replacements to import `github.com/bradtumy/credential-service/sdk/go`.
  - Adjusted top-level `README.md` and `docs/ARCHITECTURE.md` to reflect consolidated SDK locations.
  - Verified module resolution with `go list ./...` and ran `go test ./...`.

---

For details, see individual commits in the repository history.
