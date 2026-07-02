#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
# Phase A' only: register Seedance provider after fresh dev DB bootstrap.
# Requires: SEEDANCE_API_KEY, ADMIN_EMAIL, ADMIN_PASSWORD in environment (never echo values).
set -euo pipefail

BASE_URL="${SUB2API_BASE_URL:-http://127.0.0.1:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:?ADMIN_EMAIL required}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:?ADMIN_PASSWORD required}"
SEEDANCE_API_KEY="${SEEDANCE_API_KEY:?SEEDANCE_API_KEY required}"

login_resp="$(python3 - <<'PY'
import json, os, urllib.request
email = os.environ["ADMIN_EMAIL"]
password = os.environ["ADMIN_PASSWORD"]
body = json.dumps({"email": email, "password": password}).encode()
req = urllib.request.Request(
    os.environ.get("SUB2API_BASE_URL", "http://127.0.0.1:8080") + "/api/v1/auth/login",
    data=body,
    headers={"Content-Type": "application/json"},
    method="POST",
)
with urllib.request.urlopen(req, timeout=15) as resp:
    print(resp.read().decode())
PY
)"
token="$(printf '%s' "$login_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); data=d.get("data") or {}; print(data.get("access_token") or data.get("token") or "")')"
if [[ -z "$token" ]]; then
  echo "admin_login_failed"
  exit 1
fi

providers="$(curl -sS "${BASE_URL}/api/v1/admin/video/providers" \
  -H "Authorization: Bearer ${token}")"
provider_id="$(printf '%s' "$providers" | python3 -c '
import json,sys
payload=json.load(sys.stdin)
data=payload.get("data") or {}
items=data.get("items") if isinstance(data, dict) else data
if not isinstance(items, list):
    items=[]
for it in items:
    if it.get("provider")=="seedance" and it.get("enabled"):
        print(it["id"])
        break
')"

payload="$(python3 -c 'import json,os; print(json.dumps({
  "provider": "seedance",
  "display_name": "Phase A prime Seedance",
  "enabled": True,
  "api_key": os.environ["SEEDANCE_API_KEY"],
  "base_url": "https://ark.cn-beijing.volces.com/api/v3",
  "default_model": "doubao-seedance-2-0-260128",
  "metadata_json": {
    "single_smoke_authorized": True,
    "priority": 5,
    "key_status": "normal",
    "health_status": "healthy"
  }
}))')"

if [[ -n "$provider_id" ]]; then
  curl -sS -X PATCH "${BASE_URL}/api/v1/admin/video/providers/${provider_id}" \
    -H "Authorization: Bearer ${token}" \
    -H 'Content-Type: application/json' \
    -d "$payload" > /dev/null
else
  curl -sS -X POST "${BASE_URL}/api/v1/admin/video/providers" \
    -H "Authorization: Bearer ${token}" \
    -H 'Content-Type: application/json' \
    -d "$payload" > /dev/null
fi

preflight="$(curl -sS "${BASE_URL}/api/v1/admin/video/providers" \
  -H "Authorization: Bearer ${token}")"
ready="$(printf '%s' "$preflight" | python3 -c '
import json,sys
payload=json.load(sys.stdin)
data=payload.get("data") or {}
items=data.get("items") if isinstance(data, dict) else data
if not isinstance(items, list):
    items=[]
for it in items:
    if it.get("provider")!="seedance":
        continue
    if it.get("enabled") and it.get("api_key_configured") and it.get("route_available"):
        print("ready")
        raise SystemExit(0)
print("not_ready")
')"

echo "seedance_preflight=${ready}"
