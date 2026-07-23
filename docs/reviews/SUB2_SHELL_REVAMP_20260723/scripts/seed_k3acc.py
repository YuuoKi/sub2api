#!/usr/bin/env python3
"""seed k3acc scratch backend: groups for wizard acceptance"""
import json, os, urllib.request

API = "http://127.0.0.1:8088"
PW = os.environ["ADPW"]

def req(method, path, body=None, token=None, extra_headers=None):
    r = urllib.request.Request(API + path, method=method)
    r.add_header("Content-Type", "application/json")
    if token:
        r.add_header("Authorization", f"Bearer {token}")
    for k, v in (extra_headers or {}).items():
        r.add_header(k, v)
    data = json.dumps(body).encode() if body is not None else None
    with urllib.request.urlopen(r, data, timeout=30) as resp:
        return json.loads(resp.read())

login = req("POST", "/api/v1/auth/login", {"email": "admin@wujie.local", "password": PW})
token = login["data"]["access_token"]

# compliance ack header discovery: try common header names
def create_group(name, platform, allow_image, token):
    full = {"name": name, "platform": platform, "description": "", "rate_multiplier": 1, "is_exclusive": False,
            "subscription_type": "standard", "daily_limit_usd": None, "weekly_limit_usd": None, "monthly_limit_usd": None,
            "allow_image_generation": allow_image, "allow_batch_image_generation": False, "image_rate_independent": False,
            "image_rate_multiplier": 1, "batch_image_discount_multiplier": 0.5, "batch_image_hold_multiplier": 0.6,
            "image_price_1k": None, "image_price_2k": None, "image_price_4k": None, "video_rate_independent": False,
            "video_rate_multiplier": 1, "video_price_480p": None, "video_price_720p": None, "video_price_1080p": None,
            "peak_rate_enabled": False, "peak_start": "", "peak_end": "", "peak_rate_multiplier": 1,
            "claude_code_only": False, "fallback_group_id": None, "fallback_group_id_on_invalid_request": None,
            "allow_messages_dispatch": False, "require_oauth_only": False, "require_privacy_set": False,
            "model_routing_enabled": False, "supported_model_scopes": ["claude", "gemini_text", "gemini_image"],
            "mcp_xml_inject": True, "copy_accounts_from_group_ids": [], "rpm_limit": 0}
    try:
        d = req("POST", "/api/v1/admin/groups", full, token)
        print(name, "->", d.get("code"), (d.get("data") or {}).get("id"))
        return d
    except urllib.error.HTTPError as e:
        body = e.read().decode()[:300]
        print(name, "HTTP", e.code, body)
        return None

create_group("media", "openai", True, token)
create_group("video", "openai", False, token)
