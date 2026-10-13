import { ChevronRight } from 'lucide-react'
import { useEffect, type ReactNode } from 'react'
import { Link } from 'react-router'

export interface Crumb {
  label: string
  to?: string
}

export interface PageHeaderProps {
  title: string
  breadcrumbs?: Crumb[]
  description?: string
  /** Small status next to the title, e.g. a "Planned" badge. */
  status?: ReactNode
  actions?: ReactNode
}

/** Page title + breadcrumb trail. Also sets document.title. */
export function PageHeader({
  title,
  breadcrumbs = [],
  description,
  status,
  actions,
}: PageHeaderProps) {
  useEffect(() => {
    document.title = `${title} · Maa Ambey Inventory`
  }, [title])

  return (
    <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        {breadcrumbs.length > 0 && (
          <nav aria-label="Breadcrumb" className="mb-1.5">
            <ol className="flex flex-wrap items-center gap-1.5 text-xs text-muted">
              {breadcrumbs.map((c, i) => (
                <li key={`${c.label}-${String(i)}`} className="flex items-center gap-1.5">
                  {i > 0 && <ChevronRight className="size-3 text-muted/60" aria-hidden="true" />}
                  {c.to ? (
                    <Link
                      to={c.to}
                      className="rounded font-medium transition-colors duration-150 hover:text-ink focus-visible:outline-accent"
                    >
                      {c.label}
                    </Link>
                  ) : (
                    <span
                      aria-current={i === breadcrumbs.length - 1 ? 'page' : undefined}
                      className="font-medium text-ink/80"
                    >
                      {c.label}
                    </span>
                  )}
                </li>
              ))}
            </ol>
          </nav>
        )}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">{title}</h1>
          {status}
        </div>
        {description && (
          <p className="mt-1 max-w-2xl text-sm leading-relaxed text-muted">{description}</p>
        )}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </div>
  )
}
