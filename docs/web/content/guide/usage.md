---
title: How to Handle & Use
description: Effectively manage ServiceForge configuration
order: 2
---

# Handling and Using ServiceForge

ServiceForge is structured to provide an immediate operational dashboard combined with highly resilient background modules. Using the platform boils down to managing configurations, tenants, and activating respective modules.

## Using the Management UI

The simplest way to interface with ServiceForge's vast engine is via the robust Management UI. 

1. Ensure the platform is running either via local Docker or On-Premise deployments.
2. The Management UI is typically exposed locally on `http://localhost:3000`.
3. Default authentication handles basic secure log-ins, after which you have access to a visual tenant management interface.

## Tenant Setup

ServiceForge revolves heavily around the concept of multi-tenancy. For each distinct business unit, client, or segment you wish to serve:

1. Navigate to the **Tenant Module** in the UI.
2. Initialize a new Tenant by assigning a unique ID and providing contextual metadata.
3. Automatically, ServiceForge instructs the underlying Kafka backbone to register tenant-specific databases/schemas and isolation boundaries dynamically.

## Module Activation

Features in ServiceForge are standalone modules acting on independent life-cycles. 
- **Config Service**: Central directory resolving configurations for modules. You map tenant IDs to specific runtime features (e.g., turning on maximum concurrent bookings).
- **Booking Service**: By linking the Auth and Config service states, the Booking module immediately gains the ability to expose CRUD endpoints.

Whenever a new module is required by your product team, you don't rebuild. You provision it, hook its contract via Open API parameters, and activate it over your existing Tenant schema.
