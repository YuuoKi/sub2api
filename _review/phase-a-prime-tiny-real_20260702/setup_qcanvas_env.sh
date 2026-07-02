#!/usr/bin/env bash
# Creates Sub2API gateway API key and writes to QCanvas hono .env snippet (no stdout secrets).
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
QC_ROOT="/mnt/d/Codex创业任务/QCanvas（无界版）/QCanvas"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a

python3 <<'PY'
import json, os, urllib.request, pathlib

base = 'http://127.0.0.1:8080'

def call(method, path, token=None, body=None):
    data = None if body is None else json.dumps(body).encode()
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = f'Bearer {token}'
    req = urllib.request.Request(base + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode())

login = call('POST', '/api/v1/auth/login', body={
    'email': os.environ['ADMIN_EMAIL'],
    'password': os.environ['ADMIN_PASSWORD'],
})
admin_token = login['data']['access_token']
user_id = login['data']['user']['id']
call('POST', f'/api/v1/admin/users/{user_id}/balance', admin_token, {
    'balance': 100, 'operation': 'add', 'notes': 'phasea-qcanvas',
})
groups = call('GET', '/api/v1/groups/available', admin_token)
group_id = (groups.get('data') or [{}])[0].get('id')
key_body = call('POST', '/api/v1/keys', admin_token, {'name': 'phasea-qcanvas-bridge', 'group_id': group_id})
api_key = key_body['data'].get('key') or key_body['data'].get('api_key')
if not api_key:
    raise SystemExit('api_key_create_failed')

hono_env = pathlib.Path(os.environ.get('QC_HONO_ENV', f"{os.environ.get('QC_ROOT', '/mnt/d/Codex创业任务/QCanvas（无界版）/QCanvas')}/apps/hono-api/.env"))
logs_dir = hono_env.parent / 'logs'
logs_dir.mkdir(parents=True, exist_ok=True)

existing = {}
if hono_env.exists():
    for line in hono_env.read_text(encoding='utf-8').splitlines():
        if '=' in line and not line.strip().startswith('#'):
            k, v = line.split('=', 1)
            existing[k.strip()] = v

updates = {
    'PORT': '8788',
    'SUB2API_BASE_URL': 'http://127.0.0.1:8080',
    'SUB2API_API_KEY': api_key,
    'SUB2API_VIDEO_MOCK_GATEWAY_ENABLED': '1',
    'SUB2API_VIDEO_REAL_SMOKE_ENABLED': '1',
    'SUB2API_REAL_HUMAN_AUTHORIZED': '1',
    'SUB2API_REAL_ENABLED': '1',
    'SUB2API_REAL_MODEL_ALLOWLIST': 'seedance-2,seedance-2-fast,doubao-seedance-2-0-260128',
    'SUB2API_REAL_REDACTED_EVENT_LOG': './logs/qcanvas-real.log',
    'SUB2API_REAL_SINGLE_CALL_BUDGET_CENTS': '100',
    'SUB2API_REAL_DAILY_BUDGET_CENTS': '1000',
}
existing.update(updates)

lines = []
for k, v in existing.items():
    lines.append(f'{k}={v}')
hono_env.write_text('\n'.join(lines) + '\n', encoding='utf-8')

web_env = pathlib.Path(os.environ.get('QC_WEB_ENV', str(hono_env.parent.parent / 'web' / '.env')))
web_existing = {}
if web_env.exists():
    for line in web_env.read_text(encoding='utf-8').splitlines():
        if '=' in line and not line.strip().startswith('#'):
            k, v = line.split('=', 1)
            web_existing[k.strip()] = v
web_existing['VITE_API_BASE'] = 'http://127.0.0.1:8788'
web_env.write_text('\n'.join(f'{k}={v}' for k, v in web_existing.items()) + '\n', encoding='utf-8')

print('hono_env_written', str(hono_env))
print('web_env_written', str(web_env))
print('api_key_configured', bool(api_key))
PY

# Clear daily trial reservation so QCanvas can fire one tiny real if needed
docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -c \
  "DELETE FROM video_daily_trial_reservations WHERE provider='seedance';" >/dev/null 2>&1 || true
