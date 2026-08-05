# DoIt

Personal, self-hosted task management app. Event-sourced Go backend (Postgres + RabbitMQ), offline-first React PWA frontend for Safari/Apple ecosystem.

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- [Go 1.26+](https://go.dev/) and Node.js 20+ (only for running the API/frontend outside Docker)
- Google OAuth 2.0 credentials ([setup guide](#google-oauth-setup))

## Quick Start (full stack in Docker)

```bash
# 1. Copy and configure environment
cp .env.example .env
# Edit .env with your Google OAuth credentials, JWT secret, and allowed emails

# 2. Start everything — Postgres, RabbitMQ, API, workers, frontend, Caddy
docker compose up -d --build

# 3. Open https://localhost in Safari
# Accept the self-signed certificate warning, then log in with Google
```

Migrations run automatically on API startup — there is no manual migration step.

## Local Development

The full pipeline needs Postgres, RabbitMQ, the API (with outbox poller), and
the projection worker — without the worker, writes never reach the read models.

For processes running outside Docker, uncomment `DATABASE_URL` and set
`RABBITMQ_URL` in `.env` to point at localhost (ports 5432 and 5672, using
your `POSTGRES_*` and `RABBITMQ_*` credentials), then source it before each
command:

```bash
# Terminal 1: Postgres + RabbitMQ
docker compose up postgres rabbitmq -d

# Terminal 2: API (runs migrations on startup, includes outbox poller)
set -a && source .env && set +a
cd api && DEV_MODE=true SECURE_COOKIES=false go run ./cmd/api

# Terminal 3: Projection worker
set -a && source .env && set +a
cd api && go run ./cmd/worker

# Terminal 4: Recurring tasks worker (optional — creates next occurrences)
set -a && source .env && set +a
cd api && go run ./cmd/worker-recurring

# Terminal 5: Frontend dev server (hot reload)
cd web && npm install && npm run dev -- --host
```

`DEV_MODE=true` enables `POST /auth/dev` for logging in without Google.

### API smoke test (curl)

The app itself talks to the API through the sync engine, but the REST
endpoints are handy for poking the backend directly:

```bash
# Dev login (stores the auth cookie — email must be on the ALLOWED_EMAILS allowlist)
curl -s -X POST http://localhost:8080/auth/dev \
  -H "Content-Type: application/json" \
  -d '{"email":"your@email.com"}' \
  -c /tmp/doit-cookies.txt

# Create a task (omit list_id for inbox)
curl -s -b /tmp/doit-cookies.txt -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","priority":1,"position":"a"}'

# List tasks (projected asynchronously — needs the worker running)
curl -s -b /tmp/doit-cookies.txt http://localhost:8080/api/v1/tasks | python3 -m json.tool
```

## Running Tests

```bash
# Backend
make test              # unit tests
make test-integration  # integration tests (needs Postgres + RabbitMQ)
make test-fullstack    # full pipeline in-process: needs Postgres + RabbitMQ up,
                       # but NO worker processes running (stop Terminals 3–4 —
                       # the tests consume queue messages directly)
make vet               # go vet

# Frontend
cd web && npm test           # unit tests (Vitest)
cd web && npm run lint       # ESLint (includes jsx-a11y)
cd web && npm run test:e2e   # Playwright E2E: visual + accessibility + functional
```

## Changing the API Contract

`api/openapi.yaml` is the source of truth for the API. After editing it, run
`make generate` to regenerate the Go and TypeScript types — never hand-edit
`*.gen.*` files.

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
api/           Go backend (event store, domain, projections, workers, handlers)
web/           React PWA frontend (Dexie.js offline store, CRDT sync engine)
docs/          Design document, deployment guide, ADRs, architecture diagrams
deploy/        Deploy webhook sidecar (GitHub webhook → pull → rebuild)
scripts/       Backup and deploy scripts
```

- [AGENTS.md](AGENTS.md) — full architecture details
- [docs/deployment.md](docs/deployment.md) — production deployment guide
