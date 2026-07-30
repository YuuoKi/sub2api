#!/usr/bin/env bash
# Save staff-hotfix image to a Linux path (Docker daemon is WSL/Linux),
# then copy the tar to a Windows-visible folder.
set -euo pipefail

COMMIT="${1:-f8cc438f6}"
IMAGE="wujie-sub2api:staff-hotfix-${COMMIT}"
LINUX_OUT="/tmp/sub2api-staff-hotfix-deploy/wujie-sub2api-staff-hotfix-${COMMIT}.tar"
WIN_OUT="/mnt/c/Temp/sub2api-deploy/wujie-sub2api-staff-hotfix-${COMMIT}.tar"
WT_OUT="/mnt/d/sub2api-trunk/.worktrees/staff-console-hotfix-20260726/deploy/artifacts/wujie-sub2api-staff-hotfix-${COMMIT}.tar"

mkdir -p "$(dirname "$LINUX_OUT")" /mnt/c/Temp/sub2api-deploy \
  "$(dirname "$WT_OUT")"

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "ERROR: image missing: $IMAGE" >&2
  docker images 'wujie-sub2api' --format '{{.Repository}}:{{.Tag}}'
  exit 1
fi

rm -f "$LINUX_OUT"
echo "==> docker save $IMAGE -> $LINUX_OUT"
docker save -o "$LINUX_OUT" "$IMAGE"
ls -lh "$LINUX_OUT"

echo "==> copy to Windows paths"
cp -f "$LINUX_OUT" "$WIN_OUT"
cp -f "$LINUX_OUT" "$WT_OUT"
ls -lh "$WIN_OUT" "$WT_OUT"

echo "OK"