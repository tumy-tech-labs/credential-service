# Agent Lifecycle

## 1. What is an Agent?
AI agents operate on behalf of a human or service principal using delegated verifiable credentials. They hold their own DID but act with authority derived from a parent credential.

## 2. Identity Model
- **Agent DID** uniquely identifies the agent.
- **Delegated VC** is a short-lived credential scoped down from a parent token.
- **Acting On Behalf Of chain** tracks the subject that granted delegation.
- **Short-lived credentials** minimize blast radius and encourage regular rotation.

## 3. Onboarding
1. Generate a DID/keypair for the agent.
2. Store the private key securely (KMS or sealed secrets).
3. (Optional) Register the agent in a trust registry for auditable discovery.

## 4. Delegation Flow
1. Human token → delegation endpoint → agent token.
2. Scope must shrink or stay equal to the parent scope.
3. TTL must shrink relative to the parent TTL.
4. Delegation depth is capped to prevent infinite chains.

## 5. Acting On Behalf Of
Verifiers and gateways derive the ultimate delegator by walking the credential chain. Downstream services can surface this chain to show the human a given agent is acting for.

## 6. Refresh / Rotation
Agents request short-lived credentials and refresh before expiry. SDK helpers automatically re-delegate using the parent token while respecting scope/TTL constraints.

## 7. Offboarding
Revoke the parent key or VC; delegated child VCs become invalid automatically.

## 8. Security Model
- No scope escalation is allowed.
- TTL must never exceed the parent TTL.
- Delegation depth limits are enforced during verification.
