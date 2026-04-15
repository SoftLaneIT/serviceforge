SHELL := /bin/zsh

.PHONY: up down logs gateway auth tenant config booking ui fmt

up:
	docker compose -f deploy/docker/docker-compose.dev.yml up -d

down:
	docker compose -f deploy/docker/docker-compose.dev.yml down -v

logs:
	docker compose -f deploy/docker/docker-compose.dev.yml logs -f

gateway:
	cd services/api-gateway && go run ./cmd

auth:
	cd services/auth-service && go run ./cmd

tenant:
	cd services/tenant-service && go run ./cmd

config:
	cd services/config-service && go run ./cmd

booking:
	cd services/booking-service && go run ./cmd

ui:
	cd apps/management-ui && npm run dev

fmt:
	go fmt ./...
