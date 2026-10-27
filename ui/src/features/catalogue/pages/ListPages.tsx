import { Plus, SearchX, Tags } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { Pagination } from '@/components/ui/Pagination'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { Permission } from '@/features/access/api/access'
import { usePermission } from '@/features/access/hooks/useAccess'
import type {
  AuthorisationStatus,
  Brand,
  Confidence,
  LifecycleStatus,
  Variant,
  VehicleModel,
} from '../api/contracts'
import { CatalogueTabs } from '../CatalogueLayout'
import { BrandFormDialog, ModelFormDialog, VariantFormDialog } from '../components/FormDialogs'
import { FilterSelect, SearchBox, Toolbar } from '../components/ListControls'
import { confidenceOptions, lifecycleOptions, PAGE_SIZE, useListParams } from '../lib/listParams'
import {
  AuthorisationBadge,
  ConfidenceBadge,
  LifecycleBadge,
  OperationalBadge,
} from '../components/StatusBadges'
import { ResponsiveTable } from '../components/ResponsiveTable'
import { useBrands, useCatalogueItem, useModels, useVariants } from '../hooks/useCatalogue'
import { authorisationLabels, formatDateTime } from '../lib/presentation'

const OPTION_LIMIT = 100

const linkClass = 'font-medium text-primary hover:underline focus-visible:outline-accent'

function Header({ description, action }: { description: string; action?: React.ReactNode }) {
  return (
    <>
      <PageHeader
        title="Catalogue & Brands"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Catalogue & Brands' }]}
        description={description}
        actions={action}
      />
      <CatalogueTabs />
    </>
  )
}

/** Unverified data must not be read as fact (BR-004). */
function TrustNotice() {
  return (
    <p
      role="note"
      className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
    >
      Records marked <strong>Unverified</strong> or <strong>Pending verification</strong> are setup
      data, not confirmed facts: do not quote them to customers. Specifications are not recorded
      yet; prices come only from approved price history.
    </p>
  )
}

/** Brand names for list columns (brands are few; the first 100 are loaded). */
function useBrandNames() {
  const brands = useBrands({ limit: OPTION_LIMIT })
  const names = new Map((brands.data?.items ?? []).map((b) => [b.id, b.name]))
  const options = (brands.data?.items ?? []).map((b) => ({ value: b.id, label: b.name }))
  return { brands, names, options }
}

function ListBody<T>({
  query,
  caption,
  columns,
  rowKey,
  noun,
  filtered,
  emptyTitle,
  emptyText,
  emptyAction,
  onPage,
}: {
  query: {
    isPending: boolean
    isError: boolean
    error: unknown
    data?: { items: T[]; page: number; limit: number; total: number }
    refetch: () => unknown
    isPlaceholderData: boolean
  }
  caption: string
  columns: Column<T>[]
  rowKey: (row: T) => string
  noun: [string, string]
  filtered: boolean
  emptyTitle: string
  emptyText: string
  emptyAction?: React.ReactNode
  onPage: (page: number) => void
}) {
  if (query.isPending) return <LoadingState label={`Loading ${noun[1]}…`} />
  if (query.isError && !query.data) {
    return (
      <ErrorState
        error={query.error}
        title={`Could not load ${noun[1]}`}
        onRetry={() => void query.refetch()}
      />
    )
  }
  const data = query.data
  if (!data) return null
  if (data.total === 0) {
    return filtered ? (
      <EmptyState
        title={`No ${noun[1]} match these filters`}
        icon={<SearchX className="size-6 text-muted/80" />}
      >
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title={emptyTitle}
        icon={<Tags className="size-6 text-muted/80" />}
        action={emptyAction}
      >
        {emptyText}
      </EmptyState>
    )
  }
  return (
    <div
      aria-busy={query.isPlaceholderData}
      className={query.isPlaceholderData ? 'opacity-70' : undefined}
    >
      <ResponsiveTable caption={caption} columns={columns} rows={data.items} rowKey={rowKey} />
      <Pagination
        page={data.page}
        limit={data.limit}
        total={data.total}
        onPage={onPage}
        noun={noun}
      />
    </div>
  )
}

