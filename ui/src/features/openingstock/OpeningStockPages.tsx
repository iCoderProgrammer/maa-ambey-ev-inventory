import {
  AlertTriangle,
  CheckCircle2,
  Download,
  FileSpreadsheet,
  Info,
  UploadCloud,
} from 'lucide-react'
import { useId, useRef, useState, type DragEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Pagination } from '@/components/ui/Pagination'
import {
  AccessDeniedState,
  EmptyState,
  ErrorState,
  LoadingState,
  Skeleton,
} from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { formatDateTime } from '@/features/catalogue/lib/presentation'
import { FilterSelect, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { useListParams } from '@/features/catalogue/lib/listParams'
import { useUrlFilters } from '@/features/stock/lib/urlFilters'
import { cn } from '@/lib/utils/cn'
import {
  downloadTemplate,
  rowStatusSchema,
  type ImportBatch,
  type ImportRow,
  type RowStatus,
} from './api/openingStock'
import {
  useImportBatch,
  useImportList,
  useImportPerms,
  useImportRows,
  useImportSpecification,
  useUploadImport,
} from './hooks/useOpeningStock'
import {
  describeUploadError,
  formatBytes,
  PREVIEW_ONLY_TEXT,
  rowStatusLabels,
  rowStatusTones,
  severityTones,
} from './lib/labels'

/**
 * Opening stock import PREVIEW (G-06). Template → upload → validation and
 * mapping → preview → summary. Every check runs on the server; nothing on these
 * pages creates, changes or commits stock, and there is no commit button
 * (committing is the separate, authorised G-07 workflow).
 */

const basePath = '/inventory/opening-stock'
const crumbs = [
  { label: 'Home', to: '/' },
  { label: 'Opening Stock Import', to: basePath },
]
const linkClass = 'font-medium text-primary hover:underline'
const steps = ['Template', 'Upload', 'Validation', 'Preview', 'Summary'] as const

function Steps({ current }: { current: number }) {
  return (
    <ol aria-label="Import steps" className="mb-4 flex flex-wrap gap-2 text-xs">
      {steps.map((s, i) => (
        <li
          key={s}
          aria-current={i === current ? 'step' : undefined}
          className={cn(
            'rounded-full border px-3 py-1',
            i === current
              ? 'border-primary bg-primary/10 font-semibold text-primary'
              : i < current
                ? 'border-border bg-surface-muted text-ink'
                : 'border-border text-muted',
          )}
        >
          {i + 1}. {s}
        </li>
      ))}
    </ol>
  )
}

function PreviewOnlyNote() {
  return (
    <p
      role="note"
      className="mb-4 flex items-start gap-2 rounded-card border border-info/30 bg-info-subtle px-4 py-3 text-sm text-ink"
    >
      <Info className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
      <span>
        This is a preview only. Uploading a file checks it against the catalogue, locations and
        existing vehicles; it never creates or changes stock. Committing an approved preview is a
        separate, authorised step.
      </span>
    </p>
  )
}

function Gate({
  children,
  title,
  needPreview = false,
}: {
  children: ReactNode
  title: string
  needPreview?: boolean
}) {
  const perms = useImportPerms()
  const me = perms.me
  const view = needPreview ? perms.preview : perms.view
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
  else if (!view) body = <AccessDeniedState what="opening stock imports" />
  return (
    <>
      <PageHeader
        title={title}
        breadcrumbs={
          title === 'Opening Stock Import'
            ? [{ label: 'Home', to: '/' }, { label: title }]
            : [...crumbs, { label: title }]
        }
        description="Check an opening stock file against the catalogue before anything is imported."
      />
      {body}
    </>
  )
}

// ── Start: template + upload + recent previews ─────────────────────────────

export function OpeningStockImportPage() {
  return (
    <Gate title="Opening Stock Import" needPreview>
      <PreviewOnlyNote />
      <Start />
    </Gate>
  )
}

function Start() {
  const spec = useImportSpecification()
  const [step, setStep] = useState(0)
  return (
    <>
      <Steps current={step} />
      <div className="grid gap-4 lg:grid-cols-2">
        <TemplateCard spec={spec} />
        <UploadCard
          maxBytes={spec.data?.maxBytes}
          onFile={() => {
            setStep(1)
          }}
        />
      </div>
      <RecentPreviews />
    </>
  )
}

function TemplateCard({ spec }: { spec: ReturnType<typeof useImportSpecification> }) {
  const [downloadError, setDownloadError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const download = async () => {
    setBusy(true)
    setDownloadError(null)
    try {
      const blob = await downloadTemplate()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `opening-stock-template-${spec.data?.templateVersion ?? 'OS-IMPORT-1'}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setDownloadError(describeUploadError(e).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Card
      title="1. Download the template"
      description="Fill one vehicle per line from line 3. Keep the first two lines."
    >
      {spec.isPending ? (
        <Skeleton className="h-32 w-full" />
      ) : spec.isError ? (
        <ErrorState
          error={spec.error}
          title="Could not load the template rules"
          onRetry={() => void spec.refetch()}
        />
      ) : (
        <div className="flex flex-col gap-3 text-sm">
          <p>
            Template <span className="font-mono">{spec.data.templateVersion}</span> ·{' '}
            {spec.data.format} · up to {formatBytes(spec.data.maxBytes)} and {spec.data.maxRows}{' '}
            rows.
          </p>
          <div>
            <Button onClick={() => void download()} loading={busy}>
              {!busy && <Download className="size-4" aria-hidden="true" />}
              Download template (CSV)
            </Button>
          </div>
          {downloadError && (
            <p role="alert" className="text-danger">
              {downloadError}
            </p>
          )}
          <details className="rounded-control border border-border/80 p-3">
            <summary className="cursor-pointer font-medium">
              Columns ({spec.data.columns.length})
            </summary>
            <table className="mt-2 w-full text-left text-xs">
              <caption className="sr-only">Template columns</caption>
              <thead>
                <tr>
                  <th scope="col" className="py-1 pr-2">
                    Column
                  </th>
                  <th scope="col" className="py-1 pr-2">
                    Required
                  </th>
                  <th scope="col" className="py-1">
                    Meaning
                  </th>
                </tr>
              </thead>
              <tbody>
                {spec.data.columns.map((c) => (
                  <tr key={c.name} className="border-t border-border/60 align-top">
                    <td className="py-1 pr-2 font-mono">{c.name}</td>
                    <td className="py-1 pr-2">{c.required ? 'Required' : 'Optional'}</td>
                    <td className="py-1">
                      {c.description}{' '}
                      <span className="text-muted">
                        ({c.format}, max {c.maxLength})
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="mt-2 text-xs">
              Stock statuses:{' '}
              <span className="font-mono">{spec.data.stockStatuses.join(', ')}</span>
            </p>
          </details>
          <ul className="list-disc pl-5 text-xs text-muted">
            {spec.data.notes.map((n) => (
              <li key={n}>{n}</li>
            ))}
          </ul>
        </div>
      )}
    </Card>
  )
}

function newKey(): string {
  return `osi-${crypto.randomUUID()}`
}

function UploadCard({ maxBytes, onFile }: { maxBytes?: number; onFile: () => void }) {
  const navigate = useNavigate()
  const upload = useUploadImport()
  const inputRef = useRef<HTMLInputElement>(null)
  const inputId = useId()
  const hintId = useId()
  const [file, setFile] = useState<File | null>(null)
  const [key, setKey] = useState('')
  const [progress, setProgress] = useState(0)
  const [problem, setProblem] = useState<string | null>(null)
  const [dragging, setDragging] = useState(false)

  const choose = (f: File | undefined) => {
    if (!f) return
    upload.reset()
    setProgress(0)
    setProblem(null)
    if (!f.name.toLowerCase().endsWith('.csv'))
      setProblem(
        'Choose the CSV template (.csv). In Excel use Save As → “CSV UTF-8 (comma delimited)”.',
      )
    else if (maxBytes !== undefined && f.size > maxBytes)
      setProblem(
        `The file is ${formatBytes(f.size)}; the limit is ${formatBytes(maxBytes)}. Split it into smaller files.`,
      )
    else if (f.size === 0) setProblem('The file is empty.')
    setFile(f)
    setKey(newKey()) // a new file is a new request; a retry of the same file keeps its key
    onFile()
    if (inputRef.current) inputRef.current.value = ''
  }
  const send = () => {
    if (!file) return
    upload.mutate(
      { file, key, onProgress: setProgress },
      {
        onSuccess: (b) => {
          void navigate(`${basePath}/${b.id}`)
        },
      },
    )
  }
  const failure = upload.isError ? describeUploadError(upload.error) : null
  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    choose(e.dataTransfer.files[0])
  }
  return (
    <Card
      title="2. Upload the filled file"
      description="The server validates every row; nothing is imported."
    >
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => {
          setDragging(false)
        }}
        onDrop={onDrop}
        className={cn(
          'flex flex-col items-center gap-3 rounded-card border-2 border-dashed px-4 py-6 text-center',
          dragging ? 'border-primary bg-primary/5' : 'border-border bg-surface-muted/40',
        )}
      >
        <UploadCloud className="size-8 text-muted" aria-hidden="true" />
        <label htmlFor={inputId} className="text-sm font-medium">
          Choose the CSV file
        </label>
        <p id={hintId} className="text-xs text-muted">
          .csv (UTF-8){maxBytes ? ` · up to ${formatBytes(maxBytes)}` : ''} · or drop it here
        </p>
        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept=".csv,text/csv"
          tabIndex={-1}
          className="sr-only"
          onChange={(e) => {
            choose(e.target.files?.[0])
          }}
        />
        <Button
          variant="secondary"
          size="sm"
          aria-describedby={hintId}
          onClick={() => inputRef.current?.click()}
        >
          Browse files
        </Button>
      </div>
      {file && (
        <div className="mt-3 flex flex-col gap-2 text-sm">
          <p className="flex items-center gap-2">
            <FileSpreadsheet className="size-4 text-muted" aria-hidden="true" />
            <span className="font-medium">{file.name}</span>
            <span className="text-muted">{formatBytes(file.size)}</span>
          </p>
          {problem ? (
            <p role="alert" className="text-danger">
              {problem}
            </p>
          ) : (
            <>
              {upload.isPending && (
                <div role="status" aria-label="Uploading" className="flex items-center gap-2">
                  <progress
                    max={1}
                    value={progress}
                    className="h-2 w-full"
                    aria-label="Upload progress"
                  />
                  <span className="tabular-nums text-xs">{Math.round(progress * 100)}%</span>
                </div>
              )}
              {failure && (
                <div
                  role="alert"
                  className="rounded-card border border-danger/30 bg-danger-subtle p-3"
                >
                  <p className="font-semibold text-danger">
                    The upload did not complete. Nothing was stored.
                  </p>
                  <p>{failure.message}</p>
                  {failure.requestId && (
                    <p className="font-mono text-xs text-muted">Reference: {failure.requestId}</p>
                  )}
                </div>
              )}
              <div>
                <Button
                  onClick={send}
                  loading={upload.isPending}
                  disabled={upload.isPending || (failure !== null && !failure.retrySame)}
                >
                  {failure?.retrySame ? 'Try again' : 'Upload and validate'}
                </Button>
              </div>
            </>
          )}
        </div>
      )}
    </Card>
  )
}

function StatusBadge({ batch }: { batch: ImportBatch }) {
  if (batch.status === 'REJECTED') return <Badge tone="danger">File rejected</Badge>
  if (batch.commit) return <Badge tone="success">Committed</Badge>
  if (batch.approval) return <Badge tone="info">Approved</Badge>
  if (batch.expired) return <Badge tone="neutral">Expired</Badge>
  return <Badge tone="success">Previewed</Badge>
}

function RecentPreviews() {
  const { page, set } = useListParams([] as const)
  const list = useImportList(page)
  if (list.isPending) return <Skeleton className="mt-4 h-24 w-full" />
  if (list.isError && !list.data)
    return (
      <div className="mt-4">
        <ErrorState
          error={list.error}
          title="Could not load your previews"
          onRetry={() => void list.refetch()}
        />
      </div>
    )
  const cols: Column<ImportBatch>[] = [
    {
      key: 'ref',
      header: 'Preview',
      render: (b) => (
        <span className="flex flex-col">
          <Link to={`${basePath}/${b.id}`} className={`${linkClass} font-mono`}>
            {b.reference}
          </Link>
          <span className="text-xs text-muted">{b.fileName}</span>
        </span>
      ),
    },
    { key: 'when', header: 'Uploaded', render: (b) => formatDateTime(b.uploadedAt) },
    { key: 'status', header: 'Status', render: (b) => <StatusBadge batch={b} /> },
    {
      key: 'rows',
      header: 'Rows',
      render: (b) =>
        b.status === 'REJECTED'
          ? '—'
          : `${String(b.summary.totalRows)} (${String(b.summary.canProceedRows)} can proceed)`,
    },
  ]
  return (
    <section aria-labelledby="recent-previews" className="mt-6">
      <h2 id="recent-previews" className="mb-2 font-semibold">
        Your previews
      </h2>
      {list.data.total === 0 ? (
        <EmptyState title="No previews yet">
          Upload a filled template to see its validation preview.
        </EmptyState>
      ) : (
        <>
          <ResponsiveTable
            caption="Your previews"
            columns={cols}
            rows={list.data.items}
            rowKey={(b) => b.id}
          />
          <Pagination
            page={list.data.page}
            limit={list.data.limit}
            total={list.data.total}
            noun={['preview', 'previews']}
            onPage={(p) => {
              set({ page: String(p) })
            }}
          />
        </>
      )}
    </section>
  )
}

// ── Preview of one batch ────────────────────────────────────────────────────

export function OpeningStockPreviewPage() {
  const { batchId = '' } = useParams()
  return (
    <Gate title="Import preview">
      <PreviewOnlyNote />
      <Preview id={batchId} />
    </Gate>
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

const filterKeys = ['status', 'duplicate', 'unresolved'] as const

function Preview({ id }: { id: string }) {
  const batch = useImportBatch(id)
  if (batch.isPending) return <LoadingState label="Loading the preview…" />
  if (batch.isError) {
    return (
      <ErrorState
        error={batch.error}
        title="Could not load this preview"
        onRetry={() => void batch.refetch()}
      />
    )
  }
  const b = batch.data
  const s = b.summary
  return (
    <>
      <Steps current={b.status === 'REJECTED' ? 2 : 4} />
      <Card
        title={`${b.reference} · ${b.fileName}`}
        description={`Uploaded ${formatDateTime(b.uploadedAt)}${b.uploadedByName ? ` by ${b.uploadedByName}` : ''} · template ${b.templateVersion ?? '—'} · checked against ${b.businessDate}`}
        badge={<StatusBadge batch={b} />}
        className="mb-4"
      >
        {b.status === 'REJECTED' ? (
          <div role="alert" className="flex flex-col gap-2 text-sm">
            <p className="flex items-center gap-2 font-semibold text-danger">
              <AlertTriangle className="size-4" aria-hidden="true" />
              The file could not be read as the template. No row was checked.
            </p>
            <ul className="list-disc pl-5">
              {b.fileIssues.map((i) => (
                <li key={`${i.code}-${String(i.line)}-${i.column}`}>
                  {i.line > 0 && <span className="font-medium">Line {i.line}: </span>}
                  {i.message}
                </li>
              ))}
            </ul>
            <p>
              Correct the file and{' '}
              <Link to={basePath} className={linkClass}>
                upload it again
              </Link>
              .
            </p>
          </div>
        ) : (
          <>
            <dl
              aria-label="Preview summary"
              className="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7"
            >
              <Tile label="Total rows" value={s.totalRows} />
              <Tile label="Valid" value={s.validRows} tone="text-success" />
              <Tile label="Warnings" value={s.warningRows} tone="text-warning" />
              <Tile label="Errors" value={s.errorRows} tone="text-danger" />
              <Tile label="Blocked (unverified catalogue)" value={s.blockedRows} tone="text-info" />
              <Tile label="Duplicate identifiers" value={s.duplicateRows} tone="text-danger" />
              <Tile label="Unresolved mappings" value={s.unresolvedRows} tone="text-danger" />
            </dl>
            <p
              role="status"
              className="mt-3 flex items-start gap-2 rounded-control bg-surface-muted px-3 py-2 text-sm"
            >
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" aria-hidden="true" />
              <span>
                <strong>{PREVIEW_ONLY_TEXT}</strong> {s.canProceedRows} of {s.totalRows} rows could
                proceed to a commit; {s.errorRows + s.blockedRows} must be corrected or approved
                first.{' '}
                {b.expired
                  ? 'This preview has expired and is kept as history only.'
                  : `Committing is a separate, authorised step (G-07), available until ${formatDateTime(b.expiresAt)}.`}
              </span>
            </p>
          </>
        )}
      </Card>
      {b.status === 'PREVIEWED' && <NextStep batch={b} />}
      {b.status === 'PREVIEWED' && <RowsTable id={b.id} />}
      <p className="mt-4">
        <Link to={basePath} className={linkClass}>
          Upload a corrected file
        </Link>
      </p>
    </>
  )
}

/**
 * G-07 lives on its own page: the preview itself never commits. Shown to users
 * who may approve or commit, or once the batch has been approved/committed.
 */
function NextStep({ batch: b }: { batch: ImportBatch }) {
  const perms = useImportPerms()
  if (!perms.approve && !perms.commit && !b.approval && !b.commit) return null
  return (
    <p className="mb-4 rounded-control border border-border px-3 py-2 text-sm">
      {b.commit ? (
        <>
          Committed as <span className="font-mono">{b.commit.reference}</span> ·{' '}
          {b.commit.unitsCreated} vehicle unit(s) created.{' '}
        </>
      ) : b.approval ? (
        <>Approved against physical count {b.approval.physicalCountReference}. </>
      ) : (
        <>Not approved yet. </>
      )}
      <Link to={`${basePath}/${b.id}/commit`} className={linkClass}>
        {b.commit ? 'View the commit' : 'Review, approve and commit'}
      </Link>
    </p>
  )
}

function IssueList({ row }: { row: ImportRow }) {
  if (row.issues.length === 0) return <span className="text-muted">No issues</span>
  return (
    <ul className="flex flex-col gap-1">
      {row.issues.map((i, n) => (
        <li key={`${i.code}-${String(n)}`} className="text-xs">
          <Badge tone={severityTones[i.severity]} className="mr-1">
            {i.severity}
          </Badge>
          {i.field && <span className="font-mono">{i.field}: </span>}
          {i.message}
        </li>
      ))}
    </ul>
  )
}

const resolvedName = (row: ImportRow, key: string, fallback: string): string => {
  const v = row.resolved[key]
  if (v && typeof v === 'object' && 'name' in v && typeof v.name === 'string') return v.name
  return fallback
}

function RowsTable({ id }: { id: string }) {
  // useUrlFilters applies each change on top of the latest one (two quick filter changes both survive).
  const { values, page, set } = useUrlFilters(filterKeys)
  const [open, setOpen] = useState<ImportRow | null>(null)
  const rows = useImportRows(id, {
    status: (values.status || undefined) as RowStatus | undefined,
    duplicate: values.duplicate === 'true' ? true : undefined,
    unresolved: values.unresolved === 'true' ? true : undefined,
    page,
    limit: 50,
  })
  const filtered = filterKeys.some((k) => values[k] !== '')
  const clear = () => {
    set({ status: '', duplicate: '', unresolved: '' })
  }
  const show = (row: ImportRow) => (
    <Button
      variant="secondary"
      size="sm"
      onClick={() => {
        setOpen(row)
      }}
      aria-label={`Details of line ${String(row.rowNumber)}`}
    >
      Details
    </Button>
  )
  const cell = (row: ImportRow, col: string, warnCodes: string[]) => {
    const flagged = row.issues.some((i) => i.field === col && warnCodes.includes(i.severity))
    const v = row.source[col]?.trim() ?? ''
    return (
      <span
        className={cn('font-mono text-xs', flagged && 'rounded bg-danger-subtle px-1 text-danger')}
      >
        {v || '—'}
      </span>
    )
  }
  const cols: Column<ImportRow>[] = [
    {
      key: 'line',
      header: 'Line',
      render: (r) => <span className="font-mono">{r.rowNumber}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <Badge tone={rowStatusTones[r.status]}>{rowStatusLabels[r.status]}</Badge>,
    },
    {
      key: 'vehicle',
      header: 'Brand / model / variant',
      render: (r) => (
        <span className="flex flex-col text-xs">
          <span>{resolvedName(r, 'brand', r.source.brand ?? '')}</span>
          <span className="text-muted">
            {resolvedName(r, 'model', r.source.model_code ?? '')} ·{' '}
            {resolvedName(r, 'variant', r.source.variant_code ?? '')}
          </span>
        </span>
      ),
    },
    {
      key: 'chassis',
      header: 'Chassis / frame',
      render: (r) => cell(r, 'chassis_number', ['ERROR', 'WARNING']),
    },
    {
      key: 'serials',
      header: 'Serials',
      render: (r) => (
        <span className="flex flex-col gap-0.5">
          {['motor_serial', 'battery_serial', 'charger_serial']
            .filter((c) => r.source[c]?.trim())
            .map((c) => (
              <span key={c}>{cell(r, c, ['WARNING', 'ERROR'])}</span>
            ))}
        </span>
      ),
    },
    { key: 'location', header: 'Location', render: (r) => cell(r, 'location_code', ['ERROR']) },
    {
      key: 'stock',
      header: 'Status / condition',
      render: (r) => cell(r, 'stock_status', ['ERROR']),
    },
    { key: 'issues', header: 'Issues', render: (r) => <IssueList row={r} /> },
    { key: 'act', header: 'Details', render: show },
  ]
  let body: ReactNode
  if (rows.isPending) body = <Skeleton className="h-40 w-full" />
  else if (rows.isError && !rows.data)
    body = (
      <ErrorState
        error={rows.error}
        title="Could not load the rows"
        onRetry={() => void rows.refetch()}
      />
    )
  else if (rows.data.total === 0)
    body = (
      <EmptyState title={filtered ? 'No rows match these filters' : 'No rows'}>
        {filtered ? 'Change or clear the filters.' : 'This preview has no rows.'}
      </EmptyState>
    )
  else
    body = (
      <div className={rows.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Preview rows"
          columns={cols}
          rows={rows.data.items}
          rowKey={(r) => String(r.rowNumber)}
        />
        <Pagination
          page={rows.data.page}
          limit={rows.data.limit}
          total={rows.data.total}
          noun={['row', 'rows']}
          onPage={(p) => {
            set({ page: String(p) })
          }}
        />
      </div>
    )
  return (
    <section aria-labelledby="preview-rows">
      <h2 id="preview-rows" className="mb-2 font-semibold">
        Rows
      </h2>
      <Toolbar active={filtered} onClear={clear}>
        <FilterSelect
          label="Row status"
          value={values.status}
          options={rowStatusSchema.options.map((s) => ({ value: s, label: rowStatusLabels[s] }))}
          allLabel="All rows"
          onChange={(v) => {
            set({ status: v })
          }}
        />
        <FilterSelect
          label="Duplicates"
          value={values.duplicate}
          options={[{ value: 'true', label: 'Duplicate identifiers only' }]}
          allLabel="All rows"
          onChange={(v) => {
            set({ duplicate: v })
          }}
        />
        <FilterSelect
          label="Catalogue mapping"
          value={values.unresolved}
          options={[{ value: 'true', label: 'Unresolved mappings only' }]}
          allLabel="All rows"
          onChange={(v) => {
            set({ unresolved: v })
          }}
        />
      </Toolbar>
      {body}
      <Dialog
        open={open !== null}
        onClose={() => {
          setOpen(null)
        }}
        title={open ? `Line ${String(open.rowNumber)}` : 'Line'}
        description={open ? rowStatusLabels[open.status] : undefined}
      >
        {open && (
          <div className="flex flex-col gap-3 text-sm">
            <IssueList row={open} />
            {open.issues.some((i) => i.suggestion) && (
              <ul className="list-disc pl-5 text-xs">
                {open.issues
                  .filter((i) => i.suggestion)
                  .map((i) => (
                    <li key={`${i.code}-s`}>{i.suggestion}</li>
                  ))}
              </ul>
            )}
            <table className="w-full text-left text-xs">
              <caption className="sr-only">Values as uploaded</caption>
              <thead>
                <tr>
                  <th scope="col" className="py-1 pr-2">
                    Column
                  </th>
                  <th scope="col" className="py-1">
                    As uploaded
                  </th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(open.source).map(([k, v]) => (
                  <tr key={k} className="border-t border-border/60">
                    <td className="py-1 pr-2 font-mono">{k}</td>
                    <td className="py-1 font-mono break-all">{v === '' ? '—' : v}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Dialog>
    </section>
  )
}
