.PHONY: build run test test-verbose test-integration test-fullstack lint migrate rebuild-projections worker-reminder docker-up docker-down docker-up-prod docker-down-prod backup-db fmt vet help web-install web-dev web-build web-test web-lint docker-logs generate

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Go (api/)
# ---------------------------------------------------------------------------

build: ## Build the API binary
	cd api && go build -o bin/api ./cmd/api

run: ## Run the API locally
	cd api && go run ./cmd/api

test: ## Run all Go tests
	cd api && go test ./... -race -count=1

test-verbose: ## Run all Go tests with verbose output
	cd api && go test ./... -race -count=1 -v

test-integration: ## Run integration tests (requires running Postgres)
	cd api && go test -tags integration ./... -race -count=1 -v

test-fullstack: ## Run full-stack integration tests (requires Postgres + RabbitMQ)
	cd api && go test -tags integration ./internal/integration/ -race -count=1 -v

lint: ## Run golangci-lint
	cd api && golangci-lint run ./...

fmt: ## Format Go source files
	cd api && gofmt -s -w .

vet: ## Run go vet
	cd api && go vet ./...

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

migrate: ## Run database migrations (up)
	cd api && go run ./cmd/migrate up

rebuild-projections: ## Rebuild all read-model projections from the event store
	cd api && go run ./cmd/rebuild

worker-reminder: ## Run the due-date reminder worker
	cd api && go run ./cmd/worker-reminder

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

docker-up: ## Start all services (development)
	docker compose up -d --build

docker-down: ## Stop all services (development)
	docker compose down

docker-up-prod: ## Start all services (production with overrides)
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

docker-down-prod: ## Stop all services (production)
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

docker-logs: ## Tail logs from all containers
	docker compose logs -f

# ---------------------------------------------------------------------------
# Backup
# ---------------------------------------------------------------------------

backup-db: ## Run a manual database backup
	./scripts/backup.sh

# ---------------------------------------------------------------------------
# Frontend (web/)
# ---------------------------------------------------------------------------

web-install: ## Install frontend dependencies
	cd web && npm ci

web-dev: ## Start frontend dev server
	cd web && npm run dev

web-build: ## Build frontend for production
	cd web && npm run build

web-test: ## Run frontend tests
	cd web && npm test

web-lint: ## Run frontend linter
	cd web && npm run lint

# ---------------------------------------------------------------------------
# Code Generation
# ---------------------------------------------------------------------------

generate: ## Regenerate Go + TypeScript types from OpenAPI spec
	cd api && ~/go/bin/oapi-codegen --config oapi-codegen.yaml openapi.yaml
	cd web && npx openapi-typescript ../api/openapi.yaml -o src/api/types.gen.ts
