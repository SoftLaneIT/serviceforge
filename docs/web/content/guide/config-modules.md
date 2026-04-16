---
title: Configuration Modules
description: Full field reference for all six built-in configuration modules
order: 4
---

# Configuration Modules

ServiceForge ships with six configuration modules. Each is a JSON Schema stored once on the config-service. Tenants can override defaults at any time via `PUT /v1/config/:module`.

Changes take effect within **60 seconds** (the booking-service config cache TTL). Until a tenant saves an explicit config, the booking-service uses its own permissive in-process defaults — it never enforces the schema example values against a tenant that hasn't opted in.

---

## booking

Controls the core reservation rules enforced on every `POST /v1/bookings`.

| Field | Type | Default | Description |
|---|---|---|---|
| `slotDurationMinutes` | integer | 30 | Minimum slot length in minutes. Bookings shorter than this are rejected with `422`. |
| `maxBookingsPerDay` | integer | 100 | Hard cap on non-cancelled bookings per calendar day. Exceeding it returns `409`. |
| `advanceBookingDays` | integer | 365 | How far ahead (in days) a slot can be booked. Slots beyond this window return `422`. |
| `autoConfirm` | boolean | false | When `true`, new bookings are immediately created with `status: confirmed` instead of `pending`. |
| `bufferMinutes` | integer | 0 | Minimum gap required before and after each booking for the same `serviceRef`. Buffer violations return `409`. |
| `cancellationPolicy` | object | see below | Nested cancellation rules. |

**`cancellationPolicy` fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `allowCancellation` | boolean | true | Whether cancellation is permitted at all |
| `cutoffHours` | integer | 24 | Hours before slot start within which cancellation is blocked |
| `refundable` | boolean | true | Whether cancellations are eligible for a refund |

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/booking \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "slotDurationMinutes": 60,
    "maxBookingsPerDay":   50,
    "advanceBookingDays":  14,
    "autoConfirm":         true,
    "bufferMinutes":       15,
    "cancellationPolicy": {
      "allowCancellation": true,
      "cutoffHours":       2,
      "refundable":        false
    }
  }'
```

---

## business-hours

Defines when the business accepts bookings. Business-hours enforcement is **only activated** once a tenant explicitly saves this config (`isDefault: false`). Tenants using schema defaults are never blocked by business hours.

| Field | Type | Default | Description |
|---|---|---|---|
| `timezone` | string (IANA) | `"UTC"` | Timezone used for all hour comparisons, e.g. `"America/New_York"`, `"Europe/London"` |
| `allowBookingsOutsideHours` | boolean | false | When `true`, disable all business-hours enforcement |
| `monday` … `sunday` | object | see below | Per-day schedule |
| `breakDurationMinutes` | integer | 0 | Length of midday break in minutes (0 = no break) |
| `breakStartTime` | string HH:MM | `"13:00"` | Break start time in the tenant's timezone |

**Day schedule object** (for each weekday key):

| Field | Type | Description |
|---|---|---|
| `open` | boolean | Whether the business accepts bookings on this day |
| `openTime` | string HH:MM | Opening time in the tenant's timezone |
| `closeTime` | string HH:MM | Closing time in the tenant's timezone |

### How business-hours enforcement works

1. Slot start day is checked: if `open: false` → `422 business is closed on <day>`
2. `slotStart` must be ≥ `openTime` on the start day
3. `slotEnd` must be ≤ `closeTime` on the **end day** (multi-day bookings use the end day's schedule)
4. For same-day bookings, the break window is also checked

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/business-hours \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "timezone": "America/Chicago",
    "allowBookingsOutsideHours": false,
    "monday":    {"open": true,  "openTime": "08:00", "closeTime": "18:00"},
    "tuesday":   {"open": true,  "openTime": "08:00", "closeTime": "18:00"},
    "wednesday": {"open": true,  "openTime": "08:00", "closeTime": "18:00"},
    "thursday":  {"open": true,  "openTime": "08:00", "closeTime": "18:00"},
    "friday":    {"open": true,  "openTime": "08:00", "closeTime": "17:00"},
    "saturday":  {"open": true,  "openTime": "09:00", "closeTime": "13:00"},
    "sunday":    {"open": false, "openTime": "09:00", "closeTime": "12:00"},
    "breakDurationMinutes": 60,
    "breakStartTime": "12:00"
  }'
```

---

## queue

Governs waitlist behaviour when bookings are at capacity.

| Field | Type | Default | Description |
|---|---|---|---|
| `lockingTimeoutSeconds` | integer | 300 | How long a slot hold is reserved before being released |
| `maxSeats` | integer | 50 | Maximum concurrent waitlist entries |
| `queueType` | enum | `"fifo"` | `fifo` \| `priority` \| `fair-share` |
| `overflowBehaviour` | enum | `"reject"` | `reject` \| `queue` \| `redirect` |
| `maxWaitTimeSeconds` | integer | 1800 | Drop from queue after this many seconds of waiting (0 = no limit) |
| `fairnessPolicy` | object | see below | Per-customer fairness controls |

**`fairnessPolicy` fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | false | Enable per-customer booking fairness |
| `maxBookingsPerCustomerPerDay` | integer | 3 | Max bookings one `customerRef` may hold per day |

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/queue \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "lockingTimeoutSeconds": 120,
    "maxSeats":              100,
    "queueType":             "fifo",
    "overflowBehaviour":     "queue",
    "maxWaitTimeSeconds":    3600,
    "fairnessPolicy": {
      "enabled": true,
      "maxBookingsPerCustomerPerDay": 2
    }
  }'
