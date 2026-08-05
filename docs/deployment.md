# DoIt — Deployment Guide

Deploy DoIt using Docker Compose with Caddy for automatic TLS.

---

## 1. Prerequisites

- **Linux server** with Docker Engine 24+ and Docker Compose v2
- **A domain** pointed to your server's public IP
- **Ports 80 and 443** forwarded to the server
- **Google Cloud project** with OAuth 2.0 credentials (for authentication)

---

## 2. Clone and Configure

```bash
git clone https://github.com/vasi1796/doit.git /opt/doit
cd /opt/doit
cp .env.example .env
```

Edit `.env` with your values. At minimum, set:

- `DOMAIN=yourdomain.com`
- `POSTGRES_PASSWORD` (strong password)
- `RABBITMQ_PASSWORD` (strong password)
- `JWT_SECRET` (see next section)
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`
- `ALLOWED_EMAILS` (your Google email)
- `CORS_ORIGINS=https://yourdomain.com`
- `SECURE_COOKIES=true`
- `DEV_MODE=false`

Optional (for push notifications):
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`

Optional (for iCal feed):
- `ICAL_BASE_URL=https://yourdomain.com`

Optional (for auto-deploy on push to main):
- `DEPLOY_WEBHOOK_SECRET` (generate with `openssl rand -hex 32`)

Optional (event pipeline tuning; sensible defaults apply):
- `RABBITMQ_PUBLISH_TIMEOUT` — how long the outbox publisher waits for a broker
  confirmation before retrying the batch (default `5s`)
- `PROJECTION_RETRY_MAX` — projection attempts before an event is dead-lettered
  (default `5`)
- `PROJECTION_RETRY_DELAY` — wait between projection retries (default `2s`)

Optional (container log rotation; defaults apply):
- `LOG_MAX_SIZE` (default `20m`) and `LOG_MAX_FILE` (default `5`) — per-container
  json-file log caps applied to all long-running services

---

## 3. Generate Secrets

**JWT secret:**

```bash
openssl rand -base64 32
```

**VAPID keys** (for push notifications):

```bash
npx web-push generate-vapid-keys
```

---

## 4. Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/) > **APIs & Services > Credentials**.
2. Create an **OAuth 2.0 Client ID** (Web application).
3. Add authorized redirect URI: `https://yourdomain.com/auth/google/callback`
4. Copy Client ID and Client Secret into `.env`.

---

## 5. Deploy

```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

Or manually:

```bash
docker compose up -d --build
```

Caddy automatically obtains a Let's Encrypt TLS certificate for your domain.

---

## 6. Verify

```bash
# Health check
curl -s https://yourdomain.com/healthz

# Check all services
docker compose ps

# Check Caddy TLS
docker compose logs caddy | grep -i "certificate"
```

---

## 7. Auto-Start on Reboot

All services use `restart: unless-stopped` — they restart automatically when Docker starts.

---

## 8. Backups

```bash
chmod +x scripts/backup.sh
sudo mkdir -p /var/backups/doit
sudo chown "$USER" /var/backups/doit

# Daily at 3:00 AM
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/doit/scripts/backup.sh >> /var/log/doit-backup.log 2>&1") | crontab -
```

The cron entry runs as your user, so that user must own the backup directory
(hence the `chown`) and be in the `docker` group. Retention is configurable
via `BACKUP_RETAIN_DAILY` / `BACKUP_RETAIN_WEEKLY` (defaults: 7 daily +
4 weekly); set `BACKUP_S3_BUCKET` for off-VPS upload.

Only Postgres is backed up — it is the source of truth and everything else
can be rebuilt from it. Note the `caddy_data` volume (TLS certificates) is
not included: after a full-server restore, Caddy re-issues certificates,
which counts against the Let's Encrypt rate limit (50/week per domain).

---

## 9. Updates

```bash
cd /opt/doit
git pull
docker compose rm -fsv web-build   # so the one-shot frontend build re-runs
docker compose up -d --build
```

The `web-build` container is a one-shot build step — without removing it
first, `up` considers it complete and Caddy keeps serving the previous
frontend from the shared volume. `scripts/deploy.sh` does both steps for you.

Database migrations run automatically on API startup.

---

## 10. Auto-Deploy via GitHub Webhook (optional)

Set `DEPLOY_WEBHOOK_SECRET` in `.env`, then configure GitHub:
1. Repo → Settings → Webhooks → Add webhook
2. URL: `https://yourdomain.com/deploy/webhook`
3. Secret: same value from `.env`
4. Content type: `application/json`
5. Events: Just the push event

The `deployer` sidecar container receives the webhook, verifies the
HMAC-SHA256 signature, and on pushes to main runs `git pull --ff-only`,
rebuilds the five app services (`doit-api`, `web-build`, `worker`,
`worker-recurring`, `worker-reminder`), resets the one-shot `web-build`
container, and brings those services up. Non-main pushes are ignored.
If `DEPLOY_WEBHOOK_SECRET` is unset, the deployer idles and the webhook
returns 503.

Auto-deploy deliberately does **not** touch `postgres`, `rabbitmq`, `caddy`,
or the deployer itself (they rely on host bind mounts unavailable inside the
sidecar). Changes to the Caddyfile, compose infra stanzas, or `deploy/` need
a manual `docker compose up -d --build` on the host.

The sidecar pins the compose project name to `doit`, so clone the repo into
a directory named `doit` (as above) — or set `COMPOSE_PROJECT_NAME=doit` in
`.env` — so host-run compose commands share the same project.

If using a private repo, set the git remote to use a PAT:
```bash
git remote set-url origin https://<user>:<token>@github.com/user/doit.git
```

---

## 11. Troubleshooting

### Let's Encrypt certificate errors
- Ports 80 and 443 must be accessible from the internet for the ACME challenge.
- Check logs: `docker compose logs caddy`
- Rate limits: 50 certs per registered domain per week.

### Safari PWA install
- Visit in Safari > Share > "Add to Home Screen"
- Must be served over HTTPS
- If missing, clear Safari cache and retry

### Containers crash-looping or refusing to start
- Check logs: `docker compose logs <service-name>`
- Common: missing `.env` values, wrong DB password, RabbitMQ not ready
- The API refuses to start without `ALLOWED_EMAILS` when `DEV_MODE=false`
- `docker compose` itself fails fast if `RABBITMQ_PASSWORD` is unset —
  every compose command needs a complete `.env`
- Reset: `docker compose down && docker compose up -d`

### Database issues
- Health check: `docker compose exec postgres pg_isready`
- Password persists in volume — must match first-run value
- Full reset: `docker compose down -v` (destroys data)

### Docker networking / iptables
- If external traffic reaches the server but not the containers, check:
  `sudo iptables -P FORWARD` — if it's `DROP`, set it to `ACCEPT`:
  `sudo iptables -P FORWARD ACCEPT`
- Make this persistent: `sudo apt install iptables-persistent && sudo netfilter-persistent save`

### RabbitMQ connection refused on startup
- Normal — RabbitMQ takes longer to start than the API. The broker client
  auto-reconnects with exponential backoff and workers have reconnection-aware
  consume loops, so transient RabbitMQ outages are handled automatically.
  Check with `docker compose logs doit-api`
