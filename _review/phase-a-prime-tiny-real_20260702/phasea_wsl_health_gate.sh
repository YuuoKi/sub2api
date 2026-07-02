#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

for i in $(seq 1 10); do
  sleep 30
  ts="$(date +%H:%M:%S)"
  body="$(curl -s --max-time 5 http://127.0.0.1:8080/health || true)"
  printf '%s health[%02d]=%s\n' "$ts" "$i" "$body"
  if [ "$body" != '{"status":"ok"}' ]; then
    exit 23
  fi
done
