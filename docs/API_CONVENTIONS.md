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

## Authorisation (B-07)

Deny by default, in two layers: the session middleware (B-06) rejects anonymous requests, then
every feature route declares the permission it needs:

```go
api.GET("/stock/valuation", accessH.Require(access.PermCostPriceView), h.Valuation)
```

- `Require` answers **403 `FORBIDDEN`** ("You do not have permission to do this.") without naming
  the permission; it panics at start-up for a key that is not in the catalogue.
- Object/location checks belong in the handler or service: `a, ok := accessH.Current(c)`, then
  `a.Has(...)`, `a.Scope.Check(locationID)` (cross-location → 403) or `a.Scope.Restrict(clientIDs)`
  for list filters (a location outside scope is an error, never silently dropped). Never trust a
  client-supplied location, role or permission.
- Permissions come only from **APPROVED** grants (approver + reference recorded) on the user's roles.
  `PROPOSED` grants count only with `RBAC_ALLOW_PROPOSED_GRANTS=true`, accepted only with `APP_ENV=development`.
- There is no HTTP API to change grants, roles or scope; `cmd/access` does it and audits every change.

| Endpoint | Success | Errors |
| --- | --- | --- |
| `GET /api/v1/me` | `{"data":{"user":{id,username,displayName},"roles":[{key,name}],"permissions":["…"],"locationScope":{"all":false,"locations":[{id,code,name}]},"proposedGrantsActive":false}}` | 401; 503 if access cannot be loaded |
| `GET /api/v1/access/roles` (needs `access.roles.view`) | `{"data":{"matrixStatus":"PENDING_OWNER_SIGN_OFF"\|"PARTIALLY_APPROVED","approvedGrantCount":0,"roles":[{key,name,description,grants:[{permission,status,approvedBy,approvalReference,approvedAt}]}],"permissions":[{key,description,sensitive}]}}` | 401; 403 `FORBIDDEN`; 503 |

The UI renders navigation and actions from `/me` (unknown = not granted) and shows "Access denied" for
403 `FORBIDDEN`; hiding a link is never the security boundary.

## Audit events (B-08)

Sensitive changes are recorded in `audit_events` through `internal/audit`:

- Call `audit.Record(ctx, tx, audit.Event{...})` with the **same transaction** as the change, so the
  event exists exactly when the change does. Use `audit.UserActor(principal.UserID, principal.Username)`
  for signed-in requests, `audit.CLIActor(...)` for commands and `audit.AnonymousActor(...)` for
  unauthenticated attempts (e.g. a failed sign-in: the targeted account is the entity, not the actor);
  pass the request ID (`c.GetString(httpx.RequestIDKey)`).
- `Action` is a dotted past-tense key (`user.disabled`, `auth.signed_in`); `Outcome` is `SUCCESS`
  (default), `DENIED` or `FAILURE`; `EntityType`/`EntityID` name the record; `Reference` holds a
  business document number when there is one.
- `Summary` is a small JSON object of plain values (≤ 8 KB, strings ≤ 500 characters); use
  `audit.Change{From, To}` for before/after. Keys containing password, token, hash, secret, csrf,
  cookie, otp, credential … and password-hash-looking values are **rejected** (the whole write fails),
  so a mistake is caught in tests instead of leaking. Reference people by ID, not phone/address/ID numbers.
- The table is append-only at the database level: `UPDATE`, `DELETE` and `TRUNCATE` raise SQLSTATE
  `42501`. Corrections are new events, never edits. No API route may change audit history
  (`TestNoAPIRouteCanModifyAuditEvents`); a future read API may only use GET.
- Currently audited: `auth.signed_in`, `auth.signed_out` (actor = the user), `auth.sign_in_failed`
  (existing accounts only — typed unknown usernames are not stored) and `auth.sign_in_refused` (disabled
  account), both with an `ANONYMOUS` actor, and
  `cmd/user` actions `user.created`, `user.enabled`, `user.disabled`, `user.password_changed`,
  `user.sessions_revoked`. Also audited (B-07): `access.grant_proposed`, `access.grant_approved`, `access.grant_revoked`,
  `access.role_assigned`, `access.role_unassigned`, `location.created`, `access.location_scope_added`,
  `access.location_scope_removed`. Also audited (B-09): `attachment.uploaded`, `attachment.downloaded`. No read API yet (`audit.events.view` exists, not granted).

