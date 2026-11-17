import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, CheckCircle2, Info, ShieldAlert } from 'lucide-react'
import { useId, useRef, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useParams } from 'react-router'
import { z } from 'zod'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { AccessDeniedState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import { formatDate, formatDateTime } from '@/features/catalogue/lib/presentation'
import { ApiError } from '@/lib/api/client'
import { cn } from '@/lib/utils/cn'
import type { ApprovalInput, CommitResult } from './api/commit'
import type { ImportApproval, ImportBatch } from './api/openingStock'
import {
  useApproveImport,
  useCommitImport,
  useImportBatch,
  useImportCommit,
  useImportPerms,
  useImportRows,
  useImportSpecification,
} from './hooks/useOpeningStock'
import {
  batchBlockers,
  commitGuidance,
  describeCommitError,
  lineOf,
  newCommitKey,
  type CommitFailure,
} from './lib/commitFlow'
import { rowStatusLabels, rowStatusTones } from './lib/labels'

/**
 * Opening stock import REVIEW → APPROVAL → COMMIT (G-07). Separate from the
 * G-06 preview pages. The server decides everything: eligibility, counts, the
 * approval and the commit; this page shows its answers and asks the user to
 * confirm each step explicitly. Nothing is submitted automatically.
 */

const basePath = '/inventory/opening-stock'
const linkClass = 'font-medium text-primary hover:underline'

export function OpeningStockCommitPage() {
  const { batchId = '' } = useParams()
  const { me, view } = useImportPerms()
  let body: ReactNode
  if (me.isPending) body = <LoadingState label="Checking your permissions…" />
  else if (me.isError && !me.data)
    body = (
      <ErrorState
        error={me.error}
        title="Could not check your permissions"
        onRetry={() => void me.refetch()}
      />
    )
  else if (!view) body = <AccessDeniedState what="opening stock imports" />
  else body = <Workflow id={batchId} />
  return (
    <>
      <PageHeader
        title="Review and commit"
        breadcrumbs={[
          { label: 'Home', to: '/' },
          { label: 'Opening Stock Import', to: basePath },
          { label: 'Import preview', to: `${basePath}/${batchId}` },
          { label: 'Review and commit' },
        ]}
        description="Approve an opening stock preview against the signed physical count, then commit it into stock — once."
      />
      {body}
    </>
  )
}

function Workflow({ id }: { id: string }) {
  const perms = useImportPerms()
  const batch = useImportBatch(id)
  const commit = useImportCommit(id)
  // The approved statuses are part of the specification (stock.import.preview).
  const spec = useImportSpecification({ enabled: perms.preview })
  if (batch.isPending || commit.isPending) return <LoadingState label="Loading the batch…" />
  if (batch.isError)
    return (
      <ErrorState
        error={batch.error}
        title="Could not load this batch"
        onRetry={() => void batch.refetch()}
      />
    )
  if (commit.isError)
    return (
      <ErrorState
        error={commit.error}
        title="Could not check whether this batch was committed"
        onRetry={() => void commit.refetch()}
      />
    )
  const b = batch.data
  const approvedStatuses = perms.preview ? (spec.data?.approvedOpeningStatuses ?? null) : null
  return (
    <div className="flex flex-col gap-4">
      <BatchCard batch={b} committed={commit.data !== null} />
      {commit.data ? (
        <CommitResultCard result={commit.data} />
      ) : (
        <>
          <ReviewCard batch={b} approvedStatuses={approvedStatuses} />
          <ApprovalCard batch={b} />
          <CommitCard batch={b} approvedStatuses={approvedStatuses} />
        </>
      )}
    </div>
  )
}

function Definition({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2">
      {items.map(([k, v]) => (
        <div key={k} className="flex flex-col gap-0.5">
          <dt className="text-xs text-muted">{k}</dt>
          <dd className="break-words">{v}</dd>
        </div>
      ))}
    </dl>
  )
}

const mono = (v: string) => <span className="font-mono break-all">{v}</span>

function BatchCard({ batch: b, committed }: { batch: ImportBatch; committed: boolean }) {
  return (
    <Card
      title={`${b.reference} · ${b.fileName}`}
      description="As recorded by the server."
      badge={
        <span className="flex flex-wrap gap-1">
          <Badge tone={b.status === 'REJECTED' ? 'danger' : b.expired ? 'neutral' : 'success'}>
            {b.status === 'REJECTED' ? 'File rejected' : b.expired ? 'Expired' : 'Previewed'}
          </Badge>
          <Badge tone={b.approval ? 'success' : 'neutral'}>
            {b.approval ? 'Approved' : 'Not approved'}
          </Badge>
          <Badge tone={committed || b.commit ? 'success' : 'neutral'}>
            {committed || b.commit ? 'Committed' : 'Not committed'}
          </Badge>
        </span>
      }
    >
      <Definition
        items={[
          ['Batch ID', mono(b.id)],
          ['Source file', b.fileName],
          ['Template', b.templateVersion ?? '—'],
          [
            'Uploaded',
            `${formatDateTime(b.uploadedAt)}${b.uploadedByName ? ` by ${b.uploadedByName}` : ''}`,
          ],
          ['File fingerprint (SHA-256)', mono(b.fileSha256)],
          ['Preview usable until', formatDateTime(b.expiresAt)],
        ]}
      />
      <p className="mt-3 text-sm">
        <Link to={`${basePath}/${b.id}`} className={linkClass}>
          Open the preview with every line and filter
        </Link>
      </p>
    </Card>
  )
}

function Tile({ label, value, tone }: { label: string; value: number; tone?: string }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-control border border-border/80 px-3 py-2">
      <dt className="text-xs text-muted">{label}</dt>
      <dd className={cn('text-lg font-semibold tabular-nums', value > 0 ? tone : undefined)}>
        {value}
      </dd>
    </div>
  )
}

