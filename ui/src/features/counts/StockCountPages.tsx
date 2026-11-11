import { ClipboardCheck, Plus } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link, NavLink, useNavigate, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Pagination } from '@/components/ui/Pagination'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { DefinitionList } from '@/features/catalogue/components/DetailParts'
import { FilterSelect, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useListParams } from '@/features/catalogue/lib/listParams'
import { formatDateTime } from '@/features/catalogue/lib/presentation'
import { ApiError } from '@/lib/api/client'
import type { CountLine, CountStatus, StockCount } from './api/stockCounts'
import {
  CreateCountDialog,
  DecisionDialog,
  LineDialog,
  SubmitDialog,
} from './components/CountDialogs'
import { tookPart, useCountPerms, useStockCount, useStockCountList } from './hooks/useStockCounts'
import {
  countStatusLabels,
  countStatusTone,
  expectedText,
  lineLabel,
  observedText,
  outcomeLabels,
  outcomeTone,
  varianceText,
} from './lib/labels'

const listPath = '/inventory/counts'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Stock Counts & Adjustments', to: listPath },
]
const linkClass = 'font-medium text-primary hover:underline'

export function CountStatusBadge({ status }: { status: CountStatus }) {
  return <Badge tone={countStatusTone[status]}>{countStatusLabels[status]}</Badge>
}

function who(name: string | null, id: string | null, meId: string | undefined) {
  if (!id) return '—'
  return id === meId ? 'You' : (name ?? 'Unknown user')
}

/** Gate for every page: stock.adjustment.view, checked again by the API. */
function Gate({ children, header }: { children: ReactNode; header?: ReactNode }) {
  const { me, view } = useCountPerms()
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
        <AccessDeniedState what="stock counts" />
      </>
    )
  return <>{children}</>
}

function PolicyNote() {
  return (
    <p
      role="note"
      className="mb-4 rounded-card border border-border/80 bg-surface px-4 py-3 text-sm text-ink"
    >
      <strong>No approval threshold is configured</strong> — every submitted count needs a decision
      by someone with approval permission who did not create, count or submit it. Approval posts the
      parts adjustments to the stock ledger; vehicle findings are recorded only and change no
      vehicle.
    </p>
  )
}

function Tabs({ approver }: { approver: boolean }) {
  const cls = ({ isActive }: { isActive: boolean }) =>
    `rounded-control px-3 py-1.5 text-sm font-medium ${isActive ? 'bg-primary text-primary-fg' : 'text-ink hover:bg-surface-muted'}`
  return (
    <nav aria-label="Stock count views" className="mb-4 flex flex-wrap gap-2">
      <NavLink to={listPath} end className={cls}>
        All counts
      </NavLink>
      {approver && (
        <NavLink to={`${listPath}/approvals`} className={cls}>
          Awaiting approval
        </NavLink>
      )}
    </nav>
  )
}

// ── Lists ───────────────────────────────────────────────────────────────────

export function StockCountsPage() {
  return <CountsPage queue={false} />
}

/** The approval queue: SUBMITTED counts at the caller's locations. */
export function CountApprovalsPage() {
  return <CountsPage queue />
}

function CountsPage({ queue }: { queue: boolean }) {
  const perms = useCountPerms()
  const [creating, setCreating] = useState(false)
  const navigate = useNavigate()
  const add =
    perms.create && perms.view ? (
      <Button
        onClick={() => {
          setCreating(true)
        }}
      >
        <Plus className="size-4" aria-hidden="true" />
        New count
      </Button>
    ) : undefined
  return (
    <>
      <PageHeader
        title={queue ? 'Counts awaiting approval' : 'Stock Counts & Adjustments'}
        breadcrumbs={
          queue
            ? [...crumbs, { label: 'Awaiting approval' }]
            : [{ label: 'Home', to: '/' }, { label: 'Stock Counts & Adjustments' }]
        }
        description="Counts of what is physically at a location. Stock changes only when another person approves a submitted count."
        actions={add}
      />
      <Gate>
        <Tabs approver={perms.approve} />
        <PolicyNote />
        <CountList queue={queue} add={add} />
      </Gate>
      {creating && (
        <CreateCountDialog
          onClose={() => {
            setCreating(false)
          }}
          onCreated={(c) => void navigate(`${listPath}/${c.id}`)}
        />
      )}
    </>
  )
}

