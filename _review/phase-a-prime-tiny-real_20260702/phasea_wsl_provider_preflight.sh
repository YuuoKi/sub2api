#!/usr/bin/env bash
set -euo pipefail

repo_root="${1:-/mnt/d/sub2api-trunk}"
out_dir="${repo_root}/_review/phase-a-prime-tiny-real_20260702"
deploy_dir="${repo_root}/deploy"

cd "${deploy_dir}"

docker compose -p sub2api_phasea_prime -f docker-compose.dev.yml exec -T sub2api sh -lc '
  printf "GATEWAY_CONTENT_CAPTURE_ENABLED=%s\n" "$GATEWAY_CONTENT_CAPTURE_ENABLED"
  printf "GATEWAY_CONTENT_RETENTION_ENABLED=%s\n" "$GATEWAY_CONTENT_RETENTION_ENABLED"
' > "${out_dir}/wsl_content_flags.txt"

set -a
. ./.env
set +a

login_payload="$(jq -nc --arg email "${ADMIN_EMAIL}" --arg password "${ADMIN_PASSWORD}" '{email:$email,password:$password}')"
login_resp="$(curl -sS --fail-with-body -H 'content-type: application/json' -d "${login_payload}" http://127.0.0.1:8080/api/v1/auth/login)"
token="$(jq -r '.data.access_token // empty' <<<"${login_resp}")"
if [ -z "${token}" ]; then
  jq '{code,message,reason}' <<<"${login_resp}" > "${out_dir}/wsl_admin_login_failed.json"
  echo "admin-login-token-missing"
  exit 41
fi

providers_resp="$(curl -sS --fail-with-body -H "authorization: Bearer ${token}" http://127.0.0.1:8080/api/v1/admin/video/providers)"
jq '{
  checked_at: now | todate,
  items: [(.data.items // [])[] | {
    provider,
    display_name,
    enabled,
    api_key_configured,
    route_available,
    route_skip_reason,
    key_status,
    health_status,
    diagnostic_type,
    suggested_action,
    default_model,
    priority,
    current_inflight,
    today_tasks,
    today_failures
  }]
}' <<<"${providers_resp}" > "${out_dir}/wsl_provider_preflight.json"

jq -e '
  (.items // []) |
  any(.provider == "seedance" and .enabled == true and .api_key_configured == true and .route_available == true)
' "${out_dir}/wsl_provider_preflight.json" >/dev/null || {
  echo "seedance-provider-not-ready"
  exit 42
}

echo "seedance-provider-ready"
