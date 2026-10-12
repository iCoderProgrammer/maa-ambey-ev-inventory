import type { ReactNode } from 'react'
import { AccessDeniedState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import { useMe } from '../hooks/useAccess'

export interface RequirePermissionProps {
  permission: string
  /** Shown in the access-denied message, e.g. "the role and permission matrix". */
  what?: string
  children: ReactNode
}

/**
 * Renders children only when the server-confirmed permissions include `permission`.
 * This is presentation only; the API enforces the same permission on every request.
 */
export function RequirePermission({ permission, what, children }: RequirePermissionProps) {
  const me = useMe()
  if (me.isPending) return <LoadingState label="Checking your permissions…" />
  if (me.isError && !me.data) {
    return (
      <ErrorState
        error={me.error}
        title="Could not check your permissions"
        onRetry={() => void me.refetch()}
      />
    )
  }
  if (!me.data.permissions.includes(permission)) return <AccessDeniedState what={what} />
  return <>{children}</>
}
