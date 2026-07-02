#!/usr/bin/env python3
"""Register/update the Seedance provider without placing secrets in argv."""

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
            payload = {"message": raw[:120]}
        return exc.code, payload


def provider_items(payload: dict) -> list[dict]:
    data = payload.get("data") or {}
    items = data.get("items") if isinstance(data, dict) else data
    return items if isinstance(items, list) else []


def main() -> int:
    env = load_env(ROOT / "deploy/.env")
    required = ["ADMIN_EMAIL", "ADMIN_PASSWORD", "SEEDANCE_API_KEY"]
    missing = [key for key in required if not env.get(key)]
    if missing:
        print("seedance_preflight=not_ready")
        print("missing_required_env=true")
        return 1

    login_status, login = call(
        "POST",
        "/api/v1/auth/login",
        body={"email": env["ADMIN_EMAIL"], "password": env["ADMIN_PASSWORD"]},
    )
    token = (login.get("data") or {}).get("access_token") or (login.get("data") or {}).get("token")
    if login_status != 200 or not token:
        print("seedance_preflight=not_ready")
        print(f"admin_login_http={login_status}")
        return 1

    _, providers = call("GET", "/api/v1/admin/video/providers", token=token)
    provider_id = ""
    for item in provider_items(providers):
        if item.get("provider") == "seedance" and item.get("enabled"):
            provider_id = str(item.get("id") or "")
            break

    payload = {
        "provider": "seedance",
        "display_name": "Phase A prime Seedance",
        "enabled": True,
        "api_key": env["SEEDANCE_API_KEY"],
        "base_url": "https://ark.cn-beijing.volces.com/api/v3",
        "default_model": "doubao-seedance-2-0-260128",
        "metadata_json": {
            "single_smoke_authorized": True,
            "priority": 5,
            "key_status": "normal",
            "health_status": "healthy",
        },
    }
    if provider_id:
        write_status, _ = call("PATCH", f"/api/v1/admin/video/providers/{provider_id}", token=token, body=payload)
    else:
        write_status, _ = call("POST", "/api/v1/admin/video/providers", token=token, body=payload)
    if write_status >= 400:
        print("seedance_preflight=not_ready")
        print(f"provider_write_http={write_status}")
        return 1

    _, preflight = call("GET", "/api/v1/admin/video/providers", token=token)
    ready = "not_ready"
    for item in provider_items(preflight):
        if item.get("provider") != "seedance":
            continue
        if item.get("enabled") and item.get("api_key_configured") and item.get("route_available"):
            ready = "ready"
            break
    print(f"seedance_preflight={ready}")
    return 0 if ready == "ready" else 1


if __name__ == "__main__":
    raise SystemExit(main())
