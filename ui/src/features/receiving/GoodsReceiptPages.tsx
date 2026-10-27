import { Download, PackageCheck, Paperclip, Plus } from 'lucide-react'
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
import { fetchAttachment } from '@/features/attachments/api/attachments'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, SearchBox, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useListParams } from '@/features/catalogue/lib/listParams'
import {
  describeError,
  formatDate,
  formatDateTime,
  personLabel,
} from '@/features/catalogue/lib/presentation'
import { poStatusLabels } from '@/features/purchasing/lib/labels'
import { ApiError } from '@/lib/api/client'
import type { GoodsReceipt, GRNDetail, GRNDocument, GRNLine, GRNStatus } from './api/goodsReceipts'
import { CancelReceiptDialog, DocumentDialog, NewReceiptDialog } from './components/ReceiptDialogs'
import { ReceiptEditor } from './components/ReceiptEditor'
import { useGoodsReceipt, useGoodsReceiptList } from './hooks/useGoodsReceipts'
import {
  conditionLabels,
  documentTypeLabels,
  eventLabels,
  grnStatusLabels,
  lineWarningLabels,
  unitWarningLabels,
} from './lib/labels'

const listPath = '/procurement/goods-receipts'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Goods Receipts', to: listPath },
]
const linkClass = 'font-medium text-primary hover:underline'

const statusTone: Record<GRNStatus, BadgeTone> = {
  DRAFT: 'neutral',
  POSTED: 'success',
  CANCELLED: 'danger',
}

export function GRNStatusBadge({ status }: { status: GRNStatus }) {
  return <Badge tone={statusTone[status]}>{grnStatusLabels[status]}</Badge>
}

function usePerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.GoodsReceiptView),
    manage: p.includes(Permission.GoodsReceiptManage),
    attachments: p.includes(Permission.AttachmentsView),
    purchaseOrders: p.includes(Permission.PurchaseOrderView),
  }
}

