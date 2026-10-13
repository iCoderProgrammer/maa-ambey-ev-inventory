import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  Construction,
  Layers,
  ShieldCheck,
} from 'lucide-react'
import { Link } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import { navigation, type NavItem } from './navigation'

export interface PlannedModulePageProps {
  section?: string
  item: NavItem
}

/** Honest placeholder for modules that are on the backlog but not built yet. */
export function PlannedModulePage({ section, item }: PlannedModulePageProps) {
  const Icon = item.icon
  const sectionNav = navigation.find((s) => s.label === section)
  // Permission-gated pages are left out: whether to show them depends on the user's grants.
  const relatedItems =
    sectionNav?.items.filter((i) => i.path !== item.path && !i.permission).slice(0, 3) ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={item.label}
        breadcrumbs={[
          { label: 'Home', to: '/' },
          ...(section ? [{ label: section }] : []),
          { label: item.label },
        ]}
        description={item.summary}
        status={
          <Badge tone="neutral" dot>
            Planned
          </Badge>
        }
        actions={
          <Link to="/" className={buttonVariants({ variant: 'secondary', size: 'sm' })}>
            <ArrowLeft className="size-4" aria-hidden="true" />
            Back to dashboard
          </Link>
        }
      />

      <Card>
        <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
          <div className="relative shrink-0">
            <span
              className="grid size-12 place-items-center rounded-control bg-primary/10 text-primary shadow-xs ring-1 ring-primary/20"
              aria-hidden="true"
            >
              <Icon className="size-6" />
            </span>
            <span
              className="absolute -bottom-1 -right-1 grid size-5 place-items-center rounded-full bg-warning-subtle text-warning shadow-xs ring-1 ring-warning/40"
              aria-hidden="true"
            >
              <Construction className="size-3" />
            </span>
          </div>
          <div className="flex min-w-0 flex-1 flex-col gap-4">
            <div>
              <h2 className="text-base font-semibold text-ink">
                This module is not implemented yet.
              </h2>
              <p className="mt-1 max-w-prose text-sm leading-relaxed text-muted">
                Nothing on this page reads or changes inventory data. It will be delivered by the
                backlog tasks listed here.
              </p>
            </div>

            <div>
              <p className="text-xs font-semibold tracking-wider text-muted uppercase">
                Delivered by
              </p>
              <ul aria-label="Backlog tasks" className="mt-2 flex flex-wrap gap-2">
                {item.backlog.map((id) => (
                  <li key={id}>
                    <Badge tone="info" className="px-2.5 py-1 text-xs font-mono">
                      {id}
                    </Badge>
                  </li>
                ))}
              </ul>
            </div>

            <div className="pt-1">
              <Link to="/" className={buttonVariants({ variant: 'secondary', size: 'sm' })}>
                <ArrowLeft className="size-4" aria-hidden="true" />
                Back to dashboard
              </Link>
            </div>
          </div>
        </div>
      </Card>

      {/* Target Module Specifications & Architecture Preview */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card
          title="Module Architecture & Governance"
          description="Operational invariants enforced when this module is built."
          badge={<Badge tone="info">System Invariant</Badge>}
        >
          <ul className="flex flex-col gap-3 text-sm text-ink">
            <li className="flex items-start gap-2.5">
              <ShieldCheck className="mt-0.5 size-4 shrink-0 text-accent" aria-hidden="true" />
              <div>
                <span className="font-semibold">Server-Side Authorization:</span>
                <p className="text-xs text-muted">
                  Strict RBAC denies unpermitted actions by default; no client-side assumptions.
                </p>
              </div>
            </li>
            <li className="flex items-start gap-2.5">
              <Layers className="mt-0.5 size-4 shrink-0 text-accent" aria-hidden="true" />
              <div>
                <span className="font-semibold">Append-Only Audit Trails:</span>
                <p className="text-xs text-muted">
                  Every state transition generates an immutable ledger record with user, time & IP.
                </p>
              </div>
            </li>
            <li className="flex items-start gap-2.5">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-accent" aria-hidden="true" />
              <div>
                <span className="font-semibold">Multi-Brand Isolation:</span>
                <p className="text-xs text-muted">
                  Data and operations are filtered by authorized brands without cross-contamination.
                </p>
              </div>
            </li>
          </ul>
        </Card>

        {relatedItems.length > 0 && (
          <Card
            title={`Other ${section ?? ''} Modules`}
            description="Explore related operational areas in this functional domain."
          >
            <div className="flex flex-col divide-y divide-border/60">
              {relatedItems.map((rel) => {
                const RelIcon = rel.icon
                return (
                  <Link
                    key={rel.path}
                    to={rel.path}
                    className="group flex items-center justify-between py-2.5 transition-colors hover:text-primary"
                  >
                    <div className="flex items-center gap-2.5">
                      <span className="grid size-7 place-items-center rounded-control bg-surface-muted text-muted group-hover:bg-primary/10 group-hover:text-primary">
                        <RelIcon className="size-3.5" aria-hidden="true" />
                      </span>
                      <span className="text-sm font-medium text-ink group-hover:text-primary">
                        {rel.label}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Badge tone="neutral" className="text-[10px]">
                        {rel.backlog[0]}
                      </Badge>
                      <ArrowRight className="size-3.5 text-muted transition-transform group-hover:translate-x-1" />
                    </div>
                  </Link>
                )
              })}
            </div>
          </Card>
        )}
      </div>
    </div>
  )
}