## Attachments (B-09)

Private files (PDI photos, supplier bills). Never a public URL; see ADR-010.

| Route | Permission | Notes |
| --- | --- | --- |
| `GET /attachments/limits` | `attachments.upload` | `{maxBytes, allowedTypes:[{contentType, extensions, previewable}]}` — use it for client checks |
| `POST /attachments` | `attachments.upload` + CSRF | `multipart/form-data`, one part `file`; optional `?locationId=` (must be in scope). `201` → `Attachment` |
| `GET /attachments` | `attachments.view` | caller's newest 50 unlinked uploads still in scope |
| `GET /attachments/{id}` | `attachments.view` | metadata; not visible → `404 NOT_FOUND` |
| `GET /attachments/{id}/link?disposition=attachment\|inline` | `attachments.view` | `{url, expiresAt, disposition}`; `inline` only for images (`422` otherwise) |
| `GET /attachments/{id}/content?disposition&expires&signature` | `attachments.view` + same user's session | the bytes; audited |

`Attachment`: `id, fileName, contentType, sizeBytes, sha256, locationId|null, linked, uploadedBy, createdAt,
previewable`. Storage keys/paths are never returned.

Errors: `413 FILE_TOO_LARGE`; `415 UNSUPPORTED_MEDIA_TYPE` (type not allowed, or content ≠ declared type /
extension; also a non-multipart body); `400 VALIDATION_FAILED` (no `file` part); `422 VALIDATION_FAILED`
(empty file, unknown/malformed `locationId`, bad disposition); `403 FORBIDDEN` (permission, out-of-scope
location); `403 DOWNLOAD_LINK_INVALID` (tampered, other user's, wrong disposition or attachment);
`403 DOWNLOAD_LINK_EXPIRED`; `404 NOT_FOUND`. Clients: fetch the link, then the content with
`credentials: 'include'`, and show the bytes from memory (object URL) — never store or display the link.
Attachments are immutable: there is no update or delete route.

## Catalogue contracts (C-01; endpoints in C-02)

The schema exists (migration 000007, ADR-011); the API does not yet. Expected JSON shapes, mirrored by the
Zod contracts in `ui/src/features/catalogue/api/contracts.ts` (C-02 must match or update both):

- `Brand`: `id, name, slug, lifecycleStatus, authorisationStatus, manufacturerLegalName, authorisationReference,
  authorisationEvidenceId, verifiedBy, verifiedAt, version, createdAt, updatedAt`
- Provenance block on `VehicleModel`, `Variant`, `VariantColour`, `VariantSpec`: `confidence, sourceUrl,
  sourceDocumentId, sourceDate, verifiedBy, verifiedAt`
- `VehicleModel`: `id, brandId, name, internalCode, manufacturerCode, vehicleType, lifecycleStatus, …provenance, version, createdAt, updatedAt`
- `Variant`: `id, brandId, modelId, name, variantCode, manufacturerCode, lifecycleStatus, …provenance, version, …`
- `VariantColour`: `id, brandId, variantId, name, colourCode, lifecycleStatus, …provenance, version, …`
- `VariantSpec`: `id, brandId, variantId, specKey, valueText | valueNumeric (exactly one), unit, effectiveFrom, …provenance, createdAt, updatedAt`

Value sets come from DB CHECKs: lifecycle `PENDING_VERIFICATION|ACTIVE|ARCHIVED`; authorisation
`UNVERIFIED|AUTHORISED|NOT_AUTHORISED|REVOKED` (OQ-1); confidence `UNVERIFIED|SUPPLIER_CONFIRMED|CLIENT_APPROVED`.
Brands, models, variants and spec keys are data and never appear in code. `brandId` from a client is never
trusted on its own (ADR-006). Database errors to map in C-02: `23505` duplicate code/name → `409 CONFLICT`,
`23503`/`23514` invalid reference or value → `422 VALIDATION_FAILED`, `23001` delete of a used item and `42501`
edit of a verified spec → `409 CONFLICT` (archive / add a new version instead).

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
