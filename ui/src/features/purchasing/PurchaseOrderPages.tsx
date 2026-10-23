import { AlertTriangle, Pencil, Plus, Settings, ShoppingCart, Trash2 } from 'lucide-react'
import { useCallback, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge, type BadgeTone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { LifecycleBadge } from '@/features/catalogue/components/StatusBadges'
import { useBrands } from '@/features/catalogue/hooks/useCatalogue'
import { useListParams } from '@/features/catalogue/lib/listParams'
import {
  checkLabel,
  formatDate,
  formatDateTime,
  personLabel,
} from '@/features/catalogue/lib/presentation'
import { formatINR } from '@/features/pricing/lib/format'
import { SupplierStatusBadge } from '@/features/suppliers/components/SupplierBadges'
import { ApiError } from '@/lib/api/client'
import type { PODetail, POLine, POStatus, PurchaseOrder } from './api/purchaseOrders'
import {
  ApproveDialog,
  CancelDialog,
  HeaderFormDialog,
  LineFormDialog,
  RemoveLineDialog,
  ReturnDialog,
  SettingsDialog,
  SubmitDialog,
} from './components/PurchaseOrderDialogs'
import { usePOSettings, usePurchaseOrder, usePurchaseOrderList } from './hooks/usePurchaseOrders'
import { eventLabels, poStatusLabels, warningLabels } from './lib/labels'

const listPath = '/procurement/purchase-orders'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Purchase Orders', to: listPath },
]
const linkClass = 'font-medium text-primary hover:underline'

const statusTone: Record<POStatus, BadgeTone> = {
  DRAFT: 'neutral',
  SUBMITTED: 'warning',
  APPROVED: 'success',
  PARTIALLY_RECEIVED: 'info',
  RECEIVED: 'success',
  CLOSED: 'neutral',
  CANCELLED: 'danger',
}

export function POStatusBadge({ status }: { status: POStatus }) {
  return <Badge tone={statusTone[status]}>{poStatusLabels[status]}</Badge>
}

function usePerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.PurchaseOrderView),
    manage: p.includes(Permission.PurchaseOrderManage),
    settings: p.includes(Permission.PurchaseOrderSettings),
    catalogue: p.includes(Permission.CatalogueView),
  }
}

/** Gate for both pages: purchase_orders.view, checked again by the API. */
function Gate({ children }: { children: ReactNode }) {
  const { me, view } = usePerms()
  if (me.isPending) return <LoadingState label="Checking your permissions…" />
  if (me.isError && !me.data) {
    return (
      <ErrorState
        error={me.error}
        title="Could not check your permissions"
        onRetry={() => void me.refetch()}
      />
    )
  }
  if (!view) return <AccessDeniedState what="purchase orders" />
  return <>{children}</>
}

function remaining(po: PurchaseOrder) {
  return `${String(po.receivedQuantity)} / ${String(po.totalQuantity)} received · ${String(po.remainingQuantity)} remaining`
}

// ── List ────────────────────────────────────────────────────────────────────

export function PurchaseOrdersPage() {
  const { manage, view } = usePerms()
  const [creating, setCreating] = useState(false)
  const navigate = useNavigate()
  const add = () => {
    setCreating(true)
  }
  return (
    <>
      <PageHeader
        title="Purchase Orders"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Purchase Orders' }]}
        description="Orders to suppliers: lines with agreed costs, submission, approval and status history."
        actions={
          manage && view ? (
            <Button onClick={add}>
              <Plus className="size-4" aria-hidden="true" />
              New purchase order
            </Button>
          ) : undefined
        }
      />
      <Gate>
        <ApprovalPolicy />
        <PurchaseOrderList canAdd={manage} onAdd={add} />
      </Gate>
      <HeaderFormDialog
        open={creating}
        onClose={() => {
          setCreating(false)
        }}
        onSaved={(d) => void navigate(`${listPath}/${d.id}`)}
      />
    </>
  )
}

