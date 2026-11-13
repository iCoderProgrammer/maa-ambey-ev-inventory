import { ArrowLeft, ArrowRight, History, Info, X } from 'lucide-react'
import { useCallback, useId, useState, type ReactNode } from 'react'
import { Link, useLocation, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import type { SelectOption } from '@/components/ui/Select'
import {
  AccessDeniedState,
  EmptyState,
  ErrorState,
  LoadingState,
  Skeleton,
} from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import {
  timelineItemTypeSchema,
  timelineMovementTypeSchema,
  timelineSourceTypeSchema,
  type TimelineItemType,
  type TimelineMovement,
  type TimelineMovementType,
  type TimelineSourceType,
} from './api/timeline'
import { useStockMovement, useStockMovementTimeline, useTimelinePerms } from './hooks/useTimeline'
import {
  describeTimelineError,
  HIDDEN_SOURCE_TEXT,
  itemTypeLabels,
  locationLabel,
  movementTime,
  quantityChange,
  sourceTypeLabels,
  statusChange,
  timelineTypeLabels,
  utcOffsetLabel,
} from './lib/timeline'
import { dayStart, useUrlFilters } from './lib/urlFilters'

/**
 * Stock movement timeline (E-02). The append-only ledger as the server returns
 * it: filters, search and paging run on the server and live in the URL. Nothing
 * here records, edits, reverses or deletes a movement, and no history is made
 * up on the client. Links to related records appear only when the server
 * returned the reference and the user may open that screen.
 */

const listPath = '/inventory/movements'
const linkClass = 'font-medium text-primary hover:underline'
const PAGE_SIZE = 25
const filterKeys = [
  'q',
  'itemType',
  'vehicleUnitId',
  'stockItemId',
  'movementType',
  'sourceType',
  'sourceReference',
  'actorId',
  'locationId',
  'from',
  'to',
  'order',
] as const

const filterLabels: Record<string, string> = {
  q: 'Search',
  itemType: 'Item type',
  vehicleUnitId: 'Vehicle',
  stockItemId: 'Part',
  movementType: 'Movement',
  sourceType: 'Source',
  sourceReference: 'Source reference',
  actorId: 'Recorded by',
  locationId: 'Location',
  from: 'From date',
  to: 'To date',
  order: 'Order',
  page: 'Page',
  limit: 'Page size',
}

const options = <T extends string>(
  values: readonly T[],
  labels: Record<T, string>,
): SelectOption[] => values.map((v) => ({ value: v, label: labels[v] }))

/** Gate: GET /me must grant vehicle_units.view or stock.parts.view; the API checks every request again. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = useTimelinePerms()
  let body: ReactNode = children
  if (me.isPending) body = <LoadingState label="Checking your permissions…" />
  else if (me.isError && !me.data)
    body = (
      <ErrorState
        error={me.error}
        title="Could not check your permissions"
        onRetry={() => void me.refetch()}
      />
    )
  else if (!view) body = <AccessDeniedState what="stock movements" />
  if (body === children) return <>{children}</>
  return (
    <>
      {header}
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
        The stock ledger is append-only: movements are recorded by receipts, PDI, display moves,
        parts receipts and issues, and approved counts, and can never be edited or deleted. Times
        are shown in your time zone ({utcOffsetLabel()}); date filters use your calendar days.
      </span>
    </p>
  )
}

// ── Cells shared by the list and the detail ────────────────────────────────

function Actor({ m }: { m: TimelineMovement }) {
  const { meId } = useTimelinePerms()
  return <>{m.actorId === meId ? 'You' : (m.actorName ?? 'Unknown user')}</>
}

/** The vehicle or part, linked only when the user may open its screen. */
function ItemRef({ m }: { m: TimelineMovement }) {
  const perms = useTimelinePerms()
  if (m.vehicleUnit) {
    const u = m.vehicleUnit
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
      </span>
    )
  }
  if (m.stockItem) {
    const s = m.stockItem
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

function Change({ m }: { m: TimelineMovement }) {
  const s = statusChange(m)
  if (s)
    return (
      <span className="inline-flex flex-wrap items-center gap-1">
        {s.from}
        <ArrowRight className="size-3 text-muted" aria-label="to" />
        {s.to}
      </span>
    )
  return <span className="font-mono">{quantityChange(m) ?? '—'}</span>
}

function Where({ m }: { m: TimelineMovement }) {
  const to = locationLabel(m.toLocationId, m.toLocationCode, m.toLocationName)
  if (m.fromLocationId && m.fromLocationId !== m.toLocationId) {
    const from = locationLabel(m.fromLocationId, m.fromLocationCode, m.fromLocationName)
    return (
      <span className="inline-flex flex-wrap items-center gap-1">
        {from}
        <ArrowRight className="size-3 text-muted" aria-label="to" />
        {to}
      </span>
    )
  }
  return <>{to}</>
}

/**
 * The source, linked only to screens that exist and that the user may open,
 * and only with the ID the server returned (never a guessed URL).
 */
function Source({ m }: { m: TimelineMovement }) {
  const perms = useTimelinePerms()
  const label = sourceTypeLabels[m.sourceType]
  if (m.sourceHidden)
    return (
      <span className="flex flex-col">
        <span>{label}</span>
        <span className="text-xs text-muted">{HIDDEN_SOURCE_TEXT}</span>
      </span>
    )
  const ref = m.sourceReference
  let link: ReactNode = ref ? <span className="font-mono">{ref}</span> : null
  if (ref && m.sourceId && m.sourceType === 'GOODS_RECEIPT' && perms.openGRN)
    link = (
      <Link to={`/procurement/goods-receipts/${m.sourceId}`} className={`${linkClass} font-mono`}>
        {ref}
      </Link>
    )
  else if (ref && m.sourceId && m.sourceType === 'STOCK_COUNT' && perms.openCount)
    link = (
      <Link to={`/inventory/counts/${m.sourceId}`} className={`${linkClass} font-mono`}>
        {ref}
      </Link>
    )
  else if (ref && m.sourceType === 'PDI_INSPECTION' && m.vehicleUnit && perms.vehicles)
    link = (
      <Link to={`/procurement/pdi/${m.vehicleUnit.id}`} className={`${linkClass} font-mono`}>
        {ref}
      </Link>
    )
  return (
    <span className="flex flex-col">
      <span>{label}</span>
      {link && <span className="text-xs">{link}</span>}
    </span>
  )
}

// ── List ────────────────────────────────────────────────────────────────────

export function StockMovementsPage() {
  return (
    <>
      <PageHeader
        title="Stock Movements"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Stock Movements' }]}
        description="Every recorded stock and status change across vehicles and parts, as the ledger holds it."
      />
      <Gate>
        <ReadOnlyNote />
        <Timeline />
      </Gate>
    </>
  )
}

/** A text filter committed on blur or Enter (not on every keystroke). */
function CommitInput({
  label,
  value,
  onCommit,
}: {
  label: string
  value: string
  onCommit: (v: string) => void
}) {
  const id = useId()
  const [draft, setDraft] = useState(value)
  const [synced, setSynced] = useState(value)
  if (value !== synced) {
    setSynced(value)
    setDraft(value)
  }
  const commit = () => {
    const v = draft.trim()
    if (v !== value) onCommit(v)
  }
  return (
    <div className="flex min-w-0 flex-col gap-1.5 sm:w-44">
      <label htmlFor={id} className="text-xs font-medium text-muted">
        {label}
      </label>
      <Input
        id={id}
        value={draft}
        maxLength={100}
        placeholder="e.g. GRN-000001"
        onChange={(e) => {
          setDraft(e.target.value)
        }}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commit()
        }}
      />
    </div>
  )
}

