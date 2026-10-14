import {
  Archive,
  CheckCircle2,
  Clock,
  History,
  Pencil,
  Plus,
  Replace,
  Search,
  XCircle,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Badge, type BadgeTone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { personLabel } from '@/features/catalogue/lib/presentation'
import { isNoPrice, type PriceItem, type PriceList, type PriceType } from '../api/pricing'
import { usePricePerms } from '../hooks/usePricePermissions'
import { usePriceItems, usePriceLists, usePriceLookup } from '../hooks/usePricing'
import {
  formatINR,
  formatInstant,
  localInputToRFC3339,
  pricePeriod,
  toLocalInput,
  type PricePeriod,
} from '../lib/format'
import {
  ApproveDialog,
  ArchivePriceListDialog,
  CancelDraftDialog,
  CreatePriceListDialog,
  PriceFormDialog,
  SnapshotNotice,
  type PriceFormTarget,
} from './PriceDialogs'

const periodBadge: Record<PricePeriod, { tone: BadgeTone; label: string; icon: ReactNode }> = {
  CURRENT: {
    tone: 'success',
    label: 'Current',
    icon: <CheckCircle2 className="size-3" aria-hidden="true" />,
  },
  FUTURE: { tone: 'info', label: 'Future', icon: <Clock className="size-3" aria-hidden="true" /> },
  PAST: { tone: 'neutral', label: 'Past', icon: <History className="size-3" aria-hidden="true" /> },
  DRAFT: {
    tone: 'warning',
    label: 'Draft — not approved',
    icon: <Pencil className="size-3" aria-hidden="true" />,
  },
  CANCELLED: {
    tone: 'neutral',
    label: 'Cancelled draft',
    icon: <XCircle className="size-3" aria-hidden="true" />,
  },
}

export function PeriodBadge({ item }: { item: PriceItem }) {
  const b = periodBadge[pricePeriod(item)]
  return (
    <Badge tone={b.tone} icon={b.icon}>
      {b.label}
    </Badge>
  )
}

// ── Brand page: price lists ─────────────────────────────────────────────────

export function PriceListsCard({
  brandId,
  brandName,
  brandArchived,
}: {
  brandId: string
  brandName: string
  brandArchived: boolean
}) {
  const perms = usePricePerms()
  const lists = usePriceLists({ brandId, limit: 100 })
  const [adding, setAdding] = useState(false)
  const [archiving, setArchiving] = useState<PriceList | null>(null)
  const columns: Column<PriceList>[] = [
    {
      key: 'name',
      header: 'Price list',
      render: (l) => <span className="font-medium">{l.name}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      render: (l) =>
        l.lifecycleStatus === 'ACTIVE' ? (
          <Badge tone="success">Active</Badge>
        ) : (
          <Badge tone="neutral">Archived</Badge>
        ),
    },
    { key: 'currency', header: 'Currency', render: (l) => l.currency },
    {
      key: 'actions',
      header: 'Actions',
      render: (l) =>
        perms.manageLists && l.lifecycleStatus === 'ACTIVE' ? (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setArchiving(l)
            }}
          >
            <Archive className="size-3.5" aria-hidden="true" />
            Archive
          </Button>
        ) : null,
    },
  ]
  return (
    <Card
      title="Price lists"
      description="Prices are recorded per variant on a price list. Open a variant to see and manage its prices."
      actions={
        perms.manageLists &&
        !brandArchived && (
          <Button
            size="sm"
            onClick={() => {
              setAdding(true)
            }}
          >
            <Plus className="size-3.5" aria-hidden="true" />
            Add price list
          </Button>
        )
      }
    >
      {lists.isPending && <LoadingState label="Loading price lists…" />}
      {lists.isError && (
        <ErrorState
          error={lists.error}
          title="Could not load price lists"
          onRetry={() => void lists.refetch()}
        />
      )}
      {lists.data?.total === 0 && (
        <EmptyState title="No price lists yet">
          {perms.manageLists
            ? 'Add a price list, then draft prices on each variant.'
            : 'No price list has been created for this brand.'}
        </EmptyState>
      )}
      {lists.data && lists.data.total > 0 && (
        <ResponsiveTable
          caption="Price lists"
          columns={columns}
          rows={lists.data.items}
          rowKey={(l) => l.id}
        />
      )}
      <CreatePriceListDialog
        open={adding}
        onClose={() => {
          setAdding(false)
        }}
        brandId={brandId}
        brandName={brandName}
      />
      {archiving && (
        <ArchivePriceListDialog
          open
          onClose={() => {
            setArchiving(null)
          }}
          list={archiving}
        />
      )}
    </Card>
  )
}

// ── Variant page: price history ─────────────────────────────────────────────

type Dialogs =
  | { kind: 'form'; target: PriceFormTarget }
  | { kind: 'approve'; item: PriceItem }
  | { kind: 'cancel'; item: PriceItem }
  | null

