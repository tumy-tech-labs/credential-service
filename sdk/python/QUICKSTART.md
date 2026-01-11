# Python SDK Quick Start

## Installation

```bash
# Install with httpx (recommended)
pip install httpx credential-service-sdk

# Or with requests
pip install requests credential-service-sdk
```

## First Time Setup

**Important**: Before running examples, add the issuer to the trust registry:

```bash
# Make sure services are running
docker compose up -d

# Run setup script (once)
cd sdk/python
python examples/setup_trust.py
```

## 30-Second Example

```python
from credential_sdk import CredentialServiceClient, IssueRequest, AuthorizeRequest

# Connect to local services
client = CredentialServiceClient.local_dev()

# Issue a credential
issued = client.issue_vc(IssueRequest(
    subject_did="did:jwk:eyJrdHkiOiJPS1Ai...",
    ttl_seconds=600,
    claims={"aud": "my-api", "scope": "read"}
))

# Verify it
verified = client.verify(credential=issued.credential)
print(f"Valid: {verified.valid}")

# Authorize access
decision = client.authorize(AuthorizeRequest(
    credential=issued.credential,
    expected_audience="my-api",
    want_synthetic_jwt=True
))
print(f"Allowed: {decision.allowed}")
```

## Common Patterns

### Issue and Verify
```python
# Issue
issued = client.issue_vc(IssueRequest(
    subject_did="did:jwk:alice",
    ttl_seconds=600,
    claims={"role": "admin"}
))

# Verify
result = client.verify(credential=issued.credential)
if result.valid:
    print(f"Subject: {result.subject}")
```

### Selective Disclosure (SD-JWT)
```python
# Issue with selective disclosure
issued = client.issue_sd_jwt(IssueRequest(
    subject_did="did:jwk:alice",
    ttl_seconds=600,
    claims={"email": "alice@example.com", "dept": "eng"}
))

# issued.disclosures contains the disclosure tokens
# Verify with subset of disclosures to reveal only some claims
```

### Delegation
```python
# Create delegated credential
delegated = client.delegate_credential(DelegateRequest(
    parent_credential=root_credential,
    delegate_did="did:jwk:agent",
    scope=["read:orders"],  # Reduced scope
    ttl_seconds=300  # Shorter lifetime
))
```

### Gateway Authorization
```python
decision = client.authorize(AuthorizeRequest(
    credential=credential,
    expected_audience="my-api",
    resource="orders",
    action="read",
    want_synthetic_jwt=True
))

if decision.allowed:
    # Use synthetic JWT for downstream services
    downstream_api_call(
        headers={"Authorization": f"Bearer {decision.synthetic_jwt}"}
    )
```

## Configuration

### Local Development
```python
client = CredentialServiceClient.local_dev()
# Uses localhost:8080 (issuer) and localhost:8081 (verifier)
```

### Environment Variables
```python
# Set ISSUER_URL, VERIFIER_URL, GATEWAY_URL
client = CredentialServiceClient.from_env()
```

### Explicit URLs
```python
client = CredentialServiceClient(
    issuer_url="https://issuer.example.com",
    verifier_url="https://verifier.example.com",
    timeout=30.0
)
```

### Context Manager
```python
with CredentialServiceClient.local_dev() as client:
    result = client.verify(credential="...")
# Automatically closed
```

## Error Handling

```python
from credential_sdk import (
    InvalidRequestError,
    UnauthorizedError,
    ServerError,
    NetworkError
)

try:
    result = client.verify(credential=cred)
except InvalidRequestError as e:
    print(f"Bad request: {e.message}")
except UnauthorizedError as e:
    print(f"Access denied: {e.message}")
except NetworkError as e:
    print(f"Connection failed: {e.message}")
```

## Testing

```bash
# Run tests
pytest

# With coverage
pytest --cov=credential_sdk
```

## Next Steps

- Read the full [README.md](README.md) for detailed API documentation
- Check [examples/basic.py](examples/basic.py) for a complete working example
- See repository [docs/](../../docs/) for service architecture and patterns
