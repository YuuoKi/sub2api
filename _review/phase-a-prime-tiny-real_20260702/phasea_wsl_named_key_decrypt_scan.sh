#!/usr/bin/env bash
# REVIEW-ONLY: Evidence/reproduction helper. Do not run real provider or paid flows without explicit user authorization.
set -euo pipefail

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

repo_root="${1:-/mnt/d/sub2api-trunk}"
out_dir="${repo_root}/_review/phase-a-prime-tiny-real_20260702"

docker exec s2a-mock-pg sh -lc '
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atqc "
    select encrypted_api_key
    from video_provider_accounts
    where provider = '\''seedance'\'' and display_name = '\''seedance-smoke'\''
    limit 1;
  "
' > "${tmp_dir}/ciphertext.txt"

if [ ! -s "${tmp_dir}/ciphertext.txt" ]; then
  echo "ciphertext=missing" | tee "${out_dir}/wsl_named_key_decrypt_scan.txt"
  exit 0
fi

cat > "${tmp_dir}/scan.py" <<'PY'
import base64
import os
import pathlib
import re
import subprocess
import sys
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

tmp = pathlib.Path(sys.argv[1])
ciphertext = (tmp / "ciphertext.txt").read_text().strip()

roots = [
    pathlib.Path("/home/yuuoki"),
    pathlib.Path("/mnt/d/sub2api-trunk"),
    pathlib.Path("/mnt/d/Codex创业任务/企业 API 管理后台项目/02_source/sub2api"),
]

named_patterns = [
    re.compile(r'^\s*(?:export\s+)?VIDEO_GATEWAY_ENCRYPTION_KEY\s*=\s*["\']?([0-9a-fA-F]{64})["\']?\s*$'),
    re.compile(r'^\s*video_gateway\.encryption_key\s*[:=]\s*["\']?([0-9a-fA-F]{64})["\']?\s*$'),
    re.compile(r'^\s*encryption_key\s*[:=]\s*["\']?([0-9a-fA-F]{64})["\']?\s*$'),
]

skip_dirs = {
    ".git", "node_modules", ".pnpm-store", "dist", "build", ".next",
    "vendor", "__pycache__", ".cache", "go-build",
}

def iter_files(root):
    if not root.exists():
        return
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in skip_dirs and not d.endswith(".log")]
        path = pathlib.Path(dirpath)
        # keep the scan targeted to config/review/script files
        for name in filenames:
            lower = name.lower()
            if not (
                lower in {".env", "config.yaml", "config.yml", "docker-compose.yml", "docker-compose.dev.yml"}
                or lower.endswith((".env", ".yml", ".yaml", ".sh", ".ps1", ".md", ".txt"))
            ):
                continue
            f = path / name
            try:
                if f.stat().st_size > 2_000_000:
                    continue
            except OSError:
                continue
            yield f

def decrypt_ok(key_hex):
    try:
        key = bytes.fromhex(key_hex)
        data = base64.b64decode(ciphertext)
        plain = AESGCM(key).decrypt(data[:12], data[12:], None)
        return True, len(plain), ""
    except Exception as exc:
        return False, 0, exc.__class__.__name__

candidates = []
seen = set()

# Docker container envs are named sources too.
try:
    containers = subprocess.check_output(["docker", "ps", "-a", "--format", "{{.Names}}"], text=True).splitlines()
except Exception:
    containers = []
for c in containers:
    try:
        env = subprocess.check_output(["docker", "inspect", c, "--format", "{{range .Config.Env}}{{println .}}{{end}}"], text=True)
    except Exception:
        continue
    for line in env.splitlines():
        if line.startswith("VIDEO_GATEWAY_ENCRYPTION_KEY="):
            value = line.split("=", 1)[1].strip()
            if re.fullmatch(r"[0-9a-fA-F]{64}", value):
                source = f"docker:{c}:VIDEO_GATEWAY_ENCRYPTION_KEY"
                if value not in seen:
                    candidates.append((source, value))
                    seen.add(value)

for root in roots:
    for f in iter_files(root):
        try:
            text = f.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        for idx, line in enumerate(text.splitlines(), start=1):
            for pat in named_patterns:
                m = pat.match(line)
                if not m:
                    continue
                value = m.group(1)
                source = f"{f}:{idx}"
                if value not in seen:
                    candidates.append((source, value))
                    seen.add(value)

print(f"candidate_count={len(candidates)}")
matched = False
for source, value in candidates:
    ok, plain_len, reason = decrypt_ok(value)
    safe_source = str(source).replace(str(pathlib.Path.home()), "~")
    if ok:
        matched = True
        print(f"{safe_source}:decrypt_ok=true:plain_len={plain_len}")
    else:
        print(f"{safe_source}:decrypt_ok=false:{reason}")

if not matched:
    print("matched=false")
PY

python3 "${tmp_dir}/scan.py" "${tmp_dir}" | tee "${out_dir}/wsl_named_key_decrypt_scan.txt"
