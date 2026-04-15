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

SHELL := /bin/zsh

.PHONY: up down logs gateway auth tenant config booking ui fmt

up:
	docker compose -f deploy/docker/docker-compose.dev.yml up -d

down:
	docker compose -f deploy/docker/docker-compose.dev.yml down -v

logs:
	docker compose -f deploy/docker/docker-compose.dev.yml logs -f

gateway:
	cd services/api-gateway && go run ./cmd/server

auth:
	cd services/auth-service && go run ./cmd/server

tenant:
	cd services/tenant-service && go run ./cmd/server

config:
	cd services/config-service && go run ./cmd/server

booking:
	cd services/booking-service && go run ./cmd/server

ui:
	cd apps/management-ui && npm run dev

fmt:
	go fmt ./...
