---
title: TypeScript Integration
description: Code samples for integrating ServiceForge from Node.js or browser applications
order: 6
---

# TypeScript Integration

No npm package is published yet. Copy the minimal fetch wrapper below and you are production-ready in minutes.

---

## Minimal Client Setup

```typescript
// lib/serviceforge.ts

const TENANT_ID   = process.env.SERVICEFORGE_TENANT_ID!;
const BOOKING_URL = process.env.SERVICEFORGE_BOOKING_URL ?? "http://localhost:8084";
const CONFIG_URL  = process.env.SERVICEFORGE_CONFIG_URL  ?? "http://localhost:8085";
const MGMT_URL    = process.env.SERVICEFORGE_MGMT_URL    ?? "http://localhost:8081";

async function sfRequest<T>(
  baseUrl: string,
  path: string,
  opts: RequestInit & { tenantId?: string } = {},
): Promise<T> {
  const { tenantId = TENANT_ID, ...rest } = opts;
  const res = await fetch(`${baseUrl}${path}`, {
    ...rest,
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": tenantId,
      ...(rest.headers as Record<string, string>),
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as Record<string, unknown>;
    const msg = typeof body?.error === "string" ? body.error : `${res.status} ${res.statusText}`;
    throw new Error(`ServiceForge: ${msg}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as T;
}
```

---

## Types

```typescript
// types/serviceforge.ts

export type BookingStatus =
  | "pending"
  | "confirmed"
  | "completed"
  | "cancelled"
  | "no_show";

export interface Booking {
  id: string;
  tenantId: string;
  customerRef: string;
  serviceRef: string;
  slotStart: string;   // RFC 3339
  slotEnd: string;     // RFC 3339
  status: BookingStatus;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: string;
  status: string;
  createdAt: string;
}

export interface BookingConfig {
  slotDurationMinutes: number;
  maxBookingsPerDay: number;
  advanceBookingDays: number;
  autoConfirm: boolean;
  bufferMinutes: number;
  cancellationPolicy?: {
    allowCancellation: boolean;
    cutoffHours: number;
    refundable: boolean;
  };
}
```

---

## Creating a Booking

Always convert `Date` objects to ISO strings. `new Date().toISOString()` produces the RFC 3339 format the backend requires.

```typescript
async function createBooking(
  customerRef: string,
  serviceRef: string,
  start: Date,
  end: Date,
  metadata?: Record<string, unknown>,
): Promise<Booking> {
  return sfRequest<Booking>(BOOKING_URL, "/v1/bookings", {
    method: "POST",
    body: JSON.stringify({
      customerRef,
      serviceRef,
      slotStart: start.toISOString(),
      slotEnd:   end.toISOString(),
      metadata,
    }),
  });
}

// Usage
const booking = await createBooking(
  "customer_42",
  "haircut",
  new Date("2026-04-20T09:00:00Z"),
  new Date("2026-04-20T10:00:00Z"),
  { notes: "first visit" },
);

console.log(booking.id, booking.status);
// bk_abc123  pending
```

---

## Listing Bookings with Pagination

```typescript
async function listBookings(
  status?: BookingStatus,
  limit = 20,
  offset = 0,
): Promise<{ data: Booking[]; total: number }> {
  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  qs.set("limit",  String(limit));
  qs.set("offset", String(offset));

  return sfRequest(BOOKING_URL, `/v1/bookings?${qs}`);
}

// Fetch all confirmed bookings, paginated
let offset = 0;
const pageSize = 20;
let allBookings: Booking[] = [];

while (true) {
  const page = await listBookings("confirmed", pageSize, offset);
  allBookings = allBookings.concat(page.data);
  offset += pageSize;
  if (offset >= page.total) break;
}

console.log(`Fetched ${allBookings.length} confirmed bookings`);
```

---

## Updating Booking Status

```typescript
async function confirmBooking(bookingId: string): Promise<Booking> {
  return sfRequest<Booking>(BOOKING_URL, `/v1/bookings/${bookingId}`, {
    method: "PATCH",
    body: JSON.stringify({ status: "confirmed" }),
  });
}

async function completeBooking(bookingId: string): Promise<Booking> {
  return sfRequest<Booking>(BOOKING_URL, `/v1/bookings/${bookingId}`, {
    method: "PATCH",
    body: JSON.stringify({ status: "completed" }),
  });
}

async function cancelBooking(bookingId: string): Promise<Booking> {
  return sfRequest<Booking>(BOOKING_URL, `/v1/bookings/${bookingId}`, {
    method: "DELETE",
  });
}
```

---

## Reading and Updating Config

```typescript
async function getBookingConfig(): Promise<BookingConfig> {
  const res = await sfRequest<{ config: BookingConfig }>(CONFIG_URL, "/v1/config/booking");
  return res.config;
}

