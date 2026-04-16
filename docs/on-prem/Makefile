# Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
#
# SoftlaneIT licenses this file to you under the Apache License,
# Version 2.0 (the "LICENSE"); you may not use this file except
# in compliance with the LICENSE.
# You may obtain a copy of the LICENSE at
#
# https://softlaneit.com/LICENSE.txt
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the LICENSE is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the LICENSE for the
# specific language governing permissions and limitations
# under the LICENSE.

# ServiceForge Platform — Makefile

.PHONY: help up down dev core cloud reset logs build test lint migrate

# Default target
help: ## Show this help
	@echo "ServiceForge Platform Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

#  Deployment 

up: ## Start full platform (on-prem / Docker)
	@./setup.sh --full

dev: ## Start in dev mode (hot-reload + Kafka UI + pgAdmin)
	@./setup.sh --dev

core: ## Start core services only (gateway, tenant, config, web)
	@./setup.sh --core

cloud: ## Start app containers with external cloud infrastructure
	@./setup.sh --cloud

monitoring: ## Start full platform + Prometheus/Grafana/Loki
	@./setup.sh --monitoring

down: ## Stop all services
	@./setup.sh --down

reset: ## Stop + destroy all data volumes (DESTRUCTIVE)
	@./setup.sh --reset

logs: ## Tail all service logs
	@docker compose logs -f --tail=100

logs-%: ## Tail logs for a specific service (e.g., make logs-gateway)
	@docker compose logs -f --tail=100 $*

ps: ## Show service status
	@docker compose ps

#  Build 

build: ## Build all Docker images
	@docker compose build

build-%: ## Build a specific service image (e.g., make build-gateway)
	@docker compose build $*

#  Database 

migrate: ## Run database migrations
	@docker compose run --rm migrate

db-shell: ## Open PostgreSQL shell
	@docker compose exec postgres psql -U serviceforge -d serviceforge

#  Testing 

test: ## Run all tests
	@echo "Running Go tests..."
	@cd services && go test ./... -v -count=1
	@echo "Running frontend tests..."
	@cd web && npm test

test-%: ## Run tests for a specific service (e.g., make test-booking)
	@cd services/$* && go test ./... -v -count=1

lint: ## Run linters
	@echo "Linting Go..."
	@golangci-lint run ./...
	@echo "Linting TypeScript..."
	@cd web && npm run lint

#  Code Generation 

proto: ## Generate gRPC code from proto files
	@./tools/codegen/proto-gen.sh

openapi: ## Generate OpenAPI specs
	@./tools/codegen/openapi-gen.sh

sdk: ## Generate client SDKs from OpenAPI specs
	@./tools/codegen/sdk-gen.sh

#  Utilities 

seed: ## Seed development data
	@docker compose exec gateway /bin/sh -c "go run /app/tools/seed/seed.go"

clean: ## Remove build artifacts and caches
	@find . -name 'tmp' -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name '*.test' -delete 2>/dev/null || true
	@cd web && rm -rf .next node_modules/.cache 2>/dev/null || true

version: ## Show current version
	@echo "ServiceForge $$(git describe --tags --always 2>/dev/null || echo 'dev')"
