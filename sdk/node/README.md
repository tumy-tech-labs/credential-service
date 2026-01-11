# Credential Service Node SDK

A TypeScript-friendly Node.js client for the Credential Service issuer, verifier, and gateway APIs.

## Installation

```bash
npm install @id2me/credential-sdk
```

## Usage

```ts
import { CredentialServiceClient } from '@id2me/credential-sdk';

const client = new CredentialServiceClient({
  issuerUrl: process.env.ISSUER_URL || 'http://localhost:8080',
  verifierUrl: process.env.VERIFIER_URL || 'http://localhost:8081',
  gatewayUrl: process.env.GATEWAY_URL // optional fallback to verifierUrl
});

// Issue a VC-JWT
const issued = await client.issueVC({
  subject_did: 'did:example:alice',
  ttl_seconds: 600,
  claims: { scope: ['payments'] }
});

// Verify a credential or chain
const verification = await client.verify(issued.credential);

// Request an authorization decision (with optional synthetic JWT)
const decision = await client.authorize({
  credentials: [issued.credential],
  expected_audience: 'https://api.example.com',
  want_synthetic_jwt: true
});
```

See [`examples/basic.ts`](./examples/basic.ts) for a runnable script that walks through issuing, verifying, and authorizing.

## API

### `new CredentialServiceClient(options)`
- `issuerUrl` – Base URL for the issuer service.
- `verifierUrl` – Base URL for the verifier service.
- `gatewayUrl` – Base URL for the gateway; defaults to `verifierUrl` when omitted.
- `fetchImpl` – Custom `fetch` implementation (defaults to `globalThis.fetch`).

### `issueVC(request)`
Issues a JWT VC. Adds `format: 'jwt-vc'` automatically.

### `issueSDJWT(request)`
Issues an SD-JWT + disclosures. Adds `format: 'sd-jwt'` automatically.

### `verify(credential | request)`
Verifies a credential or chain using the verifier service.

### `authorize(request)`
Requests an authorization decision (and optional synthetic JWT) from the gateway service.

### `delegateCredential(request)`
Calls the delegation endpoint to mint a delegated credential. Useful for building agent flows.

## Development

```bash
npm install
npm run build
npm test
```

The published package ships the compiled JavaScript and type definitions from `dist/`.