function BlockerList({ blockers }: { blockers: ReturnType<typeof batchBlockers> }) {
  return (
    <ul className="list-disc pl-5 text-sm">
      {blockers.map((x) => (
        <li key={x.key}>{x.text}</li>
      ))}
    </ul>
  )
}

function ReviewCard({
  batch: b,
  approvedStatuses,
}: {
  batch: ImportBatch
  approvedStatuses: string[] | null
}) {
  const s = b.summary
  const blockers = batchBlockers(b, approvedStatuses)
  return (
    <Card title="1. Review" description="The server's validation of every line of the file.">
      <dl
        aria-label="Review summary"
        className="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7"
      >
        <Tile label="Lines in the file" value={s.totalRows} />
        <Tile label="Valid" value={s.validRows} tone="text-success" />
        <Tile label="Warnings" value={s.warningRows} tone="text-warning" />
        <Tile label="Errors" value={s.errorRows} tone="text-danger" />
        <Tile label="Blocked (unverified catalogue)" value={s.blockedRows} tone="text-info" />
        <Tile label="Duplicate identifiers" value={s.duplicateRows} tone="text-danger" />
        <Tile label="Unresolved mappings" value={s.unresolvedRows} tone="text-danger" />
      </dl>
      <p className="mt-3 text-sm text-muted">
        A commit creates one vehicle unit per line, and only when every line is valid. The server
        checks every line again at commit; the number it actually creates is the one it reports.
        {approvedStatuses !== null && approvedStatuses.length > 0 && (
          <> Opening statuses approved for commit: {approvedStatuses.join(', ')}.</>
        )}
      </p>
      {blockers.length > 0 ? (
        <div role="alert" className="mt-3 rounded-control border border-danger/30 bg-danger/5 p-3">
          <p className="mb-1 flex items-center gap-2 text-sm font-semibold text-danger">
            <AlertTriangle className="size-4" aria-hidden="true" />
            This batch cannot be committed as it stands.
          </p>
          <BlockerList blockers={blockers} />
          <ProblemLines id={b.id} />
        </div>
      ) : (
        <p className="mt-3 flex items-center gap-2 text-sm">
          <CheckCircle2 className="size-4 text-success" aria-hidden="true" />
          Every line passed the preview checks.
        </p>
      )}
    </Card>
  )
}

