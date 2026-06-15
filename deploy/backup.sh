#!/usr/bin/env bash
# =============================================================================
# Wujie local data backup
# =============================================================================
# Usage: ./backup.sh [retention_days]
# Default retention: 7 days
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${BACKUP_DIR:-${SCRIPT_DIR}/backups}"
RETENTION_DAYS="${1:-7}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="${BACKUP_DIR}/wujie_api_${TIMESTAMP}.sql.gz"

DB_HOST="${DATABASE_HOST:-db}"
DB_PORT="${DATABASE_PORT:-5432}"
DB_USER="${POSTGRES_USER:-wujie}"
DB_NAME="${POSTGRES_DB:-wujie_api}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-wujie-api-db}"

mkdir -p "${BACKUP_DIR}"

echo "[$(date -Iseconds)] Starting backup: ${DB_NAME}@${DB_HOST}:${DB_PORT}"

docker exec "${POSTGRES_CONTAINER}" \
    pg_dump -U "${DB_USER}" -d "${DB_NAME}" --clean --if-exists \
    | gzip > "${BACKUP_FILE}"

BACKUP_SIZE="$(du -h "${BACKUP_FILE}" | cut -f1)"
echo "[$(date -Iseconds)] Backup complete: ${BACKUP_FILE} (${BACKUP_SIZE})"

# Prune old backups
DELETED=0
while IFS= read -r -d '' old_file; do
    rm -f "${old_file}"
    ((DELETED++)) || true
done < <(find "${BACKUP_DIR}" -name "wujie_api_*.sql.gz" -mtime "+${RETENTION_DAYS}" -print0)

if [[ ${DELETED} -gt 0 ]]; then
    echo "[$(date -Iseconds)] Pruned ${DELETED} backup(s) older than ${RETENTION_DAYS} days"
fi

# Summary
TOTAL_BACKUPS=$(find "${BACKUP_DIR}" -name "wujie_api_*.sql.gz" | wc -l)
TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" | cut -f1)
echo "[$(date -Iseconds)] Total: ${TOTAL_BACKUPS} backup(s), ${TOTAL_SIZE}"
