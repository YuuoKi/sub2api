#!/usr/bin/env bash
set -euo pipefail

containers="$(docker ps -a --format '{{.Names}}' | grep -E '(postgres|pg|db)$' || true)"
if [ -z "${containers}" ]; then
  echo "no-postgres-candidates"
  exit 0
fi

for c in ${containers}; do
  echo "== ${c} =="
  docker exec "${c}" sh -lc '
    db="${POSTGRES_DB:-postgres}"
    user="${POSTGRES_USER:-postgres}"
    if ! command -v psql >/dev/null 2>&1; then
      echo "psql-missing"
      exit 0
    fi
    exists="$(psql -U "$user" -d "$db" -Atqc "select exists (select 1 from information_schema.tables where table_schema = '\''public'\'' and table_name = '\''video_provider_accounts'\'')" 2>/dev/null || true)"
    if [ "$exists" != "t" ]; then
      echo "video_provider_accounts=absent"
      exit 0
    fi
    psql -U "$user" -d "$db" -P pager=off -F $'\''\t'\'' -Atqc "
      select
        provider,
        display_name,
        enabled,
        coalesce(encrypted_api_key, '\'''\'' ) <> '\'''\'' as api_key_configured,
        default_model,
        rate_limit_per_minute,
        coalesce(metadata_json::text, '\''{}'\'') like '\''%smoke_authorized%'\'' as smoke_metadata_present
      from video_provider_accounts
      where provider in ('\''seedance'\'','\''mock'\'','\''kling'\'')
      order by provider, id;
    "
  ' || echo "query-failed"
done