// ── Brands ──────────────────────────────────────────────────────────────────

export function BrandsPage() {
  const { values, page, set } = useListParams(['q', 'status', 'authorisationStatus'] as const)
  const canManage = usePermission(Permission.CatalogueManage).allowed
  const [adding, setAdding] = useState(false)
  const navigate = useNavigate()
  const query = useBrands({
    q: values.q || undefined,
    status: (values.status || undefined) as LifecycleStatus | undefined,
    authorisationStatus: (values.authorisationStatus || undefined) as
      AuthorisationStatus | undefined,
    page,
    limit: PAGE_SIZE,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.status || values.authorisationStatus)

  const columns: Column<Brand>[] = [
    {
      key: 'name',
      header: 'Brand',
      render: (b) => (
        <div className="flex flex-col">
          <Link to={`/admin/catalogue/brands/${b.id}`} className={linkClass}>
            {b.name}
          </Link>
          <span className="font-mono text-xs text-muted">{b.slug}</span>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (b) => <LifecycleBadge status={b.lifecycleStatus} />,
    },
    {
      key: 'auth',
      header: 'Authorisation',
      render: (b) => <AuthorisationBadge status={b.authorisationStatus} />,
    },
    {
      key: 'op',
      header: 'Purchase & sales',
      render: (b) => <OperationalBadge operational={b.operational} />,
    },
    {
      key: 'updated',
      header: 'Updated',
      className: 'whitespace-nowrap text-muted',
      render: (b) => formatDateTime(b.updatedAt),
    },
  ]

  const add = canManage ? (
    <Button
      onClick={() => {
        setAdding(true)
      }}
    >
      <Plus className="size-4" aria-hidden="true" />
      Add brand
    </Button>
  ) : undefined

  return (
    <>
      <Header
        description="Brands the showroom deals in. Being listed here does not authorise a brand: purchase and sales need recorded authorisation, a verified catalogue and later approvals."
        action={add}
      />
      <TrustNotice />
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', status: '', authorisationStatus: '' })
        }}
      >
        <SearchBox label="Search brands" value={values.q} onSearch={onSearch} />
        <FilterSelect
          label="Status"
          value={values.status}
          options={lifecycleOptions}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Authorisation"
          value={values.authorisationStatus}
          options={Object.entries(authorisationLabels).map(([value, label]) => ({ value, label }))}
          allLabel="Any authorisation"
          onChange={(v) => {
            set({ authorisationStatus: v })
          }}
        />
      </Toolbar>
      <ListBody
        query={query}
        caption="Brands"
        columns={columns}
        rowKey={(b) => b.id}
        noun={['brand', 'brands']}
        filtered={filtered}
        emptyTitle="No brands yet"
        emptyText={
          canManage
            ? 'Add the first brand the showroom deals in.'
            : 'No brands have been added yet.'
        }
        emptyAction={add}
        onPage={(p) => {
          set({ page: String(p) })
        }}
      />
      <BrandFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        onSaved={(b) => void navigate(`/admin/catalogue/brands/${b.id}`)}
      />
    </>
  )
}

// ── Models ──────────────────────────────────────────────────────────────────

