---
title: Developer Guide
description: Local development setup, adding new modules, and architectural rules for contributors
order: 7
---

# Developer Guide

ServiceForge is built entirely on open-source components. Go powers all microservices; Next.js 14 (App Router) powers the management UI. The dev workflow requires only Docker for the infrastructure layer.

---

## Prerequisites

| Tool | Minimum version |
|---|---|
| Go | 1.23 |
| Node.js | 20 |
| Docker + Docker Compose | 24 / v2 |
| `make` | any modern version |

---

## Running the Full Stack

Start all infrastructure and services with one command:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml up --build -d
```

This builds images and starts:
- PostgreSQL (with automatic schema migrations via `migrate`)
- Redis
- Kafka + Zookeeper
- All Go microservices
- The Next.js management UI

---

## Running Services Locally (Hot Reload)

For faster iteration, run infrastructure in Docker and services natively.

### Step 1 — Start only infrastructure

```bash
docker compose -f deploy/docker/docker-compose.dev.yml up -d \
  postgres redis kafka zookeeper
```

### Step 2 — Run individual services

From the repository root, use the Makefile targets:

```bash
make auth       # starts auth-service on :8082
make tenant     # starts management-service on :8081
make config     # starts config-service on :8085
make booking    # starts booking-service on :8084
make gateway    # starts api-gateway on :8080
```

### Step 3 — Run the management UI

```bash
cd apps/management-ui
npm install
npm run dev
# UI available at http://localhost:3000
```

### Step 4 — Run the docs site

```bash
cd docs/web
npm install
npm run dev
# Docs available at http://localhost:3001
```

---

## Repository Structure

```
serviceforge/
├── apps/
│   └── management-ui/          # Next.js 14 control-plane dashboard
│       ├── app/                # App Router pages and layouts
│       ├── components/         # Reusable React components
│       └── lib/api/            # Typed fetch wrappers per service
│
├── services/
│   ├── auth-service/           # Authentication microservice
│   ├── booking-service/        # Booking lifecycle + policy enforcement
│   ├── config-service/         # JSON-Schema-driven config storage
│   ├── management-service/     # Tenant and API-key CRUD
│   └── api-gateway/            # Routing, CORS, auth middleware
│
├── packages/
│   └── go-common/              # Shared Go packages
│       ├── config/             # Environment variable helpers
│       ├── logger/             # slog-based structured logger
│       ├── middleware/         # CORS, request logging
│       └── tenant/             # Tenant context extraction
│
├── db/
│   └── migrations/             # SQL migrations (golang-migrate format)
│
├── deploy/
│   └── docker/                 # Docker Compose files
│
└── docs/
    ├── web/                    # This documentation site (Next.js)
    └── on-prem/                # On-premise deployment configs
```

---

## Go Service Internals

Each Go microservice follows the same four-layer structure:

```
cmd/server/main.go      ← wiring: DB, Kafka, config client, HTTP server
internal/handler/       ← HTTP request/response, policy enforcement
internal/repository/    ← data access (interface + postgres implementation)
internal/domain/        ← domain types, errors, validation
```

### Adding a new repository method

1. Add the method signature to the `Repository` interface in `repository.go`
2. Implement it in `postgres.go`
3. Use it in `handler.go`

Never call database code directly from `handler.go` — always go through the interface. This keeps the handler testable with mock repositories.

### Tenant isolation in SQL

All queries must filter by `tenant_id`. The `set_tenant_context()` PostgreSQL function sets a session variable used by RLS policies:

```go
// In repository — always include tenantID
const q = `SELECT * FROM bookings WHERE tenant_id = $1 AND id = $2`
row := pool.QueryRow(ctx, q, tenantID, id)
```

**Never** run a query without a `tenant_id` filter on a tenant-scoped table. Cross-tenant data access is the most critical bug class to prevent.

---

## Adding a New Module

ServiceForge is designed for progressive capability addition. A new module requires changes in two places.

### 1. Register the schema (config-service)

Create a new SQL migration in `db/migrations/`:

```sql
-- 000008_seed_payments_module.up.sql
INSERT INTO module_schemas (module, version, schema, defaults, description)
VALUES (
  'payments',
  1,
  '{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "title": "Payments",
    "properties": {
      "currency": {
        "type": "string",
        "enum": ["USD","GBP","EUR","AUD"],
        "default": "USD",
        "description": "Billing currency",
        "x-ui": {"component": "select"}
      },
      "requireDepositPercent": {
        "type": "integer",
        "minimum": 0,
        "maximum": 100,
        "default": 0,
        "description": "Deposit percentage required at booking time",
        "x-ui": {"component": "slider", "step": 5}
      },
      "allowPartialRefunds": {
        "type": "boolean",
        "default": true,
        "description": "Allow partial refunds",
        "x-ui": {"component": "toggle"}
      }
    },
    "additionalProperties": false
  }',
  '{"currency":"USD","requireDepositPercent":0,"allowPartialRefunds":true}',
  'Payment and billing configuration'
);
```

Run the migration:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate
```

