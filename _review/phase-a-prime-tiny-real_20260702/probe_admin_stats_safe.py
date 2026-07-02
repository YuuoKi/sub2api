#!/usr/bin/env python3
"""Read local .env, call Admin stats, and print only non-secret evidence."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path("/mnt/d/sub2api-trunk")
BASE = os.environ.get("SUB2API_BASE_URL", "http://127.0.0.1:8080")


def load_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw or raw.startswith("#") or "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        values[key] = value
    return values


def request_json(method: str, path: str, *, token: str | None = None, body: dict | None = None) -> tuple[int, dict]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        payload = exc.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(payload)
        except json.JSONDecodeError:
            parsed = {"error": payload[:120]}
        return exc.code, parsed


def main() -> int:
    env = load_env(ROOT / "deploy/.env")
    login_status, login = request_json(
        "POST",
        "/api/v1/auth/login",
        body={"email": env.get("ADMIN_EMAIL", ""), "password": env.get("ADMIN_PASSWORD", "")},
    )
    print(f"login_http={login_status}")
    token = (login.get("data") or {}).get("access_token")
    print(f"login_has_token={str(bool(token)).lower()}")
    if not token:
        err = login.get("code") or (login.get("error") or {}).get("code") or login.get("message") or "unknown"
        print(f"login_error={str(err)[:120]}")
        return 1
    stats_status, stats = request_json("GET", "/api/v1/admin/generation-content/stats", token=token)
    print(f"stats_http={stats_status}")
    data = stats.get("data") or {}
    print(f"is_live={data.get('is_live')}")
    print(f"captured_today={data.get('captured_today')}")
    return 0 if stats_status == 200 else 1


if __name__ == "__main__":
    raise SystemExit(main())
