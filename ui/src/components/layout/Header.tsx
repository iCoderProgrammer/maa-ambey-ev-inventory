import { Menu, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { ApiStatusIndicator } from '@/features/system/components/ApiStatusIndicator'
import { useAppDispatch, useAppSelector } from '@/store'
import { setMobileNavOpen, toggleSidebarCollapsed } from '@/store/uiSlice'
import { Button } from '@/components/ui/Button'
import { MOBILE_NAV_TRIGGER_ID } from './ids'
import { ThemeToggle } from './ThemeToggle'
import { UserMenu } from './UserMenu'

export function Header() {
  const dispatch = useAppDispatch()
  const collapsed = useAppSelector((s) => s.ui.sidebarCollapsed)
  const mobileNavOpen = useAppSelector((s) => s.ui.mobileNavOpen)

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-border/80 bg-surface/85 px-4 backdrop-blur-md supports-[backdrop-filter]:bg-surface/75 sm:px-6 lg:px-8">
      <Button
        variant="ghost"
        size="icon"
        className="lg:hidden"
        id={MOBILE_NAV_TRIGGER_ID}
        aria-label="Open navigation"
        aria-expanded={mobileNavOpen}
        aria-controls="mobile-nav"
        onClick={() => dispatch(setMobileNavOpen(true))}
      >
        <Menu className="size-5" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="hidden lg:inline-flex"
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        aria-pressed={collapsed}
        onClick={() => dispatch(toggleSidebarCollapsed())}
      >
        {collapsed ? (
          <PanelLeftOpen className="size-5" aria-hidden="true" />
        ) : (
          <PanelLeftClose className="size-5" aria-hidden="true" />
        )}
      </Button>

      <div className="hidden md:flex items-center gap-2 pl-2" aria-hidden="true">
        <span className="inline-flex items-center gap-1.5 rounded-full border border-border/80 bg-surface-muted/60 px-2.5 py-0.5 text-xs font-medium text-muted">
          <span className="size-1.5 rounded-full bg-accent" />
          Multi-Brand EV Operations Console
        </span>
      </div>

      <div className="ml-auto flex items-center gap-2.5 sm:gap-3">
        <ApiStatusIndicator />
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  )
}
