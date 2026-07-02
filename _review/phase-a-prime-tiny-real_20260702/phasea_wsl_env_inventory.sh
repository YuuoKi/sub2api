#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

echo "== container env key lengths =="
for c in sub2api wujie-api-day0; do
  if ! docker inspect "$c" >/dev/null 2>&1; then
    continue
  fi
  echo "-- ${c} --"
  docker inspect "$c" --format '{{range .Config.Env}}{{println .}}{{end}}' |
    awk -F= '
      /DATABASE_|POSTGRES_|REDIS_|VIDEO_GATEWAY_|GATEWAY_CONTENT_|ADMIN_EMAIL|ADMIN_PASSWORD|JWT_SECRET|TOTP_ENCRYPTION_KEY|SUB2API_/ {
        print $1 ":" length($2)
      }
    '
done

echo "== env-like files key lengths =="
for f in \
  /home/yuuoki/sub2api-deploy/.env \
  /home/yuuoki/sub2api-deploy/docker-compose.yml \
  /mnt/d/sub2api-trunk/deploy/.env \
  /mnt/d/sub2api-trunk/deploy/docker-compose.dev.yml
do
  if [ ! -f "$f" ]; then
    continue
  fi
  echo "-- ${f} --"
  awk -F= '
    /DATABASE_|POSTGRES_|REDIS_|VIDEO_GATEWAY_|GATEWAY_CONTENT_|ADMIN_EMAIL|ADMIN_PASSWORD|JWT_SECRET|TOTP_ENCRYPTION_KEY|SUB2API_/ {
      gsub(/^[[:space:]-]+/, "", $1)
      gsub(/[[:space:]]+$/, "", $1)
      print $1 ":" length($2)
    }
  ' "$f"
done
