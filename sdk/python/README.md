# Credential Service Python SDK

A modern Python client for issuing, verifying, and authorizing W3C Verifiable Credentials with the Credential Service.

## Features

- **Type-safe**: Full type hints and dataclass-based request/response models
- **Flexible HTTP**: Supports both `httpx` (async/sync) and `requests` (sync)
- **Modern Python**: Python 3.8+ with dataclasses and type annotations
- **Context Manager**: Automatic resource cleanup
- **Clear Errors**: Typed exceptions for different error categories
- **Well-documented**: Comprehensive docstrings and examples

## Installation

### From PyPI (when published)
```bash
# With httpx (recommended - supports async in future)
pip install httpx credential-service-sdk

# Or with requests (sync only)
pip install requests credential-service-sdk
```

### For Local Development

**Important**: Use a fresh virtual environment to avoid pip issues.

```bash
# Navigate to SDK directory
cd sdk/python

# Create and activate virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Upgrade pip (important!)
pip install --upgrade pip setuptools wheel

# Install in editable mode with dependencies
pip install -e ".[httpx,dev]"  # With httpx + dev tools
# OR
pip install -e ".[requests,dev]"  # With requests + dev tools
```

See [INSTALL.md](INSTALL.md) for troubleshooting and detailed instructions.

## Quick Start

```python
from credential_sdk import CredentialServiceClient, IssueRequest, AuthorizeRequest

# Use local development defaults (docker-compose)
client = CredentialServiceClient.local_dev()

# Or configure explicitly
client = CredentialServiceClient(
    issuer_url="http://localhost:8080",
    verifier_url="http://localhost:8081"
)

# Or from environment variables (ISSUER_URL, VERIFIER_URL, GATEWAY_URL)
client = CredentialServiceClient.from_env()

# Issue a credential
issued = client.issue_vc(IssueRequest(
    subject_did="did:jwk:eyJrdHkiOiJPS1Ai...",
    ttl_seconds=600,
    claims={"aud": "sample-api", "scope": "read:orders"}
))

print(f"Credential: {issued.credential}")

# Verify the credential
verified = client.verify(credential=issued.credential)
print(f"Valid: {verified.valid}, Subject: {verified.subject}")

# Authorize and get synthetic JWT
decision = client.authorize(AuthorizeRequest(
    credential=issued.credential,
    expected_audience="sample-api",
    resource="orders",
    action="read",
    want_synthetic_jwt=True
))

if decision.allowed:
    print(f"Access granted! JWT: {decision.synthetic_jwt}")
else:
    print(f"Access denied: {decision.reason}")
```

## Context Manager Usage

```python
with CredentialServiceClient.local_dev() as client:
    issued = client.issue_vc(IssueRequest(
        subject_did="did:jwk:example",
        ttl_seconds=600,
        claims={"test": True}
    ))
    verified = client.verify(credential=issued.credential)
    print(f"Valid: {verified.valid}")
# Client automatically closed
```

## API Reference

### Client Initialization

#### `CredentialServiceClient(issuer_url, verifier_url, gateway_url=None, timeout=15.0)`
Create a new client with explicit configuration.

