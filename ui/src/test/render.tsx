import type { QueryClient } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AppProviders } from '@/app/AppProviders'
import { routes } from '@/app/routes'
import { createQueryClient } from '@/app/queryClient'
import type { Me } from '@/features/access/api/access'
import { meQueryKey } from '@/features/access/hooks/useAccess'
import type { SessionState } from '@/features/auth/api/auth'
import { sessionQueryKey } from '@/features/auth/hooks/useSession'
import { setCsrfToken } from '@/lib/api/csrf'
import { createStore } from '@/store'

/** Test-only session, as GET /auth/session would return it. */
export const testSession: SessionState = {
  status: 'authenticated',
  session: {
    user: { id: 'user-1', username: 'test.user', displayName: 'Test User' },
    csrfToken: 'test-csrf-token',
    expiresAt: '2026-10-12T22:00:00Z',
  },
}

/** Test-only GET /me answer: no roles, no permissions (the real state until sign-off). */
export const testMe: Me = {
  user: { id: 'user-1', username: 'test.user', displayName: 'Test User' },
  roles: [],
  permissions: [],
  locationScope: { all: false, locations: [] },
  proposedGrantsActive: false,
}

export interface RenderAppOptions {
  /**
   * Session to place in the query cache before rendering. Defaults to a signed-in
   * user so shell tests do not depend on the auth endpoints; pass `null` to leave
   * the cache empty so GET /auth/session is called (through the mocked fetch).
   */
  session?: SessionState | null
  /** Permissions contract to seed (default `testMe`); `null` leaves GET /me to the mocked fetch. */
  me?: Me | null
}

/** Seeds the session cache the way a server answer would. */
export function seedSession(queryClient: QueryClient, session: SessionState | null) {
  if (!session) return
  queryClient.setQueryData(sessionQueryKey, session)
  setCsrfToken(session.status === 'authenticated' ? session.session.csrfToken : null)
}

/** Renders the full app at `path` with fresh store and query client. */
export function renderApp(
  path = '/',
  { session = testSession, me = testMe }: RenderAppOptions = {},
) {
  const queryClient = createQueryClient()
  queryClient.setDefaultOptions({ queries: { retry: false, refetchInterval: false } })
  seedSession(queryClient, session)
  if (me && session?.status === 'authenticated') queryClient.setQueryData(meQueryKey, me)
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  return {
    router,
    queryClient,
    ...render(
      <AppProviders store={createStore()} queryClient={queryClient}>
        <RouterProvider router={router} />
      </AppProviders>,
    ),
  }
}
