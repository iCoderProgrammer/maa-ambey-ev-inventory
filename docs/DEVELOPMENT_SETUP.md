# Development Setup

## Daily workflow

```bash
make db-up            # PostgreSQL in Docker (data persists in the postgres-data volume)
make db-migrate-up    # apply pending migrations (never automatic)
make db-seed          # idempotent reference data (initial brands)
make dev              # Go API + Vite UI together; Ctrl+C stops both
```

- UI: http://localhost:5173 — calls the API through the Vite proxy (`/api` → `API_PROXY_TARGET`).
  The browser never connects to PostgreSQL; the UI needs no database settings, only the API location
  (`ui/.env`: `API_PROXY_TARGET` for dev, `VITE_API_BASE_URL` for builds). If you change `APP_PORT`,
  set `API_PROXY_TARGET` to match.
- API: http://localhost:8080/api/v1/health
- Swagger UI: http://localhost:8080/swagger/index.html (also via http://localhost:5173/swagger/index.html)

Before pushing: `make ci` (needs the database running and migrated; see Checks and CI).

## Environment files

| File | Read by | Notes |
| --- | --- | --- |
| `.env` (from `.env.example`) | Makefile → Go API, `cmd/migrate`, Docker Compose | Never committed |
| `ui/.env` (from `ui/.env.example`) | Vite | Only `VITE_*` reaches the browser — never put secrets there |

The Makefile loads `.env` and exports it. If you run the binary directly, export it first:
`set -a; . ./.env; set +a; go run ./cmd/api`.

Ports already in use? Change `APP_PORT`, `POSTGRES_HOST_PORT` and the port inside `DATABASE_URL`
in `.env`, and `API_PROXY_TARGET` in `ui/.env` (and `CORS_ALLOWED_ORIGINS` if you changed the UI port).

Second checkout or worktree on the same machine? `docker-compose.yml` names the project
`maa-ambey-inventory`, so both copies would share one container and volume. Give the second copy its
own `COMPOSE_PROJECT_NAME=<unique-name>` in `.env` plus different ports.

### Secrets policy

- `.env`, `.env.*`, `ui/.env`, `ui/.env.*`, key/certificate files are git-ignored; only the two
  `*.env.example` files are committed and they hold placeholders (`change_me`) only.
- Every `VITE_*` value is compiled into public JavaScript. The UI needs only `VITE_API_BASE_URL`;
  never add a secret with a `VITE_` prefix. `API_PROXY_TARGET` has no prefix and stays in the dev server.
- The UI reads `import.meta.env` in one place, `ui/src/lib/config/env.ts`; ESLint rejects it elsewhere.
- The API never logs or echoes `DATABASE_URL` (config errors omit the value).

### Configuration per environment

Behaviour below is what the code enforces today (`internal/config`, `internal/server`).

