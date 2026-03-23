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

# Daily at 3:00 AM
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/doit/scripts/backup.sh >> /var/log/doit-backup.log 2>&1") | crontab -
```

Retains 7 daily + 4 weekly backups. See `scripts/backup.sh` for S3 upload options.

---

## 9. Updates

```bash
cd /opt/doit
git pull
docker compose up -d --build
```

Database migrations run automatically on API startup.

---

## 10. Troubleshooting

### Let's Encrypt certificate errors
- Ports 80 and 443 must be accessible from the internet for the ACME challenge.
- Check logs: `docker compose logs caddy`
- Rate limits: 50 certs per registered domain per week.

### Safari PWA install
- Visit in Safari > Share > "Add to Home Screen"
- Must be served over HTTPS
- If missing, clear Safari cache and retry

### Containers crash-looping
- Check logs: `docker compose logs <service-name>`
- Common: missing `.env` values, wrong DB password, RabbitMQ not ready
- Reset: `docker compose down && docker compose up -d`

### Database issues
- Health check: `docker compose exec postgres pg_isready`
- Password persists in volume — must match first-run value
- Full reset: `docker compose down -v` (destroys data)