function CountList({ queue, add }: { queue: boolean; add: ReactNode }) {
  const perms = useCountPerms()
  const { values, page, set } = useListParams(['status', 'locationId'] as const)
  const status = queue ? 'SUBMITTED' : (values.status as CountStatus | '')
  const q = useStockCountList({
    status: status || undefined,
    locationId: values.locationId || undefined,
    page,
    limit: 25,
  })
  const filtered = Boolean((!queue && values.status) || values.locationId)
  const cols: Column<StockCount>[] = [
    {
      key: 'count',
      header: 'Count',
      render: (c) => (
        <div className="flex flex-col">
          <Link to={`${listPath}/${c.id}`} className={`${linkClass} font-mono`}>
            {c.countNumber}
          </Link>
          {c.note && <span className="text-xs text-muted">{c.note}</span>}
        </div>
      ),
    },
    { key: 'loc', header: 'Location', render: (c) => `${c.locationCode} — ${c.locationName}` },
    { key: 'status', header: 'Status', render: (c) => <CountStatusBadge status={c.status} /> },
    {
      key: 'lines',
      header: 'Items',
      render: (c) => (
        <span>
          {c.lineCount}
          {perms.approve && c.varianceLines > 0 && (
            <span className="block text-xs text-muted">{c.varianceLines} differ</span>
          )}
        </span>
      ),
    },
    {
      key: 'created',
      header: 'Created',
      render: (c) => (
        <span className="text-xs">
          {who(c.createdByName, c.createdBy, perms.meId)}
          <span className="block text-muted">{formatDateTime(c.createdAt)}</span>
        </span>
      ),
    },
    {
      key: 'progress',
      header: queue ? 'Submitted' : 'Submitted / decided',
      render: (c) => (
        <span className="text-xs">
          {c.submittedAt
            ? `${who(c.submittedByName, c.submittedBy, perms.meId)}, ${formatDateTime(c.submittedAt)}`
            : '—'}
          {c.decidedAt && (
            <span className="block text-muted">
              {countStatusLabels[c.status]} by {who(c.decidedByName, c.decidedBy, perms.meId)},{' '}
              {formatDateTime(c.decidedAt)}
            </span>
          )}
          {queue && (c.createdBy === perms.meId || c.submittedBy === perms.meId) && (
            <span className="mt-1 block">
              <Badge tone="info">You took part — someone else decides</Badge>
            </span>
          )}
        </span>
      ),
    },
  ]
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading stock counts…" />
  else if (q.isError && !q.data)
    body = (
      <ErrorState
        error={q.error}
        title="Could not load stock counts"
        onRetry={() => void q.refetch()}
      />
    )
  else if (q.data.total === 0)
    body = filtered ? (
      <EmptyState title="No counts match these filters">Change or clear the filters.</EmptyState>
    ) : queue ? (
      <EmptyState title="Nothing is awaiting approval">
        Submitted counts at your locations appear here.
      </EmptyState>
    ) : (
      <EmptyState
        title="No stock counts yet"
        icon={<ClipboardCheck className="size-6 text-muted/80" />}
        action={add}
      >
        Start a count at a location, record what you find, then submit it for approval.
      </EmptyState>
    )
  else
    body = (
      <div className={q.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption={queue ? 'Counts awaiting approval' : 'Stock counts'}
          columns={cols}
          rows={q.data.items}
          rowKey={(c) => c.id}
        />
        <Pagination
          page={q.data.page}
          limit={q.data.limit}
          total={q.data.total}
          noun={['count', 'counts']}
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
          set({ status: '', locationId: '' })
        }}
      >
        {!queue && (
          <FilterSelect
            label="Status"
            value={values.status}
            options={Object.entries(countStatusLabels).map(([value, label]) => ({ value, label }))}
            allLabel="All statuses"
            onChange={(v) => {
              set({ status: v })
            }}
          />
        )}
        <FilterSelect
          label="Location"
          value={values.locationId}
          options={perms.locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
          allLabel="All my locations"
          onChange={(v) => {
            set({ locationId: v })
          }}
        />
      </Toolbar>
      {body}
    </>
  )
}

// ── Detail ──────────────────────────────────────────────────────────────────

export function StockCountDetailPage() {
  const { countId = '' } = useParams()
  return (
    <Gate
      header={
        <PageHeader title="Stock count" breadcrumbs={[...crumbs, { label: 'Stock count' }]} />
      }
    >
      <CountDetail id={countId} />
    </Gate>
  )
}

type Open =
  { kind: 'line'; line?: CountLine } | { kind: 'submit' } | { kind: 'approve' | 'reject' } | null

