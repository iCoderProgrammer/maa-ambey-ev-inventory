/**
 * CSRF token for cookie-authenticated mutations (header `X-CSRF-Token`).
 *
 * Kept in memory only: it is returned by POST /auth/login and GET /auth/session,
 * so a page reload simply fetches it again. Never written to localStorage or
 * sessionStorage. On its own it grants nothing; the session itself lives in an
 * HttpOnly cookie the browser script cannot read.
 */
let csrfToken: string | null = null

export const CSRF_HEADER = 'X-CSRF-Token'

export function setCsrfToken(token: string | null): void {
  csrfToken = token
}

export function getCsrfToken(): string | null {
  return csrfToken
}
