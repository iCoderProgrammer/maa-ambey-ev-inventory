# ──────────────────────────────────────────────
# Maa Ambey Enterprises – Inventory Management
# Go API (repo root) + React UI (ui/)
# ──────────────────────────────────────────────
-include .env
export

GO      ?= go
NPM     ?= npm
UI_DIR  := ui
API_BIN := bin/api
MIG_BIN := bin/migrate
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
TEST_DATABASE_URL ?= $(DATABASE_URL)

SWAG_GEN_FLAGS := -g main.go -d ./cmd/api,./internal --outputTypes go,json,yaml --parseInternal --quiet
SWAG_FLAGS := $(SWAG_GEN_FLAGS) -o docs/openapi

.PHONY: help setup env-files check-go check-node check-docker \
        dev dev-backend dev-frontend \
        db-up db-down db-logs db-psql db-migrate-up db-migrate-down db-migrate-version db-migrate-create db-seed \
        swagger swagger-check test test-backend test-integration test-frontend \
        lint lint-backend lint-frontend fmt typecheck typecheck-frontend build build-backend build-frontend ci clean

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ── Prerequisites ──────────────────────────

check-go:
	@command -v $(GO) >/dev/null 2>&1 || { echo "✗ Go not found. Install Go 1.26+ from https://go.dev/dl/ (or set GO=/path/to/go)."; exit 1; }

check-node:
	@command -v $(NPM) >/dev/null 2>&1 || { echo "✗ npm not found. Install Node.js 22+ from https://nodejs.org/."; exit 1; }

check-docker:
	@command -v docker >/dev/null 2>&1 || { echo "✗ Docker not found. Install Docker Desktop / Docker Engine with the compose plugin."; exit 1; }

## ── Setup ──────────────────────────────────

env-files: ## Create .env and ui/.env from the examples if missing
	@test -f .env || { cp .env.example .env && echo "▸ created .env from .env.example"; }
	@test -f $(UI_DIR)/.env || { cp $(UI_DIR)/.env.example $(UI_DIR)/.env && echo "▸ created ui/.env from ui/.env.example"; }

setup: check-go check-node env-files ## Install Go modules and UI packages
	$(GO) mod download
	cd $(UI_DIR) && $(NPM) ci
	@echo "✅ Setup complete. Next: make db-up && make db-migrate-up && make db-seed && make dev"

## ── Run ────────────────────────────────────

dev: ## Run API and UI together (Ctrl+C stops both)
	@$(MAKE) -j2 --no-print-directory dev-backend dev-frontend

dev-backend: check-go ## Run the Go API (needs .env and a running database)
	$(GO) run -ldflags "-X main.version=$(VERSION)" ./cmd/api

dev-frontend: check-node ## Run the Vite dev server
	cd $(UI_DIR) && $(NPM) run dev

## ── Database ───────────────────────────────

db-up: check-docker ## Start PostgreSQL and wait until healthy
	docker compose up -d --wait postgres

db-down: check-docker ## Stop PostgreSQL (data volume is kept)
	docker compose down

db-logs: check-docker ## Tail PostgreSQL logs
	docker compose logs -f postgres

db-psql: check-docker ## Open psql inside the container
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

db-migrate-up: check-go ## Apply all pending migrations
	$(GO) run ./cmd/migrate up

db-migrate-down: check-go ## Roll back migrations: make db-migrate-down [STEPS=1]
	$(GO) run ./cmd/migrate down $(or $(STEPS),1)

db-migrate-version: check-go ## Show the current migration version
	$(GO) run ./cmd/migrate version

db-seed: check-go ## Apply idempotent seeds from seeds/ (safe to re-run; run migrations first)
	$(GO) run ./cmd/seed

db-migrate-create: ## Create a migration pair: make db-migrate-create NAME=create_models
	@test -n "$(NAME)" || { echo "✗ usage: make db-migrate-create NAME=snake_case_name"; exit 1; }
	@last=$$(ls migrations/*.up.sql 2>/dev/null | sed -E 's|.*/([0-9]+)_.*|\1|' | sort -n | tail -1); \
	next=$$(printf "%06d" $$((10#$${last:-0} + 1))); \
	touch migrations/$${next}_$(NAME).up.sql migrations/$${next}_$(NAME).down.sql; \
	echo "▸ created migrations/$${next}_$(NAME).{up,down}.sql"

## ── API docs ───────────────────────────────

swagger: check-go ## Regenerate OpenAPI docs into docs/openapi
	$(GO) tool swag init $(SWAG_FLAGS)

swagger-check: check-go ## Fail if docs/openapi is out of date with the handler annotations
	@tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	$(GO) tool swag init $(SWAG_GEN_FLAGS) -o "$$tmp/openapi" && \
	diff -r "$$tmp/openapi" docs/openapi >/dev/null || { echo "✗ docs/openapi is stale: run 'make swagger' and commit the result"; exit 1; }

## ── Quality ────────────────────────────────

fmt: check-go ## Format Go and UI code
	gofmt -w cmd internal
	cd $(UI_DIR) && $(NPM) run format

test: test-backend test-frontend ## Run all unit tests

test-backend: check-go ## Go unit tests (DB integration tests skip without TEST_DATABASE_URL)
	TEST_DATABASE_URL= $(GO) test ./...

test-integration: check-go ## Go tests incl. PostgreSQL integration (needs db-up + db-migrate-up); fails instead of skipping without a DB URL
	@test -n "$(TEST_DATABASE_URL)" || { echo "✗ TEST_DATABASE_URL (or DATABASE_URL) is empty; integration tests cannot run."; exit 1; }
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" REQUIRE_INTEGRATION_TESTS=1 $(GO) test -count=1 ./...

test-frontend: check-node ## UI tests (Vitest)
	cd $(UI_DIR) && $(NPM) test

lint: lint-backend lint-frontend ## Lint Go and UI

lint-backend: check-go swagger-check ## gofmt check, go vet, go.mod/go.sum tidy check, OpenAPI drift check, golangci-lint if installed
	@unformatted=$$(gofmt -l cmd internal); if [ -n "$$unformatted" ]; then echo "✗ gofmt needed:"; echo "$$unformatted"; exit 1; fi
	$(GO) vet ./...
	@$(GO) mod tidy -diff || { echo "✗ go.mod/go.sum not tidy: run '$(GO) mod tidy'"; exit 1; }
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "ℹ golangci-lint not installed; skipped (go vet ran)"; fi

lint-frontend: check-node ## ESLint + Prettier check
	cd $(UI_DIR) && $(NPM) run lint && $(NPM) run format:check

typecheck: typecheck-frontend ## Type check (Go is type-checked by go vet/build in lint-backend/build-backend)

typecheck-frontend: check-node ## TypeScript type check (tsc -b --noEmit)
	cd $(UI_DIR) && $(NPM) run typecheck

build: build-backend build-frontend ## Build API binaries and UI bundle

build-backend: check-go ## Build bin/api, bin/migrate and bin/seed
	@mkdir -p bin
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(API_BIN) ./cmd/api
	$(GO) build -o $(MIG_BIN) ./cmd/migrate
	$(GO) build -o bin/seed ./cmd/seed

build-frontend: check-node ## Production UI build into ui/dist
	cd $(UI_DIR) && $(NPM) run build

## ── CI ─────────────────────────────────────

ci: lint typecheck test-integration test-frontend build ## Everything CI runs (needs a migrated DB at TEST_DATABASE_URL)

clean: ## Remove build outputs
	rm -rf bin $(UI_DIR)/dist
