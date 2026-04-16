---
title: API Reference
description: Complete REST endpoint documentation for the management, booking, and config services
order: 3
---

# API Reference

ServiceForge exposes three REST APIs. All accept and return `application/json`. Error responses always carry a `{"error": "message"}` body.

---

## Authentication

### X-Tenant-ID Header

Every call to the **booking-service** and **config-service** must include:

```
X-Tenant-ID: <tenant-uuid>
```

Without it you receive `400 Bad Request`.

### Bearer Token

API keys issued through the management UI can be passed to the **management-service**:

```
Authorization: Bearer sf_live_xxxxxxxxxxxxxxxx
```

---

## Management Service

**Base URL:** `http://localhost:8081`

### Tenants

#### List tenants

```
GET /v1/tenants?limit=20&offset=0
```

Query parameters:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `limit` | integer | 20 | Max results to return |
| `offset` | integer | 0 | Pagination offset |
| `status` | string | — | Filter by `active`, `suspended`, or `deleted` |

```bash
curl "http://localhost:8081/v1/tenants?limit=10&offset=0" | jq .
```

Response:

```json
{
  "data": [
    {
      "id": "01HZ...",
      "name": "Acme Corp",
      "slug": "acme",
      "plan": "pro",
      "status": "active",
      "createdAt": "2026-04-17T10:00:00Z"
    }
  ],
  "total": 1
}
```

#### Create tenant

```
POST /v1/tenants
```

Request body:

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Human-readable display name |
| `slug` | string | yes | URL-safe unique identifier |
| `plan` | string | yes | `free` \| `starter` \| `pro` \| `enterprise` |
| `status` | string | no | Default: `active` |

```bash
curl -X POST http://localhost:8081/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","slug":"acme","plan":"pro"}' | jq .
```

#### Get tenant

```
GET /v1/tenants/:id
```

```bash
curl http://localhost:8081/v1/tenants/01HZ... | jq .
```

#### Update tenant

```
PUT /v1/tenants/:id
```

Send the full tenant object with the fields you want to change:

```bash
curl -X PUT http://localhost:8081/v1/tenants/01HZ... \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp (Updated)","slug":"acme","plan":"enterprise"}' | jq .
```

#### Delete tenant

```
DELETE /v1/tenants/:id
```

Soft-deletes the tenant (sets `status: deleted`). Returns `204 No Content`.

---

### API Keys

#### List API keys

```
GET /v1/api-keys?tenantId=<uuid>
```

Filter by `tenantId` to list keys for a specific tenant.

```bash
curl "http://localhost:8081/v1/api-keys?tenantId=01HZ..." | jq .
```

Response:

```json
{
  "data": [
    {
      "id": "key_abc123",
      "tenantId": "01HZ...",
      "name": "acme-prod-key",
      "createdAt": "2026-04-17T10:01:00Z"
    }
  ],
  "total": 1
}
```

Note: the `key` (raw token value) is **never returned** after creation.

#### Create API key

```
POST /v1/api-keys
```

| Field | Type | Required | Description |
|---|---|---|---|
| `tenantId` | string | yes | UUID of the owning tenant |
| `name` | string | yes | Human-readable label |

```bash
curl -X POST http://localhost:8081/v1/api-keys \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"01HZ...","name":"acme-prod-key"}' | jq .
```

Response (key only shown here):

```json
{
  "id": "key_abc123",
  "tenantId": "01HZ...",
  "name": "acme-prod-key",
  "key": "sf_live_xxxxxxxxxxxxxxxx",
  "createdAt": "2026-04-17T10:01:00Z"
}
```

#### Delete API key

```
DELETE /v1/api-keys/:id
```

Permanently revokes the key. Returns `204 No Content`.

---

## Booking Service

**Base URL:** `http://localhost:8084`

All endpoints require the `X-Tenant-ID` header.

### Booking Status Lifecycle

```
pending ──► confirmed ──► completed
   │                          │
   └──► cancelled             └──► no_show
```

| Status | Meaning |
|---|---|
| `pending` | Submitted, awaiting manual confirmation |
| `confirmed` | Confirmed by operator or auto-confirmed by config |
| `completed` | Service was delivered |
| `cancelled` | Cancelled before delivery |
| `no_show` | Customer did not appear |

### List bookings

```
GET /v1/bookings
```

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `status` | string | Filter by booking status |
| `limit` | integer | Max results (default: 20) |
| `offset` | integer | Pagination offset (default: 0) |

```bash
curl "http://localhost:8084/v1/bookings?status=confirmed&limit=20" \
  -H "X-Tenant-ID: 01HZ..." | jq .
```

Response:

```json
{
  "data": [
    {
      "id": "bk_abc123",
      "tenantId": "01HZ...",
      "customerRef": "customer_42",
      "serviceRef": "haircut",
      "slotStart": "2026-04-20T09:00:00Z",
      "slotEnd": "2026-04-20T10:00:00Z",
      "status": "confirmed",
      "metadata": {},
      "createdAt": "2026-04-17T10:02:00Z",
      "updatedAt": "2026-04-17T10:02:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### Create booking

```
POST /v1/bookings
```

| Field | Type | Required | Description |
|---|---|---|---|
| `customerRef` | string | yes | Your system's customer identifier |
| `serviceRef` | string | yes | Your system's service identifier |
| `slotStart` | RFC 3339 | yes | Start time, must be in the future |
| `slotEnd` | RFC 3339 | yes | End time, must be after slotStart |
| `metadata` | object | no | Arbitrary key-value pairs |

> **Time format:** Always use full RFC 3339 with timezone suffix, e.g. `2026-04-20T09:00:00Z`. A bare `datetime-local` string like `2026-04-20T09:00` will be rejected with `400 invalid JSON body`.

```bash
curl -s -X POST http://localhost:8084/v1/bookings \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "customerRef": "customer_42",
    "serviceRef":  "haircut",
    "slotStart":   "2026-04-20T09:00:00Z",
    "slotEnd":     "2026-04-20T10:00:00Z",
    "metadata":    {"notes": "first visit"}
  }' | jq .
