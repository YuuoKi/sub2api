#!/usr/bin/env bash
set -euo pipefail

ok=0
for i in $(seq 1 10); do
  body="$(curl -sf http://127.0.0.1:8080/health)"
  echo "health_${i}=${body}"
  if [[ "${body}" == "ok" || "${body}" == '{"status":"ok"}' ]]; then
    ok=$((ok + 1))
  else
    echo "health_unexpected_${i}"
    exit 1
  fi
  sleep 1
done
echo "health_ok_count=${ok}"
