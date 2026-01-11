# Gateway Authorization API

The gateway exposes a single authorization endpoint that evaluates verifiable credentials against trust rules and authorization policies.

## Endpoint

`POST /v1/gateway/authorize`

### Purpose
- Verifies VC-JWT and SD-JWT credential chains.
- Ensures issuers are trusted for the tenant.
- Applies authorization policies (resource + action checks).
- Optionally mints a short-lived synthetic JWT for downstream services.

## Request Schema

```json
{
  "credential": "<string>",
  "credentials": ["<string>"],
  "expected_audience": "<string>",
  "want_synthetic_jwt": false,
  "resource": "<string>",
  "action": "<string>"
}
```

Notes:
- Provide either `credential` (single token) or `credentials` (delegation chain). An error is returned if neither is present.
- `expected_audience` is checked on the final credential in the chain.
- `resource` and `action` drive policy evaluation and are required.
- Set `want_synthetic_jwt` to `true` to receive a minted token on allow.

## Response Schema

### Success (HTTP 200)

```json
{
  "allowed": true,
  "subject": "did:example:subject",
  "acting_on_behalf_of": "did:example:parent",
  "delegation_depth": 1,
  "claims": {
    "scope": ["read"]
  },
  "synthetic_jwt": "<token>",
  "policy_id": 42,
  "agent": {
    "acting_on_behalf_of": "did:example:parent",
    "delegation_depth": 1,
    "scope": ["read"]
  },
  "tenant_id": "tenant",
  "api_version": "v1"
}
```

Fields without values are omitted. Synthetic JWTs are included only when requested and allowed.

### Error (HTTP 4xx/5xx)

```json
{
  "allowed": false,
  "error_code": "policy_denied",
  "message": "Policy evaluation failed for resource /orders",
  "details": {
    "policy_id": 42
  },
  "tenant_id": "tenant",
  "api_version": "v1"
}
```

- `allowed` is always present.
- `error_code` conveys the canonical reason.
- `message` is human-readable context.
- `details` is optional diagnostic data (e.g., policy id or underlying error text).

## Error Codes

| Code | Meaning | Typical HTTP Status |
| --- | --- | --- |
| `invalid_request` | Malformed or missing inputs. | 400 |
| `invalid_credential` | Signature failure, malformed token, audience mismatch, or missing SD-JWT disclosure. | 401 |
| `credential_expired` | The leaf credential in the chain is expired. | 401 |
| `credential_revoked` | Credential is revoked (when revocation checks are enabled). | 401 |
| `delegation_invalid` | Delegation depth, scope, or TTL violates parent constraints. | 401 |
| `issuer_not_trusted` | Issuer is not trusted for the tenant. | 401 |
| `tenant_mismatch` | Credential tenant does not match the request tenant. | 401 |
| `policy_denied` | Policies evaluated and denied access. | 403 |
| `rate_limited` | Rate limit exceeded. | 429 |
| `policy_error` | Internal policy engine failure. | 500 |
| `internal_error` | Unhandled server-side failure. | 500 |

## Status Codes
- **200**: Authorization allowed.
- **400**: Validation/shape errors.
- **401**: Credential/issuer/delegation issues.
- **403**: Policy denial.
- **429**: Rate limiting.
- **500**: Internal errors (including policy evaluation failures or synthetic JWT minting issues).

## Caching and Rate Limiting
- Gateway decisions may be cached when enabled; cached responses retain the same shape and error codes.
- Rate limiting uses client IP + tenant; exceeded limits return `rate_limited`.

## Versioning
`api_version` is included on all responses for forward compatibility.
