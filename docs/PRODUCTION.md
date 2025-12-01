# Production Configuration Guide

This guide highlights defaults and recommended settings for running the credential service in production.

## Configuration Recommendations
- **Database**: Run Postgres with backups enabled; size for credential issuance volume and trust registry lookups.
- **Signing Keys**: Plan for KMS or HSM-backed signing keys (future phase) to keep private keys out of containers.
- **LOG_LEVEL**: Set `info` or `warn` in production; `debug` is noisy and should be short-lived.
- **Metrics**: Enable metrics collection via the provided hooks; export to your observability stack.
- **Resource Limits**: Apply CPU and memory limits on containers to prevent noisy-neighbor issues.
- **Probes**: Use `/healthz` for liveness and `/readyz` for readiness; readiness will fail if DB connectivity is unavailable when enabled.

## Deployment Patterns
- Run issuer and verifier as separate Deployments/Services.
- Place the gateway authorize endpoint behind a stable Service or API gateway.
- Deploy Postgres with automated backups and monitoring.

## Multi-Tenant Considerations
- Maintain a tenant registry (future work) to partition trust registry entries.
- Keep per-tenant trust registry rows to avoid cross-tenant trust leakage.
- Enforce per-tenant verification using the `tenant_id` passed into trust registry lookups.
