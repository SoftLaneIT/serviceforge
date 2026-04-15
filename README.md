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

# ServiceForge

Open-source, multi-tenant business capability platform that lets teams activate configurable modules (Booking, Payment, Queue, Logging, and more) without building each capability from scratch.

## Project Identity

- Organization: SoftLaneIT
- Repository: SoftLaneIT/serviceforge
- Platform Name: ServiceForge
- License: Apache License 2.0

## Why ServiceForge

ServiceForge is built for product teams that need reusable business capabilities with strong tenant isolation, API-first integration, and configuration-driven behavior.

Core principles:

- Configuration over customization for faster adoption
- Event-driven module integration for loose coupling and resilience
- Tenant-aware request flow from gateway to data layer
- Progressive extensibility: no-code setup first, plugins later

## Current Platform Scope

Phase 1 foundation delivered in this repository includes:

- API Gateway skeleton
- Tenant Management service skeleton
- Auth service skeleton
- Configuration service skeleton
- Booking module skeleton with CRUD endpoints
- Management UI shell (Next.js)
- Local development infrastructure (PostgreSQL, Redis, Kafka)

## Architecture Overview

### Logical Layers

1. Access Layer: API Gateway + Management UI
2. Core Services: Tenant, Auth, Config, Booking
3. Integration Layer: Kafka event backbone
4. Data Layer: PostgreSQL + Redis cache
5. Platform Layer: Deployment, CI, observability hooks

### Monorepo Structure

```text
ServiceForge/
  apps/
    management-ui/
  services/
    api-gateway/
    auth-service/
    tenant-service/
    config-service/
    booking-service/
  packages/
    go-common/
    contracts/
  deploy/
    docker/
    helm/
    terraform/
  docs/
```

See full structure notes in [MONOREPO_STRUCTURE.md](MONOREPO_STRUCTURE.md).

## Technology Stack

- Backend Services: Go
- Frontend: Next.js + TypeScript
- Database: PostgreSQL
- Cache: Redis
- Event Bus: Kafka
- Infrastructure: Docker Compose (local), Helm + Terraform (target)
- CI: GitHub Actions

## Getting Started

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker + Docker Compose

### Start Local Dependencies

```bash
make up
```

### Run Services (separate terminals)

```bash
make auth
make tenant
make config
make booking
make gateway
```

### Run Management UI

```bash
cd apps/management-ui
npm install
npm run dev
```

UI runs on `http://localhost:3000`.

## API and Contracts

- Booking OpenAPI contract: [packages/contracts/openapi/booking.v1.yaml](packages/contracts/openapi/booking.v1.yaml)
- Service entrypoints live under each service `cmd/server`

## Roadmap

1. Phase 1 (Months 1-3): Foundation and first module end-to-end
2. Phase 2 (Months 3-5): Payment, Queue, Logging, catalog subscription flow
3. Phase 3 (Months 5-7): Per-tenant docs, SDKs, onboarding, sandbox
4. Phase 4 (Months 7-9): Billing tiers, rate limits, scale hardening
5. Phase 5 (Months 9-12): Plugin ecosystem, marketplace, white-label support

## Open Source Governance

- License: [LICENSE](LICENSE)
- Contributing guide: [CONTRIBUTING.md](CONTRIBUTING.md)
- Code of conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Security policy: [SECURITY.md](SECURITY.md)

## Community and Support

- Issues: use GitHub Issues for bugs and feature requests
- Discussions: use GitHub Discussions for architecture and roadmap conversations
- Security reports: follow [SECURITY.md](SECURITY.md)
