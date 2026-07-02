#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -a && source /mnt/d/sub2api-trunk/deploy/.env && set +a
python3 <<'PY'
import json, os, urllib.request
body=json.dumps({"email":os.environ["ADMIN_EMAIL"],"password":os.environ["ADMIN_PASSWORD"]}).encode()
req=urllib.request.Request("http://127.0.0.1:8080/api/v1/auth/login", data=body, headers={"Content-Type":"application/json"}, method="POST")
token=json.load(urllib.request.urlopen(req))["data"]["access_token"]
req2=urllib.request.Request("http://127.0.0.1:8080/api/v1/admin/video/providers", headers={"Authorization": f"Bearer {token}"})
payload=json.load(urllib.request.urlopen(req2))
print("top_keys", list(payload.keys()))
data=payload.get("data")
print("data_type", type(data).__name__)
if isinstance(data, dict):
    print("data_keys", list(data.keys()))
    items=data.get("items") or data.get("providers") or []
    print("items_type", type(items).__name__, "count", len(items))
    for it in items[:8]:
        print(it.get("provider"), it.get("id"), it.get("enabled"), it.get("route_available"), it.get("api_key_configured"))
elif isinstance(data, list):
    print("count", len(data))
    for it in data[:5]:
        print(it.get("provider"), it.get("id"), it.get("enabled"), it.get("route_available"))
PY
