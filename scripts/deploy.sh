#!/usr/bin/env bash
# =============================================================================
# DoIt — Deploy Script
#
# First-time setup or update: checks prerequisites, builds containers,
# waits for health checks, and prints a status summary.
#
# Usage:
#   ./scripts/deploy.sh
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

log() { echo "[$(date +%H:%M:%S)] $*"; }

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
if ! command -v docker &>/dev/null; then
    echo "ERROR: docker is not installed. See https://docs.docker.com/engine/install/" >&2
    exit 1
fi

if ! docker compose version &>/dev/null; then
    echo "ERROR: docker compose v2 is required. See https://docs.docker.com/compose/install/" >&2
    exit 1
fi

if [[ ! -f .env ]]; then
    echo "ERROR: .env file not found. Copy .env.example to .env and fill in values:" >&2
    echo "  cp .env.example .env" >&2
    exit 1
fi

# The API refuses to start without ALLOWED_EMAILS (outside dev mode), and
# compose interpolation hard-fails without RABBITMQ_PASSWORD — check up front
# so failures happen before containers are torn down.
for var in RABBITMQ_PASSWORD ALLOWED_EMAILS; do
    if ! grep -qE "^${var}=." .env; then
        echo "ERROR: ${var} is not set in .env" >&2
        exit 1
    fi
done

# ---------------------------------------------------------------------------
# Build & Start
# ---------------------------------------------------------------------------
log "Building and starting containers..."
# Remove the web-build one-shot container so it re-runs and copies
# fresh frontend assets into the shared volume Caddy serves from.
docker compose rm -fsv web-build 2>/dev/null || true
docker compose up -d --build

# ---------------------------------------------------------------------------
# Wait for health checks
# ---------------------------------------------------------------------------
log "Waiting for services to become healthy..."
DOMAIN=$(grep -E '^DOMAIN=' .env | cut -d= -f2- || echo "localhost")
# Probe the API's published port directly — Caddy only answers for $DOMAIN,
# so a localhost request through port 80 404s on any real deployment.
HEALTHZ_URL="http://127.0.0.1:8080/healthz"

MAX_ATTEMPTS=30
for i in $(seq 1 $MAX_ATTEMPTS); do
    if curl -sf "$HEALTHZ_URL" >/dev/null 2>&1; then
        log "Health check passed."
        break
    fi
    if [[ $i -eq $MAX_ATTEMPTS ]]; then
        log "WARNING: Health check did not pass after ${MAX_ATTEMPTS} attempts."
        log "Check logs with: docker compose logs"
        exit 1
    fi
    sleep 2
done

# ---------------------------------------------------------------------------
# Status Summary
# ---------------------------------------------------------------------------
echo ""
echo "========================================="
echo "  DoIt deployed successfully"
echo "========================================="
docker compose ps --format "table {{.Name}}\t{{.Status}}"
echo ""
log "Access the app at https://${DOMAIN}"
