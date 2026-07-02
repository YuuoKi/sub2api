#!/usr/bin/env bash
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a
python3 <<'PY'
import json, os, urllib.request
base='http://127.0.0.1:8080'
body=json.dumps({'email':os.environ['ADMIN_EMAIL'],'password':os.environ['ADMIN_PASSWORD']}).encode()
req=urllib.request.Request(base+'/api/v1/auth/login', data=body, headers={'Content-Type':'application/json'}, method='POST')
login=json.load(urllib.request.urlopen(req))
token=login['data']['access_token']
stats=json.load(urllib.request.urlopen(urllib.request.Request(base+'/api/v1/admin/generation-content/stats', headers={'Authorization':'Bearer '+token})))
data=stats.get('data') or {}
print('is_live', data.get('is_live'))
print('captured_today', data.get('captured_today'))
PY