function CountDetail({ id }: { id: string }) {
  const q = useStockCount(id)
  const perms = useCountPerms()
  const [open, setOpen] = useState<Open>(null)
  const [done, setDone] = useState<StockCount | null>(null)
  const close = () => {
    setOpen(null)
  }
  if (!q.data) {
    const notFound = q.error instanceof ApiError && q.error.status === 404
    return (
      <>
        <PageHeader title="Stock count" breadcrumbs={[...crumbs, { label: 'Stock count' }]} />
        {q.isPending ? (
          <LoadingState label="Loading the count…" />
        ) : notFound ? (
          <EmptyState title="This count does not exist, or it is not at one of your locations">
            <Link to={listPath} className={linkClass}>
              Back to stock counts
            </Link>
          </EmptyState>
        ) : (
          <ErrorState
            error={q.error}
            title="Could not load the count"
            onRetry={() => void q.refetch()}
          />
        )}
      </>
    )
  }
  const c = q.data
  const draft = c.status === 'DRAFT'
  const submitted = c.status === 'SUBMITTED'
  const participant = tookPart(c, perms.meId)
  const canRecord = draft && perms.create
  const canDecide = submitted && perms.approve && !participant
  // Whether counters may see expected quantities is not decided (OQ-40):
  // only people who may approve see the stock record and the difference.
  const reviewer = perms.approve
  const stale = c.lines.filter((l) => l.stockChanged)

  return (
    <>
      <PageHeader
        title={c.countNumber}
        breadcrumbs={[...crumbs, { label: c.countNumber }]}
        status={<CountStatusBadge status={c.status} />}
        description={`${c.locationCode} — ${c.locationName} · started by ${who(c.createdByName, c.createdBy, perms.meId)} on ${formatDateTime(c.createdAt)}`}
        actions={
          <>
            {canRecord && (
              <Button
                onClick={() => {
                  setDone(null)
                  setOpen({ kind: 'line' })
                }}
              >
                <Plus className="size-4" aria-hidden="true" />
                Record item
              </Button>
            )}
            {canRecord && c.lineCount > 0 && (
              <Button
                variant="secondary"
                onClick={() => {
                  setOpen({ kind: 'submit' })
                }}
              >
                Submit for approval
              </Button>
            )}
            {canDecide && (
              <>
                <Button
                  onClick={() => {
                    setOpen({ kind: 'approve' })
                  }}
                >
                  Approve
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => {
                    setOpen({ kind: 'reject' })
                  }}
                >
                  Reject
                </Button>
              </>
            )}
          </>
        }
      />
      {done && <DecisionNote count={done} />}
      {submitted && perms.approve && participant && (
        <p role="note" className="mb-4 rounded-card bg-info-subtle px-4 py-3 text-sm text-ink">
          You took part in this count, so someone else with approval permission must approve or
          reject it.
        </p>
      )}
      {submitted && !perms.approve && (
        <p role="note" className="mb-4 rounded-card bg-info-subtle px-4 py-3 text-sm text-ink">
          Awaiting a decision by someone with approval permission who did not take part. Until then
          stock is unchanged.
        </p>
      )}
      {stale.length > 0 && (
        <div
          role="status"
          className="mb-4 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink"
        >
          <p className="font-medium">
            Stock moved after{' '}
            {stale.length === 1 ? 'one item was' : `${String(stale.length)} items were`} counted:
          </p>
          <ul className="ml-5 mt-1 list-disc">
            {stale.map((l) => (
              <li key={l.id}>{lineLabel(l)}</li>
            ))}
          </ul>
          <p className="mt-1">
            {draft
              ? 'Count them again and record them before submitting.'
              : 'The server will not approve this count as it is; it can be rejected and counted again.'}
          </p>
        </div>
      )}

      <div className="grid items-start gap-6 lg:grid-cols-3">
        <Card title="Count" className="lg:col-span-2">
          <DefinitionList
            items={[
              ['Location', `${c.locationCode} — ${c.locationName}`],
              ['Status', countStatusLabels[c.status]],
              ['Approval', 'Approver required — no threshold is configured'],
              ['Note', c.note ?? '—'],
              [
                'Submitted',
                c.submittedAt
                  ? `${who(c.submittedByName, c.submittedBy, perms.meId)}, ${formatDateTime(c.submittedAt)}`
                  : '—',
              ],
              [
                'Decision',
                c.decidedAt ? (
                  <span key="d">
                    {countStatusLabels[c.status]} by {who(c.decidedByName, c.decidedBy, perms.meId)}
                    , {formatDateTime(c.decidedAt)}
                    {c.decisionNote && (
                      <span className="block text-muted">
                        {c.status === 'REJECTED' ? 'Reason' : 'Note'}: {c.decisionNote}
                      </span>
                    )}
                  </span>
                ) : (
                  '—'
                ),
              ],
            ]}
          />
        </Card>
        <Card title="Items" description="Counted by staff; the stock record is read by the server.">
          <DefinitionList
            items={[
              ['Counted items', String(c.lineCount)],
              ...(reviewer
                ? ([['Differ from the record', String(c.varianceLines)]] as [string, string][])
                : []),
            ]}
          />
        </Card>
      </div>

      <div className="mt-6">
        <LinesCard
          count={c}
          reviewer={reviewer}
          canRecord={canRecord}
          onRecord={(line) => {
            setDone(null)
            setOpen({ kind: 'line', line })
          }}
        />
      </div>

      {open?.kind === 'line' && <LineDialog count={c} line={open.line} onClose={close} />}
      {open?.kind === 'submit' && <SubmitDialog count={c} onClose={close} />}
      {(open?.kind === 'approve' || open?.kind === 'reject') && (
        <DecisionDialog
          count={c}
          decision={open.kind}
          onClose={close}
          onDone={(r) => {
            setDone(r)
            close()
          }}
        />
      )}
    </>
  )
}

