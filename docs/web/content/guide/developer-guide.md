---
title: Developer Guide
description: Building and extending ServiceForge capabilities
order: 4
---

# Developer Guide

ServiceForge is structurally sound but highly extensible. It offers engineers a robust foundation out-of-the-box using purely open-source components: Go (Microservices), Next.js (Management UI), and standard DevOps tooling.

## Local Dependencies

Before jumping into code, ensure your local development machine holds the correct toolchain:
- Go 1.23+
- Node.js 20+
- Docker + Docker Compose

At the root directory, initiate the infrastructure backing components:
```bash
make up
```

## Running Services locally

Standard module construction consists of decoupled services running cooperatively. 
To run the various backend components, open separate terminal windows and run:
```bash
make auth
make tenant
make config
make booking
make gateway
```

For the Front-end Next.js App, navigate to `apps/management-ui`:
```bash
cd apps/management-ui
npm install
npm run dev
```

## Adding a New Module

ServiceForge is primed for progressive extensibility. Adding a new module involves:
1. Creating a new directory under `services/` (e.g. `services/payment-module`).
2. Adhering to the Event Bus standards. Every new service should tap into the existing Kafka message queue, subscribing to cross-module broadcasts.
3. Defining your Open API spec under `packages/contracts/openapi/`.

Your newly built service interfaces directly with the central configuration service, letting the management UI pick up your module implicitly. 

## Architectural Nuance

Remember the Golden Rules while developing:
- **Never mutate cross-tenant contexts**. Utilize the `Tenant Management` SDK available in the Go-Common package.
- Use PostgreSQL safely separated schemas over logical partitions where strictly necessary.
- Rely on Redis exclusively for short-lived session caches—assume complete volatility here.
