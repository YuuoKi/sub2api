#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_DISABLE_ENV_FILE=1 DOCKER_BUILDKIT=0 COMPOSE_DOCKER_CLI_BUILD=0 \
  docker compose -f deploy/docker-compose.wsl.yml up -d --build

echo "系统已启动：http://127.0.0.1:8080"
