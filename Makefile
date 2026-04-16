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

SHELL := /bin/bash

COMPOSE := docker compose -f deploy/docker/docker-compose.dev.yml

#  module lists 
SERVICES := packages/go-common \
            services/api-gateway \
            services/auth-service \
            services/booking-service \
            services/config-service \
            services/tenant-service

.PHONY: up up-infra down restart logs build \
        test test-ci fmt vet tidy \
        gateway auth tenant booking config ui \
        migrate migrate-down

#  docker-compose targets 

## up: start all services (build images if needed)
up:
	$(COMPOSE) up -d --build

## up-infra: start only infrastructure (postgres, redis, kafka) — useful for
##           running services locally with `go run`
up-infra:
	$(COMPOSE) up -d postgres redis zookeeper kafka migrate

## down: stop and remove all containers and named volumes
down:
	$(COMPOSE) down -v

## restart: rebuild and restart a single service, e.g. `make restart svc=auth-service`
restart:
	$(COMPOSE) up -d --build $(svc)

## logs: tail logs from all containers
logs:
	$(COMPOSE) logs -f

## build: build all Docker images without starting containers
build:
	$(COMPOSE) build

#  migration targets ─

## migrate: run all pending DB migrations
migrate:
	$(COMPOSE) run --rm migrate up

## migrate-down: roll back the most recent migration
migrate-down:
	$(COMPOSE) run --rm migrate down 1

#  Go targets 

## test: run tests in every module
test:
	@for mod in $(SERVICES); do \
	  echo "==> $$mod"; \
	  (cd $$mod && go test ./...); \
	done

## test-ci: run tests with race detector and generate coverage
test-ci:
	@for mod in $(SERVICES); do \
	  echo "==> $$mod"; \
	  (cd $$mod && go test -race -coverprofile=coverage.out ./...); \
	done

## fmt: format all Go source files
fmt:
	@for mod in $(SERVICES); do \
	  (cd $$mod && go fmt ./...); \
	done

## vet: vet all Go modules
vet:
	@for mod in $(SERVICES); do \
	  echo "==> $$mod"; \
	  (cd $$mod && go vet ./...); \
	done

## tidy: run go mod tidy in every module then sync the workspace
tidy:
	@for mod in $(SERVICES); do \
	  echo "==> $$mod"; \
	  (cd $$mod && go mod tidy); \
	done
	go work sync

#  local run targets (infra must be up first) 

## gateway: run api-gateway locally
gateway:
	cd services/api-gateway && go run ./cmd/server

## auth: run auth-service locally
auth:
	cd services/auth-service && go run ./cmd/server

## tenant: run tenant-service locally
tenant:
	cd services/tenant-service && go run ./cmd/server

## booking: run booking-service locally
booking:
	cd services/booking-service && go run ./cmd/server

## config: run config-service locally
config:
	cd services/config-service && go run ./cmd/server

## ui: start the management UI dev server (local, hot-reload)
ui:
	cd apps/management-ui && npm run dev

## ui-install: install management UI npm dependencies
ui-install:
	cd apps/management-ui && npm install

## ui-build: build the management UI production bundle
ui-build:
	cd apps/management-ui && npm run build

#  help 

help:
	@grep -E '^##' Makefile | sed 's/## //'
