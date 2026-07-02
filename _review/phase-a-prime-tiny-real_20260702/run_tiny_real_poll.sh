#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a
python3 <<'PY'
import json, os, time, urllib.request, urllib.error
base='http://127.0.0.1:8080'

def call(method, path, token=None, body=None, timeout=30):
    data=None if body is None else json.dumps(body).encode()
    headers={'Content-Type':'application/json'}
    if token: headers['Authorization']=f'Bearer {token}'
    req=urllib.request.Request(base+path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body=e.read().decode()
        try: payload=json.loads(body)
        except Exception: payload={'raw': body[:500]}
        return e.code, payload

_, login = call('POST','/api/v1/auth/login', body={'email':os.environ['ADMIN_EMAIL'],'password':os.environ['ADMIN_PASSWORD']})
admin_token=login['data']['access_token']
user_id=login['data']['user']['id']
call('POST', f'/api/v1/admin/users/{user_id}/balance', admin_token, {'balance':100,'operation':'add','notes':'phasea'})
_, groups = call('GET','/api/v1/groups/available', admin_token)
group_id=(groups.get('data') or [{}])[0].get('id')
_, key_body = call('POST','/api/v1/keys', admin_token, {'name':'phasea-final','group_id': group_id})
api_key=key_body['data'].get('key') or key_body['data'].get('api_key')
status, created = call('POST','/v1/video/tasks', api_key, {
  'provider':'seedance','trial_mode':'tiny_real','task_type':'text_to_video',
  'model':'doubao-seedance-2-0-260128','prompt':'Phase A prime final tiny real test',
  'aspect_ratio':'16:9','duration':3,'resolution':'480p'
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
    st=polled.get('data',{}).get('status')
    print(f'poll[{i}]={st}')
    if st in ('succeeded','failed','cancelled'):
        final=polled.get('data') or {}
        break
    time.sleep(3)
if not final:
    print('poll_timeout')
    raise SystemExit(1)
print('final_status', final.get('status'))
print('has_result_url', bool(final.get('result_url') or final.get('ResultURL')))
if final.get('error_message'):
    print('error_message', final.get('error_message')[:200])
_, stats = call('GET','/api/v1/admin/generation-content/stats', admin_token)
print('is_live', (stats.get('data') or {}).get('is_live'))
print('captured_today', (stats.get('data') or {}).get('captured_today'))
PY
