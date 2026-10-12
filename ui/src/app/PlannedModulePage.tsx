import { ArrowLeft, Construction } from 'lucide-react'
import { Link } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import type { NavItem } from './navigation'

export interface PlannedModulePageProps {
  section?: string
  item: NavItem
}

/** Honest placeholder for modules that are on the backlog but not built yet. */
export function PlannedModulePage({ section, item }: PlannedModulePageProps) {
  return (
    <>
      <PageHeader
        title={item.label}
        breadcrumbs={[
          { label: 'Home', to: '/' },
          ...(section ? [{ label: section }] : []),
          { label: item.label },
        ]}
        description={item.summary}
        status={<Badge tone="neutral">Planned</Badge>}
      />
      <Card>
        <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
          <span
            className="grid size-11 shrink-0 place-items-center rounded-full bg-warning-subtle text-warning"
            aria-hidden="true"
          >
            <Construction className="size-5" />
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-4">
            <div>
              <h2 className="text-base font-semibold text-ink">
                This module is not implemented yet.
              </h2>
              <p className="mt-1 max-w-prose text-sm text-muted">
                Nothing on this page reads or changes inventory data. It will be delivered by the
                backlog tasks listed here.
              </p>
            </div>
            <div>
              <p className="text-xs font-medium tracking-wide text-muted uppercase">Delivered by</p>
              <ul aria-label="Backlog tasks" className="mt-1.5 flex flex-wrap gap-1.5">
                {item.backlog.map((id) => (
                  <li key={id}>
                    <Badge tone="info">{id}</Badge>
                  </li>
                ))}
              </ul>
            </div>
            <div>
              <Link to="/" className={buttonVariants({ variant: 'secondary', size: 'sm' })}>
                <ArrowLeft className="size-4" aria-hidden="true" />
                Back to dashboard
              </Link>
            </div>
          </div>
        </div>
      </Card>
    </>
  )
}
