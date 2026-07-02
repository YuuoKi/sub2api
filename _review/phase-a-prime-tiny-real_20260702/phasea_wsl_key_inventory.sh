#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

pattern='SEEDANCE|VOLC|ARK|SUB2API.*KEY|VIDEO.*KEY'

echo "== wsl process env =="
env | awk -F= -v pat="$pattern" '$1 ~ pat {print $1 ":" length($2)}'

echo "== docker container env =="
for c in $(docker ps -a --format '{{.Names}}'); do
  echo "-- ${c} --"
  docker inspect "$c" --format '{{range .Config.Env}}{{println .}}{{end}}' |
    awk -F= -v pat="$pattern" '$1 ~ pat {print $1 ":" length($2)}'
done