/** The first lines that need correction (display only; the preview page has every line). */
function ProblemLines({ id }: { id: string }) {
  const rows = useImportRows(id, { limit: 200 })
  if (rows.isPending) return <p className="mt-2 text-sm text-muted">Loading the lines…</p>
  if (rows.isError) return <p className="mt-2 text-sm text-danger">Could not load the lines.</p>
  const bad = rows.data.items.filter((r) => r.status !== 'VALID').slice(0, 10)
  if (bad.length === 0) return null
  return (
    <div className="mt-3">
      <p className="text-sm font-medium">Lines to correct</p>
      <ul className="mt-1 flex flex-col gap-1 text-sm">
        {bad.map((r) => (
          <li key={r.rowNumber} className="flex flex-wrap items-baseline gap-2">
            <span className="font-medium">Line {r.rowNumber}</span>
            <Badge tone={rowStatusTones[r.status]}>{rowStatusLabels[r.status]}</Badge>
            <span>
              {r.issues.map((i) => `${i.field ? `${i.field}: ` : ''}${i.message}`).join(' · ')}
            </span>
          </li>
        ))}
      </ul>
      <p className="mt-2 text-sm">
        <Link to={`${basePath}/${id}`} className={linkClass}>
          See every line in the preview
        </Link>
      </p>
    </div>
  )
}

// ── Approval ───────────────────────────────────────────────────────────────

function ApprovalDetails({ a }: { a: ImportApproval }) {
  return (
    <Definition
      items={[
        ['Approved by', `${a.approvedByName ?? 'Unknown user'} · ${formatDateTime(a.approvedAt)}`],
        ['Physical count reference', a.physicalCountReference],
        ['Physical count date', formatDate(a.physicalCountDate)],
        ['Mappings reviewed', a.mappingsReviewed ? 'Yes' : 'No'],
        ['Duplicates reviewed', a.duplicatesReviewed ? 'Yes' : 'No'],
        [
          'Received dates are the agreed opening-stock dates',
          a.receivedDatesApproved ? 'Yes' : 'No — stock age will show as not available',
        ],
        ['Note', a.note ?? '—'],
      ]}
    />
  )
}

const approvalSchema = (maxDate: string | null) =>
  z.object({
    physicalCountReference: z
      .string()
      .trim()
      .min(1, 'Enter the reference of the signed physical count.')
      .max(100, 'At most 100 characters.'),
    physicalCountDate: z.iso
      .date('Enter the date of the physical count.')
      .refine((d) => maxDate === null || d <= maxDate, 'The physical count cannot be after today.'),
    mappingsReviewed: z.literal(true, 'Confirm the mappings were reviewed.'),
    duplicatesReviewed: z.literal(true, 'Confirm the duplicate checks were reviewed.'),
    receivedDates: z.enum(['yes', 'no'], 'Choose whether the received dates are agreed.'),
    note: z.string().max(500, 'At most 500 characters.'),
  })
type ApprovalValues = z.infer<ReturnType<typeof approvalSchema>>

const fieldNames: Record<string, keyof ApprovalValues> = {
  physicalCountReference: 'physicalCountReference',
  physicalCountDate: 'physicalCountDate',
  mappingsReviewed: 'mappingsReviewed',
  duplicatesReviewed: 'duplicatesReviewed',
  receivedDatesApproved: 'receivedDates',
  note: 'note',
}

function ApprovalCard({ batch: b }: { batch: ImportBatch }) {
  const perms = useImportPerms()
  const spec = useImportSpecification({ enabled: perms.preview })
  if (b.approval)
    return (
      <Card title="2. Approval" badge={<Badge tone="success">Approved</Badge>}>
        <ApprovalDetails a={b.approval} />
      </Card>
    )
  // Statuses do not gate the approval itself; the server refuses those at commit.
  const blockers = batchBlockers(b, null)
  let body: ReactNode
  if (!perms.approve)
    body = (
      <p className="flex items-start gap-2 text-sm">
        <ShieldAlert className="mt-0.5 size-4 shrink-0 text-muted" aria-hidden="true" />
        Approval is recorded by someone with the stock.import.approve permission after the physical
        count. You do not have it.
      </p>
    )
  else if (blockers.length > 0)
    body = (
      <p className="text-sm">
        Only a preview whose every line is valid can be approved. Correct the problems listed above
        first.
      </p>
    )
  else body = <ApprovalForm batch={b} maxDate={spec.data?.businessDate ?? null} />
  return (
    <Card
      title="2. Approval"
      description="Record that this file was reconciled to a signed physical count. Completing the preview does not mean the count happened."
      badge={<Badge tone="neutral">Not approved</Badge>}
    >
      {body}
    </Card>
  )
}