| Setting | Local (`development`) | Staging (`staging`) | Production (`production`) |
| --- | --- | --- | --- |
| Source of values | `.env` copied from `.env.example` | Platform secret manager / env vars | Platform secret manager / env vars |
| Gin mode | debug | release | release |
| `DATABASE_URL` sslmode | `disable` OK (Compose DB) | `require` recommended | `disable` **rejected at startup** |
| `SWAGGER_ENABLED` default | `true` | `true` | `false` (opt in explicitly) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` or empty with the Vite proxy | Real UI origin(s); `*` rejected | Real UI origin(s); `*` rejected |
| `TRUSTED_PROXIES` | empty | Load balancer IP/CIDR | Load balancer IP/CIDR |
| `LOG_LEVEL` | `debug` | `info` | `info` (or `warn`) |
| `VITE_API_BASE_URL` (build time) | `/api/v1` via Vite proxy | `/api/v1` behind same origin, or absolute API URL | Same as staging |
| Database | Docker Compose `postgres` | Managed PostgreSQL | Managed PostgreSQL with backups (H-06) |
| Migrations | `make db-migrate-up` by hand | Explicit release step | Explicit release step after verified backup |

The hosting provider, API host, database region and backup RPO/RTO are still open
(ARCHITECTURE_DECISIONS OQ-5), so no deployment manifests exist yet.

### Staging / production

- Set `APP_ENV=staging|production`. Outside `development` Gin runs in release mode. In production
  Swagger is off unless `SWAGGER_ENABLED=true`, and `DATABASE_URL` must not use `sslmode=disable`.
- Inject secrets from the deployment platform's secret manager, never from files in the repo.
- Set `CORS_ALLOWED_ORIGINS` to the real UI origin(s) and `TRUSTED_PROXIES` to the load balancer.
- Build the UI with `VITE_API_BASE_URL` pointing at the API (or serve both behind one origin and
  keep `/api/v1`).
- Run migrations as a separate, explicit release step after a verified backup.

## Database migrations

```bash
make db-migrate-create NAME=create_models   # new empty up/down pair
make db-migrate-up                          # apply all pending
make db-migrate-version                     # show current version / dirty flag
make db-migrate-down                        # roll back ONE migration
make db-migrate-down STEPS=2                # roll back two
make db-psql                                # inspect with psql
```

Policy (enforced in code, ADR-003):
- Files are `NNNNNN_snake_case.up.sql` + `.down.sql`, starting at `000001`, no gaps, one name per
  version. They are embedded into the binaries (`migrations/migrations.go`), so `go test ./migrations`
  and every build fail on a missing `down`, a gap or a bad name.
- Every `up` has a working `down`. `make test-integration` runs `up → down → up` (plus one-step
  down/up) in a throwaway database it creates and drops, never in your dev database.
- Never edit a migration that has run anywhere shared; add a new one.
- The API never migrates, resets or drops anything. If the schema is not exactly at the version the
  build expects (unmigrated, pending, newer, or dirty), the API logs a warning and
  `GET /api/v1/ready` answers 503 with `"migrations": "down"`; the UI header shows
  "Schema not ready". Run `make db-migrate-up`; readiness recovers without a restart.
- `down` and `force` with `APP_ENV=staging|production` are refused unless `--confirm` is given
  (`make db-migrate-down CONFIRM=1`), after a verified backup. There is no "drop everything" command.
- If a migration fails midway the version is marked dirty: fix the SQL, repair the DB manually,
  then `go run ./cmd/migrate force <last good version>`.

Bad configuration fails before anything runs, with the reason and never the password: missing
`DATABASE_URL`, non-postgres scheme, no host, no database name, `sslmode=disable` in production,
unreachable server ("is `make db-up` running?") or wrong credentials.

## Seeds

- `make db-seed` runs every embedded `seeds/*.sql` in name order inside one transaction (`cmd/seed`).
  It refuses to run until the schema is current, and an empty seed set is an error.
- Every seed file **must be idempotent** (`INSERT ... ON CONFLICT DO NOTHING`), so re-running is
  always safe. `make test-integration` checks the real seed files by running them three times.
- Seed only client-confirmed reference data. Today that is the three brand identities. Catalogue,
  prices, suppliers and stock come from verified sources via C-05 / G-06 / G-07, never from seeds
  with invented values.

## Users and sign-in

There is no built-in admin account. Create the first user from the command line (needs the
database migrated):

```bash
make user ARGS="create --username asha.verma --name 'Asha Verma'"   # prompts for the password twice
make user ARGS="list"
make user ARGS="disable --username asha.verma"          # also ends all their sessions
make user ARGS="enable --username asha.verma"
make user ARGS="set-password --username asha.verma"     # also ends all their sessions
make user ARGS="revoke-sessions --username asha.verma"
```

- Passwords: 12–128 characters, not the username. Typed at a hidden prompt; when stdin is not a
  terminal the first stdin line is used (scripts). Never pass a password as a flag.
- Then sign in at http://localhost:5173/login. Every other page needs a session; the API answers
  401 for any `/api/v1` route except health, readiness, login and session.
- Local development uses `SESSION_COOKIE_SECURE=false` (plain HTTP). Behind HTTPS leave it `true`.
- Use made-up names for local test accounts, never real staff passwords.

## Tests

| Command | What runs |
| --- | --- |
| `make test-backend` | Go unit tests; DB integration tests are skipped (visible with `go test -v`) |
| `make test-integration` | Go tests incl. PostgreSQL integration (`TEST_DATABASE_URL`, defaults to `DATABASE_URL`). Writes run inside rolled-back transactions. Requires `db-up` + `db-migrate-up`. Exits 1 if the URL is empty and sets `REQUIRE_INTEGRATION_TESTS=1`, so a test that cannot reach its DB config fails instead of skipping. |
| `make test-frontend` | Vitest + Testing Library (jsdom); a run with no test files fails |

New DB integration tests must get their connection through the `testConfig` helper pattern
(skip normally, `t.Fatal` when `REQUIRE_INTEGRATION_TESTS=1`).

## Roles, permissions and locations

Nobody has any permission until the owner approves the permission matrix (RB-5). Grants, role
assignments and location scope are data, changed only with the audited `cmd/access` tool:

```bash
make access ARGS="roles"                     # roles, grants and matrix status
make access ARGS="permissions"               # permission catalogue
make access ARGS="assign --username asha.verma --role viewer"
make access ARGS="grant --role viewer --permission access.roles.view"            # PROPOSED
make access ARGS="grant --role owner --permission reports.export --approved-by 'Name' --approval-ref 'Signed matrix 2026-10-20'"  # APPROVED
make access ARGS="location-create --code SR1 --name 'Showroom name'"
make access ARGS="scope-add --username asha.verma --location SR1"
make access ARGS="show --username asha.verma"
```

- Record an **APPROVED** grant only with the owner's real sign-off. To try features locally before
  that, record **PROPOSED** grants and set `RBAC_ALLOW_PROPOSED_GRANTS=true` in `.env` (`APP_ENV=development` only — test, staging and production refuse to start with it;
  the UI then shows a "Development mode" notice). Restart the API after changing it.
- Role and grant changes apply to signed-in users on their next request.

## End-to-end browser tests (Playwright)

`ui/e2e/*.e2e.ts` drive the real stack in headless Google Chrome (installed Chrome, no browser
download), on a desktop and a phone-sized viewport. They do not start servers or create users:

```bash
make db-up && make db-migrate-up && make db-seed && make dev        # or any running stack
make user ARGS="create --username e2e.active --name 'E2E Active'"
make user ARGS="create --username e2e.disabled --name 'E2E Disabled'"
make user ARGS="disable --username e2e.disabled"
# optional, for the roles-page test: a user holding access.roles.view (needs RBAC_ALLOW_PROPOSED_GRANTS=true)
make user ARGS="create --username e2e.roles --name 'E2E Roles'"
make access ARGS="assign --username e2e.roles --role viewer"
make access ARGS="grant --role viewer --permission access.roles.view"
# optional, for the attachment tests (B-09): an uploader+viewer and a second viewer (same flag)
make user ARGS="create --username e2e.attach --name 'E2E Attach'"
make user ARGS="create --username e2e.attachviewer --name 'E2E Attach Viewer'"
make access ARGS="assign --username e2e.attach --role inventory_manager"
make access ARGS="assign --username e2e.attachviewer --role sales_executive"
make access ARGS="grant --role inventory_manager --permission attachments.upload"
make access ARGS="grant --role inventory_manager --permission attachments.view"
make access ARGS="grant --role sales_executive --permission attachments.view"
# optional, for the catalogue API tests (C-02): every catalogue key, and view only (same flag)
make user ARGS="create --username e2e.catalogue --name 'E2E Catalogue'"
make user ARGS="create --username e2e.catviewer --name 'E2E Catalogue Viewer'"
make access ARGS="assign --username e2e.catalogue --role owner"
make access ARGS="assign --username e2e.catviewer --role accountant"
for p in catalogue.view catalogue.manage catalogue.verify catalogue.activate catalogue.archive \
         catalogue.brand.authorise catalogue.brand.activate; do
  make access ARGS="grant --role owner --permission $p"; done
make access ARGS="grant --role accountant --permission catalogue.view"
# optional, for the price tests (C-04): the catalogue user drafts, a different user approves
for p in catalogue.price.view catalogue.price.manage catalogue.cost_price.view catalogue.cost_price.edit; do
  make access ARGS="grant --role owner --permission $p"; done
make user ARGS="create --username e2e.priceapprover --name 'E2E Price Approver'"
make access ARGS="assign --username e2e.priceapprover --role sales_executive"
for p in catalogue.view catalogue.price.view catalogue.price.approve; do
  make access ARGS="grant --role sales_executive --permission $p"; done
# optional, for the supplier tests (D-01)
for p in suppliers.view suppliers.manage suppliers.verify suppliers.archive attachments.upload attachments.view; do
  make access ARGS="grant --role owner --permission $p"; done
make access ARGS="grant --role accountant --permission suppliers.view"
cd ui && E2E_BASE_URL=http://localhost:5173 \
  E2E_USERNAME=e2e.active E2E_PASSWORD=... \
  E2E_DISABLED_USERNAME=e2e.disabled E2E_DISABLED_PASSWORD=... \
  E2E_ROLES_USERNAME=e2e.roles E2E_ROLES_PASSWORD=... \
  E2E_ATTACH_USERNAME=e2e.attach E2E_ATTACH_PASSWORD=... \
  E2E_ATTACH_VIEWER_USERNAME=e2e.attachviewer E2E_ATTACH_VIEWER_PASSWORD=... \
  E2E_CATALOGUE_USERNAME=e2e.catalogue E2E_CATALOGUE_PASSWORD=... \
  E2E_CATALOGUE_VIEWER_USERNAME=e2e.catviewer E2E_CATALOGUE_VIEWER_PASSWORD=... \
  E2E_PRICE_APPROVER_USERNAME=e2e.priceapprover E2E_PRICE_APPROVER_PASSWORD=... npm run test:e2e
```

- Use a non-production database and throwaway accounts; never put the passwords in files in the repo.
- The suite makes failed sign-ins on purpose (including one per viewport for the disabled account),
  and the login throttle is in memory: restart the API before re-running within 15 minutes, and
  start it with a higher `LOGIN_MAX_FAILURES_PER_IP` (e.g. 1000) since every request comes from one IP.
- The catalogue API test (C-02) sends dozens of requests in a few seconds from one IP: start the API with
  a higher `RATE_LIMIT_BURST` (e.g. `RATE_LIMIT_RPS=100 RATE_LIMIT_BURST=200`) or it can get `429 RATE_LIMITED`.
  It creates placeholder brands/models: throwaway database only. `e2e/catalogue-ui.e2e.ts` (C-03) drives the
  catalogue screens with the same two accounts (plus `e2e.active` for the no-permission case) on desktop and mobile;
  `e2e/prices.e2e.ts` (C-04) also needs `E2E_PRICE_APPROVER_*`; `e2e/isolation.e2e.ts` (C-06) uses `E2E_CATALOGUE_*`.
  `e2e/suppliers.e2e.ts` (D-01) uses `E2E_CATALOGUE_*` with the supplier grants above (throwaway database only).
- The expired-link test waits for the link to expire: start the API with `ATTACHMENT_LINK_TTL=30s`.
  Only do the attachment grants on a throwaway database: they are PROPOSED and count only with
  `RBAC_ALLOW_PROPOSED_GRANTS=true` in development.
- Failure screenshots and traces go to `ui/test-results/` (git-ignored).
- Not part of `make ci` yet: CI has no running UI/API stack.

## Checks and CI

| Target | Commands | Fails when |
| --- | --- | --- |
| `make lint-backend` | `make swagger-check` (regenerate to a temp dir and diff), `gofmt -l cmd internal`, `go vet ./...`, `go mod tidy -diff`, `golangci-lint run` (only if installed) | stale `docs/openapi`, unformatted Go, vet finding, untidy `go.mod`/`go.sum`, lint finding |
| `make lint-frontend` | `eslint . --max-warnings=0`, `prettier --check .` | any ESLint error **or warning**, unformatted file |
| `make typecheck-frontend` (`make typecheck`) | `tsc -b --noEmit` | type error |
| `make test-integration` | `go test -count=1 ./...` with DB | failing test, empty DB URL |
| `make test-frontend` | `vitest run` | failing test |
| `make build-backend` / `make build-frontend` | `go build` ×3 / `tsc -b && vite build` | compile/bundle error |
| `make ci` | all of the above (`lint typecheck test-integration test-frontend build`) | any step fails (stops at the first failure) |

GitHub Actions (`.github/workflows/ci.yml`, on push to `main` and every pull request):

- **backend:** Go from `go.mod`; PostgreSQL 18 service with throwaway CI-only credentials;
  `make lint-backend` → `make db-migrate-up` → `make db-seed` → `make test-integration` → `make build-backend`.
- **frontend:** Node 24, `npm ci`; `make lint-frontend` → `make typecheck-frontend` → `make test-frontend` → `make build-frontend`.

Not in CI yet: `golangci-lint` (optional locally, no pinned config),
browser/E2E tests (Playwright, added with the first UI task that needs them).

## Adding a backend module

1. Create `internal/modules/<name>/` with `handler.go`, `service.go`, `repository.go` and tests.
2. Use pgx with parameterised SQL; wrap multi-row stock changes in one transaction.
3. Register routes in `internal/server/server.go` on the `/api/v1` group.
4. Add swag annotations; run `make swagger` and commit `docs/openapi/*`.
5. Add any schema via `make db-migrate-create`.
6. Brand-aware queries must filter by the caller's permitted brands (server-side), never only by a
   client-supplied `brandId`.
- Sensitive create/update/approve/cancel/export actions record an audit event with `audit.Record` in the
  same transaction (see API_CONVENTIONS "Audit events").

## Adding a frontend feature

1. Create `ui/src/features/<name>/` with `api/` (functions using `apiRequest` + Zod schemas),
   `hooks/` (TanStack Query hooks and query keys), `components/` and the page component.
2. Never call `fetch` directly — ESLint blocks it outside `src/lib/api/client.ts`.
3. Reuse `src/components/ui` (Button, Field, Input, Select, Table, Badge, Dialog, Card, state views).
4. Replace the route's `PlannedModulePage` in `src/app/routes.tsx` and clear `backlog` for the item
   in `src/app/navigation.ts`.
5. Handle loading, empty, error (with request ID), 401/403, 409 conflict and offline states.
6. Keep brands, statuses and permissions API-driven; do not hardcode them.

## Toolchain notes

- Go 1.26+ (`go.mod`). `go.mod` has `ignore ./ui`, so `./...` never includes the UI or `node_modules`.
- `swag` is pinned as a Go tool (`go tool swag`), no global install needed.
- Node 22.22+ (React Router 8 requirement).
- `golangci-lint` is optional; `make lint` uses it when installed, otherwise `gofmt` + `go vet`.
- Snap-installed Go inside restricted shells: if `go version` prints nothing, run
  `make GO=/snap/go/current/bin/go <target>`.
