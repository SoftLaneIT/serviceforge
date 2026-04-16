---
title: Self-Hosting
description: Deploy ServiceForge on your own infrastructure using Docker Compose
order: 8
---

# Self-Hosting

ServiceForge ships with a production-ready Docker Compose setup. Everything runs in containers — no external managed services required.

---

## Quick Start

```bash
# Clone the repository
git clone https://github.com/SoftLaneIT/serviceforge.git
cd serviceforge

# Start all services (builds images, runs migrations)
docker compose -f deploy/docker/docker-compose.dev.yml up --build -d

# Check everything is healthy
docker compose -f deploy/docker/docker-compose.dev.yml ps
```

All services should show `healthy` or `running` within ~30 seconds.

---

## Service Ports

| Service | Port | Notes |
|---|---|---|
| management-ui | 3000 | Next.js dashboard |
| api-gateway | 8080 | Unified entry point |
| management-service | 8081 | Tenant + API key CRUD |
| auth-service | 8082 | Authentication |
| booking-service | 8084 | Booking API |
| config-service | 8085 | Configuration API |
| PostgreSQL | 5432 | Primary database |
| Redis | 6379 | Session cache |
| Kafka | 9092 | Event backbone |
| Zookeeper | 2181 | Kafka coordination |
| Kafka UI | 8080 | Topic browser (dev only) |

---

## Environment Variables

### booking-service

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8084` | HTTP listen port |
| `DATABASE_URL` | postgres://serviceforge:serviceforge@localhost:5432/serviceforge | PostgreSQL DSN |
| `CONFIG_SERVICE_URL` | `http://localhost:8085` | Config-service base URL |
| `CONFIG_CACHE_TTL` | `60` | Config cache TTL in seconds |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker addresses |
| `KAFKA_ENABLED` | `true` | Set to `false` to disable Kafka publishing |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `LOG_FORMAT` | `json` | `json` \| `text` |

### config-service

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8085` | HTTP listen port |
| `DATABASE_URL` | — | PostgreSQL DSN |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |
| `LOG_LEVEL` | `info` | Log verbosity |

### management-service

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8081` | HTTP listen port |
| `DATABASE_URL` | — | PostgreSQL DSN |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |

### management-ui

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8081` | Management service URL |
| `NEXT_PUBLIC_BOOKING_URL` | `http://localhost:8084` | Booking service URL |
| `NEXT_PUBLIC_CONFIG_URL` | `http://localhost:8085` | Config service URL |

---

## Common Operations

### Rebuild a single service after code changes

```bash
docker compose -f deploy/docker/docker-compose.dev.yml build booking-service
docker compose -f deploy/docker/docker-compose.dev.yml up -d booking-service
```

### View service logs

```bash
# All services (follow)
docker compose -f deploy/docker/docker-compose.dev.yml logs -f

# Single service
docker compose -f deploy/docker/docker-compose.dev.yml logs -f booking-service

# Last 100 lines
docker compose -f deploy/docker/docker-compose.dev.yml logs --tail=100 booking-service
```

### Run database migrations

```bash
docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate
```

### Roll back last migration

```bash
docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate down 1
```

### Stop everything

```bash
docker compose -f deploy/docker/docker-compose.dev.yml down
```

### Stop and remove volumes (full reset)

```bash
docker compose -f deploy/docker/docker-compose.dev.yml down -v
```

---

## Production Checklist

Before exposing ServiceForge to the internet:

1. **Change all default passwords** — update `POSTGRES_PASSWORD` and any service secrets in your `.env` file
2. **Set `CORS_ORIGINS`** to your actual frontend domain(s) only
3. **Set `KAFKA_ENABLED=true`** unless you are intentionally running without event publishing
4. **Use HTTPS** — put a reverse proxy (nginx, Caddy, Traefik) in front of the API Gateway
5. **Enable database backups** — mount a named volume for PostgreSQL data and set up regular backups
6. **Set `LOG_LEVEL=warn`** in production to reduce log volume
7. **Pin image tags** — replace `:latest` tags with specific version tags in your compose file
8. **Restrict port exposure** — in production, only expose the API Gateway port (8080) and management UI (3000) externally; all other ports should be internal

---

## On-Premise Deployment (Advanced)

For stricter environments, the `docs/on-prem/` directory contains a self-contained deployment package:

```bash
cd docs/on-prem/

# Copy and edit environment variables
cp .env.example .env
nano .env

# Start infrastructure (PostgreSQL, Redis, Kafka, Zookeeper)
docker-compose -f docker-compose.cloud-infra.yml up -d

# Wait for healthy, then start microservices
docker-compose up -d

# Check logs
docker-compose logs -f
```

### Maintenance

**Update images:**

```bash
docker-compose pull
docker-compose up -d
```

**PostgreSQL backup:**

```bash
docker exec serviceforge-postgres pg_dump -U serviceforge serviceforge > backup_$(date +%Y%m%d).sql
```

**PostgreSQL restore:**

```bash
docker exec -i serviceforge-postgres psql -U serviceforge serviceforge < backup_20260417.sql
```

---

## Troubleshooting

**Container exits immediately:**

```bash
docker compose logs <service-name>
```

Look for `could not connect to database` — this usually means the database isn't ready yet. The services retry with exponential backoff, but if the container restarts too fast Docker may give up. Run `docker compose up -d` again to restart it.

**`connection refused` on an API call:**

Confirm the container is running and healthy:

```bash
docker compose ps
curl http://localhost:8084/health
```

**Kafka connection errors in booking-service:**

If you disabled Kafka (`KAFKA_ENABLED=false`) but the service still tries to connect, ensure the compose file and environment variable are consistent. You can completely remove the Kafka and Zookeeper services from your compose file for Kafka-free deployments.

**Migration fails:**

Inspect the migrate container logs:

```bash
docker compose logs migrate
```

Common causes: the migration SQL has a syntax error, or a previous migration left the schema in a dirty state. To force-reset the migration version:

```bash
docker exec -it serviceforge-postgres psql -U serviceforge -c "UPDATE schema_migrations SET dirty = false;"
```
