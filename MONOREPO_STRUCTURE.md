# ServiceForge Monorepo Structure

```text
ServiceForge/
  apps/
    management-ui/            # Next.js admin shell
  services/
    api-gateway/              # Tenant-aware gateway and edge routing
    auth-service/             # Token and auth APIs
    tenant-service/           # Tenant lifecycle and plan assignment
    config-service/           # JSON-schema-driven module configuration
    booking-service/          # Phase 1 module with CRUD endpoints
  packages/
    go-common/                # Shared tenant, config, and event utilities
    contracts/
      openapi/                # Tenant/module API contracts
  deploy/
    docker/                   # Local development infra
    helm/                     # Kubernetes charts (to be added)
    terraform/                # IaC modules (to be added)
  docs/
    architecture/
    srs/
  scripts/
  .github/workflows/
```

## Scaling Design Notes
- Keep each module as a separate deployable service to scale independently.
- Use async communication (Kafka) for cross-module reactions.
- Propagate tenant context end-to-end using `X-Tenant-ID` and RLS-ready data models.
- Maintain contracts in `packages/contracts` to generate tenant-specific docs and SDKs.
