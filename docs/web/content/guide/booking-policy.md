---
title: Booking Policy
description: How the booking-service enforces tenant configuration on every POST /v1/bookings
order: 5
---

# Booking Policy Enforcement

Every `POST /v1/bookings` passes through a sequential policy pipeline before the booking is stored. The pipeline reads the tenant's `booking` and `business-hours` module configs from the config-service (cached for 60 seconds) and applies seven checks in order.

**If any check fails, the pipeline stops immediately and returns the error.** Subsequent checks are not evaluated.

---

## The Pipeline

### 1. Advance Booking Limit

**Config key:** `booking.advanceBookingDays` (default: 365)

`slotStart` must not be further into the future than `now + advanceBookingDays`.

```
slotStart > now + 14 days  →  422 "slot is too far in the future: bookings may only be created up to 14 day(s) ahead"
```

**Use case:** Prevent customers from reserving slots years in advance. Set to `14` for a two-week booking window.

---

### 2. Past-Slot Check

No config — always enforced.

`slotStart` must be in the future relative to the server's current UTC time.

```
slotStart ≤ now  →  422 "slot start must be in the future"
```

---

### 3. Minimum Duration

**Config key:** `booking.slotDurationMinutes` (default: 30)

The slot length (`slotEnd − slotStart` in minutes) must be ≥ `slotDurationMinutes`.

```
(slotEnd - slotStart) < 30 min  →  422 "slot duration (20 min) is shorter than the minimum slot duration configured for this tenant (30 min)"
```

**Use case:** Prevent zero-length or unrealistically short bookings.

---

### 4. Business Hours

**Config keys:** `business-hours` module (all fields)

This check only runs when:
- The tenant has **explicitly saved** a `business-hours` config (`isDefault: false`), **and**
- `allowBookingsOutsideHours: false`

If either condition is not met, business-hours enforcement is skipped entirely.

When active, the check validates:

1. The start day's `open` flag — if `false`: `"business is closed on <day>"`
2. `slotStart ≥ openTime` on the start day — if violated: `"slot starts before opening time (HH:MM)"`
3. `slotEnd ≤ closeTime` on the **end day** — if violated: `"slot ends after closing time (HH:MM)"`
4. For same-day bookings, the slot must not overlap the configured break window

For multi-day bookings, the closing time check uses the *end day's* schedule (not the start day's), so a booking that genuinely spans overnight is evaluated correctly.

```
slotStart on Saturday, open: false  →  422 "business is closed on saturday"
slotStart before 09:00              →  422 "slot starts before opening time (09:00)"
slotEnd after 17:00                 →  422 "slot ends after closing time (17:00)"
overlaps 12:00–13:00 break          →  422 "slot overlaps with the business break (12:00 – 60 min)"
```

---

### 5. Daily Capacity

**Config key:** `booking.maxBookingsPerDay` (default: 100)

The count of non-cancelled, non-no-show bookings on the same UTC calendar day as `slotStart` must be less than `maxBookingsPerDay`.

```
count ≥ 50  →  409 "daily booking limit reached (50/50) — no further bookings accepted on this date"
```

**Use case:** Hard-cap how many bookings a tenant's operation can handle per day.

---

### 6. Buffer Gap

**Config key:** `booking.bufferMinutes` (default: 0, enforcement skipped when 0)

No existing booking for the same `serviceRef` may fall within ±`bufferMinutes` of the new slot.

The SQL check:

```sql
slot_end   > NEW.slotStart - (bufferMinutes * interval '1 minute')
AND
slot_start < NEW.slotEnd   + (bufferMinutes * interval '1 minute')
```

```
existing booking bk_xyz overlaps the 15-min buffer  →  409 "a 15-minute buffer is required between consecutive bookings for this service"
```

**Use case:** Ensure cleanup or travel time between appointments for the same service.

---

### 7. Auto-Confirm

**Config key:** `booking.autoConfirm` (default: false)

Not a gate — this step sets the initial status.

- `autoConfirm: true` → booking is stored as `status: confirmed`
- `autoConfirm: false` → booking is stored as `status: pending` (operator must confirm manually)

---

## Graceful Degradation

If the config-service is **unreachable** when a booking is created, the booking-service falls back to its in-process defaults:

| Setting | Fallback value |
|---|---|
| `slotDurationMinutes` | 30 |
| `maxBookingsPerDay` | 1000 |
| `advanceBookingDays` | 365 |
| `autoConfirm` | false |
| `bufferMinutes` | 0 |
| Business hours | Not enforced |

A config-service outage will never prevent bookings from being created. The cache also means that even if the config-service goes down, the last known config remains in effect for up to 60 seconds.

---

## Config Propagation Timing

The booking-service caches each tenant's module config for **60 seconds** (configurable via `CONFIG_CACHE_TTL` environment variable). After you `PUT /v1/config/booking`, the new rules take effect within one cache TTL cycle.

To change the TTL:

```yaml
# deploy/docker/docker-compose.dev.yml
booking-service:
  environment:
    CONFIG_CACHE_TTL: "10"   # seconds
```

---

## Disabling Enforcement for Testing

The fastest way to disable all policy checks during development is to set permissive values:

```bash
# Turn off all restrictions
curl -X PUT http://localhost:8085/v1/config/booking \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{
    "slotDurationMinutes": 1,
    "maxBookingsPerDay":   99999,
    "advanceBookingDays":  3650,
    "autoConfirm":         true,
    "bufferMinutes":       0
  }'

# Disable business hours
curl -X PUT http://localhost:8085/v1/config/business-hours \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 01HZ..." \
  -d '{"allowBookingsOutsideHours": true}'
```