/** What the server will require on submission (PO-001). */
function ApprovalPolicy() {
  const perms = usePerms()
  const q = usePOSettings()
  const [editing, setEditing] = useState(false)
  if (!q.data) return null
  const s = q.data
  return (
    <div
      role="note"
      className="mb-4 flex flex-col gap-2 rounded-card border border-border/80 bg-surface px-4 py-3 text-sm text-ink sm:flex-row sm:items-center sm:justify-between"
    >
      <p>
        {s.approvalThreshold ? (
          <>
            Approval threshold <strong>{formatINR(s.approvalThreshold)}</strong> (owner sign-off{' '}
            {s.approvalReference}). Above it
            {s.firstTimeSupplierNeedsApproval ? ', and for a supplier’s first order,' : ''} an
            approver is needed.
          </>
        ) : (
          <>
            <strong>No approval threshold is configured</strong> — every purchase order needs an
            approver who did not create or submit it.
          </>
        )}
      </p>
      {perms.settings && (
        <Button
          size="sm"
          variant="secondary"
          onClick={() => {
            setEditing(true)
          }}
        >
          <Settings className="size-3.5" aria-hidden="true" />
          Approval settings
        </Button>
      )}
      <SettingsDialog
        open={editing}
        settings={s}
        onClose={() => {
          setEditing(false)
        }}
      />
    </div>
  )
}

