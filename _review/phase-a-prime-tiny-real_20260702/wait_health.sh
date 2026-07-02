#!/usr/bin/env bash
set -euo pipefail
for i in $(seq 1 30); do
  if curl -sf http://127.0.0.1:8080/health >/dev/null; then
    echo "health_ok_attempt_${i}"
    exit 0
  fi
  sleep 3
done
echo health_failed
exit 1
