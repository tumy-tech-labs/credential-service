# Threat Model

## Overview
The credential-service issues and verifies verifiable credentials for multiple tenants. Core components include the issuer service, verifier service, trust registry (now backed by Postgres), and a gateway authorize endpoint that bridges VC verification into authorization decisions/JWTs.

## Assets
- Issuer signing keys
- Credentials / verifiable credentials (VCs)
- Delegation chains and acting_on_behalf_of relationships
- Trust registry data (per-tenant issuer trust)
- Logs and audit trails

## Actors
- Legitimate users, services, and agents presenting credentials
- Tenant administrators managing trusted issuers
- External attackers attempting forgery or abuse
- Malicious tenants attempting cross-tenant access
- Compromised internal components or infrastructure

## Trust Boundaries
- Internet clients to issuer/verifier HTTP services
- Services to the Postgres database that stores trust registry entries
- Services to external or future key management systems (KMS/HSM)
- Gateway/service-to-service calls carrying delegated credentials

## Threats (STRIDE)
- **Spoofing**: Fake issuers or agents presenting forged credentials
- **Tampering**: Manipulating credential payloads, delegation chains, or logs
- **Repudiation**: Denial of issued or verified actions without auditability
- **Information Disclosure**: Leakage of credentials, keys, or tenant metadata
- **Denial of Service**: Flooding verifier/issuer endpoints or database dependencies
- **Elevation of Privilege**: Delegation misuse or cross-tenant trust leakage

## Mitigations (Current State)
- Signature verification and DID binding for every credential
- Delegation rules enforce reduced scope/TTL and maximum depth
- Trust registry with per-tenant entries (memory or Postgres-backed storage)
- Structured HTTP request logs that include request/tenant IDs and status
- Verification responses report explicit failure reasons
- Automated tests covering invalid chains, expired credentials, and untrusted issuers

## Open Questions / Future Work
- Use of KMS/HSM for protecting issuer keys
- Rate limiting and DoS protections at the edge
- Integrity protections for audit logs (e.g., signing logs or shipping to WORM storage)
- Formal penetration testing and cryptography reviews
