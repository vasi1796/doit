#!/usr/bin/env bash
# =============================================================================
# DoIt — Database Backup Script
#
# Strategy:
#   - Daily pg_dump, compressed with gzip
#   - Retain 7 daily backups
#   - Retain 4 weekly backups (taken on Sundays)
#   - Optionally upload to off-VPS S3-compatible storage
#
# Usage:
#   ./scripts/backup.sh          # Run manually
#   Add to cron for daily runs:
#   0 3 * * * /path/to/doit/scripts/backup.sh >> /var/log/doit-backup.log 2>&1
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration (override via environment or .env)
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Source .env if present
if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "$PROJECT_DIR/.env"
    set +a
fi

BACKUP_DIR="${BACKUP_DIR:-/var/backups/doit}"
RETAIN_DAILY="${BACKUP_RETAIN_DAILY:-7}"
RETAIN_WEEKLY="${BACKUP_RETAIN_WEEKLY:-4}"

PG_USER="${POSTGRES_USER:-doit}"
PG_DB="${POSTGRES_DB:-doit}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"

DAILY_DIR="$BACKUP_DIR/daily"
WEEKLY_DIR="$BACKUP_DIR/weekly"

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
DAY_OF_WEEK="$(date +%u)"  # 7 = Sunday

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------
mkdir -p "$DAILY_DIR" "$WEEKLY_DIR"

echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Starting backup..."

# ---------------------------------------------------------------------------
# Dump
# ---------------------------------------------------------------------------
DUMP_FILE="$DAILY_DIR/doit_${TIMESTAMP}.sql.gz"

if docker compose -f "$PROJECT_DIR/docker-compose.yml" ps --status running postgres --quiet 2>/dev/null | grep -q .; then
    # Database is running inside Docker — use docker exec
    docker compose -f "$PROJECT_DIR/docker-compose.yml" exec -T postgres \
        pg_dump -U "$PG_USER" -d "$PG_DB" --no-owner --no-acl \
        | gzip > "$DUMP_FILE"
else
    # Assume local/remote PostgreSQL
    PGPASSWORD="${POSTGRES_PASSWORD:-}" pg_dump \
        -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" \
        --no-owner --no-acl \
        | gzip > "$DUMP_FILE"
fi

DUMP_SIZE="$(du -h "$DUMP_FILE" | cut -f1)"
echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Daily backup created: $DUMP_FILE ($DUMP_SIZE)"

# ---------------------------------------------------------------------------
# Weekly backup (copy Sunday's daily to the weekly directory)
# ---------------------------------------------------------------------------
if [[ "$DAY_OF_WEEK" -eq 7 ]]; then
    WEEKLY_FILE="$WEEKLY_DIR/doit_weekly_${TIMESTAMP}.sql.gz"
    cp "$DUMP_FILE" "$WEEKLY_FILE"
    echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Weekly backup created: $WEEKLY_FILE"
fi

# ---------------------------------------------------------------------------
# Retention — prune old backups
# ---------------------------------------------------------------------------
prune_old_backups() {
    local dir="$1"
    local keep="$2"
    local count
    count="$(find "$dir" -name '*.sql.gz' -type f | wc -l | tr -d ' ')"

    if [[ "$count" -gt "$keep" ]]; then
        local to_remove=$(( count - keep ))
        find "$dir" -name '*.sql.gz' -type f -print0 \
            | sort -z \
            | head -z -n "$to_remove" \
            | xargs -0 rm -f
        echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Pruned $to_remove old backup(s) from $dir"
    fi
}

prune_old_backups "$DAILY_DIR" "$RETAIN_DAILY"
prune_old_backups "$WEEKLY_DIR" "$RETAIN_WEEKLY"

# ---------------------------------------------------------------------------
# Optional: Upload to off-VPS S3-compatible storage
# ---------------------------------------------------------------------------
if [[ -n "${BACKUP_S3_BUCKET:-}" ]]; then
    if command -v aws &>/dev/null; then
        S3_ARGS=""
        if [[ -n "${BACKUP_S3_ENDPOINT:-}" ]]; then
            S3_ARGS="--endpoint-url $BACKUP_S3_ENDPOINT"
        fi
        # shellcheck disable=SC2086
        aws s3 cp "$DUMP_FILE" "$BACKUP_S3_BUCKET/daily/" $S3_ARGS
        echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Uploaded daily backup to $BACKUP_S3_BUCKET"

        if [[ "$DAY_OF_WEEK" -eq 7 ]] && [[ -f "${WEEKLY_FILE:-}" ]]; then
            # shellcheck disable=SC2086
            aws s3 cp "$WEEKLY_FILE" "$BACKUP_S3_BUCKET/weekly/" $S3_ARGS
            echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Uploaded weekly backup to $BACKUP_S3_BUCKET"
        fi
    else
        echo "[WARN] aws CLI not found — skipping S3 upload"
    fi
fi

echo "[$(date --iso-8601=seconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)] Backup complete."
