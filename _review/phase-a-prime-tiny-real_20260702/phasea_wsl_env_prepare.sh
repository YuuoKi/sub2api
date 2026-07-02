#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

repo_root="${1:-/mnt/d/sub2api-trunk}"
deploy_dir="${repo_root}/deploy"

cd "${deploy_dir}"

POSTGRES_PASSWORD=phasea_down_only docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml down -v

rm -f .env
umask 077

rand_hex() {
  openssl rand -hex 32
}

{
  printf 'POSTGRES_USER=sub2api\n'
  printf 'POSTGRES_PASSWORD=%s\n' "$(rand_hex)"
  printf 'POSTGRES_DB=sub2api\n'
  printf 'REDIS_PASSWORD=%s\n' "$(rand_hex)"
  printf 'ADMIN_EMAIL=admin@sub2api.local\n'
  printf 'ADMIN_PASSWORD=%s\n' "$(rand_hex)"
  printf 'JWT_SECRET=%s\n' "$(rand_hex)"
  printf 'TOTP_ENCRYPTION_KEY=%s\n' "$(rand_hex)"
  printf 'VIDEO_GATEWAY_ENCRYPTION_KEY=%s\n' "$(rand_hex)"
  printf 'BIND_HOST=127.0.0.1\n'
  printf 'SERVER_PORT=8080\n'
  printf 'GATEWAY_CONTENT_CAPTURE_ENABLED=true\n'
  printf 'GATEWAY_CONTENT_RETENTION_ENABLED=true\n'
  printf 'VIDEO_GATEWAY_WORKER_ENABLED=true\n'
  printf 'VIDEO_GATEWAY_COST_PER_SECOND=0.01\n'
  printf 'VIDEO_GATEWAY_PER_CALL_BUDGET=0.08\n'
  printf 'VIDEO_GATEWAY_POLL_INTERVAL_SECONDS=3\n'
  printf 'VIDEO_GATEWAY_MAX_POLL_ATTEMPTS=140\n'
  printf 'VIDEO_GATEWAY_TASK_TIMEOUT_MINUTES=8\n'
  printf 'TZ=Asia/Shanghai\n'
} > .env

chmod 600 .env

printf 'env-written-keys\n'
cut -d= -f1 .env