```

The booking runs through the **policy enforcement pipeline** before it is stored. See the Booking Policy page for the full rule set.

### Get booking

```
GET /v1/bookings/:id
```

```bash
curl http://localhost:8084/v1/bookings/bk_abc123 \
  -H "X-Tenant-ID: 01HZ..." | jq .
```

### Update booking status

```
PATCH /v1/bookings/:id
```

| Field | Type | Required | Description |
|---|---|---|---|
| `status` | string | yes | New status value |

```bash
curl -X PATCH http://localhost:8084/v1/bookings/bk_abc123 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{"status":"completed"}' | jq .
```

Terminal statuses (`completed`, `cancelled`, `no_show`) cannot be changed again — the API returns `409 Conflict`.

### Cancel booking

```
DELETE /v1/bookings/:id
```

Sets the booking status to `cancelled`. Returns the updated booking object (not `204`).

```bash
curl -X DELETE http://localhost:8084/v1/bookings/bk_abc123 \
  -H "X-Tenant-ID: 01HZ..." | jq .
```

---

## Config Service

**Base URL:** `http://localhost:8085`

Tenant-scoped endpoints require `X-Tenant-ID`.

### List modules

```
GET /v1/modules
```

Returns all available modules with their JSON Schemas and default values. No tenant header needed.

```bash
curl http://localhost:8085/v1/modules | jq '.data[].module'
```

Output:

```
"booking"
"business-hours"
"queue"
"notifications"
"access"
"branding"
```

### Get module schema

```
GET /v1/modules/:module/schema
```

Returns the full JSON Schema for a module, including `x-ui` hints for the management UI form renderer.

```bash
curl http://localhost:8085/v1/modules/booking/schema | jq .
```

### Read tenant config

```
GET /v1/config/:module
```

Returns the tenant's current config for the module. If the tenant has never saved one, `isDefault: true` is set and the schema defaults are returned.

```bash
curl http://localhost:8085/v1/config/booking \
  -H "X-Tenant-ID: 01HZ..." | jq .
```

Response:

```json
{
  "tenantId": "01HZ...",
  "module": "booking",
  "config": {
    "slotDurationMinutes": 30,
    "maxBookingsPerDay": 100,
    "advanceBookingDays": 365,
    "autoConfirm": false,
    "bufferMinutes": 0
  },
  "schemaVersion": "1",
  "isDefault": true,
  "updatedAt": "2026-04-17T10:00:00Z"
}
```

### Upsert tenant config

```
PUT /v1/config/:module
```

The request body is validated against the module's JSON Schema. Unknown fields are rejected (`additionalProperties: false`).

```bash
curl -X PUT http://localhost:8085/v1/config/booking \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "slotDurationMinutes": 60,
    "maxBookingsPerDay":   50,
    "advanceBookingDays":  14,
    "autoConfirm":         true,
    "bufferMinutes":       15
  }' | jq .
```

Once saved, `isDefault` becomes `false` and the booking-service enforces this config within 60 seconds (cache TTL).

### Get config history

```
GET /v1/config/:module/history
```

Returns the full audit log of changes to the tenant's config for this module.

```bash
curl "http://localhost:8085/v1/config/booking/history?limit=20" \
  -H "X-Tenant-ID: 01HZ..." | jq .
```

Response:

```json
{
  "data": [
    {
      "id": "hist_xyz",
      "tenantId": "01HZ...",
      "module": "booking",
      "config": { "slotDurationMinutes": 60, "autoConfirm": true },
      "changedBy": "system",
      "changedAt": "2026-04-17T11:30:00Z"
    }
  ],
  "total": 1
}
```

---

## Error Reference

All error responses use:

```json
{ "error": "human-readable description" }
```

| HTTP Code | Meaning | Common cause |
|---|---|---|
| `200 OK` | Success | Request completed, body contains data |
| `201 Created` | Resource created | Successful POST |
| `204 No Content` | Success, no body | Successful DELETE |
| `400 Bad Request` | Malformed request | Missing field, bad JSON, missing X-Tenant-ID |
| `404 Not Found` | Resource missing | Wrong ID or wrong tenant |
| `409 Conflict` | State conflict | Daily cap hit, buffer gap, terminal status |
| `422 Unprocessable Entity` | Policy violation | Past slot, outside hours, slot too short |
| `500 Internal Server Error` | Server fault | Check service logs |

### Common booking errors

```
{"error": "slot start must be in the future"}
{"error": "slot is too far in the future: bookings may only be created up to 30 day(s) ahead"}
{"error": "slot duration (20 min) is shorter than the minimum slot duration configured for this tenant (30 min)"}
{"error": "slot starts before opening time (09:00)"}
{"error": "slot ends after closing time (17:00)"}
{"error": "business is closed on saturday"}
{"error": "slot overlaps with the business break (12:00 – 60 min)"}
{"error": "daily booking limit reached (50/50) — no further bookings accepted on this date"}
{"error": "a 15-minute buffer is required between consecutive bookings for this service"}
{"error": "time slot already booked"}
```
