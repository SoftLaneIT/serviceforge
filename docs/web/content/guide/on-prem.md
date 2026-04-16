---
title: On-Premise Guide
description: Deploying ServiceForge within your own infrastructure
order: 3
---

# On-Premise Deployments

ServiceForge allows organizations with strict compliance or security demands to operate the entire platform entirely within their own infrastructure boundaries. 

The primary deployment mechanism for on-prem installations is orchestrated via Docker Compose, scaling up to Helm charts inside Kubernetes.

## Prerequisites

- Compute resource with `docker` and `docker-compose` installed.
- Access to standard outbound network ports during initial pulling.
- Sufficient disk space for PostgreSQL persistent volumes.

## Getting Started

Inside your platform repository, navigate to the `docs/on-prem/` folder. This directory contains all the pre-configured definitions you need.

```bash
cd docs/on-prem/
```

### Initial Configuration

Before spinning up the stateful services, configure your environment variables. 
Copy the example environment securely.

```bash
cp .env.example .env
```

Review `.env` to enforce secure secrets, change default PostgreSQL passwords, and configure your exposed gateway endpoints.

### Starting the Platform

ServiceForge is structured into dependencies (databases, brokers) and volatile microservices. 
The bundled `setup.sh` orchestrates the necessary initializations.

1. **Start Core Infrastructure**:
   ```bash
   docker-compose -f docker-compose.cloud-infra.yml up -d
   ```
   This spins up PostgreSQL, Redis, Kafka, and Zookeeper.
2. **Execute Database Migrations**: 
   Ensure `init-db.sql` is run to schema boundaries.
3. **Start Microservices**:
   ```bash
   docker-compose up -d
   ```

To monitor the logs of your running platform, utilize standard docker mechanics:
```bash
docker-compose logs -f
```

## Maintenance & Upgrades

- **Updating Images**: When a new version of ServiceForge is released, `docker-compose pull` fetches the updated components.
- **Backups**: Standard backup routines must be aimed towards the mounted PostgreSQL volumes. Redis handles ephemeral volatile states which can be purged safely on restarts.
