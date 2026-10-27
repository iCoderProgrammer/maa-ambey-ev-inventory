import { MutationCache, QueryCache, QueryClient, type QueryKey } from '@tanstack/react-query'
import { meQueryKey } from '@/features/access/hooks/useAccess'
import { AuthErrorCode, type SessionState } from '@/features/auth/api/auth'
import { sessionQueryKey } from '@/features/auth/hooks/useSession'
import { ApiError } from '@/lib/api/client'
import { setCsrfToken } from '@/lib/api/csrf'

function isSessionQuery(key: QueryKey | undefined): boolean {
  return key?.[0] === sessionQueryKey[0] && key[1] === sessionQueryKey[1]
}

export function createQueryClient(): QueryClient {
  // Any 401 from a protected endpoint means the server no longer accepts this
  // session (expired, revoked, user disabled). Mark it ended so route protection
  // sends the user to sign in, instead of leaving stale screens that look usable.
  const handleAuthError = (error: Error, key?: QueryKey) => {
    if (!(error instanceof ApiError) || isSessionQuery(key)) return
    if (error.status === 401) {
      setCsrfToken(null)
      const ended: SessionState = { status: 'anonymous', expired: true }
      queryClient.setQueryData<SessionState>(sessionQueryKey, ended)
    } else if (error.code === 'FORBIDDEN') {
      // The server refused a permission the UI thought it had (e.g. revoked meanwhile):
      // re-read the permissions contract so navigation and actions follow the server.
      if (!(key?.[0] === meQueryKey[0] && key[1] === meQueryKey[1])) {
        void queryClient.invalidateQueries({ queryKey: meQueryKey })
      }
    } else if (error.code === AuthErrorCode.CsrfFailed) {
      // The token may be stale (e.g. after signing in from another tab): re-read it.
      void queryClient.invalidateQueries({ queryKey: sessionQueryKey })
    }
  }

  const queryClient: QueryClient = new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => {
        handleAuthError(error, query.queryKey)
      },
    }),
    mutationCache: new MutationCache({
      onError: (error) => {
        handleAuthError(error)
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 15_000,
        refetchOnWindowFocus: true,
        // Retry transient failures only; 4xx responses will not change on retry.
        retry: (failureCount, error) => {
          if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false
          return failureCount < 2
        },
      },
      mutations: {
        retry: false,
        // Send writes even when the browser reports offline: they fail at once with a network error
        // the dialog shows (entries kept, Cancel enabled), instead of pausing with no explanation
        // and being sent later, after the user may have given up or cancelled.
        networkMode: 'always',
      },
    },
  })
  return queryClient
}
