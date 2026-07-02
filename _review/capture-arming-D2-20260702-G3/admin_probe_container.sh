#!/bin/sh
set -eu

base="http://127.0.0.1:8080"
out="/app/data/g3_admin_result.json"
login_body="/tmp/g3_login_body.json"
login_json="/tmp/g3_login.json"
stats_json="/tmp/g3_stats.json"
samples_json="/tmp/g3_samples.json"

email="${ADMIN_EMAIL:-admin@sub2api.local}"
password="${ADMIN_PASSWORD:-}"
if [ -z "$password" ]; then
  printf '{"status":"error","phase":"env","error":"ADMIN_PASSWORD empty"}\n' > "$out"
  cat "$out"
  exit 1
fi

i=0
while [ "$i" -lt 45 ]; do
  if wget -q -T 2 -O /tmp/g3_health.json "$base/health"; then
    break
  fi
  i=$((i + 1))
  sleep 1
done
if [ "$i" -ge 45 ]; then
  printf '{"status":"error","phase":"health","health":false}\n' > "$out"
  cat "$out"
  exit 1
fi

cat > "$login_body" <<EOF
{"email":"$email","password":"$password"}
EOF

if ! wget -q -T 5 -O "$login_json" \
  --header "Content-Type: application/json" \
  --post-file "$login_body" \
  "$base/api/v1/auth/login"; then
  printf '{"status":"error","phase":"login","health":true}\n' > "$out"
  cat "$out"
  exit 1
fi

token="$(sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p' "$login_json" | head -n 1)"
rm -f "$login_body" "$login_json"
if [ -z "$token" ]; then
  printf '{"status":"error","phase":"login_token","health":true}\n' > "$out"
  cat "$out"
  exit 1
fi

if ! wget -q -T 5 -O "$stats_json" \
  --header "Authorization: Bearer $token" \
  "$base/api/v1/admin/generation-content/stats"; then
  printf '{"status":"error","phase":"stats","health":true}\n' > "$out"
  cat "$out"
  exit 1
fi

if ! wget -q -T 5 -O "$samples_json" \
  --header "Authorization: Bearer $token" \
  "$base/api/v1/admin/generation-content/samples"; then
  printf '{"status":"error","phase":"samples","health":true}\n' > "$out"
  cat "$out"
  exit 1
fi

if grep -q '"is_live":true' "$stats_json"; then
  is_live=true
else
  is_live=false
fi

if grep -q '"is_live":true' "$samples_json"; then
  samples_live=true
else
  samples_live=false
fi

if grep -Eq 'sk-[A-Za-z0-9_-]{20,}|13800138000' "$samples_json"; then
  sample_preview_safe=false
else
  sample_preview_safe=true
fi

captured_today="$(sed -n 's/.*"captured_today":\([0-9][0-9]*\).*/\1/p' "$stats_json" | head -n 1)"
if [ -z "$captured_today" ]; then
  captured_today=0
fi
sample_count="$(grep -o '"prompt_preview"' "$samples_json" | wc -l | tr -d ' ')"

cat > "$out" <<EOF
{"status":"ok","health":true,"stats_status":200,"samples_status":200,"is_live":$is_live,"samples_live":$samples_live,"captured_today":$captured_today,"sample_count":$sample_count,"sample_preview_safe":$sample_preview_safe}
EOF
cat "$out"
