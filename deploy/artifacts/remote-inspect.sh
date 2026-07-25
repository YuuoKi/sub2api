#!/usr/bin/env bash
set -euo pipefail
echo '=== host ==='
hostname
whoami
pwd
echo '=== wujie dirs ==='
ls -la ~/wujie 2>/dev/null || true
echo '=== dualkey dirs ==='
ls -1d ~/wujie/wujie-tencent-guangzhou-dualkey-* 2>/dev/null || true
echo '=== preferred dir ==='
PREF=~/wujie/wujie-tencent-guangzhou-dualkey-0561ed5-239ec7e
if [ -d "$PREF" ]; then
  echo "FOUND $PREF"
  cd "$PREF"
  pwd
  ls -la
  echo '--- compose files ---'
  ls -la compose*.yml docker-compose*.yml 2>/dev/null || true
  echo '--- running containers ---'
  docker compose --env-file .env -f compose.production.yml ps 2>/dev/null || docker compose --env-file .env ps 2>/dev/null || docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
else
  echo "MISSING $PREF"
  newest=$(ls -1d ~/wujie/wujie-tencent-guangzhou-dualkey-* 2>/dev/null | tail -1 || true)
  echo "newest=$newest"
  if [ -n "$newest" ]; then
    cd "$newest"
    pwd
    ls -la
    docker compose --env-file .env ps 2>/dev/null || docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
  fi
fi
echo '=== local health ==='
curl -fsS http://127.0.0.1:8081/health || curl -fsS http://127.0.0.1:8080/health || true
echo
curl -fsS http://127.0.0.1:8081/ 2>/dev/null | tr '"' '\n' | grep -E 'build_commit|version' | head -10 || true
