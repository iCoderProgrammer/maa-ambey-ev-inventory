import { ChevronDown, Info, UserRound } from 'lucide-react'
import { Popover } from 'radix-ui'

/**
 * Profile/menu placeholder. Authentication arrives with backlog task B-06, so this
 * shows that nobody is signed in and offers no actions that pretend to work.
 */
export function UserMenu() {
  return (
    <Popover.Root>
      <Popover.Trigger
        aria-label="Account"
        className="group flex h-9 cursor-pointer items-center gap-1.5 rounded-control px-2 text-sm text-ink transition-colors duration-150 hover:bg-surface-muted data-[state=open]:bg-surface-muted"
      >
        <span
          className="grid size-7 place-items-center rounded-full bg-surface-muted text-muted ring-1 ring-border"
          aria-hidden="true"
        >
          <UserRound className="size-4" />
        </span>
        <span className="hidden sm:inline">Not signed in</span>
        <ChevronDown
          className="size-4 text-muted transition-transform duration-150 group-data-[state=open]:rotate-180"
          aria-hidden="true"
        />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          collisionPadding={16}
          aria-label="Account"
          className="z-50 w-72 rounded-card border border-border bg-surface p-4 text-sm text-ink shadow-lg data-[state=closed]:animate-pop-out data-[state=open]:animate-pop-in"
        >
          <div className="flex items-center gap-3">
            <span
              className="grid size-9 shrink-0 place-items-center rounded-full bg-surface-muted text-muted ring-1 ring-border"
              aria-hidden="true"
            >
              <UserRound className="size-5" />
            </span>
            <div>
              <p className="font-semibold">Not signed in</p>
              <p className="text-xs text-muted">No user session</p>
            </div>
          </div>
          <p className="mt-3 flex gap-2 rounded-control bg-info-subtle p-3 text-xs text-ink">
            <Info className="mt-px size-4 shrink-0 text-info" aria-hidden="true" />
            <span>
              Sign-in, your profile and role-based access are not available yet. They will be
              delivered by backlog tasks B-06 (authentication) and B-07 (roles and permissions).
            </span>
          </p>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
