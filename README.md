# DoIt

Personal, self-hosted task management app. Event-sourced Go backend, React PWA frontend (Safari/Apple ecosystem).

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- [Go 1.26+](https://go.dev/) (for local development)
- Node.js 20+ (for frontend, not yet implemented)
- Google OAuth 2.0 credentials ([setup guide](#google-oauth-setup))

## Quick Start

```bash
# 1. Copy and configure environment
cp .env.example .env
# Edit .env with your Google OAuth credentials, JWT secret, and allowed emails

# 2. Start everything
docker compose up -d --build

# 3. Run migrations
DATABASE_URL="postgres://doit:changeme@localhost:5432/doit?sslmode=disable" make migrate

# 4. Open https://localhost/auth/google/login in Safari
# Accept the self-signed certificate warning, then log in with Google
```

## Local Development

### Running the API outside Docker

Useful for faster iteration and debugging:

```bash
# Start only Postgres
docker compose up postgres -d

# Run migrations
DATABASE_URL="postgres://doit:changeme@localhost:5432/doit?sslmode=disable" make migrate

# Start the API
set -a && source .env && set +a
DATABASE_URL="postgres://doit:changeme@localhost:5432/doit?sslmode=disable" \
SECURE_COOKIES=false make run
```

### Testing with Google OAuth (needs Caddy for TLS)

```bash
# Start Postgres + API + Caddy
docker compose up -d --build

# Run migrations (if not already applied)
DATABASE_URL="postgres://doit:changeme@localhost:5432/doit?sslmode=disable" make migrate

# Open in Safari
# https://localhost/auth/google/login
```

### Testing with dev login (no Google needed)

```bash
docker compose up postgres -d

set -a && source .env && set +a
DATABASE_URL="postgres://doit:changeme@localhost:5432/doit?sslmode=disable" \
DEV_MODE=true SECURE_COOKIES=false make run
```

In another terminal:

```bash
# Login
curl -s -X POST http://localhost:8080/auth/dev \
  -H "Content-Type: application/json" \
  -d '{"email":"your@email.com"}' \
  -c /tmp/doit.txt

# Create a list
curl -s -b /tmp/doit.txt -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name":"Work","colour":"#3b82f6","position":"a"}'

# Create a task (omit list_id for inbox)
curl -s -b /tmp/doit.txt -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","priority":1,"position":"a"}'

# List tasks
curl -s -b /tmp/doit.txt http://localhost:8080/api/v1/tasks | python3 -m json.tool

# Complete a task
curl -s -b /tmp/doit.txt -X POST http://localhost:8080/api/v1/tasks/<id>/complete

# Delete a task
curl -s -b /tmp/doit.txt -X DELETE http://localhost:8080/api/v1/tasks/<id>

# List all lists
curl -s -b /tmp/doit.txt http://localhost:8080/api/v1/lists | python3 -m json.tool

# List all labels
curl -s -b /tmp/doit.txt http://localhost:8080/api/v1/labels | python3 -m json.tool
```

## Running Tests

```bash
make test              # unit tests
make test-verbose      # unit tests with output
make test-integration  # integration tests (needs running Postgres)
make vet               # go vet
make build             # compile
```

## Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a project (or use existing)
3. APIs & Services → Credentials → Create Credentials → OAuth client ID
4. Application type: **Web application**
5. Authorized JavaScript origins: `https://localhost`
6. Authorized redirect URIs: `https://localhost/auth/google/callback`
7. Copy Client ID and Client Secret into `.env`
8. Generate JWT secret: `openssl rand -base64 32`

## Project Structure

```
api/           Go backend (event store, domain, projections, handlers)
web/           React frontend (not yet implemented)
docs/          Design document and Architecture Decision Records
scripts/       Backup utilities
```

See [AGENTS.md](AGENTS.md) for full architecture details.