function ApprovalForm({ batch: b, maxDate }: { batch: ImportBatch; maxDate: string | null }) {
  const formId = useId()
  const approve = useApproveImport(b.id)
  const [pending, setPending] = useState<ApprovalInput | null>(null)
  const [error, setError] = useState<{ message: string; denied: boolean } | null>(null)
  const form = useForm<ApprovalValues>({
    resolver: zodResolver(approvalSchema(maxDate)),
    // Nothing is preselected: every confirmation is the approver's own.
    defaultValues: {
      physicalCountReference: '',
      physicalCountDate: '',
      mappingsReviewed: false as unknown as true,
      duplicatesReviewed: false as unknown as true,
      receivedDates: undefined,
      note: '',
    },
  })
  const errs = form.formState.errors
  const review = form.handleSubmit((v) => {
    setError(null)
    setPending({
      physicalCountReference: v.physicalCountReference.trim(),
      physicalCountDate: v.physicalCountDate,
      mappingsReviewed: v.mappingsReviewed,
      duplicatesReviewed: v.duplicatesReviewed,
      receivedDatesApproved: v.receivedDates === 'yes',
      note: v.note.trim() === '' ? null : v.note.trim(),
    })
  })
  const record = () => {
    if (!pending || approve.isPending) return
    approve.mutate(pending, {
      onSuccess: () => {
        setPending(null)
      },
      onError: (e) => {
        setPending(null)
        if (e instanceof ApiError && e.status === 422 && e.details.length > 0) {
          for (const d of e.details) {
            const name = fieldNames[d.field]
            if (name) form.setError(name, { message: d.message })
          }
          setError({ message: 'Check the highlighted fields.', denied: false })
          return
        }
        if (e instanceof ApiError && e.status === 403) {
          setError({ message: 'You do not have permission to approve this batch.', denied: true })
          return
        }
        setError({
          message:
            e instanceof ApiError
              ? e.message
              : 'The approval could not be recorded. Check the batch and try again.',
          denied: false,
        })
      },
    })
  }
  const box = 'mt-0.5 size-4 shrink-0'
  return (
    <form id={formId} noValidate onSubmit={(e) => void review(e)} className="flex flex-col gap-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label="Physical count reference"
          required
          hint="The reference of the signed physical count sheet this file was reconciled to."
          error={errs.physicalCountReference?.message}
        >
          {(c) => <Input {...c} maxLength={100} {...form.register('physicalCountReference')} />}
        </Field>
        <Field
          label="Physical count date"
          required
          hint="The day the stock was physically counted."
          error={errs.physicalCountDate?.message}
        >
          {(c) => (
            <Input
              {...c}
              type="date"
              max={maxDate ?? undefined}
              {...form.register('physicalCountDate')}
            />
          )}
        </Field>
      </div>
      <fieldset className="flex flex-col gap-3 text-sm">
        <legend className="mb-1 font-medium">Confirmations (required)</legend>
        <label className="flex items-start gap-2">
          <input type="checkbox" className={box} {...form.register('mappingsReviewed')} />
          <span>
            I reviewed the brand, model, variant, colour and location mappings of every line against
            the physical stock.
            {errs.mappingsReviewed && (
              <span role="alert" className="block text-xs text-danger">
                {errs.mappingsReviewed.message}
              </span>
            )}
          </span>
        </label>
        <label className="flex items-start gap-2">
          <input type="checkbox" className={box} {...form.register('duplicatesReviewed')} />
          <span>
            I reviewed the duplicate chassis/frame and serial checks.
            {errs.duplicatesReviewed && (
              <span role="alert" className="block text-xs text-danger">
                {errs.duplicatesReviewed.message}
              </span>
            )}
          </span>
        </label>
      </fieldset>
      <fieldset className="flex flex-col gap-2 text-sm">
        <legend className="mb-1 font-medium">
          Are the file's received dates the agreed opening-stock dates? (required)
        </legend>
        <label className="flex items-start gap-2">
          <input type="radio" value="yes" className={box} {...form.register('receivedDates')} />
          <span>Yes — stock age will be counted from these dates.</span>
        </label>
        <label className="flex items-start gap-2">
          <input type="radio" value="no" className={box} {...form.register('receivedDates')} />
          <span>No — the units will show no stock age until dates are agreed.</span>
        </label>
        {errs.receivedDates && (
          <span role="alert" className="text-xs text-danger">
            {errs.receivedDates.message}
          </span>
        )}
      </fieldset>
      <Field label="Note" hint="Optional, at most 500 characters." error={errs.note?.message}>
        {(c) => <Input {...c} maxLength={500} {...form.register('note')} />}
      </Field>
      {error &&
        (error.denied ? (
          <AccessDeniedState what="approving this batch" />
        ) : (
          <p role="alert" className="text-sm text-danger">
            {error.message}
          </p>
        ))}
      <div>
        <Button type="submit">Review the approval…</Button>
      </div>
      <Dialog
        open={pending !== null}
        onClose={() => {
          if (!approve.isPending) setPending(null)
        }}
        title="Record this approval?"
        description="It is recorded once for this batch and cannot be changed afterwards."
        footer={
          <>
            <Button
              variant="secondary"
              disabled={approve.isPending}
              onClick={() => {
                setPending(null)
              }}
            >
              Cancel
            </Button>
            <Button loading={approve.isPending} disabled={approve.isPending} onClick={record}>
              Record approval
            </Button>
          </>
        }
      >
        {pending && (
          <Definition
            items={[
              ['Batch', `${b.reference} (${b.id})`],
              ['Physical count reference', pending.physicalCountReference],
              ['Physical count date', formatDate(pending.physicalCountDate)],
              ['Mappings reviewed', 'Yes'],
              ['Duplicates reviewed', 'Yes'],
              ['Received dates agreed', pending.receivedDatesApproved ? 'Yes' : 'No'],
              ['Note', pending.note ?? '—'],
            ]}
          />
        )}
      </Dialog>
    </form>
  )
}

