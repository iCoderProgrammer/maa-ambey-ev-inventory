import { Ban, CheckCircle2, ClipboardList, ListChecks, Plus, ShieldAlert } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link, NavLink, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useListParams } from '@/features/catalogue/lib/listParams'
import { formatDate, formatDateTime } from '@/features/catalogue/lib/presentation'
import type { VehicleUnit, VehicleUnitStatus } from '@/features/vehicles/api/vehicleUnits'
import { useVehicleUnit, useVehicleUnitList } from '@/features/vehicles/hooks/useVehicleUnits'
import { describeVehicleUnitReadError } from '@/features/vehicles/lib/presentation'
import { ApiError } from '@/lib/api/client'
import type { ChecklistItem, PdiOverview, PdiUnitState, RecordResult } from './api/pdi'
import { CreateChecklistItemDialog, RetireChecklistItemDialog } from './components/ChecklistDialogs'
import { InspectionForm } from './components/InspectionForm'
import { InspectionHistory } from './components/InspectionHistory'
import { useChecklist, usePdiOverview, usePdiPerms } from './hooks/usePdi'
import { dispositionLabels, dispositionTone, unitStatusLabels, unitStatusTone } from './lib/labels'

const basePath = '/procurement/pdi'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Vehicle Inspections (PDI)', to: basePath },
]
const linkClass = 'font-medium text-primary hover:underline'

export function UnitStatusBadge({ status }: { status: VehicleUnitStatus }) {
  return <Badge tone={unitStatusTone(status)}>{unitStatusLabels[status]}</Badge>
}

/** Permission gate: GET /me must say `allowed`; the API checks every request again. */
function Gate({
  allowed,
  what,
  header,
  gatedHeader,
  children,
}: {
  allowed: (p: ReturnType<typeof usePdiPerms>) => boolean
  what: string
  header?: ReactNode
  /** Heading shown only while the gate holds the page (checking, error, denied); the page renders its own. */
  gatedHeader?: ReactNode
  children: ReactNode
}) {
  const perms = usePdiPerms()
  let body: ReactNode = children
  if (perms.me.isPending) body = <LoadingState label="Checking your permissions…" />
  else if (perms.me.isError && !perms.me.data)
    body = (
      <ErrorState
        error={perms.me.error}
        title="Could not check your permissions"
        onRetry={() => void perms.me.refetch()}
      />
    )
  else if (!allowed(perms)) body = <AccessDeniedState what={what} />
  return (
    <>
      {header}
      {body !== children && gatedHeader}
      {body}
    </>
  )
}

function Tabs({ manage }: { manage: boolean }) {
  const cls = ({ isActive }: { isActive: boolean }) =>
    `rounded-control px-3 py-1.5 text-sm font-medium ${isActive ? 'bg-primary text-primary-fg' : 'text-ink hover:bg-surface-muted'}`
  return (
    <nav aria-label="PDI views" className="mb-4 flex flex-wrap gap-2">
      <NavLink to={basePath} end className={cls}>
        Awaiting PDI
      </NavLink>
      <NavLink to={`${basePath}/holds`} className={cls}>
        On quality hold
      </NavLink>
      {manage && (
        <NavLink to={`${basePath}/checklist`} className={cls}>
          Checklist
        </NavLink>
      )}
    </nav>
  )
}

// ── Queues ──────────────────────────────────────────────────────────────────

export function PdiQueuePage() {
  return <QueuePage status="RECEIVED_PENDING_PDI" />
}

export function PdiHoldsPage() {
  return <QueuePage status="QUALITY_HOLD" />
}

