# Credential Service Developer Guide

Welcome! This guide introduces the key concepts, flows, and SDK options for building on the Credential Service. Use it alongside the 5-minute quickstart and the examples in the repo.

## Core Concepts

### Decentralized Identifiers (DIDs)
- A DID is a cryptographically verifiable identifier (for example, `did:example:alice`).
- Each DID has associated public/private keys that allow holders to sign and prove control of the identifier.
- The service derives DIDs from signing keys so issuers and subjects can be uniquely addressed.

### Verifiable Credentials (VCs)
- A VC is a tamper-evident statement (JWT/JWS) that binds claims to a subject DID and is signed by an issuer DID.
- Claims can include roles, scopes, audiences, or any JSON payload your application needs.
- The verifier checks signatures, expiration, issuer trust, and audience before accepting a VC.

### Delegation
- A holder with a valid VC can delegate a subset of their permissions by requesting a delegated credential.
- Delegation is constrained by scope and time-to-live and is always tied back to the original delegator.
- The service enforces delegation depth limits to prevent unbounded chains.

### Gateway Authorization
- GatewayAuthorize verifies a credential chain and returns an allow/deny decision plus key identity fields:
  - `subject`: the DID tied to the leaf credential.
  - `acting_on_behalf_of`: the original delegator when delegation is used.
  - `delegation_depth`: how deep the chain is.
  - `claims`: merged claims from the chain.
- Use this at your API gateway or middleware to keep authorization logic simple.

### Synthetic JWTs
- Optionally, GatewayAuthorize can mint a short-lived synthetic JWT signed by the service.
- This token packages the decision (subject, claims, delegation info) so downstream services can verify quickly without re-running full VC verification.
- The token is EdDSA-signed; verify it with the published public key in your gateway or services.

## SDK Guidance

- **Go SDK (`sdk/go`)**: Best for backends and gateways written in Go. Exposes typed helpers for issuing, delegating, verifying, and authorizing credentials.
- **Node SDK (`sdk/node`)**: Ideal for JavaScript/TypeScript services or edge runtimes. Ships with TypeScript definitions for a smooth DX.

## Typical Flow

1. Issue a base credential for a subject DID.
2. (Optional) Delegate a subset of permissions to another DID.
3. Present the credential chain to GatewayAuthorize to receive an authorization decision and, if desired, a synthetic JWT.
4. Call downstream APIs with the synthetic JWT in the `Authorization: Bearer` header.

## Text Diagrams

Credential issuance:
```
[Issuer DID] --signs--> [Credential JWT] --bound to--> [Subject DID]
```

Delegation chain:
```
Issuer --issue--> Parent VC --delegate--> Child VC --delegate--> Grandchild VC
   |                                                   |
acting_on_behalf_of = Issuer DID             delegation_depth = 2
```

Gateway decision with synthetic JWT:
```
[Client] --VC--> [GatewayAuthorize] --allow + synthetic_jwt--> [API / Service]
```

Happy shipping!
