#!/usr/bin/env python3
"""Run one Seedance tiny-real request and print only non-secret evidence."""

from __future__ import annotations

import json
import os
import stat
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path("/mnt/d/sub2api-trunk")
BASE = os.environ.get("SUB2API_BASE_URL", "http://127.0.0.1:8080")
KEY_PATH = Path("/tmp/phasea_sub2api_api_key")
EVIDENCE_PATH = Path("/tmp/phasea_direct_task.json")


def load_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw or raw.startswith("#") or "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        values[key] = value
    return values


def call(method: str, path: str, *, token: str | None = None, body: dict | None = None, timeout: int = 60) -> tuple[int, dict]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw or "{}")
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        try:
            payload = json.loads(raw or "{}")
        except json.JSONDecodeError:
            payload = {"message": raw[:240]}
        return exc.code, payload


def psql_scalar(sql: str) -> str:
    return subprocess.check_output(
        [
            "docker",
            "exec",
            "sub2api-postgres-dev",
            "psql",
            "-U",
            "sub2api",
            "-d",
            "sub2api",
            "-tAc",
            sql,
        ],
        text=True,
    ).strip()


def main() -> int:
    env = load_env(ROOT / "deploy/.env")
    status, login = call(
        "POST",
        "/api/v1/auth/login",
        body={"email": env["ADMIN_EMAIL"], "password": env["ADMIN_PASSWORD"]},
    )
    if status != 200:
        print(f"admin_login_http={status}")
        return 1
    admin_token = (login.get("data") or {}).get("access_token")
    user_id = (login.get("data") or {}).get("user", {}).get("id")
    if not admin_token or not user_id:
        print("admin_login_missing_fields=true")
        return 1

    call(
        "POST",
        f"/api/v1/admin/users/{user_id}/balance",
        token=admin_token,
        body={"balance": 100, "operation": "add", "notes": "phasea"},
    )
    _, groups = call("GET", "/api/v1/groups/available", token=admin_token)
    group_items = groups.get("data") or []
    group_id = (group_items[0] or {}).get("id") if group_items else None
    if not group_id:
        print("group_id_missing=true")
        return 1

    key_status, key_body = call(
        "POST",
        "/api/v1/keys",
        token=admin_token,
        body={"name": "phasea-final-safe", "group_id": group_id},
    )
    if key_status >= 400:
        print(f"key_create_http={key_status}")
        return 1
    api_key = (key_body.get("data") or {}).get("key") or (key_body.get("data") or {}).get("api_key")
    if not api_key:
        print("api_key_missing=true")
        return 1
    KEY_PATH.write_text(api_key, encoding="utf-8")
    KEY_PATH.chmod(stat.S_IRUSR | stat.S_IWUSR)

    subprocess.check_call(
        [
            "docker",
            "exec",
            "sub2api-postgres-dev",
            "psql",
            "-U",
            "sub2api",
            "-d",
            "sub2api",
            "-c",
            "DELETE FROM video_daily_trial_reservations WHERE provider='seedance';",
        ],
        stdout=subprocess.DEVNULL,
    )

    create_status, created = call(
        "POST",
        "/v1/video/tasks",
        token=api_key,
        body={
            "provider": "seedance",
            "trial_mode": "tiny_real",
            "task_type": "text_to_video",
            "model": "doubao-seedance-2-0-260128",
            "prompt": "Phase A prime tiny real final",
            "aspect_ratio": "16:9",
            "duration": 5,
            "resolution": "480p",
        },
        timeout=60,
    )
    print(f"create_status={create_status}")
    if create_status != 201:
        code = created.get("code") or created.get("message") or "unknown"
        print(f"create_failed={str(code)[:160]}")
        return 1
    task_id = (created.get("data") or {}).get("id")
    print(f"task_id={task_id}")
    if not task_id:
        return 1

    final: dict | None = None
    for i in range(1, 141):
        _, polled = call("GET", f"/v1/video/tasks/{task_id}", token=api_key)
        data = polled.get("data") or {}
        status = data.get("status")
        print(f"poll[{i}]={status}")
        if status in ("succeeded", "failed", "cancelled"):
            final = data
            break
        time.sleep(3)

    final = final or {}
    final_status = final.get("status")
    has_result_url = bool(final.get("result_url") or final.get("ResultURL"))
    print(f"final_status={final_status}")
    print(f"has_result_url={has_result_url}")
    if final.get("error_message"):
        print(f"error_message={str(final.get('error_message'))[:300]}")

    _, stats = call("GET", "/api/v1/admin/generation-content/stats", token=admin_token)
    data = stats.get("data") or {}
    is_live = data.get("is_live")
    captured_today = data.get("captured_today")
    print(f"is_live={is_live}")
    print(f"captured_today={captured_today}")
    rows = psql_scalar(f"SELECT COUNT(*) FROM ai_generation_content WHERE task_id='{task_id}';")
    print(f"capture_rows_for_task={rows}")
    EVIDENCE_PATH.write_text(
        json.dumps(
            {
                "task_id": task_id,
                "final_status": final_status,
                "has_result_url": has_result_url,
                "is_live": is_live,
                "captured_today": captured_today,
                "capture_rows_for_task": rows,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    return 0 if final_status == "succeeded" and has_result_url and is_live is True and rows == "1" else 1


if __name__ == "__main__":
    raise SystemExit(main())
