#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a
bash "${ROOT}/_review/phase-a-prime-tiny-real_20260702/bootstrap_seedance_provider.sh"
docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -c "DELETE FROM video_daily_trial_reservations WHERE provider='seedance';" >/dev/null
python3 <<'PY'
import json, os, time, urllib.request, urllib.error
base='http://127.0.0.1:8080'

def call(method, path, token=None, body=None, timeout=60):
    data=None if body is None else json.dumps(body).encode()
    headers={'Content-Type':'application/json'}
    if token: headers['Authorization']=f'Bearer {token}'
    req=urllib.request.Request(base+path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode())

_, login = call('POST','/api/v1/auth/login', body={'email':os.environ['ADMIN_EMAIL'],'password':os.environ['ADMIN_PASSWORD']})
admin_token=login['data']['access_token']
user_id=login['data']['user']['id']
call('POST', f'/api/v1/admin/users/{user_id}/balance', admin_token, {'balance':100,'operation':'add','notes':'phasea'})
_, groups = call('GET','/api/v1/groups/available', admin_token)
group_id=(groups.get('data') or [{}])[0].get('id')
_, key_body = call('POST','/api/v1/keys', admin_token, {'name':'phasea-final2','group_id': group_id})
api_key=key_body['data'].get('key') or key_body['data'].get('api_key')
status, created = call('POST','/v1/video/tasks', api_key, {
  'provider':'seedance','trial_mode':'tiny_real','task_type':'text_to_video',
  'model':'doubao-seedance-2-0-260128','prompt':'Phase A prime tiny real final',
  'aspect_ratio':'16:9','duration':5,'resolution':'480p'
})
print('create_status', status)
if status != 201:
    print('create_failed', created)
    raise SystemExit(1)
task_id=created['data']['id']
print('task_id', task_id)
final=None
for i in range(1,141):
    _, polled = call('GET', f'/v1/video/tasks/{task_id}', api_key)
    st=(polled.get('data') or {}).get('status')
    print(f'poll[{i}]={st}')
    if st in ('succeeded','failed','cancelled'):
        final=polled.get('data') or {}
        break
    time.sleep(3)
print('final_status', (final or {}).get('status'))
print('has_result_url', bool((final or {}).get('result_url') or (final or {}).get('ResultURL')))
if (final or {}).get('error_message'):
    print('error_message', (final or {}).get('error_message')[:300])
_, stats = call('GET','/api/v1/admin/generation-content/stats', admin_token)
print('is_live', (stats.get('data') or {}).get('is_live'))
print('captured_today', (stats.get('data') or {}).get('captured_today'))
import subprocess
out=subprocess.check_output(['docker','exec','sub2api-postgres-dev','psql','-U','sub2api','-d','sub2api','-tAc',f"SELECT COUNT(*) FROM ai_generation_content WHERE task_id='{task_id}';"], text=True)
print('capture_rows_for_task', out.strip())
PY
