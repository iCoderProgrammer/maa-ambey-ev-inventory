import { ArrowRight, Boxes, FileText, History, Layers, Tag, Users, Zap } from 'lucide-react'
import { Link } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/StateViews'
import { SystemStatusCard } from '@/features/system/components/SystemStatusCard'

const quickWorkflows = [
  {
    title: 'Vehicle Units Registry',
    path: '/inventory/vehicles',
    icon: Zap,
    description: 'Serialised inventory with chassis, motor & battery tracking.',
    backlog: 'D-04',
    domain: 'Inventory',
  },
  {
    title: 'Goods Receipts',
    path: '/procurement/goods-receipts',
    icon: Boxes,
    description: 'Draft receipts with unit identifiers; posting and PDI follow (D-07, D-06).',
    backlog: 'D-03',
    available: true,
    domain: 'Procurement',
  },
  {
    title: 'Sales Quotations',
    path: '/sales/quotations',
    icon: FileText,
    description: 'Price-snapshot quotations; approval rules pending owner sign-off.',
    backlog: 'F-02',
    domain: 'Sales',
  },
  {
    title: 'Multi-Brand Catalogue',
    path: '/admin/catalogue',
    icon: Tag,
    description: 'Brands, models and variants with verification status. Prices arrive with C-04.',
    backlog: 'C-03',
    available: true,
    domain: 'Admin',
  },
  {
    title: 'Stock Movements Ledger',
    path: '/inventory/movements',
    icon: History,
    description: 'Append-only immutable audit movement history across locations.',
    backlog: 'E-01',
    domain: 'Inventory',
  },
  {
    title: 'Customer Master',
    path: '/sales/customers',
    icon: Users,
    description: 'Verified buyer profiles with strict privacy controls.',
    backlog: 'F-01',
    domain: 'Sales',
  },
]

const telemetryPipeline = [
  { stage: '1. In-Transit', desc: 'Dispatched by supplier' },
  { stage: '2. PDI & Inward', desc: 'Chassis & battery inspected' },
  { stage: '3. Saleable Stock', desc: 'Ready on showroom floor' },
  { stage: '4. Reserved Unit', desc: 'Locked with customer token' },
  { stage: '5. Invoiced & Sold', desc: 'Post-delivery checklist complete' },
]

export function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Dashboard"
        breadcrumbs={[{ label: 'Home' }]}
        description="Showroom overview across all brands you are permitted to see."
        actions={
          <div className="hidden sm:flex items-center gap-2">
            <Badge tone="info" dot>
              Showroom v1.2 Baseline
            </Badge>
          </div>
        }
      />

      {/* Quick Operations Hub */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold tracking-wider text-muted uppercase">
              Operations Quick Launch
            </h2>
            <span className="rounded-full bg-surface-muted px-2 py-0.5 text-[11px] font-medium text-muted">
              Core Modules
            </span>
          </div>
        </div>

        <div className="grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
          {quickWorkflows.map((item) => {
            const Icon = item.icon
            return (
              <Link
                key={item.path}
                to={item.path}
                className="group relative flex flex-col justify-between rounded-card border border-border/80 bg-surface p-4 shadow-xs transition-all duration-150 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-sm"
              >
                <div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="grid size-9 place-items-center rounded-control bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-fg">
                      <Icon className="size-4" aria-hidden="true" />
                    </span>
                    <div className="flex items-center gap-1.5">
                      <span className="text-[10px] font-semibold text-muted uppercase">
                        {item.domain}
                      </span>
                      <Badge tone="neutral" className="text-[10px]">
                        {item.backlog}
                      </Badge>
                    </div>
                  </div>
                  <h3 className="mt-3 text-sm font-semibold tracking-tight text-ink group-hover:text-primary transition-colors">
                    {item.title}
                  </h3>
                  <p className="mt-1 text-xs leading-relaxed text-muted">{item.description}</p>
                </div>
                {/* Planned modules open their planned-module page; built ones open the module. */}
                <div className="mt-4 flex items-center justify-between gap-2 text-xs font-medium">
                  {item.available ? (
                    <Badge tone="success">Available</Badge>
                  ) : (
                    <Badge tone="neutral">Planned</Badge>
                  )}
                  <span className="flex items-center gap-1 text-primary">
                    {item.available ? 'Open module' : 'View plan'}
                    <ArrowRight
                      className="size-3.5 transition-transform duration-150 group-hover:translate-x-1"
                      aria-hidden="true"
                    />
                  </span>
                </div>
              </Link>
            )
          })}
        </div>
      </div>

      {/* Main operational row: Inventory overview & System status */}
      <div className="grid gap-6 lg:grid-cols-3">
        <Card
          title="Inventory overview"
          className="lg:col-span-2"
          badge={
            <Badge tone="info" dot>
              Architecture Preview
            </Badge>
          }
        >
          {/* Real KPIs (saleable, reserved, in PDI/hold, in transit, sold) arrive with backlog task G-01. */}
          <EmptyState title="Stock metrics are not available yet">
            Saleable, reserved, in-PDI and in-transit counts will appear here once the vehicle
            registry and dashboard metrics (backlog G-01) are implemented. No sample figures are
            shown.
          </EmptyState>

          {/* Architecture telemetry blueprint */}
          <div className="mt-6 border-t border-border/80 pt-5">
            <div className="flex items-center justify-between">
              <h3 className="flex items-center gap-1.5 text-xs font-semibold tracking-wider text-muted uppercase">
                <Layers className="size-3.5 text-accent" aria-hidden="true" />
                Target Telemetry Pipeline (Planned Backlog G-01)
              </h3>
              <span className="text-[11px] text-muted">5 Operational Stages</span>
            </div>

            <div className="mt-3 grid gap-2 sm:grid-cols-5">
              {telemetryPipeline.map((step) => (
                <div
                  key={step.stage}
                  className="flex flex-col justify-between rounded-control border border-border/70 bg-surface-muted/30 p-2.5 text-xs"
                >
                  <span className="font-semibold text-ink">{step.stage}</span>
                  <span className="mt-1 text-[11px] leading-tight text-muted">{step.desc}</span>
                </div>
              ))}
            </div>
          </div>
        </Card>

        <SystemStatusCard />
      </div>
    </div>
  )
}
