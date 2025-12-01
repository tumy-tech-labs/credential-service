# Credential Service Node SDK

Minimal Node.js client for issuing, delegating, verifying, and authorizing Verifiable Credentials against the Credential Service APIs.

## Installation

```bash
npm install @credential-service/sdk
```

## Usage

```js
import { Client } from '@credential-service/sdk';

const client = new Client({ baseUrl: 'http://localhost:8080' });

const issued = await client.issueCredential({
  subject_did: 'did:example:alice',
  ttl_seconds: 600,
  claims: { aud: 'example-api' }
});

const decision = await client.gatewayAuthorize({
  credential: issued.credential,
  expected_audience: 'example-api',
  want_synthetic_jwt: true
});

console.log('allowed', decision.allowed, 'subject', decision.subject);
```

See the repository root quickstart for a full walk-through.
