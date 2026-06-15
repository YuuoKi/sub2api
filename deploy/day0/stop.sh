#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_DISABLE_ENV_FILE=1 docker compose -f deploy/docker-compose.wsl.yml stop

echo "系统已停止"