The management UI's Config page automatically discovers and renders new modules — no UI code changes needed.

### 2. Consume the config in your new service

Create a typed struct in your service's `configclient` package:

```go
// services/payments-service/internal/configclient/client.go
type PaymentsCfg struct {
    Currency                string `json:"currency"`
    RequireDepositPercent   int    `json:"requireDepositPercent"`
    AllowPartialRefunds     bool   `json:"allowPartialRefunds"`
}

func (c *Client) GetPaymentsConfig(ctx context.Context, tenantID string) PaymentsCfg {
    raw, isDefault, err := c.fetchFull(ctx, tenantID, "payments")
    if err != nil || isDefault {
        return PaymentsCfg{Currency: "USD", AllowPartialRefunds: true}
    }
    var cfg PaymentsCfg
    if err := decode(raw, &cfg); err != nil {
        return PaymentsCfg{Currency: "USD", AllowPartialRefunds: true}
    }
    return cfg
}
```

### 3. Add a Kafka topic

Define the topic constant and event types in your service's `events` package:

```go
const TopicPaymentInitiated = "payment.initiated"

type PaymentInitiatedEvent struct {
    EventID    string    `json:"eventId"`
    EventType  string    `json:"eventType"`
    OccurredAt time.Time `json:"occurredAt"`
    TenantID   string    `json:"tenantId"`
    BookingID  string    `json:"bookingId"`
    AmountCents int      `json:"amountCents"`
    Currency   string    `json:"currency"`
}
```

---

## Architectural Rules

These rules are enforced in code review for all contributions:

### Never mutate cross-tenant contexts

All repository methods take `tenantID` as the first argument. If you find yourself constructing a query without a tenant filter, stop and add it.

### Graceful config degradation

Every `GetXxxConfig()` method on the config client must return a safe, permissive default on any error. A config-service outage should never break the booking flow.

```go
// Good
func (c *Client) GetFooConfig(ctx context.Context, tenantID string) FooCfg {
    raw, _, err := c.fetchFull(ctx, tenantID, "foo")
    if err != nil {
        return defaultFooCfg()   // ← always fall back
    }
    // ...
}
```

### Repository interface, not pgx directly

`handler.go` must only call `repository.Repository` methods. Direct `pgx` calls in the handler are not permitted. This keeps handlers unit-testable.

### One Kafka topic per event type

Topics are named `<domain>.<event>` in lower-kebab-case, e.g. `booking.created`, `payment.initiated`. Never reuse a topic for multiple semantically different events.

### Use Redis for short-lived cache only

Redis is volatile by design. Do not store any data in Redis that you cannot reconstruct from PostgreSQL. Session tokens, rate-limit counters, and config caches are appropriate. User records are not.

---

## Running Tests

```bash
# All Go services
go test ./...

# Single service
cd services/booking-service && go test ./...

# Management UI
cd apps/management-ui && npm test
```

---

## Linting and Formatting

```bash
# Go
gofmt -w ./...
golangci-lint run

# TypeScript / Next.js
cd apps/management-ui && npm run lint
```

---

## Database Migrations

Migrations live in `db/migrations/` and follow the `golang-migrate` naming convention:

```
000001_create_tenants.up.sql
000001_create_tenants.down.sql
000002_create_bookings.up.sql
000002_create_bookings.down.sql
...
```

To apply migrations in the dev compose stack:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate
```

To roll back the last migration:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate down 1
```
