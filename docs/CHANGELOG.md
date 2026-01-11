# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Unreleased
### Added
- **did:web DID method support** - Enterprise-ready did:web resolver with HTTPS-only security, configurable timeouts, and document size limits
- Complete did:web integration with composite resolver supporting both did:jwk and did:web methods
- Comprehensive security features for did:web: HTTPS enforcement, request timeouts (10s default), document size limits (10KB default)
- Full test coverage including unit tests, integration tests, and security validation
- Updated verifier service to include did:web resolver by default
- Production-ready did:web documentation with enterprise deployment examples
- API versioning contract with health/readiness endpoints and standardized error shapes.
- OpenAPI specification for core issuer, verifier, and gateway APIs.
- Release packaging targets and documentation for tagging.
- Production configuration guidance and API overview documentation.
- Lightweight smoke test suite for rapid validation.

## v0.0.5 - 2026-01-11

### Changed
- Consolidate Go SDK module path to `github.com/bradtumy/credential-service/sdk/go` and update examples and documentation.
	- Updated `sdk/go/go.mod` module path and `sdk/go/README.md`.
	- Adjusted example `go.mod` replacement directives and example imports to use `sdk/go`.
	- Updated top-level `README.md` and `docs/ARCHITECTURE.md` to reflect consolidated SDK locations.

## [Day 10]
### Added
- API response versioning, health/readiness probes, and OpenAPI spec.
- Standard JSON error envelope with API version across services.
- Release packaging targets and changelog scaffolding.
- Production configuration and developer experience documentation updates.
- Smoke test exercising issuer, delegation, and gateway authorization flows.

## [Day 9]
### Added
- Logging and metrics hooks for core services.
- Agent lifecycle documentation and lifecycle helpers.

## [Day 8]
### Added
- Threat model and security documentation covering credential flows.
- Agent lifecycle documentation expansion and SDK references.

## [Day 7]
### Added
- Example directory and quickstart walkthrough for issuing and verifying credentials.

## [Day 6]
### Added
- Sample agent utilities and agent sample implementation.

## [Day 5]
### Added
- Node and Go SDKs for interacting with the credential service.

## [Day 4]
### Added
- Containerized developer environment and compose stack for issuer/verifier flows.

## [Day 3]
### Added
- DB-backed trust registry integration for verifier service.

## [Day 2]
### Added
- Gateway authorization endpoint wired into verifier.

## [Day 1]
### Added
- Initial issuer and verifier services with credential issuance and verification flows.
