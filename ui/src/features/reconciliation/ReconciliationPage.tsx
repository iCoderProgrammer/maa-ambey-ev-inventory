import { AlertTriangle, CheckCircle2, Info, Play, RefreshCw, Scale } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import type { SelectOption } from '@/components/ui/Select'
import { AccessDeniedState, EmptyState, LoadingState, Skeleton } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { FilterSelect, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useUrlFilters } from '@/features/stock/lib/urlFilters'
import {
  discrepancyTypeSchema,
  reconcileItemTypeSchema,
  severitySchema,
  type Discrepancy,
  type DiscrepancyType,
  type ReconcileItemType,
  type ReconcileReport,
  type Severity,
} from './api/reconciliation'
import { useReconciliation, useReconciliationPerms } from './hooks/useReconciliation'
import {
  describeReconcileError,
  discrepancyTypeLabels,
  itemTypeLabels,
  runTime,
  severityLabels,
  valueLabel,
} from './lib/labels'

/**
 * Stock ledger reconciliation (E-06). A read-only report: the server compares
 * vehicle units and parts balances with the stock movement ledger in one
 * database snapshot and returns what disagrees. Nothing here repairs, adjusts or
 * records anything — corrections go through the approved stock count and
 * adjustment flow. A run starts only when the user asks; filters and the page
 * live in the URL.
 */

const PAGE_SIZE = 25
const linkClass = 'font-medium text-primary hover:underline'
const filterKeys = ['itemType', 'severity', 'type', 'locationId'] as const
const filterLabels: Record<string, string> = {
  itemType: 'Item type',
  severity: 'Severity',
  type: 'Discrepancy',
  locationId: 'Location',
  page: 'Page',
  limit: 'Page size',
}

const options = <T extends string>(
  values: readonly T[],
  labels: Record<T, string>,
): SelectOption[] => values.map((v) => ({ value: v, label: labels[v] }))

export function ReconciliationPage() {
  const perms = useReconciliationPerms()
  let body: ReactNode
  if (perms.me.isPending) body = <LoadingState label="Checking your permissions…" />
  else if (perms.me.isError && !perms.me.data)
    body = (
      <ErrorBlock
        title="Could not check your permissions"
        message="Reload the page to try again."
        onRetry={() => void perms.me.refetch()}
      />
    )
  else if (!perms.view) body = <AccessDeniedState what="the stock reconciliation" />
  else body = <Reconciliation />
  return (
    <>
      <PageHeader
        title="Stock Reconciliation"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Stock Reconciliation' }]}
        description="Checks that every vehicle unit and parts balance agrees with the stock movement ledger."
      />
      {body}
    </>
  )
}

function ReadOnlyNote() {
  return (
    <p
      role="note"
      className="mb-4 flex items-start gap-2 rounded-card border border-info/30 bg-info-subtle px-4 py-3 text-sm text-ink"
    >
      <Info className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
      <span>
        Reconciliation only reads. It never changes a vehicle, a balance or the ledger, and it
        stores nothing. A discrepancy is corrected only through an approved stock count and
        adjustment, never from this page.
      </span>
    </p>
  )
}

function ErrorBlock({
  title,
  message,
  requestId,
  onRetry,
  children,
}: {
  title: string
  message: string
  requestId?: string
  onRetry?: () => void
  children?: ReactNode
}) {
  return (
    <div
      role="alert"
      className="flex flex-col gap-2 rounded-card border border-danger/30 bg-danger-subtle p-4 text-sm"
    >
      <p className="flex items-center gap-2 font-semibold text-danger">
        <AlertTriangle className="size-4" aria-hidden="true" />
        {title}
      </p>
      <p className="text-ink">{message}</p>
      {children}
      {requestId && <p className="font-mono text-xs text-muted">Reference: {requestId}</p>}
      {onRetry && (
        <div>
          <Button variant="secondary" size="sm" onClick={onRetry}>
            Try again
          </Button>
        </div>
      )}
    </div>
  )
}