#### `CredentialServiceClient.from_env()`
Create a client using environment variables:
- `ISSUER_URL` (default: http://localhost:8080)
- `VERIFIER_URL` (default: http://localhost:8081)
- `GATEWAY_URL` (optional, defaults to verifier_url)

#### `CredentialServiceClient.local_dev()`
Pre-configured for local development with docker-compose defaults.

### Credential Operations

#### `issue_vc(request: IssueRequest) -> IssueResponse`
Issue a standard VC-JWT credential.

**Example:**
```python
issued = client.issue_vc(IssueRequest(
    subject_did="did:jwk:alice",
    ttl_seconds=600,
    claims={"role": "admin", "aud": "api.example.com"}
))
```

#### `issue_sd_jwt(request: IssueRequest) -> IssueResponse`
Issue an SD-JWT credential with selective disclosure support.

**Example:**
```python
issued = client.issue_sd_jwt(IssueRequest(
    subject_did="did:jwk:alice",
    ttl_seconds=600,
    claims={
        "email": "alice@example.com",
        "department": "engineering",
        "scope": "read:orders"
    }
))
print(f"Credential: {issued.credential}")
print(f"Disclosures: {issued.disclosures}")
```

#### `verify(credential: str = None, request: VerifyRequest = None) -> VerifyResponse`
Verify a credential or credential chain.

**Simple usage:**
```python
result = client.verify(credential="eyJhbGci...")
print(f"Valid: {result.valid}, Subject: {result.subject}")
```

**Advanced usage:**
```python
from credential_sdk import VerifyRequest

result = client.verify(request=VerifyRequest(
    credentials=["parent_cred", "delegated_cred"],
    expected_audience="sample-api",
    disclosures=["disclosure1", "disclosure2"],  # For SD-JWT
    format="sd-jwt"
))
```

#### `authorize(request: AuthorizeRequest) -> AuthorizeResponse`
Request an authorization decision from the gateway.

**Example:**
```python
from credential_sdk import AuthorizeRequest

decision = client.authorize(AuthorizeRequest(
    credential="eyJhbGci...",
    expected_audience="sample-api",
    resource="orders",
    action="read",
    want_synthetic_jwt=True
))

if decision.allowed:
    # Use synthetic JWT for downstream services
    headers = {"Authorization": f"Bearer {decision.synthetic_jwt}"}
```

#### `delegate_credential(request: DelegateRequest) -> DelegateResponse`
Create a delegated credential with constrained scope and TTL.

**Example:**
```python
from credential_sdk import DelegateRequest

delegated = client.delegate_credential(DelegateRequest(
    parent_credential="eyJhbGci...",
    delegate_did="did:jwk:agent",
    scope=["read:orders"],  # Narrower than parent
    ttl_seconds=300,  # Shorter than parent
    claims={"agent_type": "service"}
))
```

## Type Definitions

All request and response types are strongly typed using Python dataclasses:

```python
from credential_sdk import (
    IssueRequest,
    IssueResponse,
    VerifyRequest,
    VerifyResponse,
    AuthorizeRequest,
    AuthorizeResponse,
    DelegateRequest,
    DelegateResponse,
    GatewayAgentContext,
)
```

## Error Handling

The SDK provides typed exceptions:

```python
from credential_sdk import (
    CredentialServiceError,  # Base exception
    NetworkError,            # Network/transport errors
    InvalidRequestError,     # 400 Bad Request
    UnauthorizedError,       # 401/403 errors
    ServerError,             # 5xx errors
)

try:
    result = client.verify(credential="invalid")
except InvalidRequestError as e:
    print(f"Bad request: {e.message}, status: {e.status_code}")
except UnauthorizedError as e:
    print(f"Unauthorized: {e.message}")
except NetworkError as e:
    print(f"Network error: {e.message}")
except CredentialServiceError as e:
    print(f"General error: {e.message}")
```

## Complete Example

```python
from credential_sdk import (
    CredentialServiceClient,
    IssueRequest,
    VerifyRequest,
    AuthorizeRequest,
    DelegateRequest,
    InvalidRequestError,
    UnauthorizedError,
)

def main():
    # Initialize client
    with CredentialServiceClient.local_dev() as client:
        try:
            # Issue root credential
            print("Issuing credential...")
            issued = client.issue_vc(IssueRequest(
                subject_did="did:jwk:alice",
                ttl_seconds=600,
                claims={
                    "aud": "sample-api",
                    "scope": "read:orders write:orders",
                    "role": "admin"
                }
            ))
            print(f"✓ Issued: {issued.credential[:50]}...")

            # Verify credential
            print("\nVerifying credential...")
            verified = client.verify(
                request=VerifyRequest(
                    credential=issued.credential,
                    expected_audience="sample-api"
                )
            )
            print(f"✓ Valid: {verified.valid}")
            print(f"  Subject: {verified.subject}")
            print(f"  Expires: {verified.expires_at}")

            # Create delegated credential
            print("\nDelegating credential...")
            delegated = client.delegate_credential(DelegateRequest(
                parent_credential=issued.credential,
                delegate_did="did:jwk:agent",
                scope=["read:orders"],  # Reduced scope
                ttl_seconds=300  # Shorter TTL
            ))
            print(f"✓ Delegated: {delegated.credential[:50]}...")

            # Authorize with gateway
            print("\nAuthorizing request...")
            decision = client.authorize(AuthorizeRequest(
                credentials=[issued.credential, delegated.credential],
                expected_audience="sample-api",
                resource="orders",
                action="read",
                want_synthetic_jwt=True
            ))
            print(f"✓ Allowed: {decision.allowed}")
            if decision.allowed:
                print(f"  Synthetic JWT: {decision.synthetic_jwt[:50]}...")
                print(f"  Acting on behalf of: {decision.acting_on_behalf_of}")

        except InvalidRequestError as e:
            print(f"✗ Invalid request: {e.message}")
        except UnauthorizedError as e:
            print(f"✗ Unauthorized: {e.message}")
        except Exception as e:
            print(f"✗ Error: {e}")

if __name__ == "__main__":
    main()
```

## Development

```bash
# Install development dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Type checking
mypy credential_sdk

# Linting
ruff check credential_sdk
```

## Requirements

- Python 3.8+
- httpx or requests (at least one required)

## License

Apache License 2.0 - See repository root LICENSE file.

## Related Documentation

See the repository root documentation for:
- [Architecture](../../docs/ARCHITECTURE.md)
- [API Overview](../../docs/API_OVERVIEW.md)
- [Policy Engine](../../docs/POLICY_ENGINE.md)
- [Gateway Integration](../../docs/GATEWAY_INTEGRATION.md)