async function updateBookingConfig(patch: Partial<BookingConfig>): Promise<void> {
  const current = await getBookingConfig();
  await sfRequest(CONFIG_URL, "/v1/config/booking", {
    method: "PUT",
    body: JSON.stringify({ ...current, ...patch }),
  });
}

// Enable auto-confirm and set 15-minute buffer
await updateBookingConfig({ autoConfirm: true, bufferMinutes: 15 });
```

---

## Disabling Business Hours for a Tenant

```typescript
async function disableBusinessHours(): Promise<void> {
  await sfRequest(CONFIG_URL, "/v1/config/business-hours", {
    method: "PUT",
    body: JSON.stringify({ allowBookingsOutsideHours: true }),
  });
}

// Configure 9-to-5 Monday–Friday in New York
async function setBusinessHours(): Promise<void> {
  const workday = { open: true, openTime: "09:00", closeTime: "17:00" };
  const closed   = { open: false, openTime: "09:00", closeTime: "17:00" };

  await sfRequest(CONFIG_URL, "/v1/config/business-hours", {
    method: "PUT",
    body: JSON.stringify({
      timezone: "America/New_York",
      allowBookingsOutsideHours: false,
      monday:    workday,
      tuesday:   workday,
      wednesday: workday,
      thursday:  workday,
      friday:    workday,
      saturday:  closed,
      sunday:    closed,
      breakDurationMinutes: 60,
      breakStartTime: "12:00",
    }),
  });
}
```

---

## Tenant Management

```typescript
async function createTenant(
  name: string,
  slug: string,
  plan: "free" | "starter" | "pro" | "enterprise",
): Promise<Tenant> {
  return sfRequest<Tenant>(MGMT_URL, "/v1/tenants", {
    method: "POST",
    body: JSON.stringify({ name, slug, plan }),
    tenantId: "",          // management endpoints don't scope by tenant
  });
}

async function listTenants(limit = 20): Promise<{ data: Tenant[]; total: number }> {
  return sfRequest(MGMT_URL, `/v1/tenants?limit=${limit}`, { tenantId: "" });
}
```

---

## Error Handling

```typescript
async function safeCreateBooking(
  customerRef: string,
  serviceRef: string,
  start: Date,
  end: Date,
): Promise<{ booking?: Booking; error?: string }> {
  try {
    const booking = await createBooking(customerRef, serviceRef, start, end);
    return { booking };
  } catch (err) {
    const msg = err instanceof Error ? err.message : "unknown error";

    // Strip the "ServiceForge: " prefix if present
    const clean = msg.replace(/^ServiceForge:\s*/, "");

    if (clean.includes("in the past"))          return { error: "Please choose a future time slot." };
    if (clean.includes("too far in the future")) return { error: "That date is outside our booking window." };
    if (clean.includes("daily booking limit"))   return { error: "No more slots available on that day." };
    if (clean.includes("buffer"))                return { error: "There is not enough gap before the next appointment." };
    if (clean.includes("closed"))                return { error: "We are closed on that day." };
    if (clean.includes("outside business hours"))return { error: "That time is outside our opening hours." };

    return { error: clean };
  }
}
```

---

## React Hook Example

```typescript
import { useState, useCallback } from "react";

function useCreateBooking() {
  const [loading, setLoading] = useState(false);
  const [error,   setError]   = useState<string | null>(null);

  const create = useCallback(async (
    customerRef: string,
    serviceRef: string,
    start: Date,
    end: Date,
    metadata?: Record<string, unknown>,
  ) => {
    setLoading(true);
    setError(null);
    try {
      const booking = await createBooking(customerRef, serviceRef, start, end, metadata);
      return booking;
    } catch (err) {
      const msg = err instanceof Error ? err.message.replace(/^ServiceForge:\s*/, "") : "Booking failed";
      setError(msg);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return { create, loading, error };
}

// Usage in component
function BookingForm() {
  const { create, loading, error } = useCreateBooking();

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const form = e.currentTarget;
    const start = new Date((form.elements.namedItem("slotStart") as HTMLInputElement).value);
    const end   = new Date((form.elements.namedItem("slotEnd")   as HTMLInputElement).value);
    const booking = await create("customer_42", "haircut", start, end);
    if (booking) console.log("Created:", booking.id);
  };

  return (
    <form onSubmit={handleSubmit}>
      <input type="datetime-local" name="slotStart" required />
      <input type="datetime-local" name="slotEnd"   required />
      <button type="submit" disabled={loading}>
        {loading ? "Booking…" : "Book Now"}
      </button>
      {error && <p style={{ color: "red" }}>{error}</p>}
    </form>
  );
}
```
