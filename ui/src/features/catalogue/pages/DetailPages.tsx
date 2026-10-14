import { Pencil, Plus, Power, ShieldCheck, Tags } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { ApiError } from '@/lib/api/client'
import type { Brand, Variant, VariantColour, VehicleModel } from '../api/contracts'
import {
  ActivateBrandDialog,
  ArchiveDialog,
  AuthorisationDialog,
} from '../components/ActionDialogs'
import {
  ArchivedBanner,
  DataGapsCard,
  DefinitionList,
  ItemActions,
  ProvenanceCard,
  ReadinessList,
} from '../components/DetailParts'
import {
  BrandFormDialog,
  ColourFormDialog,
  ModelFormDialog,
  VariantFormDialog,
} from '../components/FormDialogs'
import { PAGE_SIZE } from '../lib/listParams'
import {
  AuthorisationBadge,
  ConfidenceBadge,
  LifecycleBadge,
  OperationalBadge,
} from '../components/StatusBadges'
import { ResponsiveTable } from '../components/ResponsiveTable'
import {
  useBrand,
  useBrandReadiness,
  useCatalogueItem,
  useColours,
  useModels,
  useVariants,
} from '../hooks/useCatalogue'
import { formatDateTime, personLabel } from '../lib/presentation'
import { PriceListsCard, VariantPricesCard } from '@/features/pricing/components/PriceCards'
import { useCanSeePrices } from '@/features/pricing/hooks/usePricePermissions'

const crumbsRoot = [
  { label: 'Home', to: '/' },
  { label: 'Catalogue & Brands', to: '/admin/catalogue' },
]

function usePerms() {
  const me = useMe()
  const perms = me.data?.permissions ?? []
  return { has: (p: string) => perms.includes(p), meId: me.data?.user.id }
}

/** Loading / not found / error for a detail record. */
function DetailState({
  query,
  what,
  title,
}: {
  query: { isPending: boolean; isError: boolean; error: unknown; refetch: () => unknown }
  what: string
  title: string
}): ReactNode {
  const notFound = query.error instanceof ApiError && query.error.status === 404
  return (
    <>
      <PageHeader title={title} breadcrumbs={[...crumbsRoot, { label: title }]} />
      {query.isPending ? (
        <LoadingState label={`Loading ${what}…`} />
      ) : notFound ? (
        <EmptyState
          title={`This ${what} does not exist`}
          icon={<Tags className="size-6 text-muted/80" />}
        >
          It may have been removed from the link you followed.{' '}
          <Link to="/admin/catalogue" className="text-primary hover:underline">
            Back to the catalogue
          </Link>
        </EmptyState>
      ) : (
        <ErrorState
          error={query.error}
          title={`Could not load the ${what}`}
          onRetry={() => void query.refetch()}
        />
      )}
    </>
  )
}

/** A paged child list inside a card. */
function ChildList<T>({
  title,
  description,
  query,
  columns,
  rowKey,
  noun,
  page,
  onPage,
  empty,
  action,
}: {
  title: string
  description?: string
  query: {
    isPending: boolean
    isError: boolean
    error: unknown
    data?: { items: T[]; page: number; limit: number; total: number }
    refetch: () => unknown
  }
  columns: Column<T>[]
  rowKey: (row: T) => string
  noun: [string, string]
  page: number
  onPage: (page: number) => void
  empty: string
  action?: ReactNode
}) {
  return (
    <Card title={title} description={description} actions={action} contentClassName="p-0 sm:p-5">
      {query.isPending ? (
        <div className="px-5">
          <LoadingState label={`Loading ${noun[1]}…`} />
        </div>
      ) : query.isError && !query.data ? (
        <div className="p-5">
          <ErrorState
            error={query.error}
            title={`Could not load ${noun[1]}`}
            onRetry={() => void query.refetch()}
          />
        </div>
      ) : query.data && query.data.total === 0 ? (
        <EmptyState title={`No ${noun[1]} yet`}>{empty}</EmptyState>
      ) : query.data ? (
        <div className="p-3 sm:p-0">
          <ResponsiveTable
            caption={title}
            columns={columns}
            rows={query.data.items}
            rowKey={rowKey}
          />
          <Pagination
            page={page}
            limit={query.data.limit}
            total={query.data.total}
            onPage={onPage}
            noun={noun}
          />
        </div>
      ) : null}
    </Card>
  )
}

const linkClass = 'font-medium text-primary hover:underline'

// ── Brand ───────────────────────────────────────────────────────────────────

export function BrandDetailPage() {
  const { brandId = '' } = useParams()
  const brand = useBrand(brandId)
  if (!brand.data) return <DetailState query={brand} what="brand" title="Brand" />
  return <BrandDetail brand={brand.data} />
}

