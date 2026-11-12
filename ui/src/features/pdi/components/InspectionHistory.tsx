import { useMutation } from '@tanstack/react-query'
import { ArrowRight, ChevronDown, Download, History } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import { fetchAttachment, formatBytes } from '@/features/attachments/api/attachments'
import { downloadErrorMessage } from '@/features/attachments/messages'
import { formatDateTime } from '@/features/catalogue/lib/presentation'
import type { Evidence, Inspection } from '../api/pdi'
import { useInspection } from '../hooks/usePdi'
import {
  dispositionLabels,
  dispositionTone,
  outcomeLabels,
  resultLabels,
  unitStatusLabels,
} from '../lib/labels'

function who(i: Inspection, meId: string | undefined) {
  return i.inspectedBy === meId ? 'You' : (i.inspectedByName ?? 'Unknown user')
}

/** The unit's inspections, newest first (append-only: nothing here can be edited). */
export function InspectionHistory({
  inspections,
  meId,
  canSeeEvidence,
}: {
  inspections: Inspection[]
  meId: string | undefined
  canSeeEvidence: boolean
}) {
  return (
    <Card
      title="Inspection history"
      description="Every recorded inspection of this vehicle, newest first."
    >
      {inspections.length === 0 ? (
        <EmptyState
          title="No inspection recorded yet"
          icon={<History className="size-6 text-muted/80" />}
        >
          Recorded inspections appear here with who decided, when and why.
        </EmptyState>
      ) : (
        <ol aria-label="Inspections" className="flex flex-col gap-3">
          {inspections.map((i) => (
            <HistoryEntry key={i.id} inspection={i} meId={meId} canSeeEvidence={canSeeEvidence} />
          ))}
        </ol>
      )}
    </Card>
  )
}

function HistoryEntry({
  inspection: i,
  meId,
  canSeeEvidence,
}: {
  inspection: Inspection
  meId: string | undefined
  canSeeEvidence: boolean
}) {
  const [open, setOpen] = useState(false)
  const panelId = `pdi-${i.id}`
  return (
    <li className="rounded-control border border-border bg-surface">
      <div className="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-sm font-medium">{i.inspectionNumber}</span>
            <Badge tone={dispositionTone[i.disposition]}>{dispositionLabels[i.disposition]}</Badge>
            <span className="inline-flex items-center gap-1 text-xs text-muted">
              {unitStatusLabels[i.fromStatus]}
              <ArrowRight className="size-3" aria-label="to" />
              {unitStatusLabels[i.toStatus]}
            </span>
          </div>
          <p className="text-xs text-muted">
            {who(i, meId)} · {formatDateTime(i.inspectedAt)} · {resultLabels[i.result]}
            {i.itemCount > 0 &&
              ` (${String(i.itemCount)} items, ${String(i.failedCount)} failed)`}{' '}
            · {i.evidenceCount} evidence file{i.evidenceCount === 1 ? '' : 's'}
          </p>
          {i.notes && <p className="text-sm whitespace-pre-wrap text-ink">{i.notes}</p>}
        </div>
        <Button
          variant="ghost"
          size="sm"
          aria-expanded={open}
          aria-controls={panelId}
          onClick={() => {
            setOpen((o) => !o)
          }}
        >
          {open ? 'Hide details' : 'Show details'}
          <ChevronDown
            className={`size-4 transition-transform ${open ? 'rotate-180' : ''}`}
            aria-hidden="true"
          />
        </Button>
      </div>
      {open && (
        <div id={panelId} className="border-t border-border px-4 py-3">
          <InspectionDetails id={i.id} canSeeEvidence={canSeeEvidence} />
        </div>
      )}
    </li>
  )
}

function InspectionDetails({ id, canSeeEvidence }: { id: string; canSeeEvidence: boolean }) {
  const q = useInspection(id)
  if (q.isPending) return <LoadingState label="Loading inspection…" />
  if (q.isError)
    return (
      <ErrorState
        error={q.error}
        title="Could not load the inspection"
        onRetry={() => void q.refetch()}
      />
    )
  const d = q.data
  return (
    <div className="flex flex-col gap-3 text-sm">
      {d.items.length > 0 ? (
        <table className="w-full text-left text-sm">
          <caption className="sr-only">Checklist outcomes of {d.inspectionNumber}</caption>
          <thead className="text-xs text-muted">
            <tr>
              <th scope="col" className="py-1 pr-3 font-medium">
                Item
              </th>
              <th scope="col" className="py-1 pr-3 font-medium">
                Outcome
              </th>
              <th scope="col" className="py-1 font-medium">
                Note
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {d.items.map((it) => (
              <tr key={it.checklistItemId}>
                <td className="py-1.5 pr-3">
                  {it.label} <span className="font-mono text-xs text-muted">{it.code}</span>
                </td>
                <td className="py-1.5 pr-3">
                  <Badge tone={it.outcome === 'PASS' ? 'success' : 'danger'}>
                    {outcomeLabels[it.outcome]}
                  </Badge>
                </td>
                <td className="py-1.5 text-muted">{it.note ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className="text-muted">No checklist was configured for this inspection.</p>
      )}
      <div>
        <p className="font-medium">Evidence</p>
        {d.evidenceHidden || !canSeeEvidence ? (
          <p className="text-muted">
            {d.evidenceCount} file{d.evidenceCount === 1 ? '' : 's'} — viewing evidence needs
            attachment permission.
          </p>
        ) : d.evidence.length === 0 ? (
          <p className="text-muted">None.</p>
        ) : (
          <ul className="mt-1 flex flex-col gap-1.5">
            {d.evidence.map((e) => (
              <EvidenceRow key={e.attachmentId} evidence={e} />
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

/** Downloads through a fresh signed link after the server's own checks (B-09); never a public URL. */
function EvidenceRow({ evidence: e }: { evidence: Evidence }) {
  const download = useMutation({
    mutationFn: async () => {
      const blob = await fetchAttachment(e.attachmentId, 'attachment')
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = e.originalName
      a.rel = 'noopener'
      document.body.append(a)
      a.click()
      a.remove()
      setTimeout(() => {
        URL.revokeObjectURL(url)
      }, 10_000)
    },
  })
  return (
    <li className="flex flex-wrap items-center gap-2">
      <span className="min-w-0 truncate" title={e.originalName}>
        {e.originalName}
      </span>
      <span className="text-xs text-muted">{formatBytes(e.sizeBytes)}</span>
      <Button
        variant="secondary"
        size="sm"
        disabled={download.isPending}
        aria-label={`Download ${e.originalName}`}
        onClick={() => {
          download.mutate()
        }}
      >
        <Download className="size-4" aria-hidden="true" />
        {download.isPending ? 'Downloading…' : 'Download'}
      </Button>
      {download.isError && (
        <span role="alert" className="text-xs text-danger">
          {downloadErrorMessage(download.error)}
        </span>
      )}
    </li>
  )
}
