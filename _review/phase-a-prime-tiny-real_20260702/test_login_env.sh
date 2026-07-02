#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -a && source /mnt/d/sub2api-trunk/deploy/.env && set +a
python3 <<'PY'
import os, json, urllib.request, urllib.error
body = json.dumps({"email": os.environ["ADMIN_EMAIL"], "password": os.environ["ADMIN_PASSWORD"]}).encode()
req = urllib.request.Request("http://127.0.0.1:8080/api/v1/auth/login", data=body, headers={"Content-Type": "application/json"}, method="POST")
try:
    with urllib.request.urlopen(req, timeout=10) as r:
        print("login_ok", r.status)
except urllib.error.HTTPError as e:
    print("login_err", e.code, e.read().decode()[:300])
PY
