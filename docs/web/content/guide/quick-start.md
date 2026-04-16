---
title: Quick Start
description: Stand up the full platform and create your first booking in under five minutes
order: 2
---

# Quick Start

Get a booking live in under five minutes using Docker Compose. No build toolchain required.

---

## Prerequisites

- Docker 24+ and Docker Compose v2
- `curl` and `jq` (optional, for pretty-printing JSON)
- Ports 3000, 8080–8085, 5432, 6379, 9092 free on your machine

---

## Step 1 — Clone and Start

```bash
git clone https://github.com/SoftLaneIT/serviceforge.git
cd serviceforge
docker compose -f deploy/docker/docker-compose.dev.yml up -d
```

All services start automatically. Database migrations run as part of the `migrate` container. Wait about 20 seconds for everything to become healthy:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml ps
```

You should see every service show `healthy` or `running`.

---

## Step 2 — Create a Tenant

A **tenant** is the isolated workspace every resource belongs to. Create one via the management-service:

```bash
curl -s -X POST http://localhost:8081/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","slug":"acme","plan":"pro"}' | jq .
```

Response:

```json
{
  "id": "01HZ9EXAMPLE000000000000",
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "pro",
  "status": "active",
  "createdAt": "2026-04-17T10:00:00Z"
}
```

Save the `id` — you will use it as the `X-Tenant-ID` header on all booking and config calls.

---

## Step 3 — Create an API Key

```bash
curl -s -X POST http://localhost:8081/v1/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name":"acme-prod-key","tenantId":"01HZ9EXAMPLE000000000000"}' | jq .
```

Response:

```json
{
  "id": "key_abc123",
  "tenantId": "01HZ9EXAMPLE000000000000",
  "name": "acme-prod-key",
  "key": "sf_live_xxxxxxxxxxxxxxxx",
  "createdAt": "2026-04-17T10:01:00Z"
}
```

> **Important:** The `key` field is shown only once. Copy it to a secure location — it cannot be retrieved again.

---

## Step 4 — Create Your First Booking

All times must be sent in **RFC 3339 / ISO 8601** format with a timezone offset (e.g. `2026-04-20T09:00:00Z`). The `Z` suffix means UTC.

```bash
curl -s -X POST http://localhost:8084/v1/bookings \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ9EXAMPLE000000000000" \
  -d '{
    "customerRef": "customer_42",
    "serviceRef":  "haircut",
    "slotStart":   "2026-04-20T09:00:00Z",
    "slotEnd":     "2026-04-20T10:00:00Z"
  }' | jq .
```

Response:

```json
{
  "id": "bk_abcdef123456",
  "tenantId": "01HZ9EXAMPLE000000000000",
  "customerRef": "customer_42",
  "serviceRef": "haircut",
  "slotStart": "2026-04-20T09:00:00Z",
  "slotEnd": "2026-04-20T10:00:00Z",
  "status": "pending",
  "metadata": {},
  "createdAt": "2026-04-17T10:02:00Z",
  "updatedAt": "2026-04-17T10:02:00Z"
}
```

The booking is created with `status: pending`. To auto-confirm all bookings, enable `autoConfirm` in the booking module config (see the Configuration Modules page).

---

## Step 5 — Open the Management Dashboard

Navigate to `http://localhost:3000` in your browser. From the dashboard you can:

- Create and manage tenants visually
- Issue and revoke API keys
- View, create, and update bookings
- Configure all six module settings per tenant
- Explore integrations and webhook setup
- Browse this developer documentation

---

## Next Steps

| Goal | Where to look |
|---|---|
| Configure booking rules | Config Modules → `booking` |
| Set operating hours | Config Modules → `business-hours` |
| Enable webhooks | Config Modules → `notifications` |
| List / filter bookings via API | API Reference → Bookings API |
| Integrate from Node.js | TypeScript Integration |
| Deploy to a server | Self-Hosting |

---

## Troubleshooting

**Services not starting:**
Check that no other process is using the required ports. Run `docker compose ps` and look for containers in `Exit` state, then inspect their logs:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml logs booking-service
```

**`400 X-Tenant-ID header is required`:**
Every call to the booking-service and config-service must include `-H "X-Tenant-ID: <uuid>"`.

**`422 slot start must be in the future`:**
The `slotStart` you sent has already passed. Use a future UTC timestamp.

**`422 invalid JSON body`:**
Ensure `slotStart` and `slotEnd` are full RFC 3339 strings including the `Z` or offset suffix. The string `2026-04-20T09:00` (no timezone) will be rejected.
