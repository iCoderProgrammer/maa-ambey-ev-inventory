import { z } from 'zod'
import { apiRequest, ApiError } from '@/lib/api/client'

/**
 * Auth API contract (docs/API_CONVENTIONS.md, "Authentication"). The session itself
 * is an HttpOnly cookie set by the server; the browser script never sees it.
 */

/** Error codes the auth endpoints return in the standard error envelope. */
export const AuthErrorCode = {
  InvalidCredentials: 'INVALID_CREDENTIALS',
  AccountDisabled: 'ACCOUNT_DISABLED',
  Unauthenticated: 'UNAUTHENTICATED',
  SessionExpired: 'SESSION_EXPIRED',
  CsrfFailed: 'CSRF_FAILED',
  RateLimited: 'RATE_LIMITED',
} as const

const sessionUserSchema = z.object({
  id: z.string(),
  username: z.string(),
  displayName: z.string(),
})

export const sessionSchema = z.object({
  user: sessionUserSchema,
  csrfToken: z.string().min(1),
  expiresAt: z.string(),
})

export type SessionUser = z.infer<typeof sessionUserSchema>
export type Session = z.infer<typeof sessionSchema>

/**
 * What the client knows about the current session. `expired` means the server
 * rejected a session the browser presented (ended, expired, revoked or disabled).
 */
export type SessionState =
  { status: 'authenticated'; session: Session } | { status: 'anonymous'; expired: boolean }

export interface LoginInput {
  username: string
  password: string
}

/** POST /auth/login — sets the session cookie and returns the session. */
export function login(input: LoginInput): Promise<Session> {
  return apiRequest('/auth/login', sessionSchema, { method: 'POST', body: input })
}

/** POST /auth/logout — revokes the current session and clears the cookie. */
export function logout(): Promise<{ signedOut: boolean }> {
  return apiRequest('/auth/logout', z.object({ signedOut: z.boolean() }), { method: 'POST' })
}

/**
 * GET /auth/session — the current session. A 401 is an answer ("not signed in"),
 * not a failure; network and server errors still throw so the UI can say so
 * instead of guessing.
 */
export async function getSession(signal?: AbortSignal): Promise<SessionState> {
  try {
    const session = await apiRequest('/auth/session', sessionSchema, { signal })
    return { status: 'authenticated', session }
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      return { status: 'anonymous', expired: err.code === AuthErrorCode.SessionExpired }
    }
    throw err
  }
}
