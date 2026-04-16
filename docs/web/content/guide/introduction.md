---
title: Introduction
description: Overview of the ServiceForge platform
order: 1
---

# Introduction to ServiceForge

Welcome to ServiceForge! ServiceForge is an open-source, multi-tenant business capability platform designed to help teams activate highly configurable modules—such as Booking, Pricing, Payments, Queues, and more—without having to build each capability from scratch.

## Core Philosophy

ServiceForge centers around **Configuration over Customization**. Our goal is to reduce your time-to-market. Instead of writing custom logic for standard features, you configure them. The underlying architecture is event-driven to ensure loose coupling and high resilience among modules. 

Furthermore, every layer—from the API Gateway down to the caching mechanisms—is tenant-aware. This guarantees top-tier isolation without sacrificing speed. 

## Architectural Highlights

- **Access Layer**: Includes the core API Gateway and the Management UI built on modern web technologies.
- **Core Services**: Built in high-performance Go, this includes the Tenant Management, Auth, Config, and feature modules like the Booking Service.
- **Integration Layer**: A Kafka event backbone ensures scalable, asynchronous operations and data handoffs.
- **Data Layer**: Backed by PostgreSQL and Redis for steadfast transactional consistency and rapid reads.

Explore the rest of this documentation to discover how you can orchestrate your tenant management, leverage standard APIs out of the box, deploy ServiceForge in your own environments, and extend it manually as a developer!