/** Gate for both pages: goods_receipts.view, checked again by the API.
 * `header` keeps a detail page's heading and breadcrumbs while checking or denied. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = usePerms()
  if (me.isPending)
    return (
      <>
        {header}
        <LoadingState label="Checking your permissions…" />
      </>
    )
  if (me.isError && !me.data) {
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
  }
  if (!view)
    return (
      <>
        {header}
        <AccessDeniedState what="goods receipts" />
      </>
    )
  return <>{children}</>
}

function NoStockNote() {
  return (
    <p
      role="note"
      className="mb-4 rounded-card border border-info/30 bg-info-subtle px-4 py-3 text-sm text-ink"
    >
      A draft receipt <strong>does not change stock</strong>. It records what arrived, the unit
      identifiers and the supplier’s documents. Posting the receipt — which creates the vehicle
      units and stock movements — is a separate step that is not available yet.
    </p>
  )
}

function quantities(r: GoodsReceipt) {
  return `${String(r.receivedQuantity)} received · ${String(r.acceptedQuantity)} accepted · ${String(r.rejectedQuantity)} rejected`
}

// ── List ────────────────────────────────────────────────────────────────────

export function GoodsReceiptsPage() {
  const { manage, view } = usePerms()
  const [creating, setCreating] = useState(false)
  const navigate = useNavigate()
  const add = () => {
    setCreating(true)
  }
  return (
    <>
      <PageHeader
        title="Goods Receipts"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Goods Receipts' }]}
        description="Deliveries received against approved purchase orders: quantities, unit identifiers, documents and discrepancies."
        actions={
          manage && view ? (
            <Button onClick={add}>
              <Plus className="size-4" aria-hidden="true" />
              New goods receipt
            </Button>
          ) : undefined
        }
      />
      <Gate>
        <NoStockNote />
        <ReceiptList canAdd={manage} onAdd={add} />
      </Gate>
      <NewReceiptDialog
        open={creating}
        onClose={() => {
          setCreating(false)
        }}
        onSaved={(d) => void navigate(`${listPath}/${d.id}`)}
      />
    </>
  )
}

function ReceiptList({ canAdd, onAdd }: { canAdd: boolean; onAdd: () => void }) {
  const { values, page, set } = useListParams(['q', 'status'] as const)
  const query = useGoodsReceiptList({
    q: values.q || undefined,
    status: (values.status || undefined) as GRNStatus | undefined,
    page,
    limit: 25,
  })
  const onSearch = useCallback((q: string) => {
    set({ q })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- set is stable in behaviour
  }, [])
  const filtered = Boolean(values.q || values.status)
  const cols: Column<GoodsReceipt>[] = [
    {
      key: 'number',
      header: 'GRN',
      render: (r) => (
        <div className="flex flex-col">
          <Link to={`${listPath}/${r.id}`} className={`${linkClass} font-mono`}>
            {r.number}
          </Link>
          {r.supplierReference && (
            <span className="text-xs text-muted">Ref. {r.supplierReference}</span>
          )}
        </div>
      ),
    },
    {
      key: 'po',
      header: 'Purchase order',
      render: (r) => (
        <span className="flex flex-col">
          <span className="font-mono">{r.purchaseOrderNumber}</span>
          <span className="text-xs text-muted">{r.supplierName}</span>
        </span>
      ),
    },
    { key: 'status', header: 'Status', render: (r) => <GRNStatusBadge status={r.status} /> },
    { key: 'date', header: 'Received', render: (r) => formatDate(r.receivedDate) },
    {
      key: 'qty',
      header: 'Quantities',
      render: (r) => <span className="text-xs">{quantities(r)}</span>,
    },
    {
      key: 'units',
      header: 'Units captured',
      render: (r) => `${String(r.unitCount)} of ${String(r.receivedQuantity)}`,
    },
    { key: 'creator', header: 'Started by', render: (r) => r.createdByName },
  ]
  const add = canAdd ? (
    <Button onClick={onAdd}>
      <Plus className="size-4" aria-hidden="true" />
      New goods receipt
    </Button>
  ) : null

  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading goods receipts…" />
  else if (query.isError && !query.data) {
    body = (
      <ErrorState
        error={query.error}
        title="Could not load goods receipts"
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.data.total === 0) {
    body = filtered ? (
      <EmptyState title="No goods receipts match these filters">
        Change the search or clear the filters.
      </EmptyState>
    ) : (
      <EmptyState
        title="No goods receipts yet"
        icon={<PackageCheck className="size-6 text-muted/80" />}
        action={add}
      >
        When a delivery arrives, start a draft against its approved purchase order and capture each
        unit.
      </EmptyState>
    )
  } else {
    body = (
      <div className={query.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Goods receipts"
          columns={cols}
          rows={query.data.items}
          rowKey={(r) => r.id}
        />
        <Pagination
          page={query.data.page}
          limit={query.data.limit}
          total={query.data.total}
          noun={['goods receipt', 'goods receipts']}
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
          set({ q: '', status: '' })
        }}
      >
        <SearchBox
          label="Search GRN, PO, supplier or reference"
          value={values.q}
          onSearch={onSearch}
        />
        <FilterSelect
          label="Status"
          value={values.status}
          options={Object.entries(grnStatusLabels).map(([value, label]) => ({ value, label }))}
          allLabel="All statuses"
          onChange={(v) => {
            set({ status: v })
          }}
        />
      </Toolbar>
      {body}
    </>
  )
}

// ── Detail ──────────────────────────────────────────────────────────────────

export function GoodsReceiptDetailPage() {
  const { grnId = '' } = useParams()
  return (
    <Gate
      header={
        <PageHeader title="Goods receipt" breadcrumbs={[...crumbs, { label: 'Goods receipt' }]} />
      }
    >
      <ReceiptDetailBody id={grnId} />
    </Gate>
  )
}

function ReceiptDetailBody({ id }: { id: string }) {
  const q = useGoodsReceipt(id)
  const perms = usePerms()
  const [open, setOpen] = useState<'cancel' | 'document' | null>(null)
  const close = () => {
    setOpen(null)
  }

  if (!q.data) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <>
        <PageHeader title="Goods receipt" breadcrumbs={[...crumbs, { label: 'Goods receipt' }]} />
        {q.isPending ? (
          <LoadingState label="Loading goods receipt…" />
        ) : notFound ? (
          <EmptyState title="This goods receipt does not exist">
            <Link to={listPath} className={linkClass}>
              Back to goods receipts
            </Link>
          </EmptyState>
        ) : (
          <ErrorState
            error={q.error}
            title="Could not load the goods receipt"
            onRetry={() => void q.refetch()}
          />
        )}
      </>
    )
  }
  const grn = q.data
  const a = grn.actions
  const documents = (
    <DocumentsCard
      grn={grn}
      canOpen={perms.attachments}
      onAdd={() => {
        setOpen('document')
      }}
    />
  )

  return (
    <>
      <PageHeader
        title={grn.number}
        breadcrumbs={[...crumbs, { label: grn.number }]}
        status={<GRNStatusBadge status={grn.status} />}
        description={`${grn.purchaseOrderNumber} · ${grn.supplierName} · received ${formatDate(grn.receivedDate)} · started by ${grn.createdByName}`}
        actions={
          a.cancel ? (
            <Button
              variant="ghost"
              onClick={() => {
                setOpen('cancel')
              }}
            >
              Cancel draft
            </Button>
          ) : undefined
        }
      />
      {grn.status === 'DRAFT' && <NoStockNote />}
      {grn.status === 'DRAFT' && !grn.purchaseOrderReceivable && (
        <p
          role="alert"
          className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
        >
          The purchase order is now {poStatusLabels[grn.purchaseOrderStatus].toLowerCase()}, so this
          draft can no longer change. Cancel it if the delivery will not be posted.
        </p>
      )}
      {grn.status === 'CANCELLED' && (
        <p role="note" className="mb-4 rounded-card bg-neutral-subtle px-4 py-3 text-sm text-ink">
          Cancelled {formatDateTime(grn.cancelledAt)}: {grn.cancelReason}. It is kept as history and
          holds no purchase order quantity.
        </p>
      )}

      <Card title="Receipt" className="mb-6">
        <DefinitionList
          items={[
            [
              'Purchase order',
              perms.purchaseOrders ? (
                <Link
                  key="po"
                  to={`/procurement/purchase-orders/${grn.purchaseOrderId}`}
                  className={linkClass}
                >
                  {grn.purchaseOrderNumber}
                </Link>
              ) : (
                grn.purchaseOrderNumber
              ),
            ],
            ['Supplier', grn.supplierName],
            ['Supplier reference', grn.supplierReference ?? '—'],
            ['Totals (server)', quantities(grn)],
            ['Units captured', `${String(grn.unitCount)} of ${String(grn.receivedQuantity)}`],
            ['Stock', grn.stockPosted ? 'Posted' : 'Not changed (draft)'],
          ]}
        />
      </Card>

      {a.edit ? (
        <ReceiptEditor key={grn.id} grn={grn} documents={documents} />
      ) : (
        <div className="flex flex-col gap-6">
          <ReadOnlyLines grn={grn} />
          {documents}
        </div>
      )}
      <div className="mt-6">
        <HistoryCard grn={grn} meId={perms.meId} />
      </div>

      {open === 'cancel' && <CancelReceiptDialog grn={grn} onClose={close} />}
      <DocumentDialog open={open === 'document'} grn={grn} onClose={close} />
    </>
  )
}

function ReadOnlyLines({ grn }: { grn: GRNDetail }) {
  const lines = grn.lines.filter((l) => l.id !== null)
  const cols: Column<GRNLine>[] = [
    {
      key: 'item',
      header: 'Line',
      render: (l) => (
        <span className="flex flex-col">
          <span className="font-medium">
            {l.lineNo}. {l.brandName} · {l.modelName} · {l.variantName}
          </span>
          {l.discrepancyNote && <span className="text-xs text-muted">{l.discrepancyNote}</span>}
        </span>
      ),
    },
    { key: 'ordered', header: 'Ordered', render: (l) => String(l.orderedQuantity) },
    {
      key: 'qty',
      header: 'Received / accepted / rejected',
      render: (l) =>
        `${String(l.receivedQuantity)} / ${String(l.acceptedQuantity)} / ${String(l.rejectedQuantity)}`,
    },
    {
      key: 'over',
      header: 'Over-receipt',
      render: (l) =>
        l.overReceipt ? (
          <span className="text-xs">
            Authorised by {l.overReceipt.authorisedByName}: {l.overReceipt.reason}
          </span>
        ) : (
          <span className="text-xs text-muted">—</span>
        ),
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
                {lineWarningLabels[w]}
              </Badge>
            ))}
          </span>
        ),
    },
  ]
  return (
    <>
      <Card title="Delivered lines">
        {lines.length === 0 ? (
          <EmptyState title="Nothing recorded">
            No line of the purchase order was entered.
          </EmptyState>
        ) : (
          <ResponsiveTable
            caption="Delivered lines"
            columns={cols}
            rows={lines}
            rowKey={(l) => l.purchaseOrderLineId}
          />
        )}
      </Card>
      <Card title="Unit identifiers">
        {lines.every((l) => l.units.length === 0) ? (
          <EmptyState title="No units captured">No unit identifiers were recorded.</EmptyState>
        ) : (
          <ul className="flex flex-col gap-2 text-sm">
            {lines.flatMap((l) =>
              l.units.map((u) => (
                <li
                  key={u.id}
                  className="flex flex-col gap-1 rounded-control border border-border px-3 py-2"
                >
                  <span className="font-medium text-ink">
                    Line {l.lineNo}, unit {u.unitNo} · {conditionLabels[u.condition]}
                  </span>
                  <span className="font-mono text-xs text-ink">
                    Chassis {u.chassisNumber ?? '—'} · motor {u.motorSerial ?? '—'} · battery{' '}
                    {u.batterySerial ?? '—'} · charger {u.chargerSerial ?? '—'}
                  </span>
                  {u.note && <span className="text-xs text-muted">{u.note}</span>}
                  {u.warnings.length > 0 && (
                    <span className="flex flex-wrap gap-1">
                      {u.warnings.map((w) => (
                        <Badge key={w} tone="warning">
                          {unitWarningLabels[w]}
                        </Badge>
                      ))}
                    </span>
                  )}
                </li>
              )),
            )}
          </ul>
        )}
      </Card>
    </>
  )
}

function DocumentsCard({
  grn,
  canOpen,
  onAdd,
}: {
  grn: GRNDetail
  canOpen: boolean
  onAdd: () => void
}) {
  const [error, setError] = useState<string | null>(null)
  const download = async (d: GRNDocument) => {
    setError(null)
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
      setError(describeError(e).message)
    }
  }
  const cols: Column<GRNDocument>[] = [
    {
      key: 'file',
      header: 'Document',
      render: (d) => <span className="font-medium">{d.fileName}</span>,
    },
    { key: 'type', header: 'Type', render: (d) => documentTypeLabels[d.documentType] },
    {
      key: 'added',
      header: 'Added',
      render: (d) => (
        <span className="text-xs">
          {formatDateTime(d.createdAt)} · {d.addedByName}
          {d.note && <span className="block text-muted">{d.note}</span>}
        </span>
      ),
    },
    {
      key: 'open',
      header: 'Open',
      render: (d) =>
        canOpen ? (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void download(d)}
            aria-label={`Download ${d.fileName}`}
          >
            <Download className="size-3.5" aria-hidden="true" />
            Download
          </Button>
        ) : null,
    },
  ]
  return (
    <Card
      title="Supplier documents"
      description="Delivery challan, supplier invoice and photos. Private files, opened through short-lived links."
      actions={
        grn.actions.addDocument && (
          <Button size="sm" type="button" onClick={onAdd}>
            <Paperclip className="size-3.5" aria-hidden="true" />
            Attach document
          </Button>
        )
      }
    >
      {error && (
        <p role="alert" className="mb-3 text-sm text-danger">
          {error}
        </p>
      )}
      {!canOpen && grn.documents.length > 0 && (
        <p className="mb-3 text-xs text-muted">
          Opening documents needs permission to view documents.
        </p>
      )}
      {grn.documents.length === 0 ? (
        <EmptyState title="No documents">No supplier document has been attached.</EmptyState>
      ) : (
        <ResponsiveTable
          caption="Supplier documents"
          columns={cols}
          rows={grn.documents}
          rowKey={(d) => d.id}
        />
      )}
    </Card>
  )
}

function HistoryCard({ grn, meId }: { grn: GRNDetail; meId: string | undefined }) {
  return (
    <Card title="Status history" description="Every transition, oldest first. Nothing is deleted.">
      <ol className="flex flex-col gap-3 text-sm">
        {grn.events.map((e) => (
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

/** Receipts of one purchase order, for the PO page (goods_receipts.view). */
export function PurchaseOrderReceipts({
  purchaseOrderId,
  receivable,
}: {
  purchaseOrderId: string
  receivable: boolean
}) {
  const { view, manage } = usePerms()
  const navigate = useNavigate()
  const [creating, setCreating] = useState(false)
  const query = useGoodsReceiptList({ purchaseOrderId, limit: 50 }, { enabled: view })
  if (!view) return null
  let body: ReactNode
  if (query.isPending) body = <LoadingState label="Loading goods receipts…" />
  else if (query.isError && !query.data) {
    body = (
      <ErrorState
        error={query.error}
        title="Could not load goods receipts"
        onRetry={() => void query.refetch()}
      />
    )
  } else if (query.data.items.length === 0) {
    body = <p className="text-sm text-muted">No goods receipt yet.</p>
  } else {
    body = (
      <ul className="flex flex-col gap-2 text-sm">
        {query.data.items.map((r) => (
          <li key={r.id} className="flex flex-wrap items-center gap-2">
            <Link to={`${listPath}/${r.id}`} className={`${linkClass} font-mono`}>
              {r.number}
            </Link>
            <GRNStatusBadge status={r.status} />
            <span className="text-xs text-muted">
              {formatDate(r.receivedDate)} · {quantities(r)}
            </span>
          </li>
        ))}
      </ul>
    )
  }
  return (
    <Card
      title="Goods receipts"
      description="Deliveries against this purchase order. Drafts do not change received quantities."
      actions={
        manage &&
        receivable && (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => {
              setCreating(true)
            }}
          >
            <Plus className="size-3.5" aria-hidden="true" />
            Receive goods
          </Button>
        )
      }
    >
      {body}
      <NewReceiptDialog
        open={creating}
        purchaseOrderId={purchaseOrderId}
        onClose={() => {
          setCreating(false)
        }}
        onSaved={(d) => void navigate(`${listPath}/${d.id}`)}
      />
    </Card>
  )
}
