#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

printf '== containers ==\n'
docker ps -a --format '{{.Names}}	{{.Image}}	{{.Status}}	{{.Ports}}'

printf '== volumes ==\n'
docker volume ls --format '{{.Name}}' | grep -Ei 'sub2api|postgres|phasea|redis' || true

printf '== networks ==\n'
docker network ls --format '{{.Name}}' | grep -Ei 'sub2api|phasea' || true
