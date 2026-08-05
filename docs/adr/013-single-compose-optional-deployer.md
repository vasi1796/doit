# ADR-013: Single Compose File and Optional-Idle Deploy Sidecar

**Status:** Accepted

## Context

`docker-compose.prod.yml` held production overrides (log-rotation caps, CPU/memory
limits, json logging) that no documented deploy path ever applied:
`scripts/deploy.sh`, the deploy webhook sidecar, and `docs/deployment.md` all ran
plain `docker compose up -d --build`. The overrides were dead config — production
containers logged unbounded json-file logs, a disk-fill risk on a small VPS. The
file had also rotted internally: a "Phase 3 — uncomment when ready" block long
after Phase 3 shipped, and a `worker-cleanup` service that does not exist.

Separately, the `deployer` sidecar called `log.Fatal` when
`DEPLOY_WEBHOOK_SECRET` was empty, while compose starts it unconditionally with
`restart: unless-stopped` and both `.env.example` and the deployment guide
describe auto-deploy as optional. A default install therefore got a permanently
crash-looping container.

Related: the deploy sidecar was introduced without an ADR (it is documented in
`docs/deployment.md`); this ADR records its failure semantics.

## Decision

We will maintain a single `docker-compose.yml` for all environments: log rotation
moves into the base file via a shared `x-logging` anchor with env-tunable values
(`LOG_MAX_SIZE`, default `20m`; `LOG_MAX_FILE`, default `5`), the never-applied
resource limits are dropped, and `docker-compose.prod.yml` is deleted along with
the `docker-up-prod`/`docker-down-prod` Make targets.

We will make the deployer idle gracefully when `DEPLOY_WEBHOOK_SECRET` is unset:
it logs a notice, keeps serving `/deploy/healthz`, and returns 503 from the
webhook endpoint instead of exiting.

## Alternatives Considered

### Wire the prod override file into every deploy path
- **Description**: Keep the two-file split; add `-f docker-compose.yml -f docker-compose.prod.yml` to deploy.sh, the sidecar, and the docs.
- **Pros**: Preserves dev/prod separation; resource limits stay expressible.
- **Cons**: Every current and future deploy path must remember both files; the sidecar hardcodes its compose invocation; this exact forgetting already happened once.
- **Why rejected**: For a single-tenant personal VPS, environment differences are already carried by `.env`; a second compose file is one more thing that silently rots.

### Deployer behind a compose profile (opt-in)
- **Description**: Put the deployer under `profiles: [autodeploy]` so it only starts when `COMPOSE_PROFILES` enables it.
- **Pros**: Cleanest semantics — the container doesn't exist unless wanted.
- **Cons**: Requires a migration step on the existing VPS and profile awareness in every compose invocation (including the sidecar's own).
- **Why rejected**: Developer preference for zero VPS changes; an idle container is an acceptable cost.

### Make the webhook secret required
- **Description**: Keep fail-fast, mark auto-deploy as mandatory (matching the `RABBITMQ_PASSWORD:?` pattern).
- **Pros**: Simplest code; no disabled mode to reason about.
- **Cons**: Forces every install to configure a GitHub webhook it may never use, contradicting the documented "optional" contract.
- **Why rejected**: Auto-deploy is genuinely optional; the contract should stay that way.

## Consequences

### Positive
- Log rotation is actually applied — the unbounded-logs disk-fill path is closed.
- One compose file: no deploy path can forget the overrides again; deploy.sh, the sidecar, Makefile, and docs all agree.
- Default installs work out of the box; a blank webhook secret produces a clear 503 + log line instead of a crash loop.

### Negative
- No per-container CPU/memory limits anywhere; if the VPS ever runs contended workloads, limits must be reintroduced (env-interpolated in the base file).
- Local dev containers also carry log rotation (harmless) and json-file driver assumptions.
- An unused deployer container runs on installs that never configure auto-deploy.

### Cross-Repo Impact
- None — single repo. VPS operational note: existing deployments pick the logging config up on the next `docker compose up -d` (containers are recreated).

## Implementation Notes

Implemented in the same PR as this ADR (branch `fix/ops-docs-drift`):
`x-logging` anchor in `docker-compose.yml`, deletion of the prod file and Make
targets, `deploy/main.go` blank-secret gate with table-driven httptest coverage,
and doc updates in `docs/deployment.md`. Rollback is a straight revert; no data
or schema involved. If resource limits return, prefer `${API_MEM_LIMIT:-…}`-style
env interpolation in the base file over resurrecting a second compose file.
