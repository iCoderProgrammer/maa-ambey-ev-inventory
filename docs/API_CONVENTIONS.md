# API Conventions

Applies to every endpoint of the Go API. The OpenAPI spec (`docs/openapi/swagger.yaml`) is the
contract; regenerate it with `make swagger` whenever handlers or annotations change.

## Versioning and paths

- Base path: `/api/v1`. Breaking changes go to `/api/v2`; additive changes stay in v1.
- Plural, kebab-case resources: `/brands`, `/vehicle-units`, `/purchase-orders/{id}`.
- State-changing commands that are not plain updates are sub-resources with POST:
  `/goods-receipts/{id}/post`, `/sales-orders/{id}/finalise`, `/reservations/{id}/cancel`.
- JSON fields are `camelCase`. IDs are UUID strings. Timestamps are RFC 3339 UTC.

## Success envelope

```json
{ "data": { "...": "..." } }
```

Lists (from first list endpoint onward):

```json
{ "data": [ ... ], "meta": { "page": 1, "limit": 25, "total": 120 } }
```

Pagination uses `page`/`limit` with a server-enforced maximum and a stable sort.

## Error envelope

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Check the highlighted fields.",
    "requestId": "7f9d0c1e-5c1a-4c55-9e0f-0b7f1f0f2a11",
    "details": [{ "field": "slug", "message": "must be lowercase letters, digits and hyphens" }]
  }
}
```

- `code` is stable and machine-readable; `message` is safe to show users.
- Never return SQL, driver errors, stack traces, credentials or internal hostnames. Log them with
  the `request_id` instead (`httpx.InternalError` returns the generic 500).

| HTTP | `code` | When |
| --- | --- | --- |
| 400 | `BAD_REQUEST` | Malformed JSON / query |
| 422 | `VALIDATION_FAILED` | Well-formed but invalid input; `details` lists fields |
| 401 | `UNAUTHENTICATED` | No / expired session |
| 403 | `FORBIDDEN` | Authenticated but not permitted (incl. brand/location scope) |
| 404 | `NOT_FOUND` | Unknown route or resource (also used to avoid leaking existence across scopes) |
| 405 | `METHOD_NOT_ALLOWED` | Wrong method for a route |
| 409 | `CONFLICT` | Duplicate (e.g. chassis), stale `version`, state-machine violation |
| 429 | `RATE_LIMITED` | Rate limit hit; `Retry-After` header set |
| 500 | `INTERNAL_ERROR` | Unexpected failure |
| 503 | `SERVICE_UNAVAILABLE` | Dependency down |
| 503 | `REQUEST_TIMEOUT` | Request exceeded `REQUEST_TIMEOUT` and the handler wrote no response (outcome unknown: check before retrying; critical writes use idempotency keys) |

## Request IDs

- Every response carries `X-Request-ID`. A well-formed incoming `X-Request-ID`
  (8–64 chars of `A-Za-z0-9._-`) is reused, otherwise one is generated.
- Error bodies repeat it as `requestId`; the UI shows it as a support reference.

## Validation

- The server is authoritative. Validate every body/query field in the handler/service before
  touching the database; Zod in the UI mirrors but never replaces it.
- Never trust client-supplied totals, statuses, ownership or `brandId`. Brand and location scope
  come from the server-side session and are enforced in the service/repository layer.

## Concurrency and idempotency

- Editable aggregates expose `version`; updates send it (body or `If-Match`) and get `409 CONFLICT`
  when stale.
- Receipt posting, sale finalisation, payment capture/reversal and imports accept an
  `Idempotency-Key` header; replays return the original result.

## Health endpoints

| Endpoint | 200 | Non-200 |
| --- | --- | --- |
| `GET /api/v1/health` | `{"data":{"status":"ok","service","version","time"}}` — process is alive; no dependency checks | — |
| `GET /api/v1/ready` | `{"data":{"status":"ready","checks":{"database":"up","migrations":"up"}}}` | 503 with same shape, `status: "not_ready"`, failed check `"down"` (`migrations` is also `"down"` when the database is unreachable); reasons are only logged |

## Authentication (B-06)

Server-side sessions. The session token lives only in an HttpOnly, `SameSite=Strict`, `Path=/`
browser-session cookie (`__Host-maa_session` with `Secure` over HTTPS; `maa_session` when
`SESSION_COOKIE_SECURE=false` in local development). The database stores only its SHA-256 hash.

**Deny by default:** every `/api/v1` route needs a valid session except `GET /health`, `GET /ready`,
`POST /auth/login` and `GET /auth/session`. `POST`/`PUT`/`PATCH`/`DELETE` also need the session's
CSRF token in `X-CSRF-Token`. Clients keep that token in memory only (never localStorage) and get it
again from `GET /auth/session` after a reload. Permissions per route arrive with B-07.

| Endpoint | Success | Errors |
| --- | --- | --- |
| `POST /api/v1/auth/login` body `{"username","password"}` (`Content-Type: application/json` required) | 200 `{"data":{"user":{"id","username","displayName"},"csrfToken","expiresAt"}}` + `Set-Cookie` | 400 `VALIDATION_FAILED` (details `username`/`password`) or `BAD_REQUEST`; 401 `INVALID_CREDENTIALS` (unknown user and wrong password look the same); 403 `ACCOUNT_DISABLED` (only when the password was right); 415 `UNSUPPORTED_MEDIA_TYPE`; 429 `RATE_LIMITED` + `Retry-After` seconds |
| `GET /api/v1/auth/session` | 200, same `data` as login | 401 `UNAUTHENTICATED` (no cookie) or `SESSION_EXPIRED` (ended; cookie cleared); 503 if it cannot be checked |
| `POST /api/v1/auth/logout` + `X-CSRF-Token` | 200 `{"data":{"signedOut":true}}`, cookie cleared, session revoked | 401 as above; 403 `CSRF_FAILED` |
| any other protected route | — | 401 `UNAUTHENTICATED` / `SESSION_EXPIRED`; 403 `CSRF_FAILED` on mutations; 503 `SERVICE_UNAVAILABLE` if the session store is down (the session is not ended) |

`expiresAt` is when the session ends if unused: the earlier of last activity + `SESSION_IDLE_TIMEOUT`
(default 2h) and sign-in + `SESSION_ABSOLUTE_TIMEOUT` (default 12h). Sessions also end on sign-out,
a new sign-in from the same browser, password change, account disable and admin revoke. Failed
sign-ins are limited per username+IP (`LOGIN_MAX_FAILURES`, default 5) and per IP
(`LOGIN_MAX_FAILURES_PER_IP`, default 20) within `LOGIN_FAILURE_WINDOW` (default 15m).

## Middleware chain

Order (outermost first): request ID → access log → panic recovery → CORS → request timeout;
`/api/v1` adds the per-IP rate limiter, then the session check (deny by default, CSRF on mutations). The access log sits outside recovery, so a recovered panic
still logs one `http_request` line (status 500, `request_id`, route template, latency).

## OpenAPI

`make swagger` regenerates `docs/openapi/` (served at `/swagger/index.html` when `SWAGGER_ENABLED`).
`make swagger-check` (part of `make lint-backend` and CI) fails when the committed spec is stale.

## Handler layout

`internal/modules/<feature>/`: `handler.go` (Gin, request parsing, swag annotations) →
`service.go` (business rules, permissions, transactions) → `repository.go` (pgx SQL). Handlers
never build SQL; repositories never make permission decisions.
