#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail
sed -i 's/\r$//' /mnt/d/sub2api-trunk/deploy/.env
set -a && source /mnt/d/sub2api-trunk/deploy/.env && set +a
python3 <<'PY'
import json, os, time, urllib.request
base='http://127.0.0.1:8080'
body=json.dumps({'email':os.environ['ADMIN_EMAIL'],'password':os.environ['ADMIN_PASSWORD']}).encode()
req=urllib.request.Request(base+'/api/v1/auth/login', data=body, headers={'Content-Type':'application/json'}, method='POST')
login=json.load(urllib.request.urlopen(req))
admin_token=login['data']['access_token']
groups=json.load(urllib.request.urlopen(urllib.request.Request(base+'/api/v1/groups/available', headers={'Authorization':'Bearer '+admin_token})))
group_id=groups['data'][0]['id']
key_req=urllib.request.Request(base+'/api/v1/keys', data=json.dumps({'name':'poll-only','group_id':group_id}).encode(), headers={'Authorization':'Bearer '+admin_token,'Content-Type':'application/json'}, method='POST')
api_key=json.load(urllib.request.urlopen(key_req))['data']['key']
for i in range(1,141):
    polled=json.load(urllib.request.urlopen(urllib.request.Request(base+'/v1/video/tasks/1', headers={'Authorization':'Bearer '+api_key})))
    d=polled.get('data') or {}
    st=d.get('status')
    print(f'poll[{i}]={st}')
    if st in ('succeeded','failed','cancelled'):
        print('final_status', st)
        print('has_result_url', bool(d.get('result_url') or d.get('ResultURL')))
        if d.get('error_message'): print('error_message', d.get('error_message')[:300])
        break
    time.sleep(3)
stats=json.load(urllib.request.urlopen(urllib.request.Request(base+'/api/v1/admin/generation-content/stats', headers={'Authorization':'Bearer '+admin_token})))
print('is_live', stats.get('data',{}).get('is_live'))
print('captured_today', stats.get('data',{}).get('captured_today'))
PY