function Reconciliation() {
  const perms = useReconciliationPerms()
  const { values, page, set } = useUrlFilters(filterKeys)
  // A run starts only on request; the report is never repeated in the background.
  const [started, setStarted] = useState(false)
  const query = useReconciliation(
    {
      itemType: (values.itemType || undefined) as ReconcileItemType | undefined,
      severity: (values.severity || undefined) as Severity | undefined,
      type: (values.type || undefined) as DiscrepancyType | undefined,
      locationId: values.locationId || undefined,
      page,
      limit: PAGE_SIZE,
    },
    { enabled: started },
  )
  // A run already in the cache (this session) is shown with its own run time; "Run" always
  // asks the server again — it never presents a cached report as a new run.
  const shown = started || query.data !== undefined
  const running = shown && query.isFetching
  const run = () => {
    setStarted(true)
    if (shown) void query.refetch()
  }
  const filtered = filterKeys.some((k) => values[k] !== '')
  const clear = () => {
    set(Object.fromEntries(filterKeys.map((k) => [k, ''])))
  }
  const locations = perms.me.data?.locationScope.locations ?? []
  const typeValues = discrepancyTypeSchema.options.filter((t) =>
    t.startsWith('VEHICLE_') ? perms.vehicles : perms.parts,
  )

  let body: ReactNode
  if (!shown) {
    body = (
      <EmptyState
        title="Reconciliation has not been run"
        icon={<Scale className="size-6 text-muted/80" />}
        action={
          <Button onClick={run}>
            <Play className="size-4" aria-hidden="true" />
            Run reconciliation
          </Button>
        }
      >
        Compares {perms.vehicles ? 'every vehicle unit' : ''}
        {perms.vehicles && perms.parts ? ' and ' : ''}
        {perms.parts ? 'every parts balance' : ''} at your locations with the stock movement ledger,
        as it is right now.
      </EmptyState>
    )
  } else if (query.isPending) {
    body = (
      <div role="status" aria-label="Running reconciliation" className="flex flex-col gap-2">
        <p className="text-sm text-muted">Running reconciliation…</p>
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  } else if (query.isError && !query.data) {
    body = <RunError error={query.error} onRetry={run} onClear={clear} />
  } else {
    body = (
      <div
        className={query.isPlaceholderData ? 'opacity-70' : undefined}
        aria-busy={query.isFetching}
      >
        {query.isError && (
          <div className="mb-4">
            <RunError error={query.error} onRetry={run} onClear={clear} />
          </div>
        )}
        <Summary report={query.data} />
        <Results
          report={query.data}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  }

  return (
    <>
      <ReadOnlyNote />
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted" aria-live="polite">
          {running
            ? 'Running reconciliation…'
            : query.data
              ? `Last run ${runTime(query.data.runAt)}`
              : ''}
        </p>
        {shown && (
          <Button variant="secondary" onClick={run} loading={running} disabled={running}>
            {!running && <RefreshCw className="size-4" aria-hidden="true" />}
            {running ? 'Running…' : 'Run again'}
          </Button>
        )}
      </div>
      <Toolbar onClear={clear} active={filtered}>
        {perms.vehicles && perms.parts && (
          <FilterSelect
            label="Item type"
            value={values.itemType}
            options={options(reconcileItemTypeSchema.options, itemTypeLabels)}
            onChange={(v) => {
              set({ itemType: v })
            }}
          />
        )}
        <FilterSelect
          label="Severity"
          value={values.severity}
          options={options(severitySchema.options, severityLabels)}
          onChange={(v) => {
            set({ severity: v })
          }}
        />
        <FilterSelect
          label="Discrepancy"
          value={values.type}
          options={options(typeValues, discrepancyTypeLabels)}
          onChange={(v) => {
            set({ type: v })
          }}
        />
        {locations.length > 0 && (
          <FilterSelect
            label="Location"
            value={values.locationId}
            options={locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
            onChange={(v) => {
              set({ locationId: v })
            }}
          />
        )}
      </Toolbar>
      {body}
    </>
  )
}

function RunError({
  error,
  onRetry,
  onClear,
}: {
  error: unknown
  onRetry: () => void
  onClear: () => void
}) {
  const d = describeReconcileError(error)
  if (d.state === 'forbidden')
    return <AccessDeniedState what="the stock reconciliation" requestId={d.requestId} />
  if (d.state === 'invalid-filter')
    return (
      <div
        role="alert"
        className="flex flex-col gap-2 rounded-card border border-warning/30 bg-warning-subtle p-4 text-sm"
      >
        <p className="font-semibold">{d.message}</p>
        <ul className="list-disc pl-5">
          {d.fields.map((f) => (
            <li key={`${f.field}-${f.message}`}>
              <span className="font-medium">{filterLabels[f.field] ?? f.field}</span> {f.message}
            </li>
          ))}
        </ul>
        <div>
          <Button variant="secondary" size="sm" onClick={onClear}>
            Clear filters
          </Button>
        </div>
      </div>
    )
  return (
    <ErrorBlock
      title="The reconciliation did not run"
      message={d.message}
      requestId={d.requestId}
      onRetry={d.retry ? onRetry : undefined}
    />
  )
}

function Metric({ label, value, tone }: { label: string; value: ReactNode; tone?: string }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-control border border-border/80 bg-surface px-4 py-3">
      <dt className="text-xs font-medium text-muted">{label}</dt>
      <dd className={`text-xl font-semibold tabular-nums ${tone ?? 'text-ink'}`}>{value}</dd>
    </div>
  )
}

function Summary({ report }: { report: ReconcileReport }) {
  const s = report.summary
  const clean = s.discrepancies === 0
  const scope = report.coverage.allLocations
    ? 'all locations'
    : `your ${report.coverage.locationIds.length} location${report.coverage.locationIds.length === 1 ? '' : 's'}`
  const covered = [
    report.coverage.vehicleUnits ? 'vehicle units' : null,
    report.coverage.parts ? 'parts balances' : null,
  ]
    .filter(Boolean)
    .join(' and ')
  return (
    <Card
      title="Run summary"
      description={`Checked ${covered} at ${scope} · ${runTime(report.runAt)} · ${report.durationMs} ms`}
      badge={
        clean ? (
          <Badge tone="success" icon={<CheckCircle2 className="size-3" aria-hidden="true" />}>
            Ledger agrees
          </Badge>
        ) : (
          <Badge tone="danger" icon={<AlertTriangle className="size-3" aria-hidden="true" />}>
            {s.discrepancies} {s.discrepancies === 1 ? 'discrepancy' : 'discrepancies'}
          </Badge>
        )
      }
      className="mb-4"
    >
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <Metric label="Vehicle units checked" value={s.vehicleUnitsChecked} />
        <Metric label="Parts balances checked" value={s.partBalancesChecked} />
        <Metric label="Records checked" value={s.recordsChecked} />
        <Metric label="Movements checked" value={s.movementsChecked} />
        <Metric
          label="High severity"
          value={s.bySeverity.HIGH ?? 0}
          tone={(s.bySeverity.HIGH ?? 0) > 0 ? 'text-danger' : undefined}
        />
        <Metric
          label="Medium severity"
          value={s.bySeverity.MEDIUM ?? 0}
          tone={(s.bySeverity.MEDIUM ?? 0) > 0 ? 'text-warning' : undefined}
        />
      </dl>
      {!clean && (
        <ul aria-label="Discrepancies by type" className="mt-3 flex flex-wrap gap-2 text-xs">
          {Object.entries(s.byType)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([type, n]) => (
              <li
                key={type}
                className="rounded-full border border-border bg-surface-muted px-3 py-1"
              >
                {(discrepancyTypeLabels as Record<string, string | undefined>)[type] ?? type}: {n}
              </li>
            ))}
        </ul>
      )}
      <p className="mt-3 font-mono text-xs text-muted">Run {report.runId}</p>
    </Card>
  )
}

function ItemRef({ d }: { d: Discrepancy }) {
  const perms = useReconciliationPerms()
  if (d.vehicleUnit) {
    const u = d.vehicleUnit
    return (
      <span className="flex flex-col">
        {perms.vehicles ? (
          <Link to={`/inventory/vehicles/${u.id}`} className={`${linkClass} font-mono`}>
            {u.unitNumber}
          </Link>
        ) : (
          <span className="font-mono">{u.unitNumber}</span>
        )}
        {u.chassisNumber && <span className="font-mono text-xs text-muted">{u.chassisNumber}</span>}
        {d.brandName && <span className="text-xs text-muted">{d.brandName}</span>}
      </span>
    )
  }
  if (d.stockItem) {
    const s = d.stockItem
    return (
      <span className="flex flex-col">
        {perms.openSKU ? (
          <Link to={`/inventory/parts/${s.id}`} className={`${linkClass} font-mono`}>
            {s.sku}
          </Link>
        ) : (
          <span className="font-mono">{s.sku}</span>
        )}
        <span className="text-xs text-muted">{s.name}</span>
      </span>
    )
  }
  return <>—</>
}

function Movements({ d }: { d: Discrepancy }) {
  const perms = useReconciliationPerms()
  if (d.movementIds.length === 0) return <span className="text-muted">None</span>
  return (
    <ul className="flex flex-col gap-0.5">
      {d.movementIds.map((id, i) => (
        <li key={id}>
          {perms.openMovement ? (
            <Link
              to={`/inventory/movements/${id}`}
              className={`${linkClass} text-xs`}
              aria-label={`Open stock movement ${i + 1} of ${d.movementIds.length} for ${discrepancyTypeLabels[d.type]}`}
            >
              Movement {i + 1}
            </Link>
          ) : (
            <span className="font-mono text-xs">{id.slice(0, 8)}</span>
          )}
        </li>
      ))}
    </ul>
  )
}

function Results({ report, onPage }: { report: ReconcileReport; onPage: (p: number) => void }) {
  if (report.summary.discrepancies === 0)
    return (
      <EmptyState title="No discrepancies" icon={<CheckCircle2 className="size-6 text-success" />}>
        Every record checked agrees with the stock movement ledger.
      </EmptyState>
    )
  if (report.total === 0 && report.page === 1)
    return (
      <EmptyState title="No discrepancies match these filters">
        Change or clear the filters to see the other discrepancies of this run.
      </EmptyState>
    )
  const cols: Column<Discrepancy>[] = [
    {
      key: 'type',
      header: 'Discrepancy',
      render: (d) => (
        <span className="flex flex-col gap-1">
          <span className="font-medium">{discrepancyTypeLabels[d.type]}</span>
          <Badge tone={d.severity === 'HIGH' ? 'danger' : 'warning'} className="self-start">
            {severityLabels[d.severity]} severity
          </Badge>
        </span>
      ),
    },
    { key: 'item', header: 'Vehicle / part', render: (d) => <ItemRef d={d} /> },
    {
      key: 'location',
      header: 'Location',
      render: (d) =>
        !d.locationId
          ? 'No location'
          : d.locationCode
            ? `${d.locationCode}${d.locationName ? ` — ${d.locationName}` : ''}`
            : 'A location not visible to you',
    },
    {
      key: 'expected',
      header: 'Expected (ledger)',
      render: (d) => <span className="font-mono text-xs">{valueLabel(d.expected)}</span>,
    },
    {
      key: 'actual',
      header: 'Actual (record)',
      render: (d) => <span className="font-mono text-xs">{valueLabel(d.actual)}</span>,
    },
    {
      key: 'explanation',
      header: 'Explanation',
      render: (d) => (
        <span className="flex flex-col gap-1">
          <span className="text-sm">{d.explanation}</span>
          <span className="font-mono text-[11px] break-all text-muted">{d.reference}</span>
        </span>
      ),
    },
    { key: 'movements', header: 'Movements', render: (d) => <Movements d={d} /> },
  ]
  return (
    <>
      <ResponsiveTable
        caption="Reconciliation discrepancies"
        columns={cols}
        rows={report.items}
        rowKey={(d) => d.reference}
      />
      <Pagination
        page={report.page}
        limit={report.limit}
        total={report.total}
        noun={['discrepancy', 'discrepancies']}
        onPage={onPage}
      />
    </>
  )
}
