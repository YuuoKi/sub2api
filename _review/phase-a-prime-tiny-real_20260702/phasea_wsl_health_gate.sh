#!/usr/bin/env bash
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
