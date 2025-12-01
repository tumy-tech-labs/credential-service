# Policy Engine v1

The policy engine evaluates authorization decisions after credentials are verified and the tenant is resolved. Version 1 introduces a hybrid RBAC/ABAC model with tenant-scoped policies stored in Postgres (or the in-memory store for tests and local development).

## Input Model
- `TenantID`: Tenant derived from `X-Tenant-ID` or the default tenant in single-tenant mode.
- `Subject`: DID of the acting subject.
- `ActingOnBehalfOf`: Delegation chain root (if applicable).
- `Scope`: Requested scopes derived from credential claims.
- `Claims`: Raw claims from the leaf credential.
- `Resource`: Target resource path the caller is requesting.
- `Action`: Operation being authorized (verb like `read`, `write`).
- `Context`: Additional request context (path, delegation depth, etc.).

## Output Model
- `Allow`: Boolean decision.
- `Reason`: String describing the decision path (`policy_allow`, `policy_deny`, `no_matching_policy`).
- `PolicyID`: Identifier of the policy that triggered the decision (when applicable).

## Policy Schema
- `effect`: `allow` or `deny`.
- `actions`: List of verbs.
- `resources`: List of resource matchers. Exact match or prefix when ending with `*` (e.g., `orders/*`).
- `subjects`: List of subjects. Matches when:
  - DID equals the request subject.
  - Entry is `role:<name>` and the role exists in claims[`roles`].
  - Entry is `any`.
- `conditions`: Optional map supporting:
  - `scope_contains`: String or list that must be present in `Scope`.
  - `claim_equals`: Map of claim keys to exact values.
- `priority`: Lower numbers evaluated first; deny overrides allow.
- `enabled`: Disabled policies are ignored.

## Evaluation Rules
1. Gather policies for the tenant and filter by action, resource, subject, and conditions.
2. Sort matching policies by `priority` then `id`.
3. If any matching policy has `effect=deny`, the request is denied.
4. If one or more matching `allow` policies exist and no deny matched first, the request is allowed.
5. If no policies match, the engine denies by default.

## Admin API
Policies are managed per-tenant under `/v1/admin/policies`:
- `POST /v1/admin/policies` – create a policy.
- `GET /v1/admin/policies` – list policies for the tenant.
- `GET /v1/admin/policies/{id}` – fetch a policy.
- `PUT /v1/admin/policies/{id}` – update a policy.
- `DELETE /v1/admin/policies/{id}` – delete a policy.

All responses include `api_version`. Validation errors return standardized API errors.

## Gateway Integration
The gateway authorize handler calls the policy engine after credential verification. If policy evaluation denies (including no matching policy), the gateway returns `{allowed:false, reason:"policy_denied"}` with the `policy_id` when available. Allow decisions include the `policy_id` that permitted the request.

## Examples
```json
{
  "name": "orders-read",
  "effect": "allow",
  "actions": ["read"],
  "resources": ["orders/*"],
  "subjects": ["role:agent"],
  "conditions": {"scope_contains": "orders:read"},
  "priority": 10,
  "enabled": true
}
```

```json
{
  "name": "maintenance-freeze",
  "effect": "deny",
  "actions": ["write"],
  "resources": ["*"],
  "subjects": ["any"],
  "priority": 1
}
```

## Future Enhancements
- Policy bundles and versioning.
- Additional condition operators (time windows, IP ranges).
- Policy simulation and dry-run endpoints.
- Admin audit trails.
