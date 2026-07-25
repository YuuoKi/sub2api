#!/usr/bin/env bash
# Guangzhou hot-update for staff-console hotfix (Sub2API only).
#
# Prerequisites on the caller machine:
#   - SSH access to ubuntu@114.132.50.149 (key or sshpass)
#   - Image tar produced by save-and-prepare.sh
#
# Usage:
#   export GZ_HOST=ubuntu@114.132.50.149
#   ./deploy-staff-hotfix.sh /path/to/wujie-sub2api-staff-hotfix-f8cc438f6.tar
#
# On the server this will:
#   1. docker load the tar and retag as wujie-sub2api:local
#   2. Prefer deploy dir ~/wujie/wujie-tencent-guangzhou-dualkey-<qc>-<sub2>
#   3. docker compose up -d --no-deps sub2api  (NO -v; volumes preserved)
#   4. Verify /health and build_commit
set -euo pipefail

IMAGE_TAR=${1:?image tar required}
HOST=${GZ_HOST:-ubuntu@114.132.50.149}
QC_SHA=${QC_SHA:-d6e54a8}
SUB2_SHA=${SUB2_SHA:-f8cc438f6}
VERSION_LABEL=${VERSION_LABEL:-广州内部版 2026.07.26-r1}
REMOTE_DIR=~/wujie/wujie-tencent-guangzhou-dualkey-${QC_SHA}-${SUB2_SHA}
REMOTE_TAR=/tmp/wujie-sub2api-staff-hotfix-${SUB2_SHA}.tar
COMPOSE_FILE=${COMPOSE_FILE:-compose.production.yml}

if [[ ! -f "$IMAGE_TAR" ]]; then
  echo "ERROR: tar not found: $IMAGE_TAR" >&2
  exit 1
fi

echo "==> upload image to ${HOST}"
scp "$IMAGE_TAR" "${HOST}:${REMOTE_TAR}"

echo "==> load + recreate sub2api (preserve volumes)"
ssh "$HOST" bash -s -- "$REMOTE_TAR" "$SUB2_SHA" "$REMOTE_DIR" "$COMPOSE_FILE" "$VERSION_LABEL" <<'EOF'
set -euo pipefail
REMOTE_TAR=$1
SUB2_SHA=$2
REMOTE_DIR=$3
COMPOSE_FILE=$4
VERSION_LABEL=$5

docker load -i "$REMOTE_TAR"
docker tag "wujie-sub2api:staff-hotfix-${SUB2_SHA}" wujie-sub2api:local

if [ -d "$REMOTE_DIR" ]; then
  cd "$REMOTE_DIR"
else
  # Create from newest existing dualkey dir if missing
  newest=$(ls -1d ~/wujie/wujie-tencent-guangzhou-dualkey-* 2>/dev/null | tail -1 || true)
  if [ -z "${newest}" ]; then
    echo "ERROR: no dualkey deploy dir under ~/wujie" >&2
    exit 1
  fi
  echo "WARN: ${REMOTE_DIR} missing; using ${newest}"
  cd "$newest"
fi
pwd
ls -la .env "$COMPOSE_FILE" 2>/dev/null || ls -la

# Prefer production compose; fall back to docker-compose.yml
if [ ! -f "$COMPOSE_FILE" ]; then
  if [ -f compose.production.yml ]; then
    COMPOSE_FILE=compose.production.yml
  elif [ -f docker-compose.yml ]; then
    COMPOSE_FILE=docker-compose.yml
  else
    echo "ERROR: no compose file found" >&2
    exit 1
  fi
fi

docker compose --env-file .env -f "$COMPOSE_FILE" up -d --no-deps sub2api
sleep 3

echo "==> local health"
for url in http://127.0.0.1:8081/health http://127.0.0.1:8080/health; do
  if curl -fsS "$url"; then
    echo
    echo "health_ok via $url"
    break
  fi
done

echo "==> build_commit probe"
curl -fsS http://127.0.0.1:8081/ 2>/dev/null | tr '"' '\n' | grep -E 'build_commit|version' | head -20 || true
curl -fsS http://127.0.0.1:8080/ 2>/dev/null | tr '"' '\n' | grep -E 'build_commit|version' | head -20 || true

echo "==> archive hint (manual)"
echo "After verifying public http://114.132.50.149/ build_commit=${SUB2_SHA},"
echo "tar old dualkey dirs and move into ~/wujie/archive/ (keep rollback path)."
echo "Expected VERSION label: ${VERSION_LABEL}"
EOF

echo "==> public verify from caller"
curl -fsS "http://114.132.50.149/health" || true
echo
curl -fsS "http://114.132.50.149/" | tr '"' '\n' | grep -E 'build_commit|version' | head -20 || true
echo
echo "DONE. Expect build_commit short sha == ${SUB2_SHA}"