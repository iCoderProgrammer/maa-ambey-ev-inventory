import { X } from 'lucide-react'
import { Dialog as DialogPrimitive } from 'radix-ui'
import { useEffect, useRef } from 'react'
import { Outlet, useLocation } from 'react-router'
import { TooltipProvider } from '@/components/ui/Tooltip'
import { useAppDispatch, useAppSelector } from '@/store'
import { setMobileNavOpen } from '@/store/uiSlice'
import { cn } from '@/lib/utils/cn'
import { Header } from './Header'
import { MOBILE_NAV_TRIGGER_ID } from './ids'
import { Sidebar } from './Sidebar'

function Brand({ compact }: { compact: boolean }) {
  return (
    <div
      className={cn(
        'flex h-14 shrink-0 items-center gap-2.5 px-4',
        compact && 'justify-center px-0',
      )}
    >
      <span
        className="grid size-8 shrink-0 place-items-center rounded-control bg-gradient-to-br from-primary to-accent text-xs font-bold tracking-tight text-white shadow-sm ring-1 ring-white/20 transition-transform duration-150 hover:scale-105"
        aria-hidden="true"
      >
        MA
      </span>
      {!compact && (
        <span className="min-w-0 leading-tight">
          <span className="block truncate text-sm font-semibold tracking-tight text-white">
            Maa Ambey Enterprises
          </span>
          <span className="block text-[11px] font-medium text-accent">EV Inventory</span>
        </span>
      )}
    </div>
  )
}

export function AppShell() {
  const dispatch = useAppDispatch()
  const { sidebarCollapsed, mobileNavOpen } = useAppSelector((s) => s.ui)
  const { pathname } = useLocation()
  const mainRef = useRef<HTMLElement>(null)
  const previousPath = useRef(pathname)
  const navigatedRef = useRef(false)

  // On navigation: close the mobile drawer, jump to the top and move focus to the new
  // page content, so keyboard and screen-reader users are not left on a removed element.
  useEffect(() => {
    if (previousPath.current === pathname) return
    previousPath.current = pathname
    navigatedRef.current = true
    dispatch(setMobileNavOpen(false))
    window.scrollTo({ top: 0, behavior: 'instant' })
    mainRef.current?.focus({ preventScroll: true })
  }, [pathname, dispatch])

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex min-h-dvh">
        <a
          href="#main"
          className="sr-only z-50 rounded-control bg-primary px-3 py-2 text-primary-fg focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
        >
          Skip to content
        </a>

        {/* Desktop sidebar */}
        <aside
          className={cn(
            'sticky top-0 hidden h-dvh shrink-0 flex-col border-r border-white/5 bg-sidebar transition-[width] duration-200 ease-out lg:flex',
            sidebarCollapsed ? 'w-16' : 'w-68',
          )}
        >
          <Brand compact={sidebarCollapsed} />
          <Sidebar collapsed={sidebarCollapsed} />
        </aside>

        {/* Mobile / tablet drawer (Radix Dialog: focus trap, scroll lock, focus return). */}
        <DialogPrimitive.Root
          open={mobileNavOpen}
          onOpenChange={(open) => {
            if (open) navigatedRef.current = false
            dispatch(setMobileNavOpen(open))
          }}
        >
          <DialogPrimitive.Portal>
            <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/50 data-[state=closed]:animate-fade-out data-[state=open]:animate-fade-in lg:hidden" />
            <DialogPrimitive.Content
              id="mobile-nav"
              aria-describedby={undefined}
              onCloseAutoFocus={(e) => {
                // Radix only restores focus to its own Trigger; the opener lives in the Header.
                // After navigating, focus belongs to the new page instead.
                e.preventDefault()
                if (navigatedRef.current) mainRef.current?.focus({ preventScroll: true })
                else document.getElementById(MOBILE_NAV_TRIGGER_ID)?.focus()
              }}
              className="fixed inset-y-0 left-0 z-50 flex w-72 max-w-[85vw] flex-col bg-sidebar shadow-2xl data-[state=closed]:animate-drawer-out data-[state=open]:animate-drawer-in lg:hidden"
            >
              <DialogPrimitive.Title className="sr-only">Navigation</DialogPrimitive.Title>
              <div className="flex items-center justify-between pr-2">
                <Brand compact={false} />
                <DialogPrimitive.Close
                  aria-label="Close navigation"
                  className="grid size-9 cursor-pointer place-items-center rounded-control text-white/80 transition-colors hover:bg-white/10 hover:text-white"
                >
                  <X className="size-5" aria-hidden="true" />
                </DialogPrimitive.Close>
              </div>
              <Sidebar collapsed={false} onNavigate={() => dispatch(setMobileNavOpen(false))} />
            </DialogPrimitive.Content>
          </DialogPrimitive.Portal>
        </DialogPrimitive.Root>

        <div className="flex min-w-0 flex-1 flex-col">
          <Header />
          <main
            ref={mainRef}
            id="main"
            tabIndex={-1}
            className="flex-1 px-4 py-6 focus:outline-none sm:px-6 lg:px-8 lg:py-8"
          >
            <div className="mx-auto max-w-7xl">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </TooltipProvider>
  )
}
