import { Pencil, Plus, Tags, Wrench } from 'lucide-react'
import { useCallback, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { ArchivedBanner, DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useBrands } from '@/features/catalogue/hooks/useCatalogue'
import { useListParams } from '@/features/catalogue/lib/listParams'
import { formatDateTime, personLabel } from '@/features/catalogue/lib/presentation'
import { SupplierStatusBadge } from '@/features/suppliers/components/SupplierBadges'
import { useSupplierList } from '@/features/suppliers/hooks/useSuppliers'
import { ApiError } from '@/lib/api/client'
import {
  formatLevel,
  type ItemCategory,
  type ItemStatus,
  type StockItem,
  type StockItemCompatibility,
  type StockItemDetail,
  type StockItemSupplier,
  type TrackingMode,
} from './api/stockItems'
import {
  ArchiveCategoryDialog,
  ArchiveStockItemDialog,
  CategoryFormDialog,
  StockItemFormDialog,
} from './components/StockItemDialogs'
import { PartsStockPanel } from './components/PartsStockPanel'
import {
  useCategoryList,
  useStockItem,
  useStockItemList,
  useStockItemPerms,
} from './hooks/useStockItems'
import { itemStatusLabels, trackingModeHelp, trackingModeLabels } from './lib/labels'

const home = { label: 'Home', to: '/' }
const crumbs = [home, { label: 'Spare Parts & Accessories', to: '/inventory/parts' }]
const linkClass = 'font-medium text-primary hover:underline'

/** Gate for every page: stock_items.view, checked again by the API.
 * `header` keeps the page heading and breadcrumbs while checking or denied. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = useStockItemPerms()
  if (me.isPending)
    return (
      <>
        {header}
        <LoadingState label="Checking your permissions…" />
      </>
    )
  if (me.isError && !me.data)
    return (
      <>
        {header}
        <ErrorState
          error={me.error}
          title="Could not check your permissions"
          onRetry={() => void me.refetch()}
        />
      </>
    )
  if (!view)
    return (
      <>
        {header}
        <AccessDeniedState what="spare parts and accessories" />
      </>
    )
  return <>{children}</>
}

export function StatusBadge({ status }: { status: ItemStatus }) {
  return (
    <Badge tone={status === 'ACTIVE' ? 'success' : 'neutral'}>{itemStatusLabels[status]}</Badge>
  )
}

function TrackingBadge({ mode }: { mode: TrackingMode }) {
  return <Badge tone={mode === 'SERIAL' ? 'info' : 'neutral'}>{trackingModeLabels[mode]}</Badge>
}

function MasterDataNotice() {
  return (
    <p
      role="note"
      className="mb-4 rounded-card border border-border bg-neutral-subtle px-4 py-3 text-sm text-ink"
    >
      This is the parts and accessories <strong>master list</strong>. Editing a SKU never changes
      stock: quantities change only through recorded receipts and issues (on each SKU&rsquo;s page).
    </p>
  )
}

// ── SKU list ────────────────────────────────────────────────────────────────

export function PartsPage() {
  const { manage, view } = useStockItemPerms()
  const [adding, setAdding] = useState(false)
  const navigate = useNavigate()
  const header = (
    <PageHeader
      title="Spare Parts & Accessories"
      breadcrumbs={[home, { label: 'Spare Parts & Accessories' }]}
      description="SKU master: code, category, unit of measure, tracking mode, suppliers and compatibility."
      actions={
        <>
          {view && (
            <Link to="/inventory/parts/categories" className={linkClass}>
              Categories
            </Link>
          )}
          {manage && (
            <Button
              onClick={() => {
                setAdding(true)
              }}
            >
              <Plus className="size-4" aria-hidden="true" />
              Add SKU
            </Button>
          )}
        </>
      }
    />
  )
  return (
    <>
      {header}
      <Gate>
        <PartsList
          canAdd={manage}
          onAdd={() => {
            setAdding(true)
          }}
        />
      </Gate>
      <StockItemFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        onSaved={(d) => void navigate(`/inventory/parts/${d.id}`)}
      />
    </>
  )
}

const listKeys = ['q', 'status', 'categoryId', 'trackingMode', 'supplierId', 'brandId'] as const

function PartsList({ canAdd, onAdd }: { canAdd: boolean; onAdd: () => void }) {
  const perms = useStockItemPerms()
  const { values, page, set } = useListParams(listKeys)
  const categories = useCategoryList({ limit: 100 })
  // Filter choices only through the APIs the caller may use.
  const suppliers = useSupplierList({ limit: 100 }, { enabled: perms.suppliers })
  const brands = useBrands({ limit: 100 }, { enabled: perms.catalogue })
  const query = useStockItemList({
    q: values.q || undefined,
    status: (values.status || undefined) as ItemStatus | undefined,
    categoryId: values.categoryId || undefined,
    trackingMode: (values.trackingMode || undefined) as TrackingMode | undefined,
    supplierId: perms.suppliers ? values.supplierId || undefined : undefined,
    brandId: values.brandId || undefined,
    page,
    limit: 25,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = listKeys.some((k) => Boolean(values[k]))
  const cols: Column<StockItem>[] = [
    {
      key: 'sku',
      header: 'SKU',
      render: (i) => (
        <div className="flex flex-col">
          <Link to={`/inventory/parts/${i.id}`} className={`${linkClass} font-mono`}>
            {i.sku}
          </Link>
          <span className="text-xs text-muted">{i.name}</span>
        </div>
      ),
    },
    { key: 'category', header: 'Category', render: (i) => i.categoryName },
    {
      key: 'uom',
      header: 'Unit',
      render: (i) => <span className="font-mono text-xs">{i.uom}</span>,
    },
    { key: 'tracking', header: 'Tracking', render: (i) => <TrackingBadge mode={i.trackingMode} /> },
    { key: 'reorder', header: 'Reorder level', render: (i) => formatLevel(i.reorderLevel) ?? '—' },
    { key: 'status', header: 'Status', render: (i) => <StatusBadge status={i.status} /> },
  ]
  const add = canAdd ? (
    <Button onClick={onAdd}>
      <Plus className="size-4" aria-hidden="true" />
      Add SKU
    </Button>
  ) : null

  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading SKUs…" />
  else if (query.isError && !query.data)
    body = (
      <ErrorState
        error={query.error}
        title="Could not load SKUs"
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.data.total === 0)
    body = filtered ? (
      <EmptyState title="No SKUs match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No SKUs yet"
        icon={<Wrench className="size-6 text-muted/80" />}
        action={add}
      >
        Add parts and accessories from supplier paperwork. Nothing is filled in for you.
      </EmptyState>
    )
  else
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="SKUs"
          columns={cols}
          rows={query.data.items}
          rowKey={(i) => i.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['SKU', 'SKUs']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )

  return (
    <>
      <MasterDataNotice />
      <Toolbar
        active={filtered}
        onClear={() => {
          set(Object.fromEntries(listKeys.map((k) => [k, ''])))
        }}
      >
        <SearchBox label="Search SKU code or name" value={values.q} onSearch={onSearch} />
        <FilterSelect
          label="Status"
          value={values.status}
          options={Object.entries(itemStatusLabels).map(([value, label]) => ({ value, label }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Category"
          value={values.categoryId}
          options={(categories.data?.items ?? []).map((c) => ({ value: c.id, label: c.name }))}
          allLabel="Any category"
          onChange={(v) => {
            set({ categoryId: v })
          }}
        />
        <FilterSelect
          label="Tracking"
          value={values.trackingMode}
          options={Object.entries(trackingModeLabels).map(([value, label]) => ({ value, label }))}
          allLabel="Any tracking"
          onChange={(v) => {
            set({ trackingMode: v })
          }}
        />
        {perms.suppliers && (
          <FilterSelect
            label="Supplier"
            value={values.supplierId}
            options={(suppliers.data?.items ?? []).map((s) => ({
              value: s.id,
              label: s.legalName,
            }))}
            allLabel="Any supplier"
            onChange={(v) => {
              set({ supplierId: v })
            }}
          />
        )}
        {perms.catalogue && (
          <FilterSelect
            label="Compatible brand"
            value={values.brandId}
            options={(brands.data?.items ?? []).map((b) => ({ value: b.id, label: b.name }))}
            allLabel="Any brand"
            onChange={(v) => {
              set({ brandId: v })
            }}
          />
        )}
      </Toolbar>
      {body}
    </>
  )
}

// ── SKU detail ──────────────────────────────────────────────────────────────

export function PartDetailPage() {
  const { itemId = '' } = useParams()
  return (
    <Gate header={<PageHeader title="SKU" breadcrumbs={[...crumbs, { label: 'SKU' }]} />}>
      <PartDetailBody id={itemId} />
    </Gate>
  )
}

function PartDetailBody({ id }: { id: string }) {
  const q = useStockItem(id)
  const perms = useStockItemPerms()
  const [open, setOpen] = useState<'edit' | 'archive' | null>(null)
  const close = () => {
    setOpen(null)
  }
  if (!q.data) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <>
        <PageHeader title="SKU" breadcrumbs={[...crumbs, { label: 'SKU' }]} />
        {q.isPending ? (
          <LoadingState label="Loading SKU…" />
        ) : notFound ? (
          <EmptyState title="This SKU does not exist">
            <Link to="/inventory/parts" className={linkClass}>
              Back to spare parts and accessories
            </Link>
          </EmptyState>
        ) : (
          <ErrorState
            error={q.error}
            title="Could not load the SKU"
            onRetry={() => void q.refetch()}
          />
        )}
      </>
    )
  }
  const d = q.data
  const archived = d.status === 'ARCHIVED'
  return (
    <>
      <PageHeader
        title={d.sku}
        breadcrumbs={[...crumbs, { label: d.sku }]}
        description={d.name}
        status={<StatusBadge status={d.status} />}
        actions={
          !archived && (
            <>
              {perms.manage && (
                <Button
                  variant="secondary"
                  onClick={() => {
                    setOpen('edit')
                  }}
                >
                  <Pencil className="size-4" aria-hidden="true" />
                  Edit
                </Button>
              )}
              {perms.archive && (
                <Button
                  variant="ghost"
                  onClick={() => {
                    setOpen('archive')
                  }}
                >
                  Archive
                </Button>
              )}
            </>
          )
        }
      />
      {archived && <ArchivedBanner what="SKU" />}
      <MasterDataNotice />
      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Card title="Details">
          <DefinitionList
            items={[
              [
                'SKU code',
                <span key="s" className="font-mono">
                  {d.sku}
                </span>,
              ],
              ['Name', d.name],
              ['Description', d.description ?? '—'],
              [
                'Category',
                d.categoryStatus === 'ARCHIVED' ? `${d.categoryName} (archived)` : d.categoryName,
              ],
              [
                'Unit of measure',
                <span key="u" className="font-mono">
                  {d.uom}
                </span>,
              ],
              [
                'Tracking mode',
                `${trackingModeLabels[d.trackingMode]} — ${trackingModeHelp[d.trackingMode]}`,
              ],
              ['Reorder level', formatLevel(d.reorderLevel) ?? 'Not set'],
              ['Last updated', formatDateTime(d.updatedAt)],
            ]}
          />
        </Card>
        {archived ? (
          <Card title="Archive">
            <DefinitionList
              items={[
                ['Reason', d.archiveReason ?? '—'],
                ['Archived by', personLabel(d.archivedBy, perms.meId)],
                ['Archived at', formatDateTime(d.archivedAt)],
              ]}
            />
          </Card>
        ) : (
          <Card title="Record">
            <DefinitionList
              items={[
                ['Created by', personLabel(d.createdBy, perms.meId)],
                ['Created at', formatDateTime(d.createdAt)],
              ]}
            />
          </Card>
        )}
      </div>
      <div className="mt-6 flex flex-col gap-6">
        <PartsStockPanel item={d} />
        <SuppliersCard item={d} />
        <CompatibilityCard item={d} />
      </div>
      <StockItemFormDialog open={open === 'edit'} onClose={close} item={d} />
      <ArchiveStockItemDialog open={open === 'archive'} onClose={close} item={d} />
    </>
  )
}

function SuppliersCard({ item }: { item: StockItemDetail }) {
  const cols: Column<StockItemSupplier>[] = [
    {
      key: 'supplier',
      header: 'Supplier',
      render: (s) => (
        <Link to={`/procurement/suppliers/${s.supplierId}`} className={linkClass}>
          {s.supplierName}
        </Link>
      ),
    },
    {
      key: 'status',
      header: 'Supplier status',
      render: (s) => <SupplierStatusBadge status={s.supplierStatus} />,
    },
    { key: 'part', header: 'Supplier part number', render: (s) => s.supplierPartNumber ?? '—' },
    {
      key: 'link',
      header: 'Link',
      render: (s) =>
        s.endedAt ? (
          <span className="text-xs text-muted">Ended {formatDateTime(s.endedAt)}</span>
        ) : (
          <Badge tone="success">Current</Badge>
        ),
    },
  ]
  return (
    <Card title="Suppliers" description="Who supplies this item. Ended links are kept as history.">
      {item.suppliersHidden ? (
        <p className="text-sm text-muted">
          Supplier details are not shown: they need permission to view suppliers.
        </p>
      ) : item.suppliers.length === 0 ? (
        <EmptyState title="No suppliers">No supplier has been linked.</EmptyState>
      ) : (
        <ResponsiveTable
          caption="Suppliers of this SKU"
          columns={cols}
          rows={item.suppliers}
          rowKey={(s) => s.id}
        />
      )}
    </Card>
  )
}

function CompatibilityCard({ item }: { item: StockItemDetail }) {
  const cols: Column<StockItemCompatibility>[] = [
    {
      key: 'brand',
      header: 'Brand',
      render: (c) => (c.brandStatus === 'ARCHIVED' ? `${c.brandName} (archived)` : c.brandName),
    },
    { key: 'model', header: 'Model', render: (c) => c.modelName ?? 'All models' },
  ]
  return (
    <Card
      title="Compatibility"
      description="Staff-entered information, not an approved manufacturer fitment."
    >
      {item.compatibility.length === 0 ? (
        <EmptyState title="No compatibility recorded">None has been entered.</EmptyState>
      ) : (
        <ResponsiveTable
          caption="Compatibility"
          columns={cols}
          rows={item.compatibility}
          rowKey={(c) => c.id}
        />
      )}
    </Card>
  )
}

// ── Categories ──────────────────────────────────────────────────────────────

export function PartCategoriesPage() {
  const { manage } = useStockItemPerms()
  const [adding, setAdding] = useState(false)
  return (
    <>
      <PageHeader
        title="Part categories"
        breadcrumbs={[...crumbs, { label: 'Categories' }]}
        description="Groups for parts and accessories. Added by staff; none is predefined."
        actions={
          manage && (
            <Button
              onClick={() => {
                setAdding(true)
              }}
            >
              <Plus className="size-4" aria-hidden="true" />
              Add category
            </Button>
          )
        }
      />
      <Gate>
        <CategoriesList
          onAdd={
            manage
              ? () => {
                  setAdding(true)
                }
              : undefined
          }
        />
      </Gate>
      <CategoryFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
      />
    </>
  )
}

type CategoryOpen = { kind: 'edit' | 'archive'; category: ItemCategory } | null

function CategoriesList({ onAdd }: { onAdd?: () => void }) {
  const perms = useStockItemPerms()
  const { values, page, set } = useListParams(['q', 'status'] as const)
  const [open, setOpen] = useState<CategoryOpen>(null)
  const query = useCategoryList({
    q: values.q || undefined,
    status: (values.status || undefined) as ItemStatus | undefined,
    page,
    limit: 25,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.status)
  const cols: Column<ItemCategory>[] = [
    {
      key: 'name',
      header: 'Category',
      render: (c) => (
        <div className="flex flex-col">
          <span className="font-medium">{c.name}</span>
          {c.description && <span className="text-xs text-muted">{c.description}</span>}
        </div>
      ),
    },
    { key: 'items', header: 'Active SKUs', render: (c) => String(c.activeItemCount) },
    { key: 'status', header: 'Status', render: (c) => <StatusBadge status={c.status} /> },
    {
      key: 'actions',
      header: 'Actions',
      render: (c) =>
        c.status === 'ACTIVE' && (perms.manage || perms.archive) ? (
          <div className="flex flex-wrap gap-1.5 [&>button]:whitespace-nowrap">
            {perms.manage && (
              <Button
                size="sm"
                variant="secondary"
                aria-label={`Edit ${c.name}`}
                onClick={() => {
                  setOpen({ kind: 'edit', category: c })
                }}
              >
                Edit
              </Button>
            )}
            {perms.archive && (
              <Button
                size="sm"
                variant="ghost"
                aria-label={`Archive ${c.name}`}
                onClick={() => {
                  setOpen({ kind: 'archive', category: c })
                }}
              >
                Archive
              </Button>
            )}
          </div>
        ) : null,
    },
  ]
  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading categories…" />
  else if (query.isError && !query.data)
    body = (
      <ErrorState
        error={query.error}
        title="Could not load categories"
        onRetry={() => void query.refetch()}
      />
    )
  else if (query.data.total === 0)
    body = filtered ? (
      <EmptyState title="No categories match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No categories yet"
        icon={<Tags className="size-6 text-muted/80" />}
        action={
          onAdd ? (
            <Button onClick={onAdd}>
              <Plus className="size-4" aria-hidden="true" />
              Add category
            </Button>
          ) : null
        }
      >
        Add the groups your parts and accessories belong to.
      </EmptyState>
    )
  else
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Categories"
          columns={cols}
          rows={query.data.items}
          rowKey={(c) => c.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['category', 'categories']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  return (
    <>
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', status: '' })
        }}
      >
        <SearchBox label="Search categories" value={values.q} onSearch={onSearch} />
        <FilterSelect
          label="Status"
          value={values.status}
          options={Object.entries(itemStatusLabels).map(([value, label]) => ({ value, label }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
      </Toolbar>
      {body}
      {open?.kind === 'edit' && (
        <CategoryFormDialog
          open
          onClose={() => {
            setOpen(null)
          }}
          category={open.category}
        />
      )}
      {open?.kind === 'archive' && (
        <ArchiveCategoryDialog
          open
          onClose={() => {
            setOpen(null)
          }}
          category={open.category}
        />
      )}
    </>
  )
}