// ── Commit ─────────────────────────────────────────────────────────────────

type Phase =
  | { kind: 'idle' }
  | { kind: 'checking' }
  | { kind: 'not-committed'; message: string }
  | { kind: 'failed'; failure: CommitFailure }

function CommitCard({
  batch: b,
  approvedStatuses,
}: {
  batch: ImportBatch
  approvedStatuses: string[] | null
}) {
  const perms = useImportPerms()
  const commit = useCommitImport(b.id)
  const status = useImportCommit(b.id)
  const [open, setOpen] = useState(false)
  const [understood, setUnderstood] = useState(false)
  const [phase, setPhase] = useState<Phase>({ kind: 'idle' })
  // One key per commit attempt, kept for every retry of it (never renewed on a timeout).
  const keyRef = useRef<string | null>(null)
  const inFlight = useRef(false)
  const blockers = batchBlockers(b, approvedStatuses)
  const canAsk = perms.commit && b.approval !== null && blockers.length === 0 && b.commit === null

  const send = async () => {
    if (inFlight.current) return
    inFlight.current = true
    keyRef.current ??= newCommitKey()
    const key = keyRef.current
    setPhase({ kind: 'idle' })
    try {
      // Never commit to find out whether an earlier request did: ask first.
      const existing = await status.refetch()
      if (existing.data) {
        setOpen(false)
        return
      }
      // Without the server's answer nothing is sent: a failed check is reported instead.
      if (existing.isError) {
        setOpen(false)
        setPhase({ kind: 'failed', failure: describeCommitError(existing.error) })
        return
      }
      await commit.mutateAsync(key)
      setOpen(false)
    } catch (e) {
      const f = describeCommitError(e)
      setOpen(false)
      if (f.kind === 'unknown' || f.kind === 'already-committed') {
        setPhase({ kind: 'checking' })
        const after = await status.refetch()
        if (after.data) return // the page now shows the server's commit
        if (after.isError) {
          setPhase({ kind: 'failed', failure: f })
          return
        }
        setPhase({
          kind: 'not-committed',
          message:
            'The server has no commit for this batch: nothing was created. You can try again; the retry reuses the same request key, so it cannot create stock twice.',
        })
        return
      }
      if (f.kind === 'key-reused') keyRef.current = null
      setPhase({ kind: 'failed', failure: f })
    } finally {
      inFlight.current = false
    }
  }

  let body: ReactNode
  if (!perms.commit)
    body = (
      <p className="flex items-start gap-2 text-sm">
        <ShieldAlert className="mt-0.5 size-4 shrink-0 text-muted" aria-hidden="true" />
        Committing needs the stock.import.commit permission. You do not have it.
      </p>
    )
  else if (b.approval === null)
    body = <p className="text-sm">The batch must be approved before it can be committed.</p>
  else if (blockers.length > 0)
    body = (
      <div className="text-sm">
        <p className="mb-1">The commit is not available:</p>
        <BlockerList blockers={blockers} />
      </div>
    )
  else
    body = (
      <p className="text-sm">
        Committing creates one vehicle unit for each of the {b.summary.totalRows} line(s), with
        their history and opening-balance stock movements, after the server checks every line again.
        It happens once and cannot be undone.
      </p>
    )

  return (
    <Card title="3. Commit" description="Create the stock — once, all lines or none.">
      {body}
      <PhaseView
        phase={phase}
        retry={
          canAsk && phase.kind === 'not-committed'
            ? () => {
                void send()
              }
            : undefined
        }
        busy={commit.isPending}
      />
      <div className="mt-3">
        <Button
          disabled={!canAsk || commit.isPending || phase.kind === 'checking'}
          onClick={() => {
            setUnderstood(false)
            setOpen(true)
          }}
        >
          Commit this batch…
        </Button>
      </div>
      <Dialog
        open={open}
        onClose={() => {
          if (!commit.isPending) setOpen(false)
        }}
        title="Commit this batch into stock?"
        description="This creates inventory records and stock movements. It cannot be undone."
        footer={
          <>
            <Button
              variant="secondary"
              disabled={commit.isPending}
              onClick={() => {
                setOpen(false)
              }}
            >
              Cancel
            </Button>
            <Button
              loading={commit.isPending}
              disabled={!understood || commit.isPending}
              onClick={() => {
                void send()
              }}
            >
              Commit {b.summary.totalRows} line(s)
            </Button>
          </>
        }
      >
        <Definition
          items={[
            ['Batch', `${b.reference} (${b.id})`],
            ['Source file', b.fileName],
            ['Lines to create (server re-checks at commit)', String(b.summary.totalRows)],
            [
              'Approval',
              b.approval ? `Approved by ${b.approval.approvedByName ?? 'Unknown user'}` : '—',
            ],
            [
              'Physical count',
              b.approval
                ? `${b.approval.physicalCountReference} · ${formatDate(b.approval.physicalCountDate)}`
                : '—',
            ],
            [
              'Mappings / duplicates reviewed',
              b.approval
                ? `${b.approval.mappingsReviewed ? 'Yes' : 'No'} / ${b.approval.duplicatesReviewed ? 'Yes' : 'No'}`
                : '—',
            ],
            [
              'Received dates agreed',
              b.approval?.receivedDatesApproved ? 'Yes' : 'No — no stock age until agreed',
            ],
          ]}
        />
        <label className="mt-4 flex items-start gap-2 text-sm">
          <input
            type="checkbox"
            className="mt-0.5 size-4 shrink-0"
            checked={understood}
            onChange={(e) => {
              setUnderstood(e.target.checked)
            }}
          />
          <span>
            I understand that committing creates these vehicle units and their opening-balance stock
            movements.
          </span>
        </label>
      </Dialog>
    </Card>
  )
}