/** The decision exactly as the server returned it. */
function DecisionNote({ count }: { count: StockCount }) {
  const posted = count.lines.filter((l) => l.outcome === 'POSTED').length
  const findings = count.lines.filter((l) => l.outcome === 'DISCREPANCY_RECORDED').length
  return (
    <p role="status" className="mb-4 rounded-card bg-success-subtle px-4 py-3 text-sm text-ink">
      {count.status === 'APPROVED' ? (
        <>
          {count.countNumber} approved. {posted}{' '}
          {posted === 1 ? 'adjustment was' : 'adjustments were'} posted to the stock ledger
          {findings > 0 &&
            `; ${String(findings)} vehicle ${findings === 1 ? 'finding' : 'findings'} recorded (no vehicle changed)`}
          .
        </>
      ) : (
        <>{count.countNumber} rejected. Nothing was posted; stock is unchanged.</>
      )}
    </p>
  )
}

function LinesCard({
  count,
  reviewer,
  canRecord,
  onRecord,
}: {
  count: StockCount
  reviewer: boolean
  canRecord: boolean
  onRecord: (l: CountLine) => void
}) {
  const perms = useCountPerms()
  const decided = count.status === 'APPROVED' || count.status === 'REJECTED'
  const cols: Column<CountLine>[] = [
    {
      key: 'item',
      header: 'Item',
      render: (l) => (
        <span className="flex flex-col">
          <span className="font-medium">{lineLabel(l)}</span>
          {l.stockChanged && (
            <span className="pt-1">
              <Badge tone="warning">Stock moved since counted</Badge>
            </span>
          )}
        </span>
      ),
    },
    { key: 'counted', header: 'Counted', render: (l) => observedText(l) },
  ]
  if (reviewer) {
    cols.push(
      { key: 'expected', header: 'Stock record', render: (l) => expectedText(l) },
      {
        key: 'diff',
        header: 'Difference',
        render: (l) => <span className="font-medium">{varianceText(l)}</span>,
      },
    )
  }
  cols.push(
    { key: 'reason', header: 'Reason', render: (l) => l.reason ?? '—' },
    {
      key: 'by',
      header: 'Recorded',
      render: (l) => (
        <span className="text-xs">
          {who(l.recordedByName, l.recordedBy, perms.meId)}
          <span className="block text-muted">{formatDateTime(l.recordedAt)}</span>
        </span>
      ),
    },
  )
  if (decided) {
    cols.push({
      key: 'outcome',
      header: 'Outcome',
      render: (l) => (
        <Badge tone={outcomeTone[l.outcome]} className="rounded-control whitespace-normal">
          {outcomeLabels[l.outcome]}
        </Badge>
      ),
    })
  }
  if (canRecord) {
    cols.push({
      key: 'actions',
      header: 'Actions',
      render: (l) => (
        <Button
          size="sm"
          variant="secondary"
          aria-label={`Record ${lineLabel(l)} again`}
          onClick={() => {
            onRecord(l)
          }}
        >
          Record again
        </Button>
      ),
    })
  }
  return (
    <Card
      title="Counted items"
      description={
        count.status === 'DRAFT'
          ? `Items can be recorded again while the count is a draft; recording an item again replaces it.${reviewer ? '' : ' The stock record and differences are shown to approvers only.'}`
          : `Fixed since submission. Nothing is deleted.${reviewer ? '' : ' The stock record and differences are shown to approvers only.'}`
      }
    >
      {count.lines.length === 0 ? (
        <EmptyState title="Nothing counted yet">
          {canRecord
            ? 'Record each part or vehicle unit you count at this location.'
            : 'No item was recorded.'}
        </EmptyState>
      ) : (
        <ResponsiveTable
          caption="Counted items"
          columns={cols}
          rows={count.lines}
          rowKey={(l) => l.id}
        />
      )}
    </Card>
  )
}
