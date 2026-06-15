#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "== 本机网页 =="
curl -fsS http://127.0.0.1:8080/health
echo

echo "== 初始化状态 =="
curl -fsS http://127.0.0.1:8080/setup/status
echo

echo "== 监听端口 =="
if command -v ss >/dev/null 2>&1; then
  ss -ltnp | awk '$4 ~ /:8080$/ {print}'
elif command -v netstat >/dev/null 2>&1; then
  netstat -ltnp 2>/dev/null | awk '$4 ~ /:8080$/ {print}'
else
  echo "未找到 ss/netstat"
fi

echo "== 容器状态 =="
COMPOSE_DISABLE_ENV_FILE=1 docker compose -f deploy/docker-compose.wsl.yml ps
