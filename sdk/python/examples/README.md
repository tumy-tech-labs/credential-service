# Python SDK Examples

## Prerequisites

Before running these examples, make sure the credential service is running:

```bash
# From repository root
docker compose up --build -d

# Wait a few seconds for services to start
sleep 3

# Check services are healthy
curl http://localhost:8080/healthz
curl http://localhost:8081/readyz
```

## First Time Setup

**Important**: The issuer needs to be added to the verifier's trust registry before credentials can be verified.

Run the setup script once:

```bash
cd sdk/python
python examples/setup_trust.py
```

This will:
1. Extract the issuer's DID from docker logs
2. Add it to the verifier's trust registry
3. Verify the setup

You should see:
```
✅ Setup complete!
You can now run the example:
  python examples/basic.py
```

## Running Examples

### Basic Example

Demonstrates the complete flow: issue, verify, delegate, and authorize.

```bash
cd sdk/python
pip install -e .  # Install the SDK
python examples/basic.py
```

### Environment Variables

You can override the default service URLs:

```bash
export ISSUER_URL=http://localhost:8080
export VERIFIER_URL=http://localhost:8081
export GATEWAY_URL=http://localhost:8081

python examples/basic.py
```

## Example Output

```
=== Credential Service Python SDK Example ===

Initializing client...

1. Issuing root credential...
   ✓ Issued credential
   Credential (first 80 chars): eyJhbGciOiJFZERTQSIsInR5cCI6InZjK2p3dCIsImtpZCI6IjExcVlBWUtGMWJuRjNyeEh0Q19FN2I4...
   Format: jwt-vc

2. Verifying credential...
   ✓ Verification successful
   Valid: True
   Subject: did:jwk:eyJrdHk...
   Issuer: did:jwk:eyJrdHk...
   Expires at: 2025-12-06 21:30:00+00:00
   Claims: {'aud': 'sample-api', 'scope': 'read:orders write:orders', ...}

3. Creating delegated credential...
   ✓ Delegated credential created
   Credential (first 80 chars): eyJhbGciOiJFZERTQSIsInR5cCI6InZjK2p3dCIsImtpZCI6IjExcVlB...

4. Authorizing with gateway...
   ✓ Authorization decision received
   Allowed: True
   Subject: did:jwk:eyJrdHk...
   Acting on behalf of: did:jwk:eyJrdHk...
   Delegation depth: 1
   Reason: policy_allow

5. Issuing SD-JWT credential with selective disclosure...
   ✓ SD-JWT issued
   Credential (first 80 chars): eyJhbGciOiJFZERTQSIsInR5cCI6InNkK2p3dCIsImtpZCI6IjExcVlB...
   Number of disclosures: 4
   First disclosure: WyJzYWx0MSIsImVtYWlsIiwiYWxpY2VAZXhhbXBsZS5jb20iXQ...

=== Example completed successfully! ===
```

## Troubleshooting

### Connection Refused

If you get connection errors:
1. Verify services are running: `docker compose ps`
2. Check service logs: `docker compose logs issuer verifier`
3. Restart services: `docker compose down && docker compose up -d`

### Untrusted Issuer

If you get "untrusted_issuer" errors:
1. Extract the issuer DID from logs: `docker logs credential-service-issuer-1 | grep issuer_did`
2. Add to trust registry: See repository root README Quick Start section

### Import Errors

Make sure you've installed the SDK:
```bash
cd sdk/python
pip install -e .
```

Or install required HTTP client:
```bash
pip install httpx  # or: pip install requests
```
