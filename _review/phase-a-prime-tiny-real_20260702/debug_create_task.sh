#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail
ROOT="/mnt/d/sub2api-trunk"
sed -i 's/\r$//' "${ROOT}/deploy/.env"
set -a && source "${ROOT}/deploy/.env" && set +a
python3 <<'PY'
import json, os, urllib.request, urllib.error
base='http://127.0.0.1:8080'

def req(method, path, token=None, body=None):
    data=None if body is None else json.dumps(body).encode()
    headers={'Content-Type':'application/json'}
    if token: headers['Authorization']=f'Bearer {token}'
    r=urllib.request.Request(base+path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=30) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode())

login=json.dumps({'email':os.environ['ADMIN_EMAIL'],'password':os.environ['ADMIN_PASSWORD']}).encode()
_, login_body = req('POST','/api/v1/auth/login', body=json.loads(login.decode()))
admin_token=login_body['data']['access_token']
user_id=login_body['data']['user']['id']
req('POST', f'/api/v1/admin/users/{user_id}/balance', admin_token, {'balance':100,'operation':'add','notes':'phasea'})
_, groups_body = req('GET','/api/v1/groups/available', admin_token)
groups = groups_body.get('data') or []
group_id = groups[0]['id'] if groups else None
print('group_id', group_id)
_, key_body = req('POST','/api/v1/keys', admin_token, {'name':'phasea-debug','group_id': group_id})
api_key=key_body['data'].get('key') or key_body['data'].get('api_key')
print('api_key_present', bool(api_key))
status, body = req('POST','/v1/video/tasks', api_key, {
  'provider':'seedance','trial_mode':'tiny_real','task_type':'text_to_video',
  'model':'doubao-seedance-2-0-260128','prompt':'phasea debug',
  'aspect_ratio':'16:9','duration':3,'resolution':'480p'
})
print('create_status', status)
print('create_body', json.dumps(body, ensure_ascii=False)[:800])
PY
