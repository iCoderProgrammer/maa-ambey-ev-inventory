import { ArrowLeft, ArrowRight, Download, History, Info, Zap } from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { Link, useLocation, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import {
  AccessDeniedState,
  EmptyState,
  ErrorState,
  LoadingState,
  Skeleton,
} from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { fetchAttachment, formatBytes } from '@/features/attachments/api/attachments'
import { downloadErrorMessage } from '@/features/attachments/messages'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useBrands, useModels, useVariants } from '@/features/catalogue/hooks/useCatalogue'
import { useListParams } from '@/features/catalogue/lib/listParams'
import { formatDate, formatDateTime } from '@/features/catalogue/lib/presentation'
import { InspectionHistory } from '@/features/pdi/components/InspectionHistory'
import { usePdiOverview } from '@/features/pdi/hooks/usePdi'
import { unitStatusLabels } from '@/features/pdi/lib/labels'
import { UnitStatusBadge } from '@/features/pdi/PdiPages'
import { useSupplierList } from '@/features/suppliers/hooks/useSuppliers'
import {
  unitStatusSchema,
  vehicleUnitSorts,
  type VehicleUnit,
  type VehicleUnitDetail,
  type VehicleUnitDocument,
  type VehicleUnitMovement,
  type VehicleUnitSort,
  type VehicleUnitStatus,
} from './api/vehicleUnits'
import { AgeingCell, AgeingSummaryPanel } from './components/Ageing'
import { ageingRows } from './lib/ageingRows'
import {
  useAgeingConfiguration,
  useAgeingSummary,
  useVehicleUnit,
  useVehicleUnitList,
  useVehicleUnitMovements,
  useVehicleUnitPerms,
} from './hooks/useVehicleUnits'
import { originStatusLabels } from './lib/ageing'
import {
  describeVehicleUnitReadError,
  documentTypeLabels,
  isSaleable,
  movementTypeLabels,
  notSaleableReason,
  sortLabels,
  unitCostText,
} from './lib/presentation'

/**
 * Vehicle inventory list and detail (D-08). Everything comes from the vehicle
 * unit API; filters, search and paging run on the server and live in the URL.
 * Read-only: no reservation, sale, transfer or status change starts here.
 */

const listPath = '/inventory/vehicles'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Vehicle Units', to: listPath },
]
const linkClass = 'font-medium text-primary hover:underline'
const PAGE_SIZE = 25
const listKeys = [
  'q',
  'status',
  'saleable',
  'brandId',
  'modelId',
  'variantId',
  'locationId',
  'supplierId',
  'receivedFrom',
  'receivedTo',
  'minAgeDays',
  'maxAgeDays',
  'ageingBand',
  'originStatus',
  'sort',
] as const

/** Gate: GET /me must grant vehicle_units.view; the API checks every request again. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = useVehicleUnitPerms()
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
  else if (!view) body = <AccessDeniedState what="vehicle units" />
  if (body === children) return <>{children}</>
  return (
    <>
      {header}
      {body}
    </>
  )
}

/** Status badge plus, for anything but AVAILABLE, a plain "not saleable" marker. */
function StatusCell({ status }: { status: VehicleUnitStatus }) {
  return (
    <span className="flex flex-col items-start gap-0.5">
      <UnitStatusBadge status={status} />
      {!isSaleable(status) && <span className="text-xs text-muted">Not saleable</span>}
    </span>
  )
}

function identifierLines(u: VehicleUnit) {
  const extra = [
    u.motorSerial && `Motor ${u.motorSerial}`,
    u.batterySerial && `Battery ${u.batterySerial}`,
  ].filter(Boolean)
  return (
    <span className="flex flex-col">
      <span className="font-mono">{u.chassisNumber ?? '—'}</span>
      {extra.length > 0 && (
        <span className="font-mono text-xs text-muted">{extra.join(' · ')}</span>
      )}
    </span>
  )
}

function locationText(code: string | null, name: string | null) {
  if (code === null) return 'No location'
  return name ? `${code} — ${name}` : code
}

// ── List ────────────────────────────────────────────────────────────────────

