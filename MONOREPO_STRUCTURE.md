<!--
Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.

SoftlaneIT licenses this file to you under the Apache License,
Version 2.0 (the "LICENSE"); you may not use this file except
in compliance with the LICENSE.
You may obtain a copy of the LICENSE at

https://softlaneit.com/LICENSE.txt

Unless required by applicable law or agreed to in writing,
software distributed under the LICENSE is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the LICENSE for the
specific language governing permissions and limitations
under the LICENSE.
-->

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
