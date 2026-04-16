#!/usr/bin/env bash

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

# ServiceForge Platform — One-Command Setup
# Usage:
#   ./setup.sh                    # Full platform (on-prem)
#   ./setup.sh --dev              # Development mode with hot-reload + dev tools
#   ./setup.sh --core             # Core services only (gateway, tenant, config, web)
#   ./setup.sh --cloud            # App containers only, external cloud infra
#   ./setup.sh --monitoring       # Full platform + Prometheus/Grafana/Loki
#   ./setup.sh --down             # Stop everything
#   ./setup.sh --reset            # Stop + remove all data volumes

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

log()  { echo -e "${GREEN}[ServiceForge]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
err()  { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

#  Pre-flight checks 
check_deps() {
    log "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        err "Docker is not installed. Install from https://docs.docker.com/get-docker/"
    fi

    if ! docker compose version &> /dev/null; then
        err "Docker Compose V2 is not installed. Update Docker Desktop or install the compose plugin."
    fi

    DOCKER_MEM=$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo "0")
    DOCKER_MEM_GB=$((DOCKER_MEM / 1073741824))
    if [ "$DOCKER_MEM_GB" -lt 6 ]; then
        warn "Docker has ${DOCKER_MEM_GB}GB RAM allocated. Recommended: 8GB+ for full platform."
        warn "Increase in Docker Desktop > Settings > Resources"
    fi

    log "Docker $(docker --version | grep -oP '\d+\.\d+\.\d+') with Compose $(docker compose version --short)"
}

#  Environment setup 
setup_env() {
    if [ ! -f .env ]; then
        log "Creating .env from template..."
        cp .env.example .env

        # Generate a random password for Postgres
        RANDOM_PW=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/CHANGE_ME_STRONG_PASSWORD_HERE/${RANDOM_PW}/" .env
        else
            sed -i "s/CHANGE_ME_STRONG_PASSWORD_HERE/${RANDOM_PW}/" .env
        fi

        log "Generated random Postgres password"
    else
        log ".env already exists, skipping"
    fi
}

#  Print access info 
print_info() {
    local mode=$1
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  ServiceForge Platform is running! (${mode} mode)${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${BLUE}Management UI:${NC}     http://localhost"
    echo -e "  ${BLUE}API Gateway:${NC}       http://localhost/api"
    echo -e "  ${BLUE}API Documentation:${NC} http://localhost/api/docs"
    echo -e "  ${BLUE}Traefik Dashboard:${NC} http://localhost:8080"

    if [[ "$mode" == "dev" ]]; then
        echo ""
        echo -e "  ${YELLOW}Dev Tools:${NC}"
        echo -e "  ${BLUE}Kafka UI:${NC}          http://localhost:8090"
        echo -e "  ${BLUE}pgAdmin:${NC}           http://localhost:5050"
        echo -e "  ${BLUE}Redis Commander:${NC}   http://localhost:8081"
    fi

    if [[ "$mode" == *"monitoring"* ]]; then
        echo ""
        echo -e "  ${YELLOW}Monitoring:${NC}"
        echo -e "  ${BLUE}Grafana:${NC}           http://localhost:3001 (admin/admin)"
        echo -e "  ${BLUE}Prometheus:${NC}        http://localhost:9090"
    fi

    echo ""
    echo -e "  ${YELLOW}Useful commands:${NC}"
    echo -e "  docker compose logs -f gateway     # Follow gateway logs"
    echo -e "  docker compose ps                  # Service status"
    echo -e "  docker compose exec postgres psql -U serviceforge  # DB shell"
    echo -e "  ./setup.sh --down                  # Stop all services"
    echo ""
}

#  Wait for health 
wait_healthy() {
    log "Waiting for services to become healthy..."
    local timeout=120
    local elapsed=0

    while [ $elapsed -lt $timeout ]; do
        local unhealthy
        unhealthy=$(docker compose ps --format json 2>/dev/null | \
            grep -c '"Health":"starting"' || true)

        if [ "$unhealthy" -eq 0 ]; then
            log "All services are healthy!"
            return 0
        fi

        printf "."
        sleep 2
        elapsed=$((elapsed + 2))
    done

    warn "Some services may still be starting. Check: docker compose ps"
}

#  Main 
main() {
    local mode="${1:-full}"

    echo -e "${CYAN}"
    echo "  ___                 _          ___                    "
    echo " / __| ___ _ ___ _(_)__ ___| __\___  _ _ __ _ ___ "
    echo " \__ \/ -_) '_\ V / / _/ -_)  _/ _ \| '_/ _\` / -_)"
    echo " |___/\___|_|  \_/|_\__\___|_| \___/|_| \__, \___|"
    echo "                                         |___/         "
    echo -e "${NC}"

    case "$mode" in
        --down|-d)
            log "Stopping all services..."
            docker compose --profile monitoring down
            docker compose down
            log "All services stopped."
            exit 0
            ;;
        --reset)
            warn "This will DESTROY all data volumes. Are you sure? (y/N)"
            read -r confirm
            if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
                log "Cancelled."
                exit 0
            fi
            docker compose --profile monitoring down -v
            docker compose down -v
            log "All services stopped and data removed."
            exit 0
            ;;
        --dev)
            check_deps
            setup_env
            log "Starting in DEVELOPMENT mode (hot-reload + dev tools)..."
            docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
            wait_healthy
            print_info "dev"
            ;;
        --core)
            check_deps
            setup_env
            log "Starting CORE services only..."
            docker compose up -d postgres redis kafka traefik gateway tenant config-engine web migrate
            wait_healthy
            print_info "core"
            ;;
        --cloud)
            check_deps
            if [ ! -f .env ]; then
                err ".env file required with EXTERNAL_* variables set. See .env.example"
            fi
            log "Starting in HYBRID mode (local app + cloud infrastructure)..."
            docker compose -f docker-compose.yml -f docker-compose.cloud-infra.yml up -d --build
            wait_healthy
            print_info "cloud-hybrid"
            ;;
        --monitoring)
            check_deps
            setup_env
            log "Starting FULL platform with monitoring..."
            docker compose --profile monitoring up -d --build
            wait_healthy
            print_info "full+monitoring"
            ;;
        --full|*)
            check_deps
            setup_env
            log "Starting FULL platform..."
            docker compose up -d --build
            wait_healthy
            print_info "production"
            ;;
    esac
}

main "$@"