export function VariantPricesCard({
  brandId,
  variantId,
  variantUsable,
}: {
  brandId: string
  variantId: string
  /** Not archived (drafts allowed); approval rules are the server's. */
  variantUsable: boolean
}) {
  const perms = usePricePerms()
  const lists = usePriceLists({ brandId, limit: 100 })
  const [listId, setListId] = useState('')
  const types: PriceType[] = [
    ...(perms.viewSelling ? (['SELLING'] as const) : []),
    ...(perms.viewCost ? (['COST'] as const) : []),
  ]
  const [type, setType] = useState<PriceType | ''>('')
  const priceType: PriceType | undefined = type || types[0]
  const list =
    lists.data?.items.find((l) => l.id === listId) ??
    lists.data?.items.find((l) => l.lifecycleStatus === 'ACTIVE') ??
    lists.data?.items[0]
  const [page, setPage] = useState(1)
  const items = usePriceItems(
    { priceListId: list?.id, variantId, priceType, page, limit: 25 },
    { enabled: Boolean(list && priceType) },
  )
  const [dialog, setDialog] = useState<Dialogs>(null)
  const close = () => {
    setDialog(null)
  }
  const canDraft = Boolean(
    priceType && perms.canDraft(priceType) && list?.lifecycleStatus === 'ACTIVE' && variantUsable,
  )

  const columns: Column<PriceItem>[] = [
    {
      key: 'amount',
      header: 'Amount',
      render: (i) => (
        <div className="flex flex-col">
          <span className="font-semibold tabular-nums">{formatINR(i.amount)}</span>
          {i.sourceNote && <span className="text-xs text-muted">{i.sourceNote}</span>}
        </div>
      ),
    },
    { key: 'period', header: 'Status', render: (i) => <PeriodBadge item={i} /> },
    {
      key: 'dates',
      header: 'Effective',
      className: 'whitespace-nowrap',
      render: (i) => (
        <span className="text-sm">
          {formatInstant(i.effectiveFrom)} →{' '}
          {i.effectiveTo ? formatInstant(i.effectiveTo) : 'open-ended'}
        </span>
      ),
    },
    {
      key: 'approval',
      header: 'Approval',
      render: (i) =>
        i.status === 'APPROVED' ? (
          <span className="text-xs">
            {personLabel(i.approvedBy, perms.meId)} · {formatInstant(i.approvedAt)}
            {i.approvalReference && (
              <span className="block text-muted">Ref: {i.approvalReference}</span>
            )}
            {i.supersededById && <span className="block text-muted">Superseded</span>}
          </span>
        ) : (
          <span className="text-xs text-muted">
            Drafted by {personLabel(i.createdBy, perms.meId)}
            {i.status === 'DRAFT' && new Date(i.effectiveFrom) < new Date() && (
              <span className="block font-medium text-warning">
                Starts in the past — change the start before approval.
              </span>
            )}
          </span>
        ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (i) => {
        const btns: ReactNode[] = []
        if (i.status === 'DRAFT' && list && perms.canDraft(i.priceType)) {
          btns.push(
            <Button
              key="e"
              size="sm"
              variant="secondary"
              onClick={() => {
                setDialog({
                  kind: 'form',
                  target: { list, variantId, priceType: i.priceType, draft: i },
                })
              }}
            >
              Edit
            </Button>,
            <Button
              key="c"
              size="sm"
              variant="ghost"
              onClick={() => {
                setDialog({ kind: 'cancel', item: i })
              }}
            >
              Cancel
            </Button>,
          )
        }
        if (i.status === 'DRAFT' && perms.approve && i.createdBy !== perms.meId) {
          btns.push(
            <Button
              key="a"
              size="sm"
              onClick={() => {
                setDialog({ kind: 'approve', item: i })
              }}
            >
              Approve
            </Button>,
          )
        }
        const period = pricePeriod(i)
        if (i.status === 'APPROVED' && !i.supersededById && period !== 'PAST' && canDraft && list) {
          btns.push(
            <Button
              key="s"
              size="sm"
              variant="secondary"
              onClick={() => {
                setDialog({
                  kind: 'form',
                  target: { list, variantId, priceType: i.priceType, supersedes: i },
                })
              }}
            >
              <Replace className="size-3.5" aria-hidden="true" />
              Change price
            </Button>,
          )
        }
        return btns.length ? (
          <div className="flex flex-wrap gap-1.5 [&>button]:whitespace-nowrap">{btns}</div>
        ) : null
      },
    },
  ]

  let body: ReactNode
  if (lists.isPending) body = <LoadingState label="Loading price lists…" />
  else if (lists.isError)
    body = (
      <ErrorState
        error={lists.error}
        title="Could not load price lists"
        onRetry={() => void lists.refetch()}
      />
    )
  else if (!list) {
    body = (
      <EmptyState title="No price list for this brand">
        Prices are recorded on a price list.{' '}
        {perms.manageLists ? 'Add one on the brand page first.' : 'None has been created yet.'}
      </EmptyState>
    )
  } else if (items.isPending) body = <LoadingState label="Loading prices…" />
  else if (items.isError && !items.data) {
    body = (
      <ErrorState
        error={items.error}
        title="Could not load prices"
        onRetry={() => void items.refetch()}
      />
    )
  } else if (items.data.total === 0) {
    body = (
      <EmptyState title="No prices recorded">
        No {priceType === 'COST' ? 'cost' : 'selling'} price has been drafted for this variant on
        this list. Nothing is shown or implied.
      </EmptyState>
    )
  } else {
    body = (
      <>
        <ResponsiveTable
          caption="Price history"
          columns={columns}
          rows={items.data.items}
          rowKey={(i) => i.id}
        />
        {items.data.total > items.data.limit && (
          <div className="flex justify-end gap-2 pt-3">
            <Button
              size="sm"
              variant="secondary"
              disabled={page <= 1}
              onClick={() => {
                setPage(page - 1)
              }}
            >
              Previous
            </Button>
            <Button
              size="sm"
              variant="secondary"
              disabled={page * items.data.limit >= items.data.total}
              onClick={() => {
                setPage(page + 1)
              }}
            >
              Next
            </Button>
          </div>
        )}
      </>
    )
  }

  return (
    <Card
      title="Price history"
      description="Approved prices by effective date. Drafts are not used until approved by someone other than the drafter."
      actions={
        canDraft &&
        list &&
        priceType && (
          <Button
            size="sm"
            onClick={() => {
              setDialog({ kind: 'form', target: { list, variantId, priceType } })
            }}
          >
            <Plus className="size-3.5" aria-hidden="true" />
            Draft price
          </Button>
        )
      }
    >
      <div className="flex flex-col gap-4">
        <SnapshotNotice />
        {lists.data && lists.data.total > 0 && (
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <label className="flex flex-col gap-1.5 text-xs font-medium text-muted sm:w-64">
              Price list
              <Select
                value={list?.id ?? ''}
                options={lists.data.items.map((l) => ({
                  value: l.id,
                  label: l.lifecycleStatus === 'ARCHIVED' ? `${l.name} (archived)` : l.name,
                }))}
                onChange={(e) => {
                  setListId(e.target.value)
                  setPage(1)
                }}
              />
            </label>
            {types.length > 1 && (
              <label className="flex flex-col gap-1.5 text-xs font-medium text-muted sm:w-48">
                Price type
                <Select
                  value={priceType}
                  options={types.map((t) => ({
                    value: t,
                    label: t === 'COST' ? 'Cost price' : 'Selling price',
                  }))}
                  onChange={(e) => {
                    setType(e.target.value as PriceType)
                    setPage(1)
                  }}
                />
              </label>
            )}
          </div>
        )}
        {body}
        {list && priceType && (
          <PriceAtTime priceListId={list.id} variantId={variantId} priceType={priceType} />
        )}
      </div>
      {dialog?.kind === 'form' && <PriceFormDialog open onClose={close} target={dialog.target} />}
      {dialog?.kind === 'approve' && <ApproveDialog open onClose={close} item={dialog.item} />}
      {dialog?.kind === 'cancel' && <CancelDraftDialog open onClose={close} item={dialog.item} />}
    </Card>
  )
}

/** Asks the server which approved price was/is/will be valid at a chosen time. */
function PriceAtTime({
  priceListId,
  variantId,
  priceType,
}: {
  priceListId: string
  variantId: string
  priceType: PriceType
}) {
  const [input, setInput] = useState(() => toLocalInput(new Date()))
  const [asked, setAsked] = useState<string | null>(null)
  const lookup = usePriceLookup(
    { priceListId, variantId, priceType, at: asked ?? undefined },
    { enabled: asked !== null },
  )
  return (
    <div className="rounded-control border border-border p-3">
      <form
        className="flex flex-col gap-2 sm:flex-row sm:items-end"
        onSubmit={(e) => {
          e.preventDefault()
          if (input) setAsked(localInputToRFC3339(input))
        }}
      >
        <label className="flex flex-1 flex-col gap-1.5 text-xs font-medium text-muted">
          Price valid at (your local time)
          <Input
            type="datetime-local"
            value={input}
            onChange={(e) => {
              setInput(e.target.value)
            }}
          />
        </label>
        <Button type="submit" variant="secondary" loading={lookup.isFetching}>
          <Search className="size-3.5" aria-hidden="true" />
          Look up
        </Button>
      </form>
      <div aria-live="polite" className="mt-2 text-sm">
        {asked !== null && lookup.data && (
          <p>
            <span className="font-semibold">{formatINR(lookup.data.amount)}</span> — approved price
            valid from {formatInstant(lookup.data.effectiveFrom)}
            {lookup.data.effectiveTo
              ? ` until ${formatInstant(lookup.data.effectiveTo)}`
              : ' (open-ended)'}
            .
          </p>
        )}
        {asked !== null && lookup.isError && isNoPrice(lookup.error) && (
          <p className="text-muted">No approved price is valid at that time.</p>
        )}
        {asked !== null && lookup.isError && !isNoPrice(lookup.error) && (
          <ErrorState error={lookup.error} title="Could not look up the price" />
        )}
      </div>
    </div>
  )
}