function PurchaseOrderList({ canAdd, onAdd }: { canAdd: boolean; onAdd: () => void }) {
  const { catalogue } = usePerms()
  const { values, page, set } = useListParams(['q', 'status', 'brandId'] as const)
  const brands = useBrands({ limit: 100 }, { enabled: catalogue })
  const query = usePurchaseOrderList({
    q: values.q || undefined,
    status: (values.status || undefined) as POStatus | undefined,
    brandId: values.brandId || undefined,
    page,
    limit: 25,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.status || values.brandId)
  const cols: Column<PurchaseOrder>[] = [
    {
      key: 'number',
      header: 'PO',
      render: (po) => (
        <div className="flex flex-col">
          <Link to={`${listPath}/${po.id}`} className={`${linkClass} font-mono`}>
            {po.number}
          </Link>
          <span className="text-xs text-muted">{po.brandNames.join(', ') || 'No lines yet'}</span>
        </div>
      ),
    },
    { key: 'supplier', header: 'Supplier', render: (po) => po.supplierName },
    { key: 'status', header: 'Status', render: (po) => <POStatusBadge status={po.status} /> },
    { key: 'expected', header: 'Expected', render: (po) => formatDate(po.expectedDate) },
    {
      key: 'total',
      header: 'Total (excl. tax)',
      render: (po) => <span className="tabular-nums">{formatINR(po.totalAmount)}</span>,
    },
    {
      key: 'received',
      header: 'Received / remaining',
      render: (po) => <span className="text-xs">{remaining(po)}</span>,
    },
    { key: 'creator', header: 'Created by', render: (po) => po.createdByName },
  ]
  const add = canAdd ? (
    <Button onClick={onAdd}>
      <Plus className="size-4" aria-hidden="true" />
      New purchase order
    </Button>
  ) : null

  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading purchase orders…" />
  else if (query.isError && !query.data) {
    body = (
      <ErrorState
        error={query.error}
        title="Could not load purchase orders"
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.data.total === 0) {
    body = filtered ? (
      <EmptyState title="No purchase orders match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No purchase orders yet"
        icon={<ShoppingCart className="size-6 text-muted/80" />}
        action={add}
      >
        Create a draft for a supplier, add lines with the agreed costs, then submit it for approval.
      </EmptyState>
    )
  } else {
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Purchase orders"
          columns={cols}
          rows={query.data.items}
          rowKey={(po) => po.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['purchase order', 'purchase orders']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  }

  return (
    <>
      <Toolbar
        active={filtered}
        onClear={() => {
          set({ q: '', status: '', brandId: '' })
        }}
      >
        <SearchBox label="Search PO number or supplier" value={values.q} onSearch={onSearch} />
        <FilterSelect
          label="Status"
          value={values.status}
          options={Object.entries(poStatusLabels).map(([value, label]) => ({ value, label }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        {catalogue && (
          <FilterSelect
            label="Brand"
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

// ── Detail ──────────────────────────────────────────────────────────────────

export function PurchaseOrderDetailPage() {
  const { poId = '' } = useParams()
  return (
    <Gate>
      <PurchaseOrderDetailBody id={poId} />
    </Gate>
  )
}

type Open =
  | { kind: 'edit' | 'line' | 'submit' | 'approve' | 'return' | 'cancel' }
  | { kind: 'editLine' | 'removeLine'; line: POLine }
  | null

function PurchaseOrderDetailBody({ id }: { id: string }) {
  const q = usePurchaseOrder(id)
  const perms = usePerms()
  const [open, setOpen] = useState<Open>(null)
  const close = () => {
    setOpen(null)
  }

  if (!q.data) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <>
        <PageHeader title="Purchase order" breadcrumbs={[...crumbs, { label: 'Purchase order' }]} />
        {q.isPending ? (
          <LoadingState label="Loading purchase order…" />
        ) : notFound ? (
          <EmptyState title="This purchase order does not exist">
            <Link to={listPath} className={linkClass}>
              Back to purchase orders
            </Link>
          </EmptyState>
        ) : (
          <ErrorState
            error={q.error}
            title="Could not load the purchase order"
            onRetry={() => void q.refetch()}
          />
        )}
      </>
    )
  }
  const po = q.data
  const a = po.actions
  const ownSubmitted =
    po.status === 'SUBMITTED' &&
    (po.createdBy === perms.meId || po.submittedBy === perms.meId) &&
    !a.approve

  return (
    <>
      <PageHeader
        title={po.number}
        breadcrumbs={[...crumbs, { label: po.number }]}
        status={<POStatusBadge status={po.status} />}
        description={`${po.supplierName} · created by ${po.createdByName} on ${formatDateTime(po.createdAt)}`}
        actions={
          <>
            {a.submit && (
              <Button
                onClick={() => {
                  setOpen({ kind: 'submit' })
                }}
              >
                Submit
              </Button>
            )}
            {a.approve && (
              <Button
                onClick={() => {
                  setOpen({ kind: 'approve' })
                }}
              >
                Approve
              </Button>
            )}
            {a.return && (
              <Button
                variant="secondary"
                onClick={() => {
                  setOpen({ kind: 'return' })
                }}
              >
                Return to draft
              </Button>
            )}
            {a.edit && (
              <Button
                variant="secondary"
                onClick={() => {
                  setOpen({ kind: 'edit' })
                }}
              >
                <Pencil className="size-4" aria-hidden="true" />
                Edit
              </Button>
            )}
            {a.cancel && (
              <Button
                variant="ghost"
                onClick={() => {
                  setOpen({ kind: 'cancel' })
                }}
              >
                Cancel PO
              </Button>
            )}
          </>
        }
      />
      <p
        role="note"
        className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
      >
        A supplier record is <strong>not proof of authorisation</strong>. The server allows
        submission and approval only for a verified supplier with a current verified relationship to
        every brand on the order, and only for brands, models and variants activated for purchase.
      </p>
      {ownSubmitted && (
        <p role="note" className="mb-4 rounded-card bg-info-subtle px-4 py-3 text-sm text-ink">
          You created or submitted this purchase order, so another authorised person must approve
          it.
        </p>
      )}
      {po.status === 'CANCELLED' && (
        <p role="note" className="mb-4 rounded-card bg-neutral-subtle px-4 py-3 text-sm text-ink">
          Cancelled {formatDateTime(po.cancelledAt)}: {po.cancelReason}. It is kept as history.
        </p>
      )}
      <Blockers po={po} />

      <div className="grid items-start gap-6 lg:grid-cols-3">
        <Card title="Order" className="lg:col-span-2">
          <DefinitionList
            items={[
              ['Supplier', <SupplierCell key="s" po={po} />],
              ['Expected delivery', formatDate(po.expectedDate)],
              ['Notes', po.notes ?? '—'],
              ['Currency', `${po.currency}, amounts excluding tax`],
              ['Submitted', po.submittedAt ? formatDateTime(po.submittedAt) : '—'],
              ['Approval', <ApprovalCell key="a" po={po} meId={perms.meId} />],
            ]}
          />
        </Card>
        <Card title="Totals" description="Calculated by the server from the lines.">
          <DefinitionList
            items={[
              ['Total (excl. tax)', <strong key="t">{formatINR(po.totalAmount)}</strong>],
              ['Lines', String(po.lineCount)],
              ['Units ordered', String(po.totalQuantity)],
              ['Received / remaining', remaining(po)],
            ]}
          />
        </Card>
      </div>

      <div className="mt-6 flex flex-col gap-6">
        <LinesCard
          po={po}
          onAdd={() => {
            setOpen({ kind: 'line' })
          }}
          onEdit={(line) => {
            setOpen({ kind: 'editLine', line })
          }}
          onRemove={(line) => {
            setOpen({ kind: 'removeLine', line })
          }}
        />
        <HistoryCard po={po} meId={perms.meId} />
      </div>

      <HeaderFormDialog open={open?.kind === 'edit'} onClose={close} po={po} />
      <LineFormDialog open={open?.kind === 'line'} onClose={close} po={po} />
      {open?.kind === 'editLine' && (
        <LineFormDialog open onClose={close} po={po} line={open.line} />
      )}
      {open?.kind === 'removeLine' && <RemoveLineDialog po={po} line={open.line} onClose={close} />}
      {open?.kind === 'submit' && <SubmitDialog po={po} onClose={close} />}
      {open?.kind === 'approve' && <ApproveDialog po={po} onClose={close} />}
      {open?.kind === 'return' && <ReturnDialog po={po} onClose={close} />}
      {open?.kind === 'cancel' && <CancelDialog po={po} onClose={close} />}
    </>
  )
}

function SupplierCell({ po }: { po: PODetail }) {
  return (
    <span className="flex flex-wrap items-center gap-2">
      <Link to={`/procurement/suppliers/${po.supplierId}`} className={linkClass}>
        {po.supplierName}
      </Link>
      <SupplierStatusBadge status={po.supplierStatus} />
    </span>
  )
}

function ApprovalCell({ po, meId }: { po: PODetail; meId: string | undefined }) {
  if (po.approvalBasis === 'APPROVER') {
    return (
      <>
        Approved by {personLabel(po.approvedBy, meId)} on {formatDateTime(po.approvedAt)}
        {po.approvalNote && <span className="block text-muted">{po.approvalNote}</span>}
      </>
    )
  }
  if (po.approvalBasis === 'BELOW_THRESHOLD') {
    return (
      <>
        Not required: total within the signed-off threshold of{' '}
        {po.thresholdApplied ? formatINR(po.thresholdApplied) : '—'}
      </>
    )
  }
  if (po.status === 'SUBMITTED') return <>Awaiting an approver</>
  if (po.status === 'DRAFT' && po.approvalReason) return <>{po.approvalReason}</>
  return <>—</>
}

/** Unmet prerequisites the server reports for the next step. */
function Blockers({ po }: { po: PODetail }) {
  if (po.blockers.length === 0) return null
  const step = po.status === 'DRAFT' ? 'submitted' : 'approved'
  return (
    <div
      role="status"
      className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
    >
      <p className="flex items-center gap-2 font-medium">
        <AlertTriangle className="size-4 text-warning" aria-hidden="true" />
        Cannot be {step} yet:
      </p>
      <ul className="ml-6 mt-1 list-disc space-y-0.5">
        {po.blockers.map((b, i) => (
          <li key={`${b.key}-${String(i)}`}>
            <span className="font-medium">{checkLabel(b.key)}</span>
            <span className="text-muted"> — {b.message}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function LinesCard({
  po,
  onAdd,
  onEdit,
  onRemove,
}: {
  po: PODetail
  onAdd: () => void
  onEdit: (l: POLine) => void
  onRemove: (l: POLine) => void
}) {
  const edit = po.actions.edit
  const cols: Column<POLine>[] = [
    {
      key: 'item',
      header: 'Line',
      render: (l) => (
        <div className="flex flex-col">
          <span className="font-medium">
            {l.lineNo}. {l.brandName} · {l.modelName} · {l.variantName}
          </span>
          <span className="flex flex-wrap gap-1 pt-1">
            {l.brandStatus !== 'ACTIVE' && <LifecycleBadge status={l.brandStatus} />}
            {l.variantStatus !== 'ACTIVE' && <LifecycleBadge status={l.variantStatus} />}
          </span>
          {l.note && <span className="text-xs text-muted">{l.note}</span>}
        </div>
      ),
    },
    { key: 'qty', header: 'Quantity', render: (l) => String(l.quantity) },
    {
      key: 'cost',
      header: 'Agreed unit cost',
      render: (l) => <span className="tabular-nums">{formatINR(l.unitCost)}</span>,
    },
    {
      key: 'ref',
      header: 'Catalogue cost (snapshot)',
      render: (l) =>
        l.referenceUnitCost ? (
          <span className="text-xs">
            {formatINR(l.referenceUnitCost)}
            <span className="block text-muted">
              taken {formatDateTime(l.referenceCapturedAt)}
              {l.costVariance && ` · difference ${formatINR(l.costVariance)}`}
            </span>
          </span>
        ) : l.referenceHidden ? (
          <span className="text-xs text-muted">Recorded (not visible to you)</span>
        ) : (
          <span className="text-xs text-muted">None</span>
        ),
    },
    {
      key: 'total',
      header: 'Line total',
      render: (l) => <span className="tabular-nums">{formatINR(l.lineTotal)}</span>,
    },
    {
      key: 'remaining',
      header: 'Received / remaining',
      render: (l) => `${String(l.receivedQuantity)} / ${String(l.remainingQuantity)}`,
    },
    {
      key: 'warnings',
      header: 'Warnings',
      render: (l) =>
        l.warnings.length === 0 ? (
          <span className="text-xs text-muted">—</span>
        ) : (
          <span className="flex flex-col gap-1">
            {l.warnings.map((w) => (
              <Badge key={w} tone="warning">
                {warningLabels[w]}
              </Badge>
            ))}
          </span>
        ),
    },
  ]
  if (edit) {
    cols.push({
      key: 'actions',
      header: 'Actions',
      render: (l) => (
        <span className="flex gap-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => {
              onEdit(l)
            }}
            aria-label={`Edit line ${String(l.lineNo)}`}
          >
            <Pencil className="size-3.5" aria-hidden="true" />
            Edit
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              onRemove(l)
            }}
            aria-label={`Remove line ${String(l.lineNo)}`}
          >
            <Trash2 className="size-3.5" aria-hidden="true" />
            Remove
          </Button>
        </span>
      ),
    })
  }
  return (
    <Card
      title="Lines"
      description={
        edit
          ? 'One line per variant. Lines can change only while the purchase order is a draft.'
          : 'Lines are fixed after submission.'
      }
      actions={
        edit && (
          <Button size="sm" variant="secondary" onClick={onAdd}>
            <Plus className="size-3.5" aria-hidden="true" />
            Add line
          </Button>
        )
      }
    >
      {po.lines.length === 0 ? (
        <EmptyState title="No lines yet">
          {edit ? 'Add the variants and quantities agreed with the supplier.' : 'Nothing ordered.'}
        </EmptyState>
      ) : (
        <ResponsiveTable caption="Lines" columns={cols} rows={po.lines} rowKey={(l) => l.id} />
      )}
    </Card>
  )
}

function HistoryCard({ po, meId }: { po: PODetail; meId: string | undefined }) {
  return (
    <Card title="Status history" description="Every transition, oldest first. Nothing is deleted.">
      <ol className="flex flex-col gap-3 text-sm">
        {po.events.map((e) => (
          <li key={e.id} className="flex flex-col border-l-2 border-border pl-3">
            <span className="font-medium text-ink">{eventLabels[e.action]}</span>
            <span className="text-xs text-muted">
              {formatDateTime(e.createdAt)} ·{' '}
              {e.actorId === meId ? 'You' : (e.actorName ?? personLabel(e.actorId, meId))}
            </span>
            {e.note && <span className="text-xs text-ink">{e.note}</span>}
          </li>
        ))}
      </ol>
    </Card>
  )
}
