#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a

echo "== bootstrap seedance =="
bash "${ROOT}/_review/phase-a-prime-tiny-real_20260702/bootstrap_seedance_provider.sh"

echo "== create api key =="
login_resp="$(python3 - <<'PY'
import json, os, urllib.request
body=json.dumps({"email":os.environ["ADMIN_EMAIL"],"password":os.environ["ADMIN_PASSWORD"]}).encode()
req=urllib.request.Request("http://127.0.0.1:8080/api/v1/auth/login", data=body, headers={"Content-Type":"application/json"}, method="POST")
print(urllib.request.urlopen(req, timeout=15).read().decode())
PY
)"
admin_token="$(printf '%s' "$login_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("data") or {}).get("access_token",""))')"
user_id="$(printf '%s' "$login_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(((d.get("data") or {}).get("user") or {}).get("id",""))')"

curl -sS -X POST "http://127.0.0.1:8080/api/v1/admin/users/${user_id}/balance" \
  -H "Authorization: Bearer ${admin_token}" \
  -H 'Content-Type: application/json' \
  -d '{"balance":100,"operation":"add","notes":"phasea-prime-test"}' >/dev/null
echo "balance_topped_up=yes"

apikey_resp="$(curl -sS -X POST "http://127.0.0.1:8080/api/v1/keys" \
  -H "Authorization: Bearer ${admin_token}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"phasea-prime-test"}')"
api_key="$(printf '%s' "$apikey_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); data=d.get("data") or {}; print(data.get("key") or data.get("api_key") or "")')"
if [[ -z "$api_key" ]]; then
  echo "api_key_create_failed"
  exit 1
fi
echo "api_key_created=yes"

echo "== tiny real post =="
create_resp="$(curl -sS -X POST "http://127.0.0.1:8080/v1/video/tasks" \
  -H "Authorization: Bearer ${api_key}" \
  -H 'Content-Type: application/json' \
  -d '{"provider":"seedance","trial_mode":"tiny_real","task_type":"text_to_video","model":"doubao-seedance-2-0-260128","prompt":"Phase A prime tiny real smoke test","aspect_ratio":"16:9","duration":3,"resolution":"480p"}')"
task_id="$(printf '%s' "$create_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("data") or {}).get("id",""))')"
status="$(printf '%s' "$create_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("data") or {}).get("status",""))')"
echo "task_id=${task_id} initial_status=${status}"
if [[ -z "$task_id" ]]; then
  printf '%s' "$create_resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("create_error", d.get("message"), d.get("code"))'
  exit 1
fi

for i in $(seq 1 140); do
  poll="$(curl -sS "http://127.0.0.1:8080/v1/video/tasks/${task_id}" -H "Authorization: Bearer ${api_key}")"
  st="$(printf '%s' "$poll" | python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("data") or {}).get("status",""))')"
  echo "poll[$i]=${st}"
  if [[ "$st" == "succeeded" || "$st" == "failed" || "$st" == "cancelled" ]]; then
    final="$poll"
    break
  fi
  sleep 3
done

printf '%s' "$final" | python3 -c 'import json,sys; d=json.load(sys.stdin).get("data") or {}; print("final_status", d.get("status")); print("has_result_url", bool(d.get("result_url") or d.get("ResultURL")))'

echo "== sql capture check =="
docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -tAc "SELECT COUNT(*) FROM ai_generation_content WHERE task_id='${task_id}';"

echo "== admin stats =="
curl -sS "http://127.0.0.1:8080/api/v1/admin/generation-content/stats" -H "Authorization: Bearer ${admin_token}" | python3 -c 'import json,sys; d=json.load(sys.stdin).get("data") or {}; print("is_live", d.get("is_live"), "captured_today", d.get("captured_today"))'
