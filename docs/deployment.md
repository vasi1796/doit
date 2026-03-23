# DoIt — Deployment Guide

Step-by-step guide for deploying DoIt on an Ubuntu VM behind a MikroTik router
with DuckDNS for dynamic DNS and Caddy for automatic TLS.

---

## 1. Prerequisites

- **Ubuntu 22.04+** (or any systemd-based Linux distro)
- **Docker Engine** 24+ with **Docker Compose v2**
- **A DuckDNS subdomain** pointed to your public IP
- **MikroTik router** (or any router that supports port forwarding)
- **Google Cloud project** with OAuth 2.0 credentials (for production auth)

Install Docker on Ubuntu:

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for group membership to take effect
```

Verify:

```bash
docker --version
docker compose version
```

---

## 2. MikroTik Router Setup

Forward ports 80 (HTTP) and 443 (HTTPS) from your public IP to the VM's
internal IP. In WinBox or terminal:

```
/ip firewall nat add chain=dstnat protocol=tcp dst-port=80 \
    action=dst-nat to-addresses=<VM_INTERNAL_IP> to-ports=80
/ip firewall nat add chain=dstnat protocol=tcp dst-port=443 \
    action=dst-nat to-addresses=<VM_INTERNAL_IP> to-ports=443
```

Replace `<VM_INTERNAL_IP>` with your VM's LAN address (e.g., `192.168.1.100`).

Make sure port 443/udp is also forwarded if you want HTTP/3 support.

---

## 3. DuckDNS Setup

1. Sign up at [duckdns.org](https://www.duckdns.org/) and create a subdomain.
2. Set up a cron job on the VM to keep the IP updated:

```bash
echo "*/5 * * * * curl -s 'https://www.duckdns.org/update?domains=YOURSUBDOMAIN&token=YOUR_TOKEN&ip=' >/dev/null" \
    | crontab -
```

Verify DNS resolves to your public IP:

```bash
dig +short yoursubdomain.duckdns.org
```

---

## 4. Clone and Configure

```bash
sudo mkdir -p /opt/doit
sudo chown $USER:$USER /opt/doit
git clone https://github.com/vasi1796/doit.git /opt/doit
cd /opt/doit

cp .env.example .env
```

Edit `.env` with your values. At minimum, set:

- `DOMAIN=yoursubdomain.duckdns.org`
- `POSTGRES_PASSWORD` (choose a strong password)
- `RABBITMQ_PASSWORD` (choose a strong password)
- `JWT_SECRET` (see next section)
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`
- `ALLOWED_EMAILS` (your Google email)
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`
- `CORS_ORIGINS=https://yoursubdomain.duckdns.org`
- `ICAL_BASE_URL=https://yoursubdomain.duckdns.org`
- `SECURE_COOKIES=true`
- `DEV_MODE=false`

---

## 5. Generate Secrets

**JWT secret:**

```bash
openssl rand -base64 32
```

Paste the output into `JWT_SECRET` in `.env`.

**VAPID keys** (for push notifications):

```bash
npx web-push generate-vapid-keys
```

Copy the public and private keys into `VAPID_PUBLIC_KEY` and
`VAPID_PRIVATE_KEY`. Set `VAPID_SUBJECT` to your email address.

---

## 6. Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/).
2. Create a project (or use an existing one).
3. Navigate to **APIs & Services > Credentials**.
4. Click **Create Credentials > OAuth 2.0 Client ID**.
5. Set application type to **Web application**.
6. Add authorized redirect URI:
   `https://yoursubdomain.duckdns.org/auth/google/callback`
7. Copy the Client ID and Client Secret into `.env`:
   ```
   GOOGLE_CLIENT_ID=xxxx.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=GOCSPX-xxxx
   GOOGLE_REDIRECT_URL=https://yoursubdomain.duckdns.org/auth/google/callback
   ```

---

## 7. Deploy

Using the deploy script:

```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

Or manually:

```bash
docker compose up -d --build
```

The first build takes a few minutes. Caddy will automatically obtain a
Let's Encrypt TLS certificate for your domain.

---

## 8. Verify

Check the health endpoint:

```bash
curl -s https://yoursubdomain.duckdns.org/healthz
```

Check Caddy TLS logs (look for successful certificate issuance):

```bash
docker compose logs caddy | grep -i "certificate"
```

Check all services are running:

```bash
docker compose ps
```

Test login by visiting `https://yoursubdomain.duckdns.org` in Safari and
signing in with your Google account.

---

## 9. Systemd Auto-Start

To ensure DoIt starts automatically after a VM reboot:

```bash
sudo cp scripts/doit.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable doit.service
```

Manage the service:

```bash
sudo systemctl start doit     # Start
sudo systemctl stop doit      # Stop
sudo systemctl status doit    # Check status
sudo journalctl -u doit       # View logs
```

---

## 10. Backups

Set up automated daily backups using the included backup script:

```bash
chmod +x scripts/backup.sh
sudo mkdir -p /var/backups/doit

# Add to crontab — runs daily at 3:00 AM
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/doit/scripts/backup.sh >> /var/log/doit-backup.log 2>&1") | crontab -
```

The script creates daily compressed PostgreSQL dumps and retains 7 daily +
4 weekly backups. See `scripts/backup.sh` for S3 upload options.

---

## 11. Updates

Pull the latest code and rebuild:

```bash
cd /opt/doit
git pull
docker compose up -d --build
```

If database migrations are included in the update, they run automatically
on API container startup.

---

## 12. Troubleshooting

### Port forwarding not working

- Verify MikroTik NAT rules are active: `/ip firewall nat print`
- Check that the VM firewall allows ports 80 and 443:
  ```bash
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  ```
- Test from outside your network (e.g., mobile data)

### DNS not resolving

- DuckDNS updates can take a few minutes to propagate.
- Verify your token and subdomain in the cron job.
- Check from an external DNS: `dig +short yoursubdomain.duckdns.org @8.8.8.8`

### Let's Encrypt certificate errors

- Caddy needs ports 80 and 443 accessible from the internet for the ACME
  challenge.
- Let's Encrypt has [rate limits](https://letsencrypt.org/docs/rate-limits/):
  50 certificates per registered domain per week.
- Check Caddy logs: `docker compose logs caddy`
- If rate-limited, wait or use a different subdomain temporarily.

### Safari PWA install

- Visit the site in Safari, tap the Share button, then "Add to Home Screen."
- The app must be served over HTTPS for PWA installation.
- If the install option does not appear, clear Safari cache and retry.
- Ensure the `manifest.json` is being served (check Network tab in
  Safari Web Inspector).

### Containers crash-looping

- Check logs: `docker compose logs <service-name>`
- Common causes: missing `.env` values, wrong database password, RabbitMQ
  not ready yet (workers depend on health checks, but timing can vary).
- Restart everything cleanly: `docker compose down && docker compose up -d`

### Database connection issues

- Verify Postgres is healthy: `docker compose exec postgres pg_isready`
- Check that `POSTGRES_PASSWORD` in `.env` matches what was used on first
  run (Postgres persists the password in the volume).
- To reset: `docker compose down -v` (destroys data), then `docker compose up -d`
