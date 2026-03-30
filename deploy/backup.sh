#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${DB_PATH:-/var/lib/roasti/data.db}"
BACKUP_TEMP="${BACKUP_TEMP:-/tmp/roasti-backup-$$.db}"

cleanup() {
    rm -f "$BACKUP_TEMP"
}
trap cleanup EXIT

echo "[backup] Creating SQLite snapshot..."
sqlite3 "$DB_PATH" ".backup '$BACKUP_TEMP'"

echo "[backup] Pushing to restic repository..."
restic backup "$BACKUP_TEMP" --tag roasti --tag sqlite

echo "[backup] Applying retention policy..."
restic forget \
    --tag roasti \
    --keep-daily 7 \
    --keep-weekly 4 \
    --keep-monthly 3 \
    --prune

echo "[backup] Done."
