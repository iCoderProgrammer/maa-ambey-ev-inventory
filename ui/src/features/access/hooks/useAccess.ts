import { useQuery } from '@tanstack/react-query'
import { getMe, getRoleMatrix } from '../api/access'

export const meQueryKey = ['access', 'me'] as const

/** The signed-in user's permissions contract from the server. */
export function useMe() {
  return useQuery({
    queryKey: meQueryKey,
    queryFn: ({ signal }) => getMe(signal),
    staleTime: 60_000,
  })
}

/**
 * Whether the server granted `permission`. Unknown (still loading or failed)
 * counts as not granted: the UI never shows more than the server allows.
 */
export function usePermission(permission: string): { allowed: boolean; isPending: boolean } {
  const me = useMe()
  return {
    allowed: me.data?.permissions.includes(permission) ?? false,
    isPending: me.isPending,
  }
}

export function useRoleMatrix() {
  return useQuery({
    queryKey: ['access', 'roles'],
    queryFn: ({ signal }) => getRoleMatrix(signal),
  })
}
