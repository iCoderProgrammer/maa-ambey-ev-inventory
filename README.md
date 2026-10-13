# Maa Ambey Enterprises — Multi-Brand EV Inventory Management System

Internal staff console for an electric two-wheeler showroom: serialised vehicle units, multi-brand
catalogue, procurement, stock ledger, reservations, sales and reports. Multi-brand support is a
P0 requirement: brands are data, never code.

**Status:** foundation only. The Go API, PostgreSQL, migrations, OpenAPI docs, sign-in/sessions and
the React shell work end to end; business modules are on the backlog (see `documentation/` and `prompt-pack/`).

## Stack

| Layer | Tech |
| --- | --- |
| UI (`ui/`) | React 19, TypeScript (strict), Vite, React Router, Tailwind CSS v4, TanStack Query, Redux Toolkit (shell state), React Hook Form + Zod, Vitest |
| API (repo root) | Go 1.26+, Gin, pgx v5, golang-migrate, slog, swaggo OpenAPI |
| Database | PostgreSQL 18 (Docker Compose for local dev) |

## Repository layout

One Git repository; Go at the root, React in `ui/` (same pattern as `saral-sampark`).

```text
.
├── cmd/
│   ├── api/            # API entry point (graceful shutdown)
│   ├── migrate/        # migration runner (up / down / version / force)
│   └── seed/           # idempotent seed runner
├── internal/
│   ├── config/         # env loading + validation
│   ├── database/       # pgx pool
│   ├── httpx/          # response + error envelopes
│   ├── logger/         # slog JSON logger
│   ├── middleware/     # request ID, logging, recovery, timeout, CORS, rate limit
│   ├── server/         # router + http.Server
│   └── modules/
│       └── health/     # /health, /ready
├── migrations/         # versioned SQL (000001_create_brands)
├── seeds/              # idempotent reference data (initial brands only)
├── templates/          # blank opening-stock CSV template (draft)
├── docs/
│   ├── openapi/        # generated OpenAPI (make swagger)
│   ├── REQUIREMENTS_BASELINE.md
│   ├── ARCHITECTURE_DECISIONS.md
│   ├── API_CONVENTIONS.md
│   └── DEVELOPMENT_SETUP.md
├── ui/                 # React app
│   └── src/{app,components/{ui,layout},features,hooks,lib/{api,config,utils},store,styles,types,test}
├── documentation/      # approved PRD, architecture, business rules, design spec, tasks (.docx)
├── prompt-pack/        # per-task Claude Code prompts
├── docker-compose.yml
└── Makefile
```

## Prerequisites

- Go 1.26+
- Node.js 22.22+ and npm
- Docker with the Compose plugin
- `make`

## Quick start

```bash
make setup            # creates .env and ui/.env from examples, installs Go + npm deps
make db-up            # start PostgreSQL, wait until healthy
make db-migrate-up    # apply migrations
make db-seed          # initial brands (idempotent)
make user ARGS="create --username asha.verma --name 'Asha Verma'"   # first account (password prompt)
make dev              # API on :8080 + UI on :5173
```

Open http://localhost:5173 and sign in with that account. There is no built-in admin user; every
page except sign-in needs a session (B-06). The header pill and the dashboard "System status"
card show live results from the Go API's `/health` and `/ready` endpoints.

## Run the backend and frontend

Run from the repository root. PostgreSQL must be running first:

```bash
make db-up            # start PostgreSQL (skip if already running)
make db-migrate-up    # apply any pending migrations (no-op when up to date)
```

Both servers in one terminal (Ctrl+C stops both):

```bash
make dev
```

Or each in its own terminal:

```bash
# Terminal 1 — Go API (port from APP_PORT in .env, default 8080)
make dev-backend      # = go run ./cmd/api

# Terminal 2 — React UI on http://localhost:5173
make dev-frontend     # = cd ui && npm run dev
```

The UI proxies `/api` to `API_PROXY_TARGET` in `ui/.env`; keep it on the same port as `APP_PORT` in
`.env` (e.g. both 8081). Check the API at `http://localhost:<APP_PORT>/api/v1/health`.
Restart `make dev-backend` after pulling backend changes; the UI reloads on its own.

## Environment

