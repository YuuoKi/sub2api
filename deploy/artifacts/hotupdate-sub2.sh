#!/usr/bin/env bash
# Hot-update Guangzhou Sub2API image in the active dualkey compose project.
# Auth: SSHPASS_FILE must point to a chmod 600 password file (never committed).
set -euo pipefail

HOST=${GZ_HOST:-ubuntu@114.132.50.149}
SUB2_SHA=${SUB2_SHA:-758f5c419}
QC_SHA=${QC_SHA:-d6e54a8}
LOCAL_TAR=${LOCAL_TAR:-/mnt/c/Temp/sub2api-deploy/wujie-sub2api-staff-hotfix-${SUB2_SHA}.tar}
REMOTE_TAR=/tmp/wujie-sub2api-staff-hotfix-${SUB2_SHA}.tar
# Use absolute remote paths — never pass a local-expanded ~
ACTIVE_DIR=/home/ubuntu/wujie/wujie-tencent-guangzhou-dualkey-${QC_SHA}-35d5f77
NEW_DIR=/home/ubuntu/wujie/wujie-tencent-guangzhou-dualkey-${QC_SHA}-${SUB2_SHA}
ARCHIVE_DIR=/home/ubuntu/wujie/archive
PASSFILE=${SSHPASS_FILE:?set SSHPASS_FILE to password file}

SSH_OPTS=(
  -o StrictHostKeyChecking=accept-new
  -o PreferredAuthentications=password
  -o PubkeyAuthentication=no
  -o NumberOfPasswordPrompts=1
  -o ConnectTimeout=30
  -o ServerAliveInterval=15
)

ssh_cmd() { sshpass -f "${PASSFILE}" ssh "${SSH_OPTS[@]}" "${HOST}" "$@"; }
scp_cmd() { sshpass -f "${PASSFILE}" scp "${SSH_OPTS[@]}" "$@"; }

[[ -f "${LOCAL_TAR}" ]] || { echo "missing tar ${LOCAL_TAR}" >&2; exit 1; }
[[ -f "${PASSFILE}" ]] || { echo "missing passfile" >&2; exit 1; }

echo "==> upload $(du -h "${LOCAL_TAR}" | awk '{print $1}')"
scp_cmd "${LOCAL_TAR}" "${HOST}:${REMOTE_TAR}"

echo "==> load + recreate sub2api in active project"
ssh_cmd bash -s -- "${REMOTE_TAR}" "${SUB2_SHA}" "${ACTIVE_DIR}" "${NEW_DIR}" "${ARCHIVE_DIR}" <<'EOF'
set -euo pipefail
REMOTE_TAR=$1
SUB2_SHA=$2
ACTIVE_DIR=$3
NEW_DIR=$4
ARCHIVE_DIR=$5

docker load -i "${REMOTE_TAR}"
SRC="wujie-sub2api:staff-hotfix-${SUB2_SHA}"
docker tag "${SRC}" wujie-production-sub2api:latest
docker tag "${SRC}" "wujie-sub2api:20260726-r3"
docker tag "${SRC}" wujie-sub2api:local

cd "${ACTIVE_DIR}"
# Recreate only sub2api; keep named volumes; do not rebuild from ./sources
docker compose --env-file .env -f compose.production.yml up -d --no-deps --force-recreate --no-build sub2api
sleep 5
docker compose --env-file .env -f compose.production.yml ps sub2api
curl -fsS http://127.0.0.1:8081/health
echo

EXPECTED_SHA="${SUB2_SHA}" python3 - <<'PY'
import json,os,re,urllib.request
html=urllib.request.urlopen('http://127.0.0.1:8081/',timeout=15).read().decode('utf-8','replace')
m=re.search(r'window\.__APP_CONFIG__=(\{.*?\});</script>',html)
assert m, 'missing __APP_CONFIG__'
cfg=json.loads(m.group(1))
for k in ('version','build_commit','build_date'):
    print(f'{k}={cfg.get(k)}')
commit=str(cfg.get('build_commit') or '')
expected=os.environ.get('EXPECTED_SHA','')
if not expected or not commit.startswith(expected[:7]):
    raise SystemExit(f'UNEXPECTED_COMMIT {commit} want_prefix={expected[:7]}')
print('COMMIT_OK')
PY

# Bookkeeping: mirror deploy dir name with new sub2 sha (no second stack)
if [ ! -d "${NEW_DIR}" ]; then
  mkdir -p "${NEW_DIR}"
  rsync -a --exclude 'backups' --exclude 'sources' "${ACTIVE_DIR}/" "${NEW_DIR}/"
  cp -a "${ACTIVE_DIR}/.env" "${NEW_DIR}/.env"
fi
printf 'QCanvas %s\nSub2API %s\nHotfix %s staff-console\n' \
  'd6e54a8228327294ed7e90da3dcf2638d2ccd9e4' \
  "$(python3 - <<'PY'
import json,re,urllib.request
html=urllib.request.urlopen('http://127.0.0.1:8081/',timeout=15).read().decode('utf-8','replace')
cfg=json.loads(re.search(r'window\.__APP_CONFIG__=(\{.*?\});</script>',html).group(1))
print(cfg.get('build_commit'))
PY
)" \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" | tee "${ACTIVE_DIR}/COMMITS.txt" "${NEW_DIR}/COMMITS.txt"

mkdir -p "${ARCHIVE_DIR}"
# Archive older dualkey trees that are not active/new (keep rollback)
ts=$(date +%Y%m%dT%H%M%SZ)
for d in /home/ubuntu/wujie/wujie-tencent-guangzhou-dualkey-*; do
  base=$(basename "$d")
  case "$base" in
    *d6e54a8-35d5f77*|*d6e54a8-${SUB2_SHA}*) continue ;;
  esac
  out="${ARCHIVE_DIR}/${base}-${ts}.tar.gz"
  if [ ! -f "${out}" ]; then
    echo "archive $d -> $out"
    tar -C /home/ubuntu/wujie -czf "${out}" "${base}" || true
  fi
done

rm -f "${REMOTE_TAR}"
echo REMOTE_DONE
EOF

echo "==> public verify"
EXPECTED_SHA="${SUB2_SHA}" python3 - <<'PY'
import json,os,re,urllib.request
html=urllib.request.urlopen('http://114.132.50.149/',timeout=20).read().decode('utf-8','replace')
cfg=json.loads(re.search(r'window\.__APP_CONFIG__=(\{.*?\});</script>',html).group(1))
for k in ('version','build_commit','build_date'):
    print(f'public_{k}={cfg.get(k)}')
expected=os.environ.get('EXPECTED_SHA','')
assert expected and str(cfg.get('build_commit','')).startswith(expected[:7]), cfg
print('PUBLIC_OK')
PY
curl -fsS http://114.132.50.149/health
echo
echo ALL_DONE
