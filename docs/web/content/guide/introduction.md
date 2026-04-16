---
title: Introduction
description: Overview of the ServiceForge platform — architecture, core concepts, and service map
order: 1
---

# Introduction to ServiceForge

**ServiceForge** is an open-source, multi-tenant business-capability platform. It gives every tenant their own isolated booking engine, runtime configuration, and API keys — all managed through a single control plane, without writing any custom logic per tenant.

## Core Philosophy

ServiceForge is built on **Configuration over Customization**. Standard capabilities — booking schedules, business hours, queue behaviour, notifications, access controls, branding — are not hard-coded; they are JSON documents validated against a published schema and stored per-tenant. Change behaviour at runtime without a redeploy.

The underlying architecture is **event-driven**: every state change publishes a Kafka event so downstream integrations stay decoupled from the core services.

Every layer — API Gateway, services, database — is **tenant-aware by default**, enforcing row-level isolation so one tenant can never touch another's data.

---

## Service Map

| Service | Default Port | Language | Responsibility |
|---|---|---|---|
| `management-ui` | 3000 | Next.js 14 | Control-plane dashboard for operators |
| `management-service` | 8081 | Go | Tenant and API-key CRUD |
| `booking-service` | 8084 | Go | Booking lifecycle + policy enforcement |
| `config-service` | 8085 | Go | JSON-Schema-driven runtime configuration |
| `auth-service` | 8082 | Go | Authentication and session management |
| `api-gateway` | 8080 | Go | Unified entry point with routing + CORS |

**Infrastructure:**

| Component | Port | Purpose |
|---|---|---|
| PostgreSQL | 5432 | Primary relational store (RLS per tenant) |
| Redis | 6379 | Session cache, rate-limit counters |
| Kafka + Zookeeper | 9092 / 2181 | Async event backbone |
| Kafka UI | 8080 | Browser UI to inspect topics and messages |

---

## Key Concepts

### Tenant

A **tenant** is the fundamental isolation unit. Every resource — bookings, API keys, configuration — is scoped to a tenant UUID. Tenants are created once and referenced via the `X-Tenant-ID` HTTP header on every subsequent call.

### API Key

A **per-tenant bearer token** created through the management UI or the management-service API. Pass it as `Authorization: Bearer <key>` for protected management operations.

### Booking

A **time-slot reservation** linking a `customerRef` and a `serviceRef` to a calendar window (`slotStart` → `slotEnd`). Bookings move through a status lifecycle: `pending → confirmed → completed` (or `cancelled` / `no_show`).

### Module Configuration

A **JSON document** stored per-tenant, validated against a JSON Schema. There are six built-in modules:

| Module | Controls |
|---|---|
| `booking` | Slot duration, daily cap, advance limit, auto-confirm, buffer gap |
| `business-hours` | Day schedules, timezone, break windows |
| `queue` | Waitlist type, overflow behaviour, seat limits |
| `notifications` | Webhooks, email triggers, SMS triggers, retry policy |
| `access` | Session timeout, MFA, IP allowlist, rate limits |
| `branding` | Company name, colours, logo, locale, date/time format |

### X-Tenant-ID Header

The **primary authentication signal** for the booking-service and config-service. All calls to these services must include it. Without it the service returns `400 Bad Request`.

---

## Architecture Diagram

```
Browser / Mobile App
        │
        ▼
  ┌─────────────┐
  │ API Gateway │  :8080
  └──────┬──────┘
         │  routes by path prefix
    ┌────┴──────────────────────┐
    │                           │
    ▼                           ▼
┌──────────────┐     ┌──────────────────┐
│  management  │     │    booking       │
│   service    │     │    service       │
│   :8081      │     │    :8084         │
└──────┬───────┘     └───────┬──────────┘
       │                     │  reads config
       │                     ▼
       │             ┌──────────────────┐
       │             │  config service  │
       │             │   :8085          │
       │             └──────────────────┘
       │
       ▼
┌─────────────┐     ┌──────────┐
│ PostgreSQL  │     │  Redis   │
│   :5432     │     │  :6379   │
└─────────────┘     └──────────┘
       │
       ▼
┌─────────────┐
│    Kafka    │  :9092
└─────────────┘
```

---

## Explore the Docs

Use the sidebar to navigate to:

- **Quick Start** — stand up the full platform in under five minutes and create your first booking
- **API Reference** — complete endpoint documentation with request/response examples
- **Configuration Modules** — detailed field reference for all six modules
- **Booking Policy** — how the backend enforces tenant configuration on every booking
- **TypeScript Integration** — code samples for integrating from a Node.js or browser application
- **Developer Guide** — local development setup, adding new modules, architectural rules
- **Self-Hosting** — Docker Compose and on-premise deployment instructions