export function ModelsPage() {
  const { values, page, set } = useListParams(['q', 'brandId', 'status', 'confidence'] as const)
  const canManage = usePermission(Permission.CatalogueManage).allowed
  const [adding, setAdding] = useState(false)
  const navigate = useNavigate()
  const { names, options } = useBrandNames()
  const query = useModels({
    q: values.q || undefined,
    brandId: values.brandId || undefined,
    status: (values.status || undefined) as LifecycleStatus | undefined,
    confidence: (values.confidence || undefined) as Confidence | undefined,
    page,
    limit: PAGE_SIZE,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.brandId || values.status || values.confidence)

  const columns: Column<VehicleModel>[] = [
    {
      key: 'name',
      header: 'Model',
      render: (m) => (
        <div className="flex flex-col">
          <Link to={`/admin/catalogue/models/${m.id}`} className={linkClass}>
            {m.name}
          </Link>
          <span className="font-mono text-xs text-muted">{m.internalCode}</span>
        </div>
      ),
    },
    {
      key: 'brand',
      header: 'Brand',
      render: (m) => (
        <Link to={`/admin/catalogue/brands/${m.brandId}`} className="text-ink hover:underline">
          {names.get(m.brandId) ?? 'Brand'}
        </Link>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (m) => <LifecycleBadge status={m.lifecycleStatus} />,
    },
    {
      key: 'conf',
      header: 'Verification',
      render: (m) => <ConfidenceBadge confidence={m.confidence} />,
    },
    {
      key: 'updated',
      header: 'Updated',
      className: 'whitespace-nowrap text-muted',
      render: (m) => formatDateTime(m.updatedAt),
    },
  ]

  const add = canManage ? (
    <Button
      onClick={() => {
        setAdding(true)
      }}
    >
      <Plus className="size-4" aria-hidden="true" />
      Add model
    </Button>
  ) : undefined

  return (
    <>
      <Header
        description="Vehicle models by brand. A model belongs to one brand and never moves."
        action={add}
      />
      <TrustNotice />
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', brandId: '', status: '', confidence: '' })
        }}
      >
        <SearchBox
          key={values.q}
          label="Search models or codes"
          value={values.q}
          onSearch={onSearch}
        />
        <FilterSelect
          label="Brand"
          value={values.brandId}
          options={options}
          allLabel="All brands"
          onChange={(v) => {
            set({ brandId: v })
          }}
        />
        <FilterSelect
          label="Status"
          value={values.status}
          options={lifecycleOptions}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Verification"
          value={values.confidence}
          options={confidenceOptions}
          allLabel="Any verification"
          onChange={(v) => {
            set({ confidence: v })
          }}
        />
      </Toolbar>
      <ListBody
        query={query}
        caption="Models"
        columns={columns}
        rowKey={(m) => m.id}
        noun={['model', 'models']}
        filtered={filtered}
        emptyTitle="No models yet"
        emptyText="Models are added per brand from supplier or client sources. None have been supplied yet."
        emptyAction={add}
        onPage={(p) => {
          set({ page: String(p) })
        }}
      />
      <ModelFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        onSaved={(m) => void navigate(`/admin/catalogue/models/${m.id}`)}
      />
    </>
  )
}

// ── Variants ────────────────────────────────────────────────────────────────