```

---

## notifications

Configures outbound webhooks, email triggers, and SMS triggers for booking lifecycle events.

### Webhook fields

| Field | Type | Default | Description |
|---|---|---|---|
| `webhookEnabled` | boolean | false | Enable outbound webhook delivery |
| `webhookUrl` | string (URL) | — | HTTPS endpoint to receive POST requests |
| `webhookEvents` | array | — | Events to deliver (see list below) |

**Available webhook events:**

| Event | Fires when |
|---|---|
| `booking.created` | A new booking is submitted |
| `booking.confirmed` | A booking moves to `confirmed` |
| `booking.cancelled` | A booking is cancelled |
| `booking.completed` | A booking is marked completed |
| `booking.no_show` | A booking is marked no-show |

### Email fields

| Field | Type | Default | Description |
|---|---|---|---|
| `emailEnabled` | boolean | false | Enable email notifications |
| `emailTriggers` | object | — | `{onCreate, onConfirm, onCancel, onReminder}` (booleans) |

### SMS fields

| Field | Type | Default | Description |
|---|---|---|---|
| `smsEnabled` | boolean | false | Enable SMS notifications |
| `smsTriggers` | object | — | `{onCreate, onConfirm, onCancel}` (booleans) |

### Retry policy

| Field | Type | Default | Description |
|---|---|---|---|
| `retryPolicy.maxAttempts` | integer | 3 | Retry count on non-2xx webhook response |
| `retryPolicy.backoffSeconds` | integer | 30 | Wait between retries in seconds |

### Webhook payload shape

Every webhook POST carries:

```json
{
  "event":     "booking.confirmed",
  "tenantId":  "01HZ...",
  "timestamp": "2026-04-17T10:05:00Z",
  "data": {
    "id":          "bk_abc123",
    "customerRef": "customer_42",
    "serviceRef":  "haircut",
    "slotStart":   "2026-04-20T09:00:00Z",
    "slotEnd":     "2026-04-20T10:00:00Z",
    "status":      "confirmed",
    "metadata":    {}
  }
}
```

Your endpoint must return HTTP `2xx` within 5 seconds or the delivery is retried.

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/notifications \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "webhookEnabled": true,
    "webhookUrl":     "https://my-app.example.com/hooks/serviceforge",
    "webhookEvents":  ["booking.created","booking.confirmed","booking.cancelled"],
    "emailEnabled":   true,
    "emailTriggers":  {"onCreate": true, "onConfirm": true, "onCancel": true},
    "retryPolicy": {
      "maxAttempts":    3,
      "backoffSeconds": 30
    }
  }'
```

---

## access

Controls security policy for the tenant's dashboard sessions.

| Field | Type | Default | Description |
|---|---|---|---|
| `sessionTimeoutMinutes` | integer (slider) | 60 | Dashboard idle session timeout |
| `maxConcurrentSessions` | integer | 5 | Per-user session cap |
| `enforceIpAllowlist` | boolean | false | Block requests not in `ipAllowlist` |
| `ipAllowlist` | array of strings | — | CIDR blocks or exact IPs, e.g. `["10.0.0.0/8","203.0.113.5"]` |
| `requireMfa` | boolean | false | Require TOTP / MFA on login |
| `rateLimiting` | object | see below | Request rate controls |
| `loginFailurePolicy` | object | see below | Account lockout on failed logins |

**`rateLimiting` fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | false | Enable rate limiting |
| `requestsPerMinute` | integer | 60 | Allowed requests per minute per client |
| `burstSize` | integer | 10 | Burst headroom above the steady rate |

**`loginFailurePolicy` fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `maxAttempts` | integer | 5 | Failed logins before lockout |
| `lockoutMinutes` | integer | 30 | How long the lockout lasts |

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/access \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "sessionTimeoutMinutes": 30,
    "maxConcurrentSessions": 3,
    "enforceIpAllowlist":    true,
    "ipAllowlist":           ["10.0.0.0/8"],
    "requireMfa":            true,
    "rateLimiting": {
      "enabled":            true,
      "requestsPerMinute":  120,
      "burstSize":          20
    },
    "loginFailurePolicy": {
      "maxAttempts":    3,
      "lockoutMinutes": 60
    }
  }'
```

---

## branding

White-labelling settings used by emails, SMS, and the customer-facing booking widget.

| Field | Type | Default | Description |
|---|---|---|---|
| `companyName` | string | — | Displayed in email headers and the widget title |
| `colors` | object | — | `{primary, accent, background}` — hex colour values |
| `logoUrl` | string (URL) | — | HTTPS URL to a PNG or SVG logo |
| `customDomain` | string (URL) | — | HTTPS domain used for widget embedding |
| `locale` | enum | `"en"` | `en` \| `es` \| `fr` \| `de` \| `ja` \| `zh` |
| `dateFormat` | string | `"YYYY-MM-DD"` | `DD/MM/YYYY` \| `MM/DD/YYYY` \| `YYYY-MM-DD` |
| `timeFormat` | string | `"24h"` | `12h` \| `24h` |
| `hidePoweredBy` | boolean | false | Remove "Powered by ServiceForge" badge |

### Example

```bash
curl -X PUT http://localhost:8085/v1/config/branding \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "companyName": "Acme Corp",
    "colors": {
      "primary":    "#2563eb",
      "accent":     "#16a34a",
      "background": "#f8fafc"
    },
    "logoUrl":       "https://acme.com/logo.png",
    "customDomain":  "https://book.acme.com",
    "locale":        "en",
    "dateFormat":    "MM/DD/YYYY",
    "timeFormat":    "12h",
    "hidePoweredBy": true
  }'
```
