# Tenancy Fundamentals

Multi-tenancy allows isolated credential issuance, verification, and policy evaluation for distinct customers or environments while sharing a common platform runtime. It keeps keys, trust anchors, and audit trails scoped to the tenant boundary and enables safer delegation and rotation workflows.

## Modes
- **Single-tenant (default):** Behaves like prior releases. The `DEFAULT_TENANT_ID` is injected automatically when the `X-Tenant-ID` header is absent.
- **Multi-tenant:** Requires explicit tenant selection via `X-Tenant-ID`. Requests without a tenant fail fast to prevent cross-tenant leakage.

## Request Flow
```
[Client] --X-Tenant-ID--> [Tenant Middleware] --validates--> [Handler]
      |                                            |
      |                         injects tenant into context
      +--> Trust registry + policy engine consulted with tenant scope
```

## Trust Registry per Tenant
Trusted issuers are stored per-tenant (database or in-memory). Lookups always include the resolved tenant ID, ensuring issuers trusted for one tenant are not implicitly trusted for another.

## Tenant Discovery
Tenants are resolved in this order:
1. `X-Tenant-ID` header (required in multi-tenant mode)
2. Default tenant when running in single-tenant mode
3. Validation against the tenant store (Postgres or in-memory)

Disabled or unknown tenants return a clear API error before any business logic executes.

## Deployment Notes
- Set `TENANCY_MODE=multi` to enforce explicit tenant selection.
- Seed the `tenants` table (or in-memory store) with enabled tenants and default signing keys.
- Database migrations add `tenants` and `tenant_trusted_issuers` tables; existing flows continue to work with the legacy single-tenant defaults.
- Docker Compose continues to run in single-tenant mode unless overridden via environment variables.