Backend + Compose: `.env` (template `.env.example`).

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_ENV` | `development` | `development` / `test` / `staging` / `production` |
| `APP_PORT` | `8080` | API port |
| `DATABASE_URL` | — (required) | `postgres://user:pass@host:port/db?sslmode=...` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` / `POSTGRES_HOST_PORT` | see example | Compose database |
| `DB_MAX_CONNS` / `DB_MIN_CONNS` / `DB_MAX_CONN_LIFETIME` / `DB_MAX_CONN_IDLE_TIME` | 10 / 1 / 1h / 30m | pgx pool |
| `CORS_ALLOWED_ORIGINS` | empty | Comma list of explicit origins (`*` rejected) |
| `TRUSTED_PROXIES` | empty | Proxies allowed to set `X-Forwarded-For` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `REQUEST_TIMEOUT` / `SHUTDOWN_TIMEOUT` | 15s / 20s | |
| `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | 20 / 40 | Per client IP on `/api/v1` |
| `SWAGGER_ENABLED` | true (false in production) | Swagger UI |

UI: `ui/.env` (template `ui/.env.example`).

| Variable | Default | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `/api/v1` | API base URL (public — no secrets) |
| `API_PROXY_TARGET` | `http://localhost:8080` | Vite dev proxy target (not exposed to browser) |

The API validates configuration at startup and lists every problem at once.

## Database

```bash
make db-up / db-down          # start / stop (volume kept)
make db-migrate-up            # apply
make db-migrate-down [STEPS=1]# roll back
make db-migrate-version
make db-migrate-create NAME=create_models
make db-seed                  # apply seeds/*.sql; safe to re-run
make db-psql
```

Seeds add only the three client-confirmed brands (Lectrix EV, Ceeon India, Ujagar Electric) as
`PENDING_VERIFICATION`. No models, prices, warranties, suppliers or stock are seeded.
The API never runs migrations or resets data. See `docs/DEVELOPMENT_SETUP.md`.

## URLs

| What | URL |
| --- | --- |
| UI | http://localhost:5173 |
| Health | http://localhost:8080/api/v1/health |
| Readiness | http://localhost:8080/api/v1/ready |
| Swagger UI | http://localhost:8080/swagger/index.html |
| OpenAPI JSON | http://localhost:8080/swagger/doc.json (file: `docs/openapi/swagger.json`) |

## Quality commands

```bash
make lint             # OpenAPI drift, gofmt, go vet, go mod tidy (+ golangci-lint if installed), ESLint (0 warnings), Prettier
make typecheck        # tsc
make test             # Go unit + Vitest (DB integration tests skip)
make test-integration # Go tests incl. PostgreSQL (needs db-up + migrations); fails, never skips, without a DB
make build            # bin/api, bin/migrate, bin/seed, ui/dist
make ci               # lint + typecheck + test-integration + test-frontend + build (what CI runs)
make swagger          # regenerate OpenAPI (make swagger-check fails if docs/openapi is stale)
```

Every target exits non-zero on failure. CI (`.github/workflows/ci.yml`) runs the same targets on
pushes to `main` and on pull requests: a backend job with a PostgreSQL 18 service (migrate, seed,
integration tests) and a frontend job. Details: `docs/DEVELOPMENT_SETUP.md` → Checks and CI.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `bind: address already in use` | Change `APP_PORT` / `POSTGRES_HOST_PORT` (+ port in `DATABASE_URL`) in `.env` and `API_PROXY_TARGET` in `ui/.env` |
| `invalid configuration: DATABASE_URL is required` | Run `make env-files` or create `.env`; use make targets (they export `.env`) |
| `connect to PostgreSQL ... (is make db-up running?)` | `make db-up`, check `docker compose ps` |
| `/ready` returns 503 | Database down or unreachable; see API logs (`readiness check failed`) |
| UI shows "API offline" | API not running, or `API_PROXY_TARGET` points to the wrong port |
| CORS error in the browser | Using an absolute `VITE_API_BASE_URL`: add the UI origin to `CORS_ALLOWED_ORIGINS`, or use `/api/v1` |
| `Dirty database version` | Fix SQL, repair schema, `go run ./cmd/migrate force <version>` |
| `go version` prints nothing (snap Go in sandboxed shell) | `make GO=/snap/go/current/bin/go <target>` |

## Documentation

- `documentation/` — approved PRD, System Architecture, Business Rules, Design Spec, Tasks
- `docs/REQUIREMENTS_BASELINE.md` — client-confirmed decisions (brands, billing, catalogue, import, roles, delivery checklist) and pending approvals
- `docs/ARCHITECTURE_DECISIONS.md` — stack decisions and open questions
- `docs/API_CONVENTIONS.md` — endpoint, error and versioning conventions
- `docs/DEVELOPMENT_SETUP.md` — workflow, migrations, adding modules