export function VariantsPage() {
  const { values, page, set } = useListParams([
    'q',
    'brandId',
    'modelId',
    'status',
    'confidence',
  ] as const)
  const canManage = usePermission(Permission.CatalogueManage).allowed
  const [adding, setAdding] = useState(false)
  const navigate = useNavigate()
  const { names: brandNames, options: brandOptions } = useBrandNames()
  // Brand → model: the model filter lists only the chosen brand's models.
  const models = useModels({ brandId: values.brandId || undefined, limit: OPTION_LIMIT })
  const modelNames = new Map((models.data?.items ?? []).map((m) => [m.id, m.name]))
  const modelOptions = values.brandId
    ? (models.data?.items ?? [])
        .filter((m) => m.brandId === values.brandId)
        .map((m) => ({ value: m.id, label: `${m.name} (${m.internalCode})` }))
    : []
  // A link can carry a model of another brand (stale or edited URL). The server
  // refuses that pair (C-06), so check the model's brand first and drop a
  // mismatched model filter instead of showing an error.
  const pairToCheck = Boolean(values.brandId && values.modelId)
  const filterModel = useCatalogueItem('models', values.modelId || undefined, {
    enabled: pairToCheck,
  })
  const modelMatches = !pairToCheck
    ? true
    : filterModel.data
      ? filterModel.data.brandId === values.brandId
      : filterModel.isError
        ? false
        : undefined
  useEffect(() => {
    if (modelMatches === false) set({ modelId: '' })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [modelMatches])
  const query = useVariants(
    {
      q: values.q || undefined,
      brandId: values.brandId || undefined,
      modelId: modelMatches ? values.modelId || undefined : undefined,
      status: (values.status || undefined) as LifecycleStatus | undefined,
      confidence: (values.confidence || undefined) as Confidence | undefined,
      page,
      limit: PAGE_SIZE,
    },
    { enabled: modelMatches !== undefined },
  )
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(
    values.q || values.brandId || values.modelId || values.status || values.confidence,
  )

  const columns: Column<Variant>[] = [
    {
      key: 'name',
      header: 'Variant',
      render: (v) => (
        <div className="flex flex-col">
          <Link to={`/admin/catalogue/variants/${v.id}`} className={linkClass}>
            {v.name}
          </Link>
          <span className="font-mono text-xs text-muted">{v.variantCode}</span>
        </div>
      ),
    },
    {
      key: 'model',
      header: 'Model',
      render: (v) => (
        <Link to={`/admin/catalogue/models/${v.modelId}`} className="text-ink hover:underline">
          {modelNames.get(v.modelId) ?? 'Model'}
        </Link>
      ),
    },
    { key: 'brand', header: 'Brand', render: (v) => brandNames.get(v.brandId) ?? 'Brand' },
    {
      key: 'status',
      header: 'Status',
      render: (v) => <LifecycleBadge status={v.lifecycleStatus} />,
    },
    {
      key: 'conf',
      header: 'Verification',
      render: (v) => <ConfidenceBadge confidence={v.confidence} />,
    },
  ]

  const add = canManage ? (
    <Button
      onClick={() => {
        setAdding(true)
      }}
    >
      <Plus className="size-4" aria-hidden="true" />
      Add variant
    </Button>
  ) : undefined

  return (
    <>
      <Header
        description="Variants of each model. Choose a brand to narrow the model list to that brand."
        action={add}
      />
      <TrustNotice />
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', brandId: '', modelId: '', status: '', confidence: '' })
        }}
      >
        <SearchBox
          key={values.q}
          label="Search variants or codes"
          value={values.q}
          onSearch={onSearch}
        />
        <FilterSelect
          label="Brand"
          value={values.brandId}
          options={brandOptions}
          allLabel="All brands"
          onChange={(v) => {
            set({ brandId: v, modelId: '' })
          }}
        />
        <FilterSelect
          label="Model"
          value={values.modelId}
          options={modelOptions}
          allLabel={values.brandId ? 'All models of this brand' : 'Choose a brand first'}
          disabled={!values.brandId}
          onChange={(v) => {
            set({ modelId: v })
          }}
        />
        <FilterSelect
          label="Status"
          value={values.status}
          options={lifecycleOptions}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Verification"
          value={values.confidence}
          options={confidenceOptions}
          allLabel="Any verification"
          onChange={(v) => {
            set({ confidence: v })
          }}
        />
      </Toolbar>
      <ListBody
        query={query}
        caption="Variants"
        columns={columns}
        rowKey={(v) => v.id}
        noun={['variant', 'variants']}
        filtered={filtered}
        emptyTitle="No variants yet"
        emptyText="Variants are added under a model from supplier or client sources. None have been supplied yet."
        emptyAction={add}
        onPage={(p) => {
          set({ page: String(p) })
        }}
      />
      <VariantFormDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        onSaved={(v) => void navigate(`/admin/catalogue/variants/${v.id}`)}
      />
    </>
  )
}
