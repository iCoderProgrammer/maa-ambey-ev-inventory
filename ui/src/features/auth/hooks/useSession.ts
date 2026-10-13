import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { setCsrfToken } from '@/lib/api/csrf'
import {
  getSession,
  login,
  logout,
  type LoginInput,
  type Session,
  type SessionState,
} from '../api/auth'

export const sessionQueryKey = ['auth', 'session'] as const

/** Keeps the in-memory CSRF token in step with what the server last confirmed. */
function syncCsrf(state: SessionState): SessionState {
  setCsrfToken(state.status === 'authenticated' ? state.session.csrfToken : null)
  return state
}

/** Server-confirmed session state. Never derived from local storage or guesses. */
export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: async ({ signal }) => syncCsrf(await getSession(signal)),
    staleTime: 60_000,
    // Re-check now and then so a session that ended on the server (idle timeout,
    // sign-out elsewhere, account disabled) is noticed even while nothing else is loading.
    refetchInterval: 5 * 60_000,
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: LoginInput) => login(input),
    onSuccess: (session: Session) => {
      // Data cached for a previous user must not leak into this session.
      queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== 'auth' })
      queryClient.setQueryData<SessionState>(
        sessionQueryKey,
        syncCsrf({ status: 'authenticated', session }),
      )
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== 'auth' })
      const signedOut: SessionState = { status: 'anonymous', expired: false }
      queryClient.setQueryData<SessionState>(sessionQueryKey, syncCsrf(signedOut))
    },
  })
}
