#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

for c in s2a-mock-pg s2a-mock-redis sub2api wujie-api-day0; do
  if ! docker inspect "$c" >/dev/null 2>&1; then
    continue
  fi
  echo "== ${c} labels =="
  docker inspect "$c" --format '{{json .Config.Labels}}'
  echo
  echo "== ${c} mounts =="
  docker inspect "$c" --format '{{range .Mounts}}{{.Type}}	{{.Name}}	{{.Source}}	{{.Destination}}{{println}}{{end}}'
  echo "== ${c} networks =="
  docker inspect "$c" --format '{{range $name, $v := .NetworkSettings.Networks}}{{$name}} {{end}}'
  echo
done
