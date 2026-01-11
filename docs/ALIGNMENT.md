# Alignment Report

## What is aligned
- Services already expose issue/delegate/verify/authorize endpoints plus synthetic JWT minting, matching the gateway + SDK focus.
- Docker Compose spins up issuer, verifier/gateway, Postgres, and a sample API, giving us a deterministic local stack to build the demo on.
- Default gateway policy seeds an `orders:read` allow rule and synthetic JWT decoding exists in the sample API, showing legacy bearer compatibility.
- Go SDK already provides simple constructors (`NewLocalDevClient`, `NewClientFromEnv`) and thin HTTP wrappers for issuance, verification, delegation, and authorization.
- Trust registry and policy engine are tenant-scoped with memory/DB backends, aligning with “credential-aware gateway” positioning rather than CIAM logins.

## What is not aligned
- README and docs market a broad CIAM/decentralized identity platform with complex deployment diagrams (centralized vs distributed, multi-org federations) rather than a focused developer tool + gateway story.
- Quickstart omits agents/constraints and requires manual log scraping to register issuers; there is no 5-minute human→agent→API path.
- SDK `AuthorizeRequest` omits required `resource`/`action`, so the typed client cannot drive policy enforcement—the current example only proves verification.
- No single scripted “killer demo”; users must glue together issuer, trust registry, gateway, and sample API calls manually.
- Documentation is sprawling (15+ docs) with deep DID/tenancy/policy detail before a simple path to success; “when NOT to use VCs” and scope narrowing are absent.
- Tests/examples don’t cover the synthetic JWT + agent flow; there is no integration guardrail for the happy path.

## What is too complex or confusing now
- Dual deployment topologies and long architecture sections obscure the minimal gateway + SDK value prop.
- Prerequisites (keygen binary, log parsing for issuer DID, optional admin bootstrap) add friction to the first run.
- Policy model is described in detail but the SDK and docs don’t surface the minimal fields (resource/action) needed for a deny reason.
- Multiple SDKs and examples are listed without a clear “use this one” entry point.

## What should be simplified or removed
- De-emphasize CIAM/federated identity messaging; lead with “credential-aware gateway + tiny SDK.”
- Collapse docs into quickstart + killer demo first, with advanced topics linked later.
- Provide a single copy/paste script for the full flow (issue → spawn agent → authorize → synthetic JWT → sample API → denied call) and remove manual log scraping instructions.
- Offer a high-level Go client that hides resource/action wiring and delegation mechanics.
- Keep DID/SD-JWT details in an advanced section instead of the main README.

## Current developer happy path and where it breaks
1. `docker compose up` brings up issuer/verifier but does not surface issuer DID; user must parse logs to trust it.
2. `make keygen` + `/v1/credentials/issue` can mint a VC, but scope/TTL constraints for agents are undocumented.
3. `/v1/trust/issuers` must be called manually with the issuer DID; this step is easy to miss and yields `issuer_not_trusted` errors.
4. `/v1/gateway/authorize` works for a single credential but SDK types omit `resource/action`, so policy enforcement isn’t exercised.
5. Sample API `/orders` can be called with a synthetic JWT, but there’s no scripted flow to prove deny reasons or agent delegation.

## Killer demo readiness
- **Can we do a 5-minute flow today?** Not reliably. The pieces exist but require manual DID lookup, ad-hoc curl commands, and guessing the required fields.
- **Missing for 5-minute demo:**
  - One-shot script to run the full sequence and print clear outputs.
  - SDK surface that wires resource/action and exposes an opinionated `SpawnAgent` helper.
  - README that starts with the killer demo and clarifies when *not* to use VCs.
  - Example code mirroring the demo.

---

# Simplicity Refactor Plan (PR-sized steps)
1. **SDK: Magical facade** — Add a high-level Go client (`MagicClient`) with `IssueVC`, `Verify`, `Authorize(resource, action)`, and `SpawnAgent` helpers that default to local compose URLs; align `AuthorizeRequest` with gateway schema.
2. **CLI/Demo flow** — Provide a single `scripts/demo_killer_5min.sh` that starts (or verifies) the compose stack and runs human→agent→gateway authorize→synthetic JWT→sample API→deny outside scope.
3. **Docs rewrite** — Reorder README: 60-second quickstart, 5-minute killer demo (calling the script), concise “how it works,” explicit “when NOT to use VCs,” and links to advanced topics.
4. **Examples** — Add a Go example (`examples/go/killer-demo`) that uses the high-level client to mirror the demo calls.
5. **Testing guardrail** — Add/extend integration coverage to exercise the killer demo flow (gateway allow + deny). (Future PR placeholder—automate the script steps.)

