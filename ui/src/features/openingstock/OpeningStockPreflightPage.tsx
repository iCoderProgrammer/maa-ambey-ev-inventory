import {
  AlertTriangle,
  CheckCircle2,
  Download,
  RefreshCw,
  ShieldCheck,
  XCircle,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { FilterSelect } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { formatDateTime } from '@/features/catalogue/lib/presentation'
import { cn } from '@/lib/utils/cn'
import {
  exceptionsCsv,
  type Preflight,
  type PreflightCategory,
  type PreflightQuantity,
  type PreflightRow,
} from './api/preflight'
import { useImportPerms, usePreflight } from './hooks/useOpeningStock'
import {
  formatBytes,
  PREFLIGHT_NO_CHANGE_TEXT,
  preflightCategoryLabels,
  preflightCategoryTones,
  rowStatusLabels,
  severityTones,
} from './lib/labels'

/**
 * Opening stock RECONCILIATION PREFLIGHT (H-08). Shows the server's read-only
 * re-check of a batch: file integrity, row categories, exceptions, imported
 * quantities beside the physical count, and what still blocks the batch. This
 * page has no approve or commit action and computes no counts of its own.
 */

const basePath = '/inventory/opening-stock'
const linkClass = 'font-medium text-primary hover:underline'

export function OpeningStockPreflightPage() {
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
  else body = <PreflightView id={batchId} />
  return (
    <>
      <PageHeader
        title="Reconciliation preflight"
        breadcrumbs={[
          { label: 'Home', to: '/' },
          { label: 'Opening Stock Import', to: basePath },
          { label: 'Import preview', to: `${basePath}/${batchId}` },
          { label: 'Reconciliation preflight' },
        ]}
        description="Re-check an opening stock file against today's catalogue and inventory, and compare it with the physical count — without changing anything."
      />
      {body}
    </>
  )
}

function PreflightView({ id }: { id: string }) {
  const q = usePreflight(id)
  if (q.isPending) return <LoadingState label="Running the preflight checks…" />
  if (q.isError)
    return (
      <ErrorState
        error={q.error}
        title="Could not run the preflight"
        onRetry={() => void q.refetch()}
      />
    )
  const p = q.data
  return (
    <div className="flex flex-col gap-4">
      <p
        role="status"
        className="flex flex-wrap items-start gap-2 rounded-card border border-success/30 bg-success-subtle px-4 py-3 text-sm text-ink"
      >
        <ShieldCheck className="mt-0.5 size-4 shrink-0 text-success" aria-hidden="true" />
        <span className="min-w-0 flex-1">
          <strong>{PREFLIGHT_NO_CHANGE_TEXT}</strong> Checked {formatDateTime(p.checkedAt)} against{' '}
          {p.businessDate}.
        </span>
        <Button
          variant="secondary"
          size="sm"
          loading={q.isFetching}
          disabled={q.isFetching}
          onClick={() => void q.refetch()}
        >
          <RefreshCw className="size-3.5" aria-hidden="true" />
          Run again
        </Button>
      </p>
      <BatchCard p={p} />
      <BlockersCard p={p} />
      {p.rowsChecked && <SummaryCard p={p} />}
      {p.rowsChecked && <QuantitiesCard p={p} />}
      {p.rowsChecked && <ExceptionsCard p={p} />}
      <p>
        <Link to={`${basePath}/${p.batchId}`} className={linkClass}>
          Back to the preview
        </Link>
      </p>
    </div>
  )
}

function Check({ ok, label }: { ok: boolean; label: string }) {
  return (
    <li className="flex items-center gap-2">
      {ok ? (
        <CheckCircle2 className="size-4 text-success" aria-hidden="true" />
      ) : (
        <XCircle className="size-4 text-danger" aria-hidden="true" />
      )}
      <span>
        {label}
        <span className="sr-only">{ok ? ': passed' : ': failed'}</span>
      </span>
    </li>
  )
}

const sourceTone = { VERIFIED: 'success', INVALID: 'danger', NOT_CHECKED: 'neutral' } as const
const sourceLabel = { VERIFIED: 'Verified', INVALID: 'Failed', NOT_CHECKED: 'Not checked' } as const

function BatchCard({ p }: { p: Preflight }) {
  const s = p.source
  return (
    <Card
      title={`${p.batchReference} · ${p.fileName}`}
      description={`Uploaded ${formatDateTime(p.uploadedAt)}${p.uploadedByName ? ` by ${p.uploadedByName}` : ''} · template ${p.templateVersion ?? '—'} · ${formatBytes(p.fileSizeBytes)}`}
      badge={
        p.commit ? (
          <Badge tone="success">Committed</Badge>
        ) : p.batchStatus === 'REJECTED' ? (
          <Badge tone="danger">Rejected file</Badge>
        ) : p.expired ? (
          <Badge tone="neutral">Expired</Badge>
        ) : (
          <Badge tone="info">Preview</Badge>
        )
      }
    >
      <div className="grid gap-4 text-sm sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <p className="text-xs text-muted">File fingerprint (SHA-256)</p>
          <p className="font-mono text-xs break-all">{p.fileSha256}</p>
          <p className="mt-2 text-xs text-muted">Preview usable until</p>
          <p>
            {formatDateTime(p.expiresAt)}
            {p.expired && ' (expired — history only)'}
          </p>
        </div>
        <div className="flex flex-col gap-2">
          <p className="flex items-center gap-2">
            <span className="text-xs text-muted">Source file integrity</span>
            <Badge tone={sourceTone[s.status]}>{sourceLabel[s.status]}</Badge>
          </p>
          {s.status !== 'NOT_CHECKED' && (
            <ul aria-label="Source file checks" className="flex flex-col gap-1">
              <Check ok={s.fileAvailable} label="Stored file available" />
              <Check ok={s.checksumMatches} label="SHA-256 matches the upload" />
              <Check ok={s.sizeMatches} label="Size matches the upload" />
              <Check ok={s.templateCompatible} label="Readable as the template" />
              <Check ok={s.matchesStagedRows} label="Lines match the staged preview" />
            </ul>
          )}
          <p className={cn(s.status === 'INVALID' && 'font-medium text-danger')}>{s.message}</p>
        </div>
      </div>
    </Card>
  )
}

function BlockersCard({ p }: { p: Preflight }) {
  return (
    <Card
      title="What still blocks this batch"
      description="Data problems must be corrected in a new file. Items with a decision reference wait for the owner; nothing here is decided by the system."
    >
      <ul aria-label="Blockers" className="flex flex-col gap-2 text-sm">
        {p.blockers.map((b) => (
          <li key={b.code} className="flex items-start gap-2">
            <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
            <span className="min-w-0 flex-1">
              {b.message}
              {b.decision && (
                <Badge tone="neutral" className="ml-2">
                  Pending: {b.decision}
                </Badge>
              )}
            </span>
          </li>
        ))}
      </ul>
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

function SummaryCard({ p }: { p: Preflight }) {
  const s = p.summary
  return (
    <Card
      title="Records today"
      description={`At preview: ${String(p.previewSummary.validRows)} of ${String(p.previewSummary.totalRows)} valid. Warnings are shown but never accepted automatically (OQ-49).`}
    >
      <dl
        aria-label="Preflight summary"
        className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5"
      >
        <Tile label="Total records" value={s.totalRows} />
        <Tile label="Valid" value={s.validRows} tone="text-success" />
        <Tile label="Warnings" value={s.warningRows} tone="text-warning" />
        <Tile label="Blocking errors" value={s.blockingErrorRows} tone="text-danger" />
        <Tile label="Unverified catalogue" value={s.unverifiedRows} tone="text-info" />
        <Tile
          label="Manual reconciliation"
          value={s.manualReconciliationRows}
          tone="text-warning"
        />
        <Tile label="Duplicate identifiers" value={s.duplicateRows} tone="text-danger" />
        <Tile label="Missing chassis" value={s.missingChassisRows} tone="text-danger" />
        <Tile label="Changed since preview" value={s.changedSincePreviewRows} tone="text-warning" />
      </dl>
      {p.issueCounts.length > 0 && (
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer font-medium">
            Issues by type ({p.issueCounts.length})
          </summary>
          <ul className="mt-2 flex flex-col gap-1">
            {p.issueCounts.map((c) => (
              <li key={`${c.code}-${c.severity}`} className="flex items-center gap-2">
                <Badge tone={severityTones[c.severity]}>{c.severity}</Badge>
                <span className="font-mono text-xs">{c.code}</span>
                <span className="text-muted">
                  {c.rows} row{c.rows === 1 ? '' : 's'}
                </span>
              </li>
            ))}
          </ul>
        </details>
      )}
    </Card>
  )
}

const notRecorded = <span className="text-muted">Not recorded</span>

const quantityColumns: Column<PreflightQuantity>[] = [
  {
    key: 'group',
    header: 'Location · catalogue',
    render: (q) => (
      <span>
        <span className="font-mono">{q.locationCode || '—'}</span> · {q.brand || '—'}{' '}
        <span className="font-mono">
          {q.modelCode || '—'}/{q.variantCode || '—'}
        </span>
      </span>
    ),
  },
  { key: 'status', header: 'Stock status', render: (q) => q.stockStatus || '—' },
  {
    key: 'file',
    header: 'Imported (file)',
    render: (q) => <span className="tabular-nums">{q.fileRows}</span>,
  },
  {
    key: 'valid',
    header: 'Valid today',
    render: (q) => (
      <span className={cn('tabular-nums', q.validRows < q.fileRows && 'font-semibold text-danger')}>
        {q.validRows}
      </span>
    ),
  },
  {
    key: 'physical',
    header: 'Physical count',
    render: (q) => (q.physicalCount === null ? notRecorded : q.physicalCount),
  },
  {
    key: 'difference',
    header: 'Difference',
    render: (q) => (q.difference === null ? <span className="text-muted">—</span> : q.difference),
  },
]

function QuantitiesCard({ p }: { p: Preflight }) {
  const pc = p.physicalCount
  return (
    <Card
      title="Imported quantity and physical count"
      description="Quantities from the file, grouped as uploaded. The physical count is shown separately and is never filled in by the system."
    >
      <p
        role="note"
        className="mb-3 rounded-control border border-warning/30 bg-warning-subtle px-3 py-2 text-sm"
      >
        <strong>Physical count: </strong>
        {pc.status === 'REFERENCE_ONLY' && pc.reference
          ? `signed count ${pc.reference}${pc.countDate ? ` of ${pc.countDate}` : ''} is named in the approval. `
          : ''}
        {pc.message} Differences cannot be calculated until counted quantities are recorded (H-08).
      </p>
      {p.quantities.length === 0 ? (
        <EmptyState title="No records in this file" />
      ) : (
        <ResponsiveTable
          caption="Imported quantity and physical count"
          columns={quantityColumns}
          rows={p.quantities}
          rowKey={(q) =>
            [q.locationCode, q.brand, q.modelCode, q.variantCode, q.stockStatus].join('|')
          }
        />
      )}
    </Card>
  )
}

const exceptionColumns: Column<PreflightRow>[] = [
  {
    key: 'line',
    header: 'Line',
    render: (r) => (
      <span className="flex flex-wrap items-center gap-2">
        <span className="font-medium">Line {r.rowNumber}</span>
        <Badge tone={preflightCategoryTones[r.category]}>
          {preflightCategoryLabels[r.category]}
        </Badge>
      </span>
    ),
  },
  {
    key: 'chassis',
    header: 'Chassis/frame',
    render: (r) =>
      r.source.chassis_number?.trim() ? (
        <span className="font-mono break-all">{r.source.chassis_number}</span>
      ) : (
        <span className="text-danger">Missing</span>
      ),
  },
  {
    key: 'status',
    header: 'Preview → today',
    render: (r) => (
      <span className={cn(r.changedSincePreview && 'font-semibold text-warning')}>
        {rowStatusLabels[r.previewStatus]} → {rowStatusLabels[r.currentStatus]}
        {r.changedSincePreview && <span className="sr-only"> (changed since the preview)</span>}
      </span>
    ),
  },
  {
    key: 'issues',
    header: 'Issues',
    render: (r) => (
      <ul className="flex flex-col gap-1">
        {r.issues.map((i, n) => (
          <li key={`${i.code}-${String(n)}`} className="text-xs">
            <Badge tone={severityTones[i.severity]} className="mr-1">
              {i.severity}
            </Badge>
            {i.field && <span className="font-mono">{i.field}: </span>}
            {i.message}
            {i.suggestion && <span className="block text-muted">{i.suggestion}</span>}
          </li>
        ))}
      </ul>
    ),
  },
]

const categoryOptions = (Object.keys(preflightCategoryLabels) as PreflightCategory[]).map((c) => ({
  value: c,
  label: preflightCategoryLabels[c],
}))

function ExceptionsCard({ p }: { p: Preflight }) {
  const [category, setCategory] = useState('')
  const rows = category ? p.exceptions.filter((r) => r.category === category) : p.exceptions
  const download = () => {
    const blob = new Blob([exceptionsCsv(p)], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${p.batchReference}-preflight-exceptions.csv`
    a.click()
    URL.revokeObjectURL(url)
  }
  return (
    <Card
      title="Records needing attention"
      description="Every record that is not valid today, with each reason. Correct the file and upload it again; nothing is corrected automatically."
      actions={
        p.exceptions.length > 0 ? (
          <Button variant="secondary" size="sm" onClick={download}>
            <Download className="size-3.5" aria-hidden="true" />
            Download exceptions (CSV)
          </Button>
        ) : undefined
      }
    >
      {p.exceptions.length === 0 ? (
        <EmptyState title="No exceptions">
          Every record passes today&apos;s checks. The batch still needs the items listed under
          &ldquo;What still blocks this batch&rdquo;.
        </EmptyState>
      ) : (
        <>
          <div className="mb-3">
            <FilterSelect
              label="Category"
              value={category}
              options={categoryOptions}
              onChange={setCategory}
              allLabel="All categories"
            />
          </div>
          {rows.length === 0 ? (
            <EmptyState title="No records in this category" />
          ) : (
            <ResponsiveTable
              caption="Records needing attention"
              columns={exceptionColumns}
              rows={rows}
              rowKey={(r) => String(r.rowNumber)}
            />
          )}
        </>
      )}
    </Card>
  )
}
