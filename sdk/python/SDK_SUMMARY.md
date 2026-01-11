# Python SDK Summary

## Overview

A modern, production-ready Python SDK for the Credential Service that provides a type-safe, well-documented interface for issuing, verifying, and authorizing W3C Verifiable Credentials.

## Key Features

✅ **Type-Safe**: Full type hints with dataclasses
✅ **Flexible HTTP**: Supports both httpx and requests
✅ **Modern Python**: Python 3.8+ compatibility
✅ **Well-Tested**: Comprehensive unit test coverage
✅ **Developer-Friendly**: Context managers, clear errors, good docs
✅ **Production-Ready**: Error handling, timeouts, resource cleanup

## What Was Created

### Core SDK (`credential_sdk/`)
```
credential_sdk/
├── __init__.py          # Public API exports
├── client.py            # Main CredentialServiceClient (343 lines)
├── types.py             # Dataclass type definitions (107 lines)
└── exceptions.py        # Typed exception hierarchy (34 lines)
```

### Tests (`tests/`)
- `test_client.py` - Client functionality tests (mocked HTTP)
- `test_types.py` - Type validation tests
- Full pytest configuration in pyproject.toml

### Examples (`examples/`)
- `basic.py` - Complete end-to-end example
- `README.md` - Example documentation and troubleshooting

### Documentation
- `README.md` - Comprehensive API documentation (9,515 chars)
- `QUICKSTART.md` - 30-second quick start guide
- `CONTRIBUTING.md` - Development setup and guidelines
- `SDK_SUMMARY.md` - This file

### Configuration
- `pyproject.toml` - Modern Python packaging with all metadata
- `setup.py` - Backward compatibility wrapper
- `MANIFEST.in` - Distribution manifest
- `.gitignore` - Python-specific ignores

## API Design Principles

### 1. Parity with Go and Node SDKs
The Python SDK mirrors the API surface of the existing Go and Node.js SDKs:
- Same method names (snake_case for Python conventions)
- Same request/response structures
- Same error handling patterns

### 2. Pythonic Conventions
- Snake_case naming (not camelCase)
- Dataclasses instead of dicts
- Context managers for resource cleanup
- Type hints throughout
- Docstrings on all public APIs

### 3. Flexible HTTP Backend
Unlike the Go (always net/http) and Node (always fetch) SDKs, Python supports both:
- **httpx** - Modern, async-capable (preferred)
- **requests** - Traditional, widely-used

This allows users to choose based on their existing dependencies.

### 4. Developer Experience
```python
# Quick start with sensible defaults
client = CredentialServiceClient.local_dev()

# Or from environment
client = CredentialServiceClient.from_env()

# Auto cleanup with context manager
with CredentialServiceClient.local_dev() as client:
    result = client.verify(credential="...")

# Clear, typed errors
try:
    client.issue_vc(...)
except InvalidRequestError as e:
    print(f"Bad request: {e.message}, status: {e.status_code}")
```

## API Surface

### Client Initialization
- `CredentialServiceClient()` - Explicit configuration
- `CredentialServiceClient.from_env()` - From environment variables
- `CredentialServiceClient.local_dev()` - Local development defaults

### Credential Operations
- `issue_vc(request)` - Issue VC-JWT credential
- `issue_sd_jwt(request)` - Issue SD-JWT with selective disclosure
- `verify(credential|request)` - Verify credential or chain
- `authorize(request)` - Gateway authorization decision
- `delegate_credential(request)` - Create delegated credential

### Type System
All requests and responses are strongly typed:
- `IssueRequest` / `IssueResponse`
- `VerifyRequest` / `VerifyResponse`
- `AuthorizeRequest` / `AuthorizeResponse`
- `DelegateRequest` / `DelegateResponse`
- `GatewayAgentContext`

### Exception Hierarchy
```
CredentialServiceError (base)
├── NetworkError (network/transport)
├── InvalidRequestError (400)
├── UnauthorizedError (401/403)
└── ServerError (5xx)
```

## Comparison with Other SDKs

| Feature | Go SDK | Node SDK | Python SDK |
|---------|--------|----------|------------|
| Type Safety | ✅ (structs) | ✅ (TypeScript) | ✅ (dataclasses) |
| HTTP Client | net/http | fetch | httpx or requests |
| Async Support | ❌ | ✅ | Future ✅ |
| Context Manager | ❌ | ❌ | ✅ |
| Error Types | sentinel errors | Error classes | Exception classes |
| Naming | PascalCase | camelCase | snake_case |
| Package Manager | go get | npm | pip |

## Installation & Usage

### Installation
```bash
pip install httpx credential-service-sdk
# or
pip install requests credential-service-sdk
```

### Quick Start
```python
from credential_sdk import CredentialServiceClient, IssueRequest

client = CredentialServiceClient.local_dev()
issued = client.issue_vc(IssueRequest(
    subject_did="did:jwk:...",
    ttl_seconds=600,
    claims={"aud": "my-api"}
))
print(issued.credential)
```

## Testing

Run the test suite:
```bash
cd sdk/python
pip install -e ".[dev]"
pytest
```

Run example against live services:
```bash
# From repo root
docker compose up -d

# Run example
cd sdk/python
python examples/basic.py
```

## Future Enhancements

1. **Async Support**: Add async variants of all methods
   ```python
   async with CredentialServiceClient.local_dev() as client:
       result = await client.issue_vc_async(...)
   ```

2. **Response Caching**: Optional caching for verification results

3. **Retry Logic**: Built-in exponential backoff for transient failures

4. **Metrics Integration**: Optional OpenTelemetry/Prometheus support

5. **CLI Tool**: Command-line interface for testing
   ```bash
   credential-sdk issue --subject did:jwk:... --claims '{"test":true}'
   ```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and contribution guidelines.

## License

Apache License 2.0 - Same as the main repository.