export function VehicleUnitsPage() {
  return (
    <>
      <PageHeader
        title="Vehicle Units"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Vehicle Units' }]}
        description="Every registered scooter as an individual unit: identifiers, status, location and where it came from."
      />
      <Gate>
        <p
          role="note"
          className="mb-4 flex items-start gap-2 rounded-card border border-info/30 bg-info-subtle px-4 py-3 text-sm text-ink"
        >
          <Info className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
          <span>
            Only <strong>Available</strong> units are saleable stock. Units awaiting PDI, on hold,
            rejected, on display, reserved, sold or delivered are never offered for sale.
          </span>
        </p>
        <VehicleList />
      </Gate>
    </>
  )
}

/** A whole number of days, committed on blur or Enter (not on every keystroke). */
function DaysInput({
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
    <div className="flex min-w-0 flex-col gap-1.5 sm:w-32">
      <label htmlFor={id} className="text-xs font-medium text-muted">
        {label}
      </label>
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={0}
        step={1}
        value={draft}
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

/** A URL value as a number when it is one; otherwise the raw text, so the server reports it. */
function numberOrRaw(v: string): number | string | undefined {
  if (!v) return undefined
  return /^\d+$/.test(v) ? Number(v) : v
}

function VehicleList() {
  const perms = useVehicleUnitPerms()
  const { values, page, set } = useListParams(listKeys)
  const location = useLocation()
  const query = useVehicleUnitList({
    q: values.q || undefined,
    status: (values.status || undefined) as VehicleUnitStatus | undefined,
    saleable: values.saleable === '' ? undefined : values.saleable === 'true',
    brandId: values.brandId || undefined,
    modelId: values.modelId || undefined,
    variantId: values.variantId || undefined,
    locationId: values.locationId || undefined,
    supplierId: values.supplierId || undefined,
    receivedFrom: values.receivedFrom || undefined,
    receivedTo: values.receivedTo || undefined,
    minAgeDays: numberOrRaw(values.minAgeDays),
    maxAgeDays: numberOrRaw(values.maxAgeDays),
    ageingBand: values.ageingBand || undefined,
    originStatus: (values.originStatus || undefined) as 'KNOWN' | 'MISSING' | undefined,
    sort: (values.sort || undefined) as VehicleUnitSort | undefined,
    page,
    limit: PAGE_SIZE,
  })
  // E-07: the same filters, counted by ageing band on the server.
  const ageingSummary = useAgeingSummary({
    q: values.q || undefined,
    status: (values.status || undefined) as VehicleUnitStatus | undefined,
    saleable: values.saleable === '' ? undefined : values.saleable === 'true',
    brandId: values.brandId || undefined,
    modelId: values.modelId || undefined,
    variantId: values.variantId || undefined,
    locationId: values.locationId || undefined,
    supplierId: values.supplierId || undefined,
    receivedFrom: values.receivedFrom || undefined,
    receivedTo: values.receivedTo || undefined,
    minAgeDays: numberOrRaw(values.minAgeDays),
    maxAgeDays: numberOrRaw(values.maxAgeDays),
    ageingBand: values.ageingBand || undefined,
    originStatus: (values.originStatus || undefined) as 'KNOWN' | 'MISSING' | undefined,
  })
  const ageingConfig = useAgeingConfiguration()
  const bandOptions = (ageingConfig.data?.bandSet?.bands ?? []).map((b) => ({
    value: b.key,
    label: b.label,
  }))
  // Option lists come from the catalogue, supplier and access APIs (never hardcoded).
  const brands = useBrands({ limit: 100 }, { enabled: perms.viewCatalogue })
  const models = useModels(
    { brandId: values.brandId, limit: 100 },
    { enabled: perms.viewCatalogue && Boolean(values.brandId) },
  )
  const variants = useVariants(
    { modelId: values.modelId, limit: 100 },
    { enabled: perms.viewCatalogue && Boolean(values.modelId) },
  )
  const suppliers = useSupplierList({ limit: 100 }, { enabled: perms.viewSuppliers })
  const locations = perms.me.data?.locationScope.locations ?? []
  // The debounced search must apply on top of the filters chosen meanwhile: call the
  // latest setter (an older one would rebuild the URL from stale parameters).
  const setRef = useRef(set)
  useEffect(() => {
    setRef.current = set
  })
  const onSearch = useCallback((q: string) => {
    setRef.current({ q })
  }, [])
  const filtered = listKeys.some((k) => k !== 'sort' && values[k] !== '')
  const clear = () => {
    set(Object.fromEntries(listKeys.map((k) => [k, ''])))
  }
  const from = { from: location.search }

  const cols: Column<VehicleUnit>[] = [
    {
      key: 'unit',
      header: 'Unit',
      render: (u) => (
        <span className="flex flex-col">
          <Link to={`${listPath}/${u.id}`} state={from} className={`${linkClass} font-mono`}>
            {u.unitNumber}
          </Link>
          <span className="text-xs text-muted">
            {u.brandName} · {u.modelName} · {u.variantName}
          </span>
        </span>
      ),
    },
    { key: 'ids', header: 'Chassis / frame', render: identifierLines },
    { key: 'status', header: 'Status', render: (u) => <StatusCell status={u.status} /> },
    {
      key: 'location',
      header: 'Location',
      render: (u) => locationText(u.locationCode, u.locationName),
    },
    { key: 'supplier', header: 'Supplier', render: (u) => u.supplierName ?? '—' },
    { key: 'received', header: 'Received', render: (u) => formatDate(u.receivedDate) },
    { key: 'age', header: 'Stock age', render: (u) => <AgeingCell ageing={u.ageing} /> },
    ...(perms.viewCost
      ? [
          {
            key: 'cost',
            header: 'Cost',
            render: (u: VehicleUnit) => unitCostText(u, perms.viewCost) ?? '—',
          },
        ]
      : []),
    {
      key: 'act',
      header: 'Actions',
      render: (u) => (
        <Link
          to={`${listPath}/${u.id}`}
          state={from}
          className={buttonVariants({ variant: 'secondary', size: 'sm' })}
          aria-label={`View ${u.unitNumber}`}
        >
          View
        </Link>
      ),
    },
  ]

  let body: ReactNode
  if (query.isPending) {
    body = (
      <div role="status" aria-label="Loading vehicle units" className="flex flex-col gap-2">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  } else if (query.isError && !query.data) {
    const d = describeVehicleUnitReadError(query.error)
    if (d.state === 'forbidden')
      body = <AccessDeniedState what="vehicle units" requestId={d.requestId} />
    else if (d.state === 'invalid-filter') {
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
    } else {
      body = (
        <ErrorState
          error={query.error}
          title="Could not load vehicle units"
          onRetry={d.retry ? () => void query.refetch() : undefined}
        />
      )
    }
  } else if (query.data.total === 0 && query.data.page === 1) {
    body = filtered ? (
      <EmptyState title="No vehicles match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState title="No vehicle units yet" icon={<Zap className="size-6 text-muted/80" />}>
        Vehicles appear here when a goods receipt is posted: each accepted unit becomes a vehicle
        awaiting PDI.
      </EmptyState>
    )
  } else {
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Vehicle units"
          columns={cols}
          rows={query.data.items}
          rowKey={(u) => u.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['vehicle', 'vehicles']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  }

  const catalogueHint = perms.viewCatalogue ? undefined : 'Not available to you'
  return (
    <>
      <Toolbar active={filtered} onClear={clear}>
        <SearchBox
          label="Search unit number, chassis, motor or battery"
          value={values.q}
          onSearch={onSearch}
        />
        <FilterSelect
          label="Status"
          value={values.status}
          options={unitStatusSchema.options.map((s) => ({ value: s, label: unitStatusLabels[s] }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Saleable"
          value={values.saleable}
          options={[
            { value: 'true', label: 'Saleable (Available) only' },
            { value: 'false', label: 'Not saleable' },
          ]}
          allLabel="All units"
          onChange={(v) => {
            set({ saleable: v })
          }}
        />
        <FilterSelect
          label="Brand"
          value={values.brandId}
          options={(brands.data?.items ?? []).map((b) => ({ value: b.id, label: b.name }))}
          allLabel={catalogueHint ?? 'All brands'}
          disabled={!perms.viewCatalogue}
          onChange={(v) => {
            set({ brandId: v, modelId: '', variantId: '' })
          }}
        />
        <FilterSelect
          label="Model"
          value={values.modelId}
          options={(models.data?.items ?? []).map((m) => ({ value: m.id, label: m.name }))}
          allLabel={catalogueHint ?? (values.brandId ? 'All models' : 'Choose a brand first')}
          disabled={!perms.viewCatalogue || !values.brandId}
          onChange={(v) => {
            set({ modelId: v, variantId: '' })
          }}
        />
        <FilterSelect
          label="Variant"
          value={values.variantId}
          options={(variants.data?.items ?? []).map((v) => ({ value: v.id, label: v.name }))}
          allLabel={catalogueHint ?? (values.modelId ? 'All variants' : 'Choose a model first')}
          disabled={!perms.viewCatalogue || !values.modelId}
          onChange={(v) => {
            set({ variantId: v })
          }}
        />
        <FilterSelect
          label="Location"
          value={values.locationId}
          options={locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
          allLabel={locations.length === 0 ? 'No assigned location' : 'All my locations'}
          disabled={locations.length === 0}
          onChange={(v) => {
            set({ locationId: v })
          }}
        />
        <FilterSelect
          label="Supplier"
          value={values.supplierId}
          options={(suppliers.data?.items ?? []).map((s) => ({
            value: s.id,
            label: s.tradeName ?? s.legalName,
          }))}
          allLabel={perms.viewSuppliers ? 'All suppliers' : 'Not available to you'}
          disabled={!perms.viewSuppliers}
          onChange={(v) => {
            set({ supplierId: v })
          }}
        />
        <DateFilter
          label="Received from"
          value={values.receivedFrom}
          onChange={(v) => {
            set({ receivedFrom: v })
          }}
        />
        <DateFilter
          label="Received to"
          value={values.receivedTo}
          onChange={(v) => {
            set({ receivedTo: v })
          }}
        />
        <DaysInput
          label="Min. days in stock"
          value={values.minAgeDays}
          onCommit={(v) => {
            set({ minAgeDays: v })
          }}
        />
        <DaysInput
          label="Max. days in stock"
          value={values.maxAgeDays}
          onCommit={(v) => {
            set({ maxAgeDays: v })
          }}
        />
        <FilterSelect
          label="Ageing band"
          value={values.ageingBand}
          options={bandOptions}
          allLabel={
            ageingConfig.data && !ageingConfig.data.configured ? 'No bands configured' : 'All bands'
          }
          disabled={bandOptions.length === 0}
          onChange={(v) => {
            set({ ageingBand: v, ...(v ? { originStatus: '' } : {}) })
          }}
        />
        <FilterSelect
          label="Origin date"
          value={values.originStatus}
          options={[
            { value: 'KNOWN', label: originStatusLabels.KNOWN },
            { value: 'MISSING', label: originStatusLabels.MISSING },
          ]}
          allLabel="All vehicles"
          onChange={(v) => {
            set({
              originStatus: v,
              ...(v === 'MISSING' ? { ageingBand: '', minAgeDays: '', maxAgeDays: '' } : {}),
            })
          }}
        />
        <FilterSelect
          label="Sort by"
          value={values.sort}
          options={vehicleUnitSorts
            .filter((s) => s !== '-createdAt')
            .map((s) => ({ value: s, label: sortLabels[s] }))}
          allLabel={sortLabels['-createdAt']}
          onChange={(v) => {
            set({ sort: v })
          }}
        />
      </Toolbar>
      <AgeingSummaryPanel
        summary={ageingSummary.data}
        isPending={ageingSummary.isPending}
        error={ageingSummary.error}
        onRetry={() => void ageingSummary.refetch()}
        activeBand={values.ageingBand}
        activeOrigin={values.originStatus}
        onBand={(key) => {
          set({ ageingBand: key, originStatus: '' })
        }}
        onMissing={() => {
          set(
            values.originStatus === 'MISSING'
              ? { originStatus: '' }
              : { originStatus: 'MISSING', ageingBand: '', minAgeDays: '', maxAgeDays: '' },
          )
        }}
      />
      {body}
    </>
  )
}

/** Field names the server may report on an invalid list request, in screen words. */
const filterLabels: Record<string, string> = {
  q: 'Search',
  status: 'Status',
  saleable: 'Saleable',
  brandId: 'Brand',
  modelId: 'Model',
  variantId: 'Variant',
  locationId: 'Location',
  supplierId: 'Supplier',
  receivedFrom: 'Received from',
  receivedTo: 'Received to',
  minAgeDays: 'Min. days in stock',
  maxAgeDays: 'Max. days in stock',
  ageingBand: 'Ageing band',
  originStatus: 'Origin date',
  sort: 'Sort by',
  page: 'Page',
  limit: 'Page size',
}

// ── Detail ──────────────────────────────────────────────────────────────────

export function VehicleUnitDetailPage() {
  const { unitId = '' } = useParams()
  const location = useLocation()
  const fromState: unknown = location.state
  const back =
    typeof fromState === 'object' &&
    fromState !== null &&
    'from' in fromState &&
    typeof fromState.from === 'string' &&
    fromState.from.startsWith('?')
      ? `${listPath}${fromState.from}`
      : listPath
  const header = (
    <PageHeader title="Vehicle unit" breadcrumbs={[...crumbs, { label: 'Vehicle unit' }]} />
  )
  return (
    <Gate header={header}>
      <DetailBody id={unitId} back={back} header={header} />
    </Gate>
  )
}

function BackLink({ to }: { to: string }) {
  return (
    <Link to={to} className={`${linkClass} mb-4 inline-flex items-center gap-1 text-sm`}>
      <ArrowLeft className="size-4" aria-hidden="true" />
      Back to vehicle units
    </Link>
  )
}

function DetailBody({ id, back, header }: { id: string; back: string; header: ReactNode }) {
  const q = useVehicleUnit(id)
  const perms = useVehicleUnitPerms()
  if (!q.data) {
    let body: ReactNode = (
      <div role="status" aria-label="Loading vehicle unit" className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
    if (q.isError) {
      const d = describeVehicleUnitReadError(q.error)
      if (d.state === 'not-found')
        body = (
          <EmptyState title="Vehicle unit not found">
            {d.message} Check the link, or find the vehicle from the list.
          </EmptyState>
        )
      else if (d.state === 'forbidden')
        body = <AccessDeniedState what="vehicle units" requestId={d.requestId} />
      else
        body = (
          <ErrorState
            error={q.error}
            title="Could not load the vehicle unit"
            onRetry={d.retry ? () => void q.refetch() : undefined}
          />
        )
    }
    return (
      <>
        {header}
        <BackLink to={back} />
        {body}
      </>
    )
  }
  const u = q.data
  const cost = unitCostText(u, perms.viewCost)
  return (
    <>
      <PageHeader
        title={u.unitNumber}
        breadcrumbs={[...crumbs, { label: u.unitNumber }]}
        description={`${u.brandName} · ${u.modelName} · ${u.variantName}`}
        status={<UnitStatusBadge status={u.status} />}
      />
      <BackLink to={back} />
      {isSaleable(u.status) ? (
        <p
          role="status"
          className="mb-4 rounded-card border border-success/30 bg-success-subtle px-4 py-3 text-sm"
        >
          Available — this vehicle is saleable stock.
        </p>
      ) : (
        <p
          role="status"
          className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm"
        >
          {notSaleableReason[u.status] ?? 'Not saleable.'}
        </p>
      )}
      <div className="grid gap-4 lg:grid-cols-2">
        <Card title="Vehicle">
          <DefinitionList
            items={[
              ['Unit number', <span className="font-mono">{u.unitNumber}</span>],
              ['Internal ID', <span className="font-mono text-xs break-all">{u.id}</span>],
              ['Brand', u.brandName],
              ['Model', u.modelName],
              ['Variant', u.variantName],
              ['Status', <UnitStatusBadge status={u.status} />],
              ['Location', locationText(u.locationCode, u.locationName)],
              ...ageingRows(u.ageing),
              ...(cost !== null
                ? ([['Acquisition cost (excl. tax)', cost]] as [string, string][])
                : []),
            ]}
          />
        </Card>
        <Card title="Identifiers" description="As captured on the goods receipt.">
          <DefinitionList
            items={[
              ['Chassis / frame', <Ident value={u.chassisNumber} />],
              ['Motor', <Ident value={u.motorSerial} />],
              ['Battery', <Ident value={u.batterySerial} />],
              ['Charger', <Ident value={u.chargerSerial} />],
            ]}
          />
        </Card>
        <ProcurementCard unit={u} />
        <DocumentsCard unit={u} />
      </div>
      <div className="mt-4 flex flex-col gap-4">
        <PdiSection unitId={u.id} />
        <MovementHistory unitId={u.id} />
      </div>
    </>
  )
}

function Ident({ value }: { value: string | null }) {
  return value ? (
    <span className="font-mono">{value}</span>
  ) : (
    <span className="text-muted">Not recorded</span>
  )
}

function ProcurementCard({ unit: u }: { unit: VehicleUnitDetail }) {
  const perms = useVehicleUnitPerms()
  const s = u.source
  // The registration entry (the only history action so far; later changes are movements).
  const registered = u.events[0]
  const po =
    s.purchaseOrderNumber === null ? (
      '—'
    ) : perms.viewPO && s.purchaseOrderId ? (
      <Link
        to={`/procurement/purchase-orders/${s.purchaseOrderId}`}
        className={`${linkClass} font-mono`}
      >
        {s.purchaseOrderNumber}
      </Link>
    ) : (
      <span className="font-mono">{s.purchaseOrderNumber}</span>
    )
  const grn =
    s.goodsReceiptNumber === null ? (
      '—'
    ) : perms.viewGRN && s.goodsReceiptId ? (
      <Link
        to={`/procurement/goods-receipts/${s.goodsReceiptId}`}
        className={`${linkClass} font-mono`}
      >
        {s.goodsReceiptNumber}
      </Link>
    ) : (
      <span className="font-mono">{s.goodsReceiptNumber}</span>
    )
  // Opening stock (G-07): no purchase order or goods receipt exists; the unit names its import line.
  const opening =
    s.type === 'OPENING_STOCK' && s.openingStockBatchId ? (
      <Link
        to={`/inventory/opening-stock/${s.openingStockBatchId}`}
        className={`${linkClass} font-mono`}
      >
        {s.openingStockBatchReference ?? 'Import batch'}
        {s.openingStockRowNumber !== null ? ` · line ${String(s.openingStockRowNumber)}` : ''}
      </Link>
    ) : null
  return (
    <Card title="Procurement" description="Where this vehicle came from.">
      <DefinitionList
        items={[
          ...(opening ? ([['Opening stock import', opening]] as [string, ReactNode][]) : []),
          ['Supplier', s.supplierName ?? '—'],
          ['Purchase order', po],
          ['Goods receipt', grn],
          ['Supplier reference', s.supplierReference ?? '—'],
          ['Received', formatDate(s.receivedDate)],
          [
            'Registered',
            registered
              ? `${formatDateTime(registered.createdAt)} by ${registered.actorName ?? 'Unknown user'}`
              : '—',
          ],
        ]}
      />
    </Card>
  )
}

function DocumentsCard({ unit: u }: { unit: VehicleUnitDetail }) {
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const download = async (d: VehicleUnitDocument) => {
    setError(null)
    setBusy(d.attachmentId)
    try {
      const blob = await fetchAttachment(d.attachmentId, 'attachment')
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = d.fileName
      link.rel = 'noopener'
      document.body.append(link)
      link.click()
      link.remove()
      setTimeout(() => {
        URL.revokeObjectURL(url)
      }, 10_000)
    } catch (e) {
      setError(downloadErrorMessage(e))
    } finally {
      setBusy(null)
    }
  }
  let body: ReactNode
  if (u.documentsHidden)
    body = (
      <p className="text-sm text-muted">
        You do not have permission to open documents, so none are listed.
      </p>
    )
  else if (u.documents.length === 0)
    body = (
      <p className="text-sm text-muted">
        No documents available to you for this vehicle (goods receipt documents and PDI evidence
        appear here).
      </p>
    )
  else
    body = (
      <ul aria-label="Vehicle documents" className="flex flex-col divide-y divide-border">
        {u.documents.map((d) => (
          <li
            key={d.attachmentId}
            className="flex flex-wrap items-center justify-between gap-2 py-2 text-sm"
          >
            <span className="flex min-w-0 flex-col">
              <span className="font-medium break-all">{d.fileName}</span>
              <span className="text-xs text-muted">
                {documentTypeLabels[d.documentType]} · {d.sourceReference} ·{' '}
                {formatBytes(d.sizeBytes)}
              </span>
            </span>
            <Button
              variant="secondary"
              size="sm"
              disabled={busy !== null}
              onClick={() => void download(d)}
              aria-label={`Download ${d.fileName}`}
            >
              <Download className="size-3.5" aria-hidden="true" />
              {busy === d.attachmentId ? 'Opening…' : 'Download'}
            </Button>
          </li>
        ))}
      </ul>
    )
  return (
    <Card title="Documents" description="Private files: each download is checked again.">
      {body}
      {error && (
        <p role="alert" className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </Card>
  )
}

function PdiSection({ unitId }: { unitId: string }) {
  const perms = useVehicleUnitPerms()
  const q = usePdiOverview(unitId)
  const pdiLink = (
    <Link
      to={`/procurement/pdi/${unitId}`}
      className={buttonVariants({ variant: 'secondary', size: 'sm' })}
    >
      {perms.pdiRecord && q.data?.allowedDispositions.length ? 'Open PDI inspection' : 'Open PDI'}
    </Link>
  )
  if (q.isPending) return <LoadingState label="Loading PDI…" />
  if (q.isError)
    return (
      <Card title="Inspection history" actions={pdiLink}>
        <ErrorState
          error={q.error}
          title="Could not load PDI information"
          onRetry={() => void q.refetch()}
        />
      </Card>
    )
  return (
    <div className="flex flex-col gap-2">
      <div className="flex justify-end">{pdiLink}</div>
      <InspectionHistory
        inspections={q.data.inspections}
        meId={perms.me.data?.user.id}
        canSeeEvidence={perms.attachmentsView}
      />
    </div>
  )
}

const MOVEMENTS_PER_PAGE = 20

function MovementHistory({ unitId }: { unitId: string }) {
  const [page, setPage] = useState(1)
  const q = useVehicleUnitMovements(unitId, { order: 'asc', page, limit: MOVEMENTS_PER_PAGE })
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading movement history…" />
  else if (q.isError && !q.data)
    body = (
      <ErrorState
        error={q.error}
        title="Could not load the movement history"
        onRetry={() => void q.refetch()}
      />
    )
  else if (q.data.total === 0)
    body = (
      <EmptyState title="No movements yet" icon={<History className="size-6 text-muted/80" />}>
        Stock movements of this vehicle appear here.
      </EmptyState>
    )
  else
    body = (
      <div className={q.isPlaceholderData ? 'opacity-70' : undefined}>
        <ol aria-label="Stock movements" className="flex flex-col gap-2">
          {q.data.items.map((m) => (
            <MovementEntry key={m.id} m={m} unitId={unitId} />
          ))}
        </ol>
        <Pagination
          page={q.data.page}
          limit={q.data.limit}
          total={q.data.total}
          noun={['movement', 'movements']}
          onPage={setPage}
        />
      </div>
    )
  return (
    <Card
      title="Movement history"
      description="Every stock movement of this vehicle, oldest first. Append-only: nothing here can be edited."
    >
      {body}
    </Card>
  )
}

function MovementSource({ m, unitId }: { m: VehicleUnitMovement; unitId: string }) {
  const perms = useVehicleUnitPerms()
  if (m.sourceType === 'GOODS_RECEIPT' && m.sourceReference) {
    return perms.viewGRN && m.sourceId ? (
      <Link to={`/procurement/goods-receipts/${m.sourceId}`} className={`${linkClass} font-mono`}>
        {m.sourceReference}
      </Link>
    ) : (
      <span className="font-mono">{m.sourceReference}</span>
    )
  }
  if (m.sourceType === 'PDI_INSPECTION' && m.sourceReference) {
    return (
      <Link to={`/procurement/pdi/${unitId}`} className={`${linkClass} font-mono`}>
        {m.sourceReference}
      </Link>
    )
  }
  if (m.sourceType === 'OPENING_STOCK' && m.sourceReference && m.sourceId) {
    return (
      <Link to={`/inventory/opening-stock/${m.sourceId}`} className={`${linkClass} font-mono`}>
        {m.sourceReference}
      </Link>
    )
  }
  return <span>Staff action</span>
}

function MovementEntry({ m, unitId }: { m: VehicleUnitMovement; unitId: string }) {
  const perms = useVehicleUnitPerms()
  const meId = perms.me.data?.user.id
  const to = m.toLocationId
    ? m.toLocationCode
      ? locationText(m.toLocationCode, m.toLocationName)
      : 'A location not visible to you'
    : 'No location'
  return (
    <li className="rounded-control border border-border bg-surface px-4 py-3 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">{movementTypeLabels[m.movementType]}</span>
        <span className="inline-flex items-center gap-1 text-xs text-muted">
          {m.fromStatus ? unitStatusLabels[m.fromStatus] : 'Not in stock'}
          <ArrowRight className="size-3" aria-label="to" />
          {unitStatusLabels[m.toStatus]}
        </span>
      </div>
      <p className="mt-1 flex flex-wrap gap-x-2 text-xs text-muted">
        <span>{formatDateTime(m.createdAt)}</span>
        <span>· {m.actorId === meId ? 'You' : (m.actorName ?? 'Unknown user')}</span>
        <span>· {to}</span>
        <span>
          · Source: <MovementSource m={m} unitId={unitId} />
        </span>
      </p>
      {m.note && <p className="mt-1 whitespace-pre-wrap text-ink">{m.note}</p>}
    </li>
  )
}
