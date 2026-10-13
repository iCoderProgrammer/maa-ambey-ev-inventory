import { WifiOff } from 'lucide-react'
import type { ReactNode } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router'
import { ErrorState, LoadingState } from '@/components/ui/StateViews'
import { useSession } from '../hooks/useSession'

function CenteredStatus({ children }: { children: ReactNode }) {
  return (
    <div className="grid min-h-dvh place-items-center bg-canvas px-4">
      <div className="w-full max-w-md">{children}</div>
    </div>
  )
}

/**
 * Renders protected routes only for a session the server has confirmed. Anything
 * else (no session, ended session) goes to the sign-in page; if the session cannot
 * be checked, say so rather than guessing either way.
 */
export function RequireAuth() {
  const session = useSession()
  const location = useLocation()

  // A confirmed session stays usable when a later re-check fails (API briefly
  // unreachable): the header shows the outage, and any 401 still ends the session.
  if (session.data?.status === 'authenticated') return <Outlet />

  if (session.data?.status === 'anonymous') {
    const from = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to="/login" replace state={{ from }} />
  }

  if (session.isPending) {
    if (session.fetchStatus === 'paused') {
      return (
        <CenteredStatus>
          <p
            role="alert"
            className="flex items-start gap-2.5 rounded-control border border-warning/30 bg-warning-subtle p-3.5 text-sm text-ink"
          >
            <WifiOff className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
            No network connection. Your session will be checked when you are back online.
          </p>
        </CenteredStatus>
      )
    }
    return (
      <CenteredStatus>
        <LoadingState label="Checking your session…" />
      </CenteredStatus>
    )
  }

  // Never checked successfully: say so rather than guessing either way.
  return (
    <CenteredStatus>
      <ErrorState
        error={session.error}
        title="Could not check your session"
        onRetry={() => void session.refetch()}
      />
    </CenteredStatus>
  )
}