function PhaseView({ phase, retry, busy }: { phase: Phase; retry?: () => void; busy: boolean }) {
  if (phase.kind === 'idle') return null
  if (phase.kind === 'checking')
    return (
      <p role="status" className="mt-3 text-sm">
        Checking with the server whether the commit went through…
      </p>
    )
  if (phase.kind === 'not-committed')
    return (
      <div role="status" className="mt-3 rounded-control bg-surface-muted p-3 text-sm">
        <p>{phase.message}</p>
        {retry && (
          <Button
            className="mt-2"
            variant="secondary"
            loading={busy}
            disabled={busy}
            onClick={retry}
          >
            Try the commit again
          </Button>
        )}
      </div>
    )
  const f = phase.failure
  if (f.kind === 'forbidden')
    return (
      <div className="mt-3">
        <AccessDeniedState what="committing this batch" />
      </div>
    )
  return (
    <div
      role="alert"
      className="mt-3 rounded-control border border-danger/30 bg-danger/5 p-3 text-sm"
    >
      <p className="font-semibold text-danger">Nothing was committed.</p>
      <p>{f.message}</p>
      {f.kind === 'unknown' && (
        <p className="mt-1">
          Its outcome could not be checked. Reload this page: it asks the server whether the batch
          was committed before offering anything else.
        </p>
      )}
      {f.kind === 'refused' && f.code && commitGuidance[f.code] && (
        <p className="mt-1">{commitGuidance[f.code]}</p>
      )}
      {f.kind === 'rows-blocked' && (
        <ul className="mt-2 list-disc pl-5">
          {f.lines.map((l, i) => {
            const { line, column } = lineOf(l.field)
            return (
              <li key={`${l.field}-${String(i)}`}>
                {line !== null && <span className="font-medium">Line {line}</span>}
                {column && <span className="font-mono"> {column}</span>}: {l.message}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

function CommitResultCard({ result: c }: { result: CommitResult }) {
  return (
    <Card
      title={`Committed · ${c.reference}`}
      description="As confirmed by the server."
      badge={<Badge tone="success">{c.batchStatus}</Badge>}
    >
      <p role="status" className="mb-3 flex items-start gap-2 text-sm">
        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" aria-hidden="true" />
        <span>
          {c.unitsCreated} vehicle unit(s) were created from {c.batchReference}, each with an
          opening-balance stock movement.
          {c.replayed &&
            ' This answer repeats the earlier successful commit; nothing new was created.'}
        </span>
      </p>
      <Definition
        items={[
          ['Commit reference', mono(c.reference)],
          ['Batch', `${c.batchReference} (${c.batchId})`],
          ['Status', c.batchStatus],
          ['Units created', String(c.unitsCreated)],
          [
            'Committed by',
            `${c.committedByName ?? 'Unknown user'} · ${formatDateTime(c.committedAt)}`,
          ],
          ['File fingerprint (SHA-256)', mono(c.fileSha256)],
          ['Template', c.templateVersion],
          ['Provenance', `Source ${c.sourceType}, movement ${c.movementType}`],
          ['Approval', mono(c.approvalId)],
        ]}
      />
      <div className="mt-4 overflow-x-auto">
        <table className="w-full text-left text-sm">
          <caption className="mb-1 text-left font-medium">Units created</caption>
          <thead className="text-xs text-muted">
            <tr>
              <th scope="col" className="py-1 pr-3">
                Line
              </th>
              <th scope="col" className="py-1 pr-3">
                Vehicle unit
              </th>
              <th scope="col" className="py-1">
                Status
              </th>
            </tr>
          </thead>
          <tbody>
            {c.units.map((u) => (
              <tr key={u.vehicleUnitId} className="border-t border-border">
                <td className="py-1 pr-3 tabular-nums">{u.rowNumber}</td>
                <td className="py-1 pr-3">
                  <Link
                    to={`/inventory/vehicles/${u.vehicleUnitId}`}
                    className={`${linkClass} font-mono`}
                  >
                    {u.unitNumber}
                  </Link>
                </td>
                <td className="py-1">{u.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-3 flex items-start gap-2 text-xs text-muted">
        <Info className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
        The batch cannot be committed again.
      </p>
    </Card>
  )
}
