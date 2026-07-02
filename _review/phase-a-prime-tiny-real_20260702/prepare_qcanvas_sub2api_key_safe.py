#!/usr/bin/env python3
"""Prepare a Sub2API API key for QCanvas without creating video tasks."""

from __future__ import annotations

import json
import os
import stat
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path("/mnt/d/sub2api-trunk")
BASE = os.environ.get("SUB2API_BASE_URL", "http://127.0.0.1:8080")
KEY_PATH = Path("/tmp/phasea_sub2api_api_key")


def load_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw or raw.startswith("#") or "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        values[key] = value
    return values


def call(method: str, path: str, *, token: str | None = None, body: dict | None = None) -> tuple[int, dict]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw or "{}")
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        try:
            payload = json.loads(raw or "{}")
        except json.JSONDecodeError:
            payload = {"message": raw[:160]}
        return exc.code, payload


def main() -> int:
    env = load_env(ROOT / "deploy/.env")
    status, login = call(
        "POST",
        "/api/v1/auth/login",
        body={"email": env["ADMIN_EMAIL"], "password": env["ADMIN_PASSWORD"]},
    )
    print(f"admin_login_http={status}")
    token = (login.get("data") or {}).get("access_token")
    user_id = (login.get("data") or {}).get("user", {}).get("id")
    print(f"admin_login_has_token={str(bool(token)).lower()}")
    if status != 200 or not token or not user_id:
        return 1

    balance_status, _ = call(
        "POST",
        f"/api/v1/admin/users/{user_id}/balance",
        token=token,
        body={"balance": 100, "operation": "add", "notes": "phasea-qcanvas"},
    )
    print(f"balance_http={balance_status}")

    groups_status, groups = call("GET", "/api/v1/groups/available", token=token)
    print(f"groups_http={groups_status}")
    group_items = groups.get("data") or []
    group_id = (group_items[0] or {}).get("id") if group_items else None
    print(f"group_id_present={str(bool(group_id)).lower()}")
    if not group_id:
        return 1

    key_status, key_body = call(
        "POST",
        "/api/v1/keys",
        token=token,
        body={"name": "phasea-qcanvas-longpoll", "group_id": group_id},
    )
    print(f"key_create_http={key_status}")
    api_key = (key_body.get("data") or {}).get("key") or (key_body.get("data") or {}).get("api_key")
    print(f"api_key_present={str(bool(api_key)).lower()}")
    if key_status >= 400 or not api_key:
        return 1
    KEY_PATH.write_text(api_key, encoding="utf-8")
    KEY_PATH.chmod(stat.S_IRUSR | stat.S_IWUSR)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
