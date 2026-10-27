import { NavLink, Outlet } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { RequirePermission } from '@/features/access/components/RequirePermission'
import { useMe } from '@/features/access/hooks/useAccess'
import { Permission } from '@/features/access/api/access'
import { cn } from '@/lib/utils/cn'

const tabs = [
  { to: '/admin/catalogue', label: 'Brands', end: true },
  { to: '/admin/catalogue/models', label: 'Models', end: false },
  { to: '/admin/catalogue/variants', label: 'Variants', end: false },
]

/** Tabs for the catalogue lists; detail pages render without them. */
export function CatalogueTabs() {
  return (
    <nav aria-label="Catalogue sections" className="mb-5 border-b border-border">
      <ul className="-mb-px flex gap-1 overflow-x-auto">
        {tabs.map((t) => (
          <li key={t.to}>
            <NavLink
              to={t.to}
              end={t.end}
              className={({ isActive }) =>
                cn(
                  'inline-flex h-10 items-center border-b-2 px-3 text-sm font-medium whitespace-nowrap transition-colors duration-150',
                  isActive
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted hover:border-border hover:text-ink',
                )
              }
            >
              {t.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

/**
 * Catalogue & Brands (C-03). Viewing needs catalogue.view; every action has its
 * own permission. The server checks every request again (deny by default).
 */
export function CatalogueLayout() {
  const me = useMe()
  if (me.data?.permissions.includes(Permission.CatalogueView)) return <Outlet />
  return (
    <>
      <PageHeader
        title="Catalogue & Brands"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Catalogue & Brands' }]}
      />
      <RequirePermission permission={Permission.CatalogueView} what="the catalogue">
        <Outlet />
      </RequirePermission>
    </>
  )
}
