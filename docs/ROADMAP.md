# Credential Service — Roadmap

This document defines the **official roadmap** for the Credential Service project.  
It is written to support **AI-assisted development (Codex/Copilot)** and enables developers to quickly understand what is implemented, what is in progress, and what comes next.

The roadmap is structured in **phases**, each covering features and responsibilities across the system:

- DID Service  
- Issuer Service  
- Verifier Service  
- Gateway / PEP  
- Trust Registry  
- Key Management  
- SDKs  
- Documentation  
- DX & Tooling  
- Testing & Security  

---

# 📌 Current Status (Summary)

The project currently provides:

- Working local dev environment via Docker Compose  
- Issuer, Verifier, Gateway, and Sample API  
- Issuer DID creation  
- VC issuance and verification  
- Authorization gateway + Synthetic JWT generation  
- Basic tenant support  
- Go and Node.js SDKs  
- Initial policy engine  
- Demo-ready end-to-end flow

This is a **strong MVP** that demonstrates core functionality.

Next steps focus on **identity lifecycle, key management, security hardening, delegation, revocation, testing, and DX**.

---

# 🗺️ Roadmap Overview (High-Level)

## Phase 0 — **MVP Foundation (Complete / Nearly Complete)**

- [x] Issuer service  
- [x] Verifier service  
- [x] Gateway service  
- [x] VC issuance (short-lived)  
- [x] VC verification  
- [x] Basic policy enforcement  
- [x] Synthetic JWT integration for legacy APIs  
- [x] Demo API (payments)  
- [x] Docker Compose environment  
- [x] Basic DID service (issuer only)  
- [x] Initial docs & SDKs (Go/Node)

---

## Phase 1 — **Platform Hardening (In Progress)**

### Identity Lifecycle (DIDs)
- [ ] Full DID lifecycle for holders (users, services)  
- [ ] Full DID lifecycle for agents (ephemeral, delegated)  
- [ ] DID metadata storage (owner_id, owner_type, created_at)  
- [ ] DID → internal identity mapping  
- [ ] DID rotation  
- [ ] DID deactivation  

### Key Management
- [ ] Unified key registry service  
- [ ] Key metadata (kid, status, creation/rotation)  
- [ ] Multiple active keys per DID  
- [ ] Key rotation API  
- [ ] Admin CLI or SDK support  

### Platform Authentication
- [ ] VC-native admin authentication  
- [ ] Super-root user bootstrap  
- [ ] Admin roles (tenant admin, issuer admin, policy admin)  
- [ ] Admin endpoints to manage:  
  - Tenants  
  - Issuers  
  - Trust registry  
  - Policies  
  - Key rotation  

### Credential Lifecycle
- [ ] Credential revocation  
- [ ] Credential status (StatusList or CRL-like)  
- [ ] Long-lived credential support (optional)  

### Delegation & Agents
- [ ] Delegation VC format  
- [ ] Delegation chain enforcement  
- [ ] TTL narrowing logic  
- [ ] Delegation depth limits  
- [ ] Agent DIDs  
- [ ] Agent impersonation prevention logic  

### Multi-Tenant Hardening
- [ ] Strict tenant boundary checks across all endpoints  
- [ ] Per-tenant issuer registry  
- [ ] Tenant-level isolation tests  

---

## Phase 2 — **Developer Experience & DX Acceleration**

### SDK Enhancements
- [ ] Expand Go SDK with typed models + helper flows  
- [ ] Expand Node SDK with examples and tests  
- [ ] Python SDK (optional but recommended)  
- [ ] Unified error handling & response typing  

### Sample Apps (DX Kits)
- [ ] Go sample app (protected API)  
- [ ] Node/Express protected API  
- [ ] Delegation chain example  
- [ ] AI agent sample using VC for identity  
- [ ] Demo kit with one-click setup + curl flows  

### Documentation
- [ ] Rewrite top-level README.md (in progress)  
- [ ] Full architecture document with diagrams  
- [ ] Credential format specification  
- [ ] DID and key management spec  
- [ ] Admin guide  
- [ ] Threat model updates  

---