function DateFilter({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (v: string) => void
}) {
  const id = useId()
  return (
    <div className="flex min-w-0 flex-col gap-1.5 sm:w-40">
      <label htmlFor={id} className="text-xs font-medium text-muted">
        {label}
      </label>
      <Input
        id={id}
        type="date"
        value={value}
        onChange={(e) => {
          onChange(e.target.value)
        }}
      />
    </div>
  )
}

/** A pinned filter that has no control of its own (set from a detail page link). */
function Pinned({ label, onClear }: { label: string; onClear: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 self-start rounded-full border border-border bg-surface-muted px-3 py-1 text-xs sm:self-end sm:mb-1.5">
      {label}
      <button
        type="button"
        onClick={onClear}
        className="rounded-full p-0.5 text-muted hover:text-ink"
        aria-label={`Remove filter: ${label}`}
      >
        <X className="size-3" aria-hidden="true" />
      </button>
    </span>
  )
}

function Timeline() {
  const perms = useTimelinePerms()
  const { values, page, set } = useUrlFilters(filterKeys)
  const location = useLocation()
  const query = useStockMovementTimeline({
    q: values.q || undefined,
    itemType: (values.itemType || undefined) as TimelineItemType | undefined,
    vehicleUnitId: values.vehicleUnitId || undefined,
    stockItemId: values.stockItemId || undefined,
    movementType: (values.movementType || undefined) as TimelineMovementType | undefined,
    sourceType: (values.sourceType || undefined) as TimelineSourceType | undefined,
    sourceReference: values.sourceReference || undefined,
    actorId: values.actorId || undefined,
    locationId: values.locationId || undefined,
    from: dayStart(values.from),
    to: dayStart(values.to, 1),
    order: values.order === 'asc' ? 'asc' : undefined,
    page,
    limit: PAGE_SIZE,
  })
  const rows = query.data?.items ?? []
  const locations = perms.me.data?.locationScope.locations ?? []
  const actorOptions: SelectOption[] = perms.meId ? [{ value: perms.meId, label: 'Me' }] : []
  if (values.actorId && values.actorId !== perms.meId) {
    const name = rows.find((m) => m.actorId === values.actorId)?.actorName
    actorOptions.push({ value: values.actorId, label: name ?? 'Selected person' })
  }
  const unitPin = values.vehicleUnitId
    ? (rows.find((m) => m.vehicleUnit?.id === values.vehicleUnitId)?.vehicleUnit?.unitNumber ??
      'Selected vehicle')
    : ''
  const skuPin = values.stockItemId
    ? (rows.find((m) => m.stockItem?.id === values.stockItemId)?.stockItem?.sku ?? 'Selected part')
    : ''
  const filtered = filterKeys.some((k) => k !== 'order' && values[k] !== '')
  const clear = () => {
    set(Object.fromEntries(filterKeys.map((k) => [k, ''])))
  }
  const from = { from: location.search }
  // Stable (set is stable), so the search box's pause is not restarted by every render.
  const onSearch = useCallback(
    (q: string) => {
      set({ q })
    },
    [set],
  )

  const cols: Column<TimelineMovement>[] = [
    {
      key: 'event',
      header: 'Movement',
      render: (m) => (
        <span className="flex flex-col">
          <Link to={`${listPath}/${m.id}`} state={from} className={linkClass}>
            {timelineTypeLabels[m.movementType]}
          </Link>
          <time dateTime={m.createdAt} className="text-xs text-muted">
            {movementTime(m.createdAt)}
          </time>
        </span>
      ),
    },
    { key: 'item', header: 'Vehicle / part', render: (m) => <ItemRef m={m} /> },
    { key: 'change', header: 'Change', render: (m) => <Change m={m} /> },
    { key: 'location', header: 'Location', render: (m) => <Where m={m} /> },
    { key: 'actor', header: 'Recorded by', render: (m) => <Actor m={m} /> },
    { key: 'source', header: 'Source', render: (m) => <Source m={m} /> },
    {
      key: 'act',
      header: 'Details',
      render: (m) => (
        <Link
          to={`${listPath}/${m.id}`}
          state={from}
          className={buttonVariants({ variant: 'secondary', size: 'sm' })}
          aria-label={`View ${timelineTypeLabels[m.movementType]} of ${movementTime(m.createdAt)}`}
        >
          View
        </Link>
      ),
    },
  ]

  let body: ReactNode
  if (query.isPending) {
    body = (
      <div role="status" aria-label="Loading stock movements" className="flex flex-col gap-2">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  } else if (query.isError && !query.data) {
    const d = describeTimelineError(query.error)
    if (d.state === 'forbidden')
      body = <AccessDeniedState what="stock movements" requestId={d.requestId} />
    else if (d.state === 'invalid-filter')
      body = (
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
            <Button variant="secondary" size="sm" onClick={clear}>
              Clear filters
            </Button>
          </div>
        </div>
      )
    else
      body = (
        <ErrorState
          error={query.error}
          title="Could not load stock movements"
          onRetry={d.retry ? () => void query.refetch() : undefined}
        />
      )
  } else if (query.data.total === 0 && query.data.page === 1) {
    body = filtered ? (
      <EmptyState title="No movements match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No stock movements yet"
        icon={<History className="size-6 text-muted/80" />}
      >
        Movements appear here when goods receipts are posted, vehicles pass PDI, and parts are
        received, issued or counted.
      </EmptyState>
    )
  } else {
    body = (
      <div
        className={query.isPlaceholderData ? 'opacity-70' : undefined}
        aria-busy={query.isFetching}
      >
        <ResponsiveTable
          caption="Stock movements"
          columns={cols}
          rows={query.data.items}
          rowKey={(m) => m.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['movement', 'movements']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  }

  return (
    <>
      <Toolbar onClear={clear} active={filtered}>
        <SearchBox
          label="Search unit number, chassis or SKU"
          value={values.q}
          onSearch={onSearch}
        />
        {perms.vehicles && perms.parts && (
          <FilterSelect
            label="Item type"
            value={values.itemType}
            options={options(timelineItemTypeSchema.options, itemTypeLabels)}
            allLabel="Vehicles and parts"
            onChange={(v) => {
              set({ itemType: v })
            }}
          />
        )}
        <FilterSelect
          label="Movement"
          value={values.movementType}
          options={options(timelineMovementTypeSchema.options, timelineTypeLabels)}
          allLabel="All movements"
          onChange={(v) => {
            set({ movementType: v })
          }}
        />
        <FilterSelect
          label="Source"
          value={values.sourceType}
          options={options(timelineSourceTypeSchema.options, sourceTypeLabels)}
          allLabel="All sources"
          onChange={(v) => {
            set({ sourceType: v })
          }}
        />
        <CommitInput
          label="Source reference"
          value={values.sourceReference}
          onCommit={(v) => {
            set({ sourceReference: v })
          }}
        />
        <FilterSelect
          label="Recorded by"
          value={values.actorId}
          options={actorOptions}
          allLabel="Anyone"
          onChange={(v) => {
            set({ actorId: v })
          }}
        />
        <FilterSelect
          label="Location"
          value={values.locationId}
          options={locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
          allLabel={perms.me.data?.locationScope.all ? 'All locations' : 'All my locations'}
          onChange={(v) => {
            set({ locationId: v })
          }}
        />
        <DateFilter
          label="From date"
          value={values.from}
          onChange={(v) => {
            set({ from: v })
          }}
        />
        <DateFilter
          label="To date"
          value={values.to}
          onChange={(v) => {
            set({ to: v })
          }}
        />
        <FilterSelect
          label="Order"
          value={values.order === 'asc' ? 'asc' : ''}
          options={[{ value: 'asc', label: 'Oldest first' }]}
          allLabel="Newest first"
          onChange={(v) => {
            set({ order: v })
          }}
        />
        {unitPin && (
          <Pinned
            label={`Vehicle ${unitPin}`}
            onClear={() => {
              set({ vehicleUnitId: '' })
            }}
          />
        )}
        {skuPin && (
          <Pinned
            label={`Part ${skuPin}`}
            onClear={() => {
              set({ stockItemId: '' })
            }}
          />
        )}
      </Toolbar>
      {body}
    </>
  )
}

// ── Detail ──────────────────────────────────────────────────────────────────

export function StockMovementDetailPage() {
  const { movementId } = useParams()
  const location = useLocation()
  const back = (location.state as { from?: string } | null)?.from ?? ''
  const q = useStockMovement(movementId)
  const header = (title: string) => (
    <PageHeader
      title={title}
      breadcrumbs={[
        { label: 'Home', to: '/' },
        { label: 'Stock Movements', to: listPath },
        { label: title },
      ]}
    />
  )
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading the movement…" />
  else if (q.isError) {
    const d = describeTimelineError(q.error)
    body =
      d.state === 'forbidden' ? (
        <AccessDeniedState what="stock movements" requestId={d.requestId} />
      ) : d.state === 'not-found' ? (
        <EmptyState title="Movement not found">{d.message}</EmptyState>
      ) : (
        <ErrorState
          error={q.error}
          title="Could not load the movement"
          onRetry={d.retry ? () => void q.refetch() : undefined}
        />
      )
  } else body = <MovementDetail m={q.data} />
  const title = q.data ? timelineTypeLabels[q.data.movementType] : 'Stock movement'
  return (
    <>
      {header(title)}
      <Link
        to={`${listPath}${back}`}
        className={`${linkClass} mb-4 inline-flex items-center gap-1 text-sm`}
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        Back to stock movements
      </Link>
      <Gate>{body}</Gate>
    </>
  )
}

function MovementDetail({ m }: { m: TimelineMovement }) {
  const perms = useTimelinePerms()
  const s = statusChange(m)
  const items: [string, ReactNode][] = [
    ['Movement', timelineTypeLabels[m.movementType]],
    [
      'Recorded at',
      <time key="t" dateTime={m.createdAt} title={m.createdAt}>
        {movementTime(m.createdAt)}
      </time>,
    ],
    ['Recorded by', <Actor key="a" m={m} />],
    [m.itemType === 'VEHICLE_UNIT' ? 'Vehicle' : 'Part', <ItemRef key="i" m={m} />],
  ]
  if (s) {
    items.push(['Status before', s.from], ['Status after', s.to])
  } else {
    items.push([
      'Quantity',
      <span key="q" className="font-mono">
        {quantityChange(m)}
      </span>,
    ])
  }
  items.push(
    ['Location before', locationLabel(m.fromLocationId, m.fromLocationCode, m.fromLocationName)],
    ['Location after', locationLabel(m.toLocationId, m.toLocationCode, m.toLocationName)],
    ['Source', <Source key="s" m={m} />],
  )
  if (m.supplierName) {
    items.push([
      'Supplier',
      perms.openSupplier && m.supplierId ? (
        <Link key="sp" to={`/procurement/suppliers/${m.supplierId}`} className={linkClass}>
          {m.supplierName}
        </Link>
      ) : (
        m.supplierName
      ),
    ])
  }
  if (m.reason)
    items.push([
      'Reason',
      <span key="r" className="whitespace-pre-wrap">
        {m.reason}
      </span>,
    ])
  if (m.note)
    items.push([
      'Note',
      <span key="n" className="whitespace-pre-wrap">
        {m.note}
      </span>,
    ])
  items.push([
    'Movement ID',
    <span key="id" className="font-mono text-xs">
      {m.id}
    </span>,
  ])

  const more: [string, string][] = []
  if (m.vehicleUnit)
    more.push([`All movements of ${m.vehicleUnit.unitNumber}`, `vehicleUnitId=${m.vehicleUnit.id}`])
  if (m.stockItem)
    more.push([`All movements of ${m.stockItem.sku}`, `stockItemId=${m.stockItem.id}`])
  more.push([
    `All movements recorded by ${m.actorId === perms.meId ? 'you' : (m.actorName ?? 'this person')}`,
    `actorId=${m.actorId}`,
  ])
  if (m.sourceReference && !m.sourceHidden)
    more.push([
      `All movements from ${m.sourceReference}`,
      `sourceType=${m.sourceType}&sourceReference=${encodeURIComponent(m.sourceReference)}`,
    ])
  return (
    <div className="flex flex-col gap-4">
      <Card
        title="Movement"
        description="As recorded in the append-only stock ledger. It cannot be edited or deleted."
      >
        <DefinitionList items={items} />
      </Card>
      <Card title="Related movements">
        <ul className="flex flex-col gap-2 text-sm">
          {more.map(([label, search]) => (
            <li key={search}>
              <Link to={`${listPath}?${search}`} className={linkClass}>
                {label}
              </Link>
            </li>
          ))}
        </ul>
      </Card>
    </div>
  )
}