function BrandDetail({ brand }: { brand: Brand }) {
  const { has, meId } = usePerms()
  const canSeePrices = useCanSeePrices()
  const readiness = useBrandReadiness(brand.id)
  const [modelPage, setModelPage] = useState(1)
  const models = useModels({ brandId: brand.id, page: modelPage, limit: PAGE_SIZE })
  const [open, setOpen] = useState<'edit' | 'auth' | 'activate' | 'archive' | 'model' | null>(null)
  const close = () => {
    setOpen(null)
  }
  const archived = brand.lifecycleStatus === 'ARCHIVED'

  const modelColumns: Column<VehicleModel>[] = [
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
      key: 'status',
      header: 'Status',
      render: (m) => <LifecycleBadge status={m.lifecycleStatus} />,
    },
    {
      key: 'conf',
      header: 'Verification',
      render: (m) => <ConfidenceBadge confidence={m.confidence} />,
    },
  ]

  return (
    <>
      <PageHeader
        title={brand.name}
        breadcrumbs={[...crumbsRoot, { label: brand.name }]}
        status={<LifecycleBadge status={brand.lifecycleStatus} />}
        description="Catalogue status, dealership authorisation and purchase/sales readiness are tracked separately."
        actions={
          !archived && (
            <>
              {has(Permission.CatalogueManage) && (
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
              {has(Permission.CatalogueArchive) && (
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
      {archived && <ArchivedBanner what="brand" />}
      <div className="mb-6 flex flex-wrap items-center gap-2" aria-label="Brand status">
        <AuthorisationBadge status={brand.authorisationStatus} />
        <OperationalBadge operational={brand.operational} />
        <span className="font-mono text-xs text-muted">/{brand.slug}</span>
      </div>
      {brand.authorisationStatus !== 'AUTHORISED' && (
        <p
          role="note"
          className="mb-6 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
        >
          <strong>Authorisation not verified.</strong> Listing this brand does not authorise the
          showroom to buy or sell it. Do not treat it as an authorised dealership brand until the
          supplier/manufacturer document is recorded.
        </p>
      )}

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Card
          title="Authorisation"
          description="Supplier/manufacturer legal identity and evidence (BR-002)."
          actions={
            !archived &&
            has(Permission.BrandAuthorise) && (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => {
                  setOpen('auth')
                }}
              >
                <ShieldCheck className="size-3.5" aria-hidden="true" />
                Record
              </Button>
            )
          }
        >
          <DefinitionList
            items={[
              ['Status', <AuthorisationBadge key="s" status={brand.authorisationStatus} />],
              ['Legal name', brand.manufacturerLegalName ?? 'Not recorded'],
              ['Reference', brand.authorisationReference ?? 'Not recorded'],
              [
                'Evidence',
                brand.authorisationEvidenceId ? 'Private document attached' : 'Not attached',
              ],
              ['Verified by', personLabel(brand.verifiedBy, meId)],
              ['Verified at', formatDateTime(brand.verifiedAt)],
            ]}
          />
        </Card>

        <Card
          title="Activation readiness"
          description="What the server requires before this brand can be used for purchase and sales."
          actions={
            brand.lifecycleStatus === 'PENDING_VERIFICATION' &&
            has(Permission.BrandActivate) && (
              <Button
                size="sm"
                onClick={() => {
                  setOpen('activate')
                }}
              >
                <Power className="size-3.5" aria-hidden="true" />
                Enable…
              </Button>
            )
          }
        >
          {readiness.isPending && <LoadingState label="Checking requirements…" />}
          {readiness.isError && (
            <ErrorState
              error={readiness.error}
              title="Could not load readiness"
              onRetry={() => void readiness.refetch()}
            />
          )}
          {readiness.data && <ReadinessList readiness={readiness.data} />}
        </Card>

        <Card title="Supplier relationship" description="Verified supplier ↔ brand links.">
          {readiness.data?.checks.find((c) => c.key === 'supplier_link')?.met ? (
            <p className="text-sm text-ink">A verified supplier relationship is recorded.</p>
          ) : (
            <p className="text-sm text-muted">
              No supplier relationship is recorded. The supplier master (D-01) is not built yet, so
              no supplier is shown or assumed for this brand.
            </p>
          )}
        </Card>

        <DataGapsCard />
      </div>

      {canSeePrices && (
        <div className="mt-6">
          <PriceListsCard brandId={brand.id} brandName={brand.name} brandArchived={archived} />
        </div>
      )}

      <div className="mt-6">
        <ChildList
          title="Models"
          description="Models of this brand."
          query={models}
          columns={modelColumns}
          rowKey={(m) => m.id}
          noun={['model', 'models']}
          page={modelPage}
          onPage={setModelPage}
          empty={archived ? 'This brand is archived.' : 'No models have been added for this brand.'}
          action={
            !archived &&
            has(Permission.CatalogueManage) && (
              <Button
                size="sm"
                onClick={() => {
                  setOpen('model')
                }}
              >
                <Plus className="size-3.5" aria-hidden="true" />
                Add model
              </Button>
            )
          }
        />
      </div>

      <BrandFormDialog open={open === 'edit'} onClose={close} brand={brand} />
      <AuthorisationDialog open={open === 'auth'} onClose={close} brand={brand} />
      <ActivateBrandDialog
        open={open === 'activate'}
        onClose={close}
        brand={brand}
        ready={readiness.data?.ready ?? false}
      />
      <ArchiveDialog
        open={open === 'archive'}
        onClose={close}
        kind="brands"
        item={brand}
        childrenNote="Its models and variants keep their own status; none of them can be used while the brand is archived."
      />
      <ModelFormDialog open={open === 'model'} onClose={close} fixedBrand={brand} />
    </>
  )
}

// ── Model ───────────────────────────────────────────────────────────────────

export function ModelDetailPage() {
  const { modelId = '' } = useParams()
  const model = useCatalogueItem('models', modelId)
  if (!model.data) return <DetailState query={model} what="model" title="Model" />
  return <ModelDetail model={model.data} />
}

/** The parent brand's name and purchase/sales status for a child page. */
function BrandContext({ brandId }: { brandId: string }) {
  const brand = useBrand(brandId)
  return (
    <div className="mb-6 flex flex-wrap items-center gap-2 text-sm" aria-label="Brand">
      <span className="text-muted">Brand:</span>
      <Link to={`/admin/catalogue/brands/${brandId}`} className={linkClass}>
        {brand.data?.name ?? 'Brand'}
      </Link>
      {brand.data && <OperationalBadge operational={brand.data.operational} />}
    </div>
  )
}

function ModelDetail({ model }: { model: VehicleModel }) {
  const { has } = usePerms()
  const brand = useBrand(model.brandId)
  const [variantPage, setVariantPage] = useState(1)
  const variants = useVariants({ modelId: model.id, page: variantPage, limit: PAGE_SIZE })
  const [open, setOpen] = useState<'edit' | 'variant' | null>(null)
  const close = () => {
    setOpen(null)
  }
  const archived = model.lifecycleStatus === 'ARCHIVED'
  const brandName = brand.data?.name ?? 'Brand'

  const variantColumns: Column<Variant>[] = [
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

  return (
    <>
      <PageHeader
        title={model.name}
        breadcrumbs={[
          ...crumbsRoot,
          { label: brandName, to: `/admin/catalogue/brands/${model.brandId}` },
          { label: model.name },
        ]}
        status={<LifecycleBadge status={model.lifecycleStatus} />}
        actions={
          <ItemActions
            kind="models"
            item={model}
            status={model.lifecycleStatus}
            confidence={model.confidence}
            onEdit={() => {
              setOpen('edit')
            }}
            parentNote="Its brand must not be archived. Using it for purchase and sales also needs the brand to be enabled."
            childrenNote="Its variants keep their own status and references."
          />
        }
      />
      {archived && <ArchivedBanner what="model" />}
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <ConfidenceBadge confidence={model.confidence} />
      </div>
      <BrandContext brandId={model.brandId} />

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Card title="Details">
          <DefinitionList
            items={[
              [
                'Internal code',
                <span key="c" className="font-mono">
                  {model.internalCode}
                </span>,
              ],
              ['Manufacturer code', model.manufacturerCode ?? '—'],
              ['Vehicle type', model.vehicleType ?? '—'],
              ['Last updated', formatDateTime(model.updatedAt)],
            ]}
          />
        </Card>
        <ProvenanceCard provenance={model} />
      </div>

      <div className="mt-6">
        <ChildList
          title="Variants"
          description="Variants of this model."
          query={variants}
          columns={variantColumns}
          rowKey={(v) => v.id}
          noun={['variant', 'variants']}
          page={variantPage}
          onPage={setVariantPage}
          empty={
            archived ? 'This model is archived.' : 'No variants have been added for this model.'
          }
          action={
            !archived &&
            has(Permission.CatalogueManage) && (
              <Button
                size="sm"
                onClick={() => {
                  setOpen('variant')
                }}
              >
                <Plus className="size-3.5" aria-hidden="true" />
                Add variant
              </Button>
            )
          }
        />
      </div>

      <ModelFormDialog open={open === 'edit'} onClose={close} model={model} />
      <VariantFormDialog open={open === 'variant'} onClose={close} fixedModel={model} />
    </>
  )
}

// ── Variant ─────────────────────────────────────────────────────────────────

export function VariantDetailPage() {
  const { variantId = '' } = useParams()
  const variant = useCatalogueItem('variants', variantId)
  if (!variant.data) return <DetailState query={variant} what="variant" title="Variant" />
  return <VariantDetail variant={variant.data} />
}

function ColourRowActions({ colour, variant }: { colour: VariantColour; variant: Variant }) {
  const [editing, setEditing] = useState(false)
  return (
    <>
      <ItemActions
        compact
        kind="colours"
        item={colour}
        status={colour.lifecycleStatus}
        confidence={colour.confidence}
        onEdit={() => {
          setEditing(true)
        }}
        parentNote="The variant must be active first."
      />
      <ColourFormDialog
        open={editing}
        onClose={() => {
          setEditing(false)
        }}
        colour={colour}
        variant={variant}
      />
    </>
  )
}

function VariantDetail({ variant }: { variant: Variant }) {
  const { has } = usePerms()
  const canSeePrices = useCanSeePrices()
  const brand = useBrand(variant.brandId)
  const model = useCatalogueItem('models', variant.modelId)
  const [colourPage, setColourPage] = useState(1)
  const colours = useColours({ variantId: variant.id, page: colourPage, limit: PAGE_SIZE })
  const [open, setOpen] = useState<'edit' | 'colour' | null>(null)
  const close = () => {
    setOpen(null)
  }
  const archived = variant.lifecycleStatus === 'ARCHIVED'

  const colourColumns: Column<VariantColour>[] = [
    {
      key: 'name',
      header: 'Colour',
      render: (c) => (
        <div className="flex flex-col">
          <span className="font-medium">{c.name}</span>
          {c.colourCode && <span className="font-mono text-xs text-muted">{c.colourCode}</span>}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (c) => <LifecycleBadge status={c.lifecycleStatus} />,
    },
    {
      key: 'conf',
      header: 'Verification',
      render: (c) => <ConfidenceBadge confidence={c.confidence} />,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (c) => <ColourRowActions colour={c} variant={variant} />,
    },
  ]

  return (
    <>
      <PageHeader
        title={variant.name}
        breadcrumbs={[
          ...crumbsRoot,
          { label: brand.data?.name ?? 'Brand', to: `/admin/catalogue/brands/${variant.brandId}` },
          { label: model.data?.name ?? 'Model', to: `/admin/catalogue/models/${variant.modelId}` },
          { label: variant.name },
        ]}
        status={<LifecycleBadge status={variant.lifecycleStatus} />}
        actions={
          <ItemActions
            kind="variants"
            item={variant}
            status={variant.lifecycleStatus}
            confidence={variant.confidence}
            onEdit={() => {
              setOpen('edit')
            }}
            parentNote="The model must be active first."
            childrenNote="Its colours keep their own status and references."
          />
        }
      />
      {archived && <ArchivedBanner what="variant" />}
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <ConfidenceBadge confidence={variant.confidence} />
      </div>
      <BrandContext brandId={variant.brandId} />

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Card title="Details">
          <DefinitionList
            items={[
              [
                'Variant code',
                <span key="c" className="font-mono">
                  {variant.variantCode}
                </span>,
              ],
              ['Manufacturer code', variant.manufacturerCode ?? '—'],
              [
                'Model',
                <Link
                  key="m"
                  to={`/admin/catalogue/models/${variant.modelId}`}
                  className={linkClass}
                >
                  {model.data?.name ?? 'Open model'}
                </Link>,
              ],
              ['Last updated', formatDateTime(variant.updatedAt)],
            ]}
          />
        </Card>
        <ProvenanceCard provenance={variant} />
        <DataGapsCard />
      </div>

      {canSeePrices && (
        <div className="mt-6">
          <VariantPricesCard
            brandId={variant.brandId}
            variantId={variant.id}
            variantUsable={!archived}
          />
        </div>
      )}

      <div className="mt-6">
        <ChildList
          title="Colours"
          description="Colours offered for this variant."
          query={colours}
          columns={colourColumns}
          rowKey={(c) => c.id}
          noun={['colour', 'colours']}
          page={colourPage}
          onPage={setColourPage}
          empty={
            archived ? 'This variant is archived.' : 'No colours have been added for this variant.'
          }
          action={
            !archived &&
            has(Permission.CatalogueManage) && (
              <Button
                size="sm"
                onClick={() => {
                  setOpen('colour')
                }}
              >
                <Plus className="size-3.5" aria-hidden="true" />
                Add colour
              </Button>
            )
          }
        />
      </div>

      <VariantFormDialog open={open === 'edit'} onClose={close} variant={variant} />
      <ColourFormDialog open={open === 'colour'} onClose={close} variant={variant} />
    </>
  )
}
