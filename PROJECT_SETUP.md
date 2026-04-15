# ServiceForge Platform Setup

## Identity
- Project Name: ServiceForge
- Repository Name: SoftLaneIT/serviceforge
- Proposed Platform Repo Name: serviceforge-platform
- Tagline: Build less. Ship more.

## Recommended Technology Stack
- Backend services: Go (API gateway, tenant, auth, config, booking)
- Management UI: Next.js + TypeScript
- Database: PostgreSQL with row-level tenant isolation
- Event bus: Apache Kafka
- Cache: Redis
- Observability: OpenTelemetry + Prometheus + Grafana (next increment)
- Deployment: Kubernetes + Helm, Terraform for cloud infra

## Why this stack
- Go provides high concurrency and low resource usage for multi-tenant APIs.
- Next.js provides a fast dashboard UX and clean routing for B2B admin panels.
- PostgreSQL RLS balances isolation, cost, and operational simplicity.
- Kafka enables decoupled modules and replayable domain events.

## Phase-1 implementation in this repo
- API gateway skeleton
- Tenant service skeleton
- Auth service skeleton
- Configuration service skeleton
- Booking service skeleton
- Management UI shell
- Local infra via Docker Compose (Postgres, Redis, Kafka)

## Local startup
1. Start dependencies
   - `make up`
2. Run services (one terminal each)
   - `make auth`
   - `make tenant`
   - `make config`
   - `make booking`
   - `make gateway`
3. Run UI
   - `make ui`

## GitHub repository creation
Create the repository with GitHub CLI:

```bash
gh repo create SoftLaneIT/serviceforge-platform --private --source=. --push
```

Or create manually from github.com/new with the name `serviceforge-platform`.