function QueuePage({ status }: { status: 'RECEIVED_PENDING_PDI' | 'QUALITY_HOLD' }) {
  const perms = usePdiPerms()
  const pending = status === 'RECEIVED_PENDING_PDI'
  return (
    <Gate
      what="vehicle inspections"
      allowed={(p) => p.view}
      header={
        <PageHeader
          title="Vehicle Inspections (PDI)"
          breadcrumbs={
            pending
              ? [{ label: 'Home', to: '/' }, { label: 'Vehicle Inspections (PDI)' }]
              : [...crumbs, { label: 'On quality hold' }]
          }
          description="Pre-delivery inspection of each received vehicle. A vehicle becomes saleable only when it passes; held and rejected vehicles are never saleable."
        />
      }
    >
      <Tabs manage={perms.manage} />
      <UnitQueue status={status} />
    </Gate>
  )
}

function UnitQueue({ status }: { status: 'RECEIVED_PENDING_PDI' | 'QUALITY_HOLD' }) {
  const { values, page, set } = useListParams(['q'] as const)
  const q = useVehicleUnitList({ status, q: values.q || undefined, page, limit: 25 })
  const pending = status === 'RECEIVED_PENDING_PDI'
  const cols: Column<VehicleUnit>[] = [
    {
      key: 'unit',
      header: 'Vehicle',
      render: (u) => (
        <div className="flex flex-col">
          <Link to={`${basePath}/${u.id}`} className={`${linkClass} font-mono`}>
            {u.unitNumber}
          </Link>
          <span className="text-xs text-muted">
            {u.brandName} · {u.modelName} · {u.variantName}
          </span>
        </div>
      ),
    },
    {
      key: 'chassis',
      header: 'Chassis / frame',
      render: (u) =>
        u.chassisNumber ? (
          <span className="font-mono text-xs">{u.chassisNumber}</span>
        ) : (
          <span className="text-xs text-warning">Not captured</span>
        ),
    },
    {
      key: 'loc',
      header: 'Location',
      render: (u) => (u.locationCode ? `${u.locationCode} — ${u.locationName ?? ''}` : '—'),
    },
    { key: 'received', header: 'Received', render: (u) => formatDate(u.receivedDate) },
    { key: 'status', header: 'Status', render: (u) => <UnitStatusBadge status={u.status} /> },
    {
      key: 'act',
      header: 'Action',
      render: (u) => (
        <Link
          to={`${basePath}/${u.id}`}
          className={buttonVariants({ variant: 'secondary', size: 'sm' })}
        >
          {pending ? 'Inspect' : 'Review'}
        </Link>
      ),
    },
  ]
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading vehicles…" />
  else if (q.isError && !q.data) {
    const d = describeVehicleUnitReadError(q.error)
    body =
      d.state === 'forbidden' ? (
        <AccessDeniedState what="vehicle units" requestId={d.requestId} />
      ) : (
        <ErrorState
          error={q.error}
          title="Could not load vehicles"
          onRetry={d.retry ? () => void q.refetch() : undefined}
        />
      )
  } else if (q.data.total === 0)
    body = values.q ? (
      <EmptyState title="No vehicles match this search">Change or clear the search.</EmptyState>
    ) : (
      <EmptyState
        title={pending ? 'No vehicle is awaiting PDI' : 'No vehicle is on quality hold'}
        icon={<ClipboardList className="size-6 text-muted/80" />}
      >
        {pending
          ? 'Vehicles appear here once their goods receipt is posted, at the locations you can see.'
          : 'Vehicles put on hold by an inspection appear here until they are rejected.'}
      </EmptyState>
    )
  else
    body = (
      <div className={q.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption={pending ? 'Vehicles awaiting PDI' : 'Vehicles on quality hold'}
          columns={cols}
          rows={q.data.items}
          rowKey={(u) => u.id}
        />
        <Pagination
          page={q.data.page}
          limit={q.data.limit}
          total={q.data.total}
          noun={['vehicle', 'vehicles']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  return (
    <>
      <Toolbar
        active={Boolean(values.q)}
        onClear={() => {
          set({ q: '' })
        }}
      >
        <SearchBox
          label="Search unit number or chassis"
          value={values.q}
          onSearch={(v) => {
            set({ q: v })
          }}
        />
      </Toolbar>
      {body}
    </>
  )
}

// ── Inspection ──────────────────────────────────────────────────────────────

export function PdiInspectionPage() {
  const { unitId = '' } = useParams()
  return (
    <Gate
      what="vehicle inspections"
      allowed={(p) => p.view}
      gatedHeader={
        <PageHeader title="Vehicle inspection" breadcrumbs={[...crumbs, { label: 'Inspection' }]} />
      }
    >
      <Inspection unitId={unitId} />
    </Gate>
  )
}

function SaleabilityNotice({ unit }: { unit: PdiUnitState }) {
  if (unit.saleable)
    return (
      <p
        role="status"
        className="flex items-start gap-2 rounded-card border border-success/30 bg-success-subtle px-4 py-3 text-sm text-success"
      >
        <CheckCircle2 className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <span>
          <strong>Saleable.</strong> This vehicle passed PDI and is available stock.
        </span>
      </p>
    )
  const why: Partial<Record<VehicleUnitStatus, string>> = {
    RECEIVED_PENDING_PDI: 'It is awaiting PDI.',
    QUALITY_HOLD: 'It is on quality hold after a failed inspection.',
    REJECTED: 'It was rejected at PDI.',
  }
  const danger = unit.status === 'REJECTED' || unit.status === 'QUALITY_HOLD'
  return (
    <p
      role="status"
      className={`flex items-start gap-2 rounded-card border px-4 py-3 text-sm ${danger ? 'border-danger/30 bg-danger-subtle text-danger' : 'border-warning/30 bg-warning-subtle text-warning'}`}
    >
      <Ban className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <span>
        <strong>Not saleable.</strong>{' '}
        {why[unit.status] ?? `Its status is ${unitStatusLabels[unit.status]}.`} It cannot be
        reserved or sold.
      </span>
    </p>
  )
}

function Inspection({ unitId }: { unitId: string }) {
  const perms = usePdiPerms()
  const overview = usePdiOverview(unitId)
  const unit = useVehicleUnit(unitId)
  const [recorded, setRecorded] = useState<RecordResult | null>(null)

  const header = (
    <PageHeader title="Vehicle inspection" breadcrumbs={[...crumbs, { label: 'Inspection' }]} />
  )
  if (overview.isPending)
    return (
      <>
        {header}
        <LoadingState label="Loading the vehicle's PDI…" />
      </>
    )
  if (overview.isError && !overview.data) {
    const e = overview.error
    let body: ReactNode = (
      <ErrorState
        error={e}
        title="Could not load the vehicle's PDI"
        onRetry={() => void overview.refetch()}
      />
    )
    if (e instanceof ApiError && e.status === 404)
      body = (
        <EmptyState title="Vehicle not found">
          This vehicle does not exist or is not at a location you can see.{' '}
          <Link to={basePath} className={linkClass}>
            Back to the PDI queue
          </Link>
        </EmptyState>
      )
    else if (e instanceof ApiError && e.status === 403)
      body = <AccessDeniedState what="this vehicle's inspections" requestId={e.requestId} />
    return (
      <>
        {header}
        {body}
      </>
    )
  }
  const o = overview.data
  const formKey = `${o.unit.status}:${o.allowedDispositions.join(',')}:${o.checklist.map((c) => c.id).join(',')}`
  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={o.unit.unitNumber}
        breadcrumbs={[...crumbs, { label: o.unit.unitNumber }]}
        status={<UnitStatusBadge status={o.unit.status} />}
        description="Inspect the vehicle, record the outcome of every checklist item and decide its disposition."
      />
      {recorded && (
        <p
          role="status"
          className="flex items-start gap-2 rounded-card border border-border bg-surface px-4 py-3 text-sm"
        >
          <Badge tone={dispositionTone[recorded.inspection.disposition]}>
            {dispositionLabels[recorded.inspection.disposition]}
          </Badge>
          <span>
            {recorded.inspection.inspectionNumber} recorded. The vehicle is now{' '}
            <strong>{unitStatusLabels[recorded.unit.status]}</strong>
            {recorded.unit.saleable ? ' and saleable.' : ' and not saleable.'}
          </span>
        </p>
      )}
      <SaleabilityNotice unit={o.unit} />
      <VehiclePanel overview={o} unit={unit} />
      <RecordArea
        overview={o}
        canUpload={perms.upload}
        formKey={formKey}
        onRecorded={setRecorded}
      />
      <InspectionHistory
        inspections={o.inspections}
        meId={perms.meId}
        canSeeEvidence={perms.attachmentsView}
      />
    </div>
  )
}

function VehiclePanel({
  overview,
  unit,
}: {
  overview: PdiOverview
  unit: ReturnType<typeof useVehicleUnit>
}) {
  let content: ReactNode
  if (unit.isPending) content = <LoadingState label="Loading vehicle details…" />
  else if (unit.isError && !unit.data)
    content = (
      <ErrorState
        error={unit.error}
        title="Could not load vehicle details"
        onRetry={() => void unit.refetch()}
      />
    )
  else {
    const u = unit.data
    content = (
      <DefinitionList
        items={[
          ['Brand / model / variant', `${u.brandName} · ${u.modelName} · ${u.variantName}`],
          [
            'Chassis / frame',
            u.chassisNumber ?? (
              <span className="text-warning">Not captured — the vehicle cannot leave PDI</span>
            ),
          ],
          ['Motor serial', u.motorSerial ?? '—'],
          ['Battery serial', u.batterySerial ?? '—'],
          ['Charger serial', u.chargerSerial ?? '—'],
          ['Location', u.locationCode ? `${u.locationCode} — ${u.locationName ?? ''}` : 'None'],
          ['Supplier', u.supplierName ?? '—'],
          ['Goods receipt', u.source.goodsReceiptNumber ?? '—'],
          ['Received', formatDate(u.receivedDate)],
          ['Last change', formatDateTime(u.updatedAt)],
        ]}
      />
    )
  }
  return (
    <Card
      title="Vehicle"
      description={`Unit ${overview.unit.unitNumber}, version ${String(overview.unit.version)}`}
    >
      {content}
    </Card>
  )
}

function RecordArea({
  overview: o,
  canUpload,
  formKey,
  onRecorded,
}: {
  overview: PdiOverview
  canUpload: boolean
  formKey: string
  onRecorded: (r: RecordResult) => void
}) {
  if (o.allowedDispositions.length === 0)
    return (
      <Card title="Record inspection">
        <p role="note" className="text-sm text-muted">
          No inspection can be recorded: the vehicle is{' '}
          {unitStatusLabels[o.unit.status].toLowerCase()}. PDI applies to vehicles awaiting PDI, and
          a vehicle on quality hold can only be rejected.
        </p>
      </Card>
    )
  if (!o.canRecord)
    return (
      <Card title="Record inspection">
        <p role="note" className="flex items-start gap-2 text-sm text-muted">
          <ShieldAlert className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          You can view this vehicle's inspections. Recording an inspection needs PDI permission.
        </p>
      </Card>
    )
  return <InspectionForm key={formKey} overview={o} canUpload={canUpload} onRecorded={onRecorded} />
}

// ── Checklist configuration ─────────────────────────────────────────────────

export function PdiChecklistPage() {
  const perms = usePdiPerms()
  const [creating, setCreating] = useState(false)
  const add = perms.manage ? (
    <Button
      onClick={() => {
        setCreating(true)
      }}
    >
      <Plus className="size-4" aria-hidden="true" />
      Add item
    </Button>
  ) : undefined
  return (
    <Gate
      what="the PDI checklist"
      allowed={(p) => p.view || p.record || p.manage}
      header={
        <PageHeader
          title="PDI checklist"
          breadcrumbs={[...crumbs, { label: 'Checklist' }]}
          description="The items every inspection must answer. Nothing is pre-filled: the inspection criteria are the business's to define."
          actions={add}
        />
      }
    >
      <Tabs manage={perms.manage} />
      <ChecklistList add={add} manage={perms.manage} />
      {creating && (
        <CreateChecklistItemDialog
          canListBrands={perms.catalogueView}
          onClose={() => {
            setCreating(false)
          }}
        />
      )}
    </Gate>
  )
}

function ChecklistList({ add, manage }: { add: ReactNode; manage: boolean }) {
  const { values, page, set } = useListParams(['status'] as const)
  const status = (values.status || 'ACTIVE') as 'ACTIVE' | 'RETIRED'
  const q = useChecklist({ status, page, limit: 50 })
  const [retiring, setRetiring] = useState<ChecklistItem | null>(null)
  const cols: Column<ChecklistItem>[] = [
    { key: 'order', header: 'Order', render: (c) => c.sortOrder },
    {
      key: 'item',
      header: 'Item',
      render: (c) => (
        <div className="flex flex-col">
          <span className="font-medium">{c.label}</span>
          <span className="font-mono text-xs text-muted">{c.code}</span>
          {c.description && <span className="text-xs text-muted">{c.description}</span>}
        </div>
      ),
    },
    { key: 'scope', header: 'Applies to', render: (c) => c.brandName ?? 'All brands' },
    {
      key: 'status',
      header: 'Status',
      render: (c) =>
        c.status === 'ACTIVE' ? (
          <Badge tone="success">Active</Badge>
        ) : (
          <span className="flex flex-col gap-0.5">
            <Badge tone="neutral">Retired</Badge>
            {c.retireReason && <span className="text-xs text-muted">{c.retireReason}</span>}
          </span>
        ),
    },
    ...(manage && status === 'ACTIVE'
      ? [
          {
            key: 'act',
            header: 'Action',
            render: (c: ChecklistItem) => (
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Retire ${c.code}`}
                onClick={() => {
                  setRetiring(c)
                }}
              >
                Retire
              </Button>
            ),
          },
        ]
      : []),
  ]
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading the checklist…" />
  else if (q.isError && !q.data)
    body = (
      <ErrorState
        error={q.error}
        title="Could not load the checklist"
        onRetry={() => void q.refetch()}
      />
    )
  else if (q.data.total === 0)
    body =
      status === 'ACTIVE' ? (
        <EmptyState
          title="The PDI checklist is not configured"
          icon={<ListChecks className="size-6 text-muted/80" />}
          action={add}
        >
          Until at least one item applies to a vehicle, it cannot pass PDI (it can still be put on
          hold).
        </EmptyState>
      ) : (
        <EmptyState title="No retired items">Retired items appear here.</EmptyState>
      )
  else
    body = (
      <div className={q.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="PDI checklist items"
          columns={cols}
          rows={q.data.items}
          rowKey={(c) => c.id}
        />
        <Pagination
          page={q.data.page}
          limit={q.data.limit}
          total={q.data.total}
          noun={['item', 'items']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  return (
    <>
      <Toolbar
        active={status !== 'ACTIVE'}
        onClear={() => {
          set({ status: '' })
        }}
      >
        <FilterSelect
          label="Status"
          value={values.status === 'RETIRED' ? 'RETIRED' : ''}
          options={[{ value: 'RETIRED', label: 'Retired' }]}
          allLabel="Active"
          onChange={(v) => {
            set({ status: v })
          }}
        />
      </Toolbar>
      {body}
      {retiring && (
        <RetireChecklistItemDialog
          item={retiring}
          onClose={() => {
            setRetiring(null)
          }}
        />
      )}
    </>
  )
}