## Phase 3 — **Security, Testing & Observability**

### Automated Tests
- [ ] Full suite of unit tests  
- [ ] End-to-end tests (issue → verify → authorize → API)  
- [ ] Negative tests  
  - Tampered VC  
  - Expired VC  
  - Wrong tenant  
  - Insufficient scope  
  - Revoked credential  
  - Invalid delegation  
- [ ] Load/concurrency testing  
- [ ] Testing for multi-tenant isolation  

### Observability
- [ ] Structured logging across services  
- [ ] Metrics (OTel)  
- [ ] Tracing (OpenTelemetry)  
- [ ] Audit logs for all identity/key/credential actions  

### Trust Registry
- [ ] Hardening and full CRUD APIs  
- [ ] Trust policy enforcement per tenant  
- [ ] Admin UI or endpoints  

---

## Phase 4 — **Enterprise Readiness**

- [ ] Secure deployment guides  
- [ ] Vault hardened configuration  
- [ ] Rotation policies for issuer and admin keys  
- [ ] Cross-region replication patterns  
- [ ] Compliance-oriented logging (PII-safe)  
- [ ] SLA observability  
- [ ] Integration patterns with OAuth/OIDC providers  
- [ ] Optional support for Apple/Google wallets (future)  

---

# 🧩 Feature Matrix (Codex-Friendly)

### DID Support
- [x] Issuer DIDs  
- [ ] Holder DIDs  
- [ ] Service DIDs  
- [ ] Agent DIDs  
- [ ] DID rotation  
- [ ] DID deactivation  

### Key Management
- [x] Issuer signing keys  
- [ ] Rotation  
- [ ] Key registry  
- [ ] Multi-key support  
- [ ] Key metadata  

### Credentials
- [x] Issue credentials  
- [x] Verify credentials  
- [ ] Revoke credentials  
- [ ] Credential status service  
- [ ] Long-lived credentials  
- [ ] Credential refresh  

### Delegation
- [ ] Delegation VC format  
- [ ] Chain validation  
- [ ] TTL narrowing  
- [ ] Depth limits  
- [ ] Agent DIDs  

### Gateway / Authorization
- [x] Policy checks  
- [x] Synthetic JWT bridge  
- [ ] Complex policy rules  
- [ ] Delegation-aware policy enforcement  

### Tenancy
- [x] Basic multi-tenant support  
- [ ] Trust registry per tenant  
- [ ] Full isolation testing  

### Documentation
- [ ] Comprehensive README  
- [ ] Architecture diagrams  
- [ ] Detailed specs  
- [ ] Developer onboarding kits  

### Testing
- [ ] Unit tests  
- [ ] Integration tests  
- [ ] E2E tests  
- [ ] Security tests  
- [ ] Load tests  

---

# 🗓️ Development Rhythm (Recommended)

## Weekly Cadence
- 1–2 major roadmap items  
- Daily PRs from Codex with incremental progress  
- End-of-week roadmap update  

## Daily Cadence
When opening VS Code each day:

1. Review `ROADMAP.md`  
2. Open a task-specific command to Codex  
3. Codex generates code → commit → PR  
4. Update ROADMAP.md and TODO comments accordingly  

---

# 🧠 Codex Usage Guidelines

Codex should use the roadmap this way:

- Before writing code, analyze which section of the roadmap applies  
- Use TODO comments in service files to localize work  
- Follow component boundaries (issuer/verifier/gateway/did-service)  
- Use small, incremental PR-style diffs  
- Update ROADMAP.md automatically when tasks are completed or partially implemented  
- Suggest improvements at the end of each implementation  

### Example Codex Command
Analyze ROADMAP.md and implement the next highest-priority item for the DID Service.

# 🏁 Final Summary

The credential-service platform is now in the **MVP+** stage, with critical work ahead in:

- DID lifecycle  
- Key management & rotation  
- Credential revocation  
- Delegation chains  
- Multi-tenant isolation  
- Automated tests  
- Developer experience  

ROADMAP.md will serve as the **source of truth** for progress, priorities, and guidance for both human and AI-driven development.

