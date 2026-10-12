import { NavLink, useMatch, useResolvedPath } from 'react-router'
import { navigation, type NavItem } from '@/app/navigation'
import { Tooltip } from '@/components/ui/Tooltip'
import { cn } from '@/lib/utils/cn'

export interface SidebarProps {
  collapsed: boolean
  onNavigate?: () => void
}

interface SidebarLinkProps {
  item: NavItem
  collapsed: boolean
  onNavigate?: () => void
}

/**
 * Computes the active state itself and passes NavLink a plain string className:
 * Radix Tooltip's `asChild` Slot merges className as strings, which would break
 * NavLink's function form.
 */
function SidebarLink({ item, collapsed, onNavigate }: SidebarLinkProps) {
  const Icon = item.icon
  const planned = item.backlog.length > 0
  const end = item.path === '/'
  const resolved = useResolvedPath(item.path)
  const isActive = useMatch({ path: resolved.pathname, end }) !== null

  return (
    <Tooltip
      enabled={collapsed}
      side="right"
      content={planned ? `${item.label} · Planned` : item.label}
    >
      <NavLink
        to={item.path}
        end={end}
        onClick={onNavigate}
        className={cn(
          // `relative` also keeps the absolutely positioned sr-only text inside the scrolling nav.
          'group relative flex min-h-10 items-center gap-3 rounded-control px-3 py-1.5 text-sm transition-colors duration-150',
          'focus-visible:outline-accent',
          isActive
            ? 'bg-white/10 font-medium text-white'
            : 'text-white/70 hover:bg-white/5 hover:text-white',
          collapsed && 'justify-center px-0',
        )}
      >
        {isActive && (
          <span
            className="absolute top-2 bottom-2 left-0 w-1 rounded-full bg-accent shadow-[0_0_8px_rgba(18,184,166,0.5)]"
            aria-hidden="true"
          />
        )}
        <Icon
          className={cn(
            'size-4 shrink-0 transition-colors duration-150',
            isActive ? 'text-accent' : 'text-white/60 group-hover:text-white',
          )}
          aria-hidden="true"
        />
        <span className={cn('line-clamp-2 leading-snug', collapsed && 'sr-only')}>
          {item.label}
        </span>
        {planned && (
          <>
            <span className="sr-only">, planned</span>
            {!collapsed && (
              <span
                aria-hidden="true"
                className="ml-auto rounded-full border border-white/20 bg-white/5 px-2 py-0.5 text-[10px] leading-3 font-medium text-white/70"
              >
                Planned
              </span>
            )}
          </>
        )}
      </NavLink>
    </Tooltip>
  )
}

export function Sidebar({ collapsed, onNavigate }: SidebarProps) {
  return (
    <nav
      aria-label="Main"
      className={cn(
        'flex h-full flex-col overflow-y-auto px-2 py-3',
        collapsed ? 'scrollbar-none' : 'scrollbar-sidebar',
      )}
    >
      <div className="flex-1">
        {navigation.map((section, si) => (
          <div key={section.label ?? `section-${String(si)}`} className="mb-2">
            {section.label &&
              (collapsed ? (
                <div className="mx-3 my-2 border-t border-white/10" aria-hidden="true" />
              ) : (
                <p className="px-3 pt-3 pb-1 text-[11px] font-semibold tracking-wider text-white/50 uppercase">
                  {section.label}
                </p>
              ))}
            <ul className="flex flex-col gap-0.5">
              {section.items.map((item) => (
                <li key={item.path}>
                  <SidebarLink item={item} collapsed={collapsed} onNavigate={onNavigate} />
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </nav>
  )
}
