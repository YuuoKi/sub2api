#!/usr/bin/env bash
set -euo pipefail

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

docker exec s2a-mock-pg sh -lc '
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atqc "
    select encrypted_api_key
    from video_provider_accounts
    where provider = '\''seedance'\'' and display_name = '\''seedance-smoke'\''
    limit 1;
  "
' > "${tmp_dir}/ciphertext.txt"

docker inspect wujie-api-day0 --format '{{range .Config.Env}}{{println .}}{{end}}' |
  awk -F= '$1 == "VIDEO_GATEWAY_ENCRYPTION_KEY" {print $2}' > "${tmp_dir}/wujie_day0_video_key.txt"

if [ ! -s "${tmp_dir}/ciphertext.txt" ]; then
  echo "ciphertext=missing"
  exit 0
fi

if ! python3 -c 'from cryptography.hazmat.primitives.ciphers.aead import AESGCM' >/dev/null 2>&1; then
  echo "cryptography=missing"
  exit 0
fi

python3 - "$tmp_dir" <<'PY'
import base64
import pathlib
import sys
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

root = pathlib.Path(sys.argv[1])
ciphertext = (root / "ciphertext.txt").read_text().strip()
keys = {
    "wujie-api-day0:VIDEO_GATEWAY_ENCRYPTION_KEY": (root / "wujie_day0_video_key.txt").read_text().strip(),
}

for name, key_hex in keys.items():
    ok = False
    plain_len = 0
    reason = ""
    try:
        key = bytes.fromhex(key_hex)
        data = base64.b64decode(ciphertext)
        nonce = data[:12]
        sealed = data[12:]
        plain = AESGCM(key).decrypt(nonce, sealed, None)
        ok = True
        plain_len = len(plain)
    except Exception as exc:
        reason = exc.__class__.__name__
    if ok:
        print(f"{name}:decrypt_ok=true:plain_len={plain_len}")
    else:
        print(f"{name}:decrypt_ok=false:{reason}")
PY
