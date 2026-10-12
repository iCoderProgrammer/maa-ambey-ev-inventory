import { ChevronDown, Info, LogOut, UserRound } from 'lucide-react'
import { Popover } from 'radix-ui'
import { useNavigate } from 'react-router'
import { Button } from '@/components/ui/Button'
import { useLogout, useSession } from '@/features/auth/hooks/useSession'
import { ApiError } from '@/lib/api/client'

/**
 * Account menu for the signed-in user (B-06). It only renders inside protected
 * routes, so the session shown here has been confirmed by the server. Sign-out
 * waits for the server to revoke the session before leaving the app.
 */
export function UserMenu() {
  const session = useSession()
  const logout = useLogout()
  const navigate = useNavigate()
  const user = session.data?.status === 'authenticated' ? session.data.session.user : null
  if (!user) return null

  const signOut = () => {
    if (logout.isPending) return
    logout.mutate(undefined, {
      onSuccess: () => void navigate('/login', { replace: true }),
    })
  }

  return (
    <Popover.Root>
      <Popover.Trigger
        aria-label={`Account: ${user.displayName}`}
        className="group flex h-9 cursor-pointer items-center gap-2 rounded-control px-2 text-sm text-ink transition-colors duration-150 hover:bg-surface-muted data-[state=open]:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
      >
        <span
          className="grid size-7 place-items-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/20 shadow-xs"
          aria-hidden="true"
        >
          <UserRound className="size-3.5" />
        </span>
        <span className="hidden max-w-40 truncate font-medium sm:inline">{user.displayName}</span>
        <ChevronDown
          className="size-3.5 text-muted transition-transform duration-150 group-data-[state=open]:rotate-180"
          aria-hidden="true"
        />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          collisionPadding={16}
          aria-label="Account"
          className="z-50 w-76 rounded-card border border-border/90 bg-surface p-4 text-sm text-ink shadow-xl ring-1 ring-black/5 data-[state=closed]:animate-pop-out data-[state=open]:animate-pop-in dark:ring-white/5"
        >
          <div className="flex items-center gap-3">
            <span
              className="grid size-10 shrink-0 place-items-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/20 shadow-xs"
              aria-hidden="true"
            >
              <UserRound className="size-5" />
            </span>
            <div className="min-w-0">
              <p className="truncate font-semibold tracking-tight text-ink">{user.displayName}</p>
              <p className="truncate text-xs text-muted">{user.username}</p>
            </div>
          </div>
          <div className="mt-3.5 flex gap-2.5 rounded-control border border-info/20 bg-info-subtle p-3 text-xs leading-relaxed text-ink">
            <Info className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
            <p>Roles and permissions are not assigned yet; they arrive with backlog task B-07.</p>
          </div>
          {logout.isError && (
            <p role="alert" className="mt-3 text-xs font-medium text-danger">
              {logout.error instanceof ApiError
                ? logout.error.message
                : 'Sign-out failed. Please try again.'}{' '}
              You are still signed in.
            </p>
          )}
          <Button
            variant="secondary"
            size="sm"
            className="mt-3.5 w-full"
            loading={logout.isPending}
            onClick={signOut}
          >
            {!logout.isPending && <LogOut className="size-3.5" aria-hidden="true" />}
            {logout.isPending ? 'Signing out…' : 'Sign out'}
          </Button>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
