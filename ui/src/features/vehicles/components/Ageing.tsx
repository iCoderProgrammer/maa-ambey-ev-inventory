import { AlertTriangle, CalendarClock } from 'lucide-react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/StateViews'
import { formatDate } from '@/features/catalogue/lib/presentation'
import type { AgeingSummary, StockAgeing } from '../api/vehicleUnits'
import {
  bandSetNote,
  missingReasonLabels,
  NO_ORIGIN_TEXT,
  originSourceLabels,
  stockAgeText,
} from '../lib/ageing'
import { describeVehicleUnitReadError } from '../lib/presentation'

/**
 * Stock ageing views (E-07). Everything shown is the server's: age, band,
 * origin and missing-origin reason (ADR-029). A unit without a reliable origin
 * date is shown as such — never as 0 days.
 */

export function AgeingBadge({ ageing }: { ageing: StockAgeing }) {
  if (ageing.status === 'MISSING_ORIGIN')
    return (
      <Badge tone="warning" icon={<AlertTriangle className="size-3" aria-hidden="true" />}>
        {NO_ORIGIN_TEXT}
      </Badge>
    )
  if (ageing.bandLabel) return <Badge tone="neutral">{ageing.bandLabel}</Badge>
  return <Badge tone="neutral">No band</Badge>
}

/** List cell: age, band and origin, or the missing-origin label and reason. */
export function AgeingCell({ ageing }: { ageing: StockAgeing }) {
  if (ageing.status === 'MISSING_ORIGIN')
    return (
      <span className="flex flex-col items-start gap-1">
        <AgeingBadge ageing={ageing} />
        {ageing.missingReason && (
          <span className="text-xs text-muted" title={ageing.missingExplanation ?? undefined}>
            {missingReasonLabels[ageing.missingReason]}
          </span>
        )}
      </span>
    )
  return (
    <span className="flex flex-col items-start gap-1">
      <span className="font-medium tabular-nums">{stockAgeText(ageing)}</span>
      <AgeingBadge ageing={ageing} />
      {ageing.originSource && ageing.originDate && (
        <span className="text-xs text-muted">
          from {originSourceLabels[ageing.originSource].toLowerCase()} of{' '}
          {formatDate(ageing.originDate)}
        </span>
      )}
    </span>
  )
}

/**
 * The ageing summary of the filtered units (a report on the list page): totals,
 * one entry per band, and units without a reliable origin date. Choosing an
 * entry filters the list by it (server-side).
 */
export function AgeingSummaryPanel({
  summary,
  isPending,
  error,
  onRetry,
  activeBand,
  activeOrigin,
  onBand,
  onMissing,
}: {
  summary: AgeingSummary | undefined
  isPending: boolean
  error: unknown
  onRetry: () => void
  activeBand: string
  activeOrigin: string
  onBand: (key: string) => void
  onMissing: () => void
}) {
  if (isPending)
    return (
      <div role="status" aria-label="Loading stock ageing" className="mb-4">
        <Skeleton className="h-24 w-full" />
      </div>
    )
  if (!summary) {
    if (!error) return null
    const d = describeVehicleUnitReadError(error)
    if (d.state === 'invalid-filter' || d.state === 'forbidden') return null // the list explains it
    return (
      <div
        role="alert"
        className="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-card border border-danger/30 bg-danger-subtle px-4 py-3 text-sm"
      >
        <span>Stock ageing could not be loaded. {d.message}</span>
        {d.retry && (
          <Button variant="secondary" size="sm" onClick={onRetry}>
            Try again
          </Button>
        )}
      </div>
    )
  }
  const tile =
    'flex min-w-0 flex-col items-start gap-0.5 rounded-control border px-3 py-2 text-left transition-colors hover:border-primary focus-visible:outline-accent'
  return (
    <section
      aria-labelledby="ageing-summary-title"
      className="mb-4 rounded-card border border-border/80 bg-surface p-4 shadow-xs"
    >
      <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 id="ageing-summary-title" className="flex items-center gap-2 font-semibold">
            <CalendarClock className="size-4 text-muted" aria-hidden="true" />
            Stock ageing
          </h2>
          <p className="text-xs text-muted">
            {summary.total} {summary.total === 1 ? 'vehicle' : 'vehicles'} as of{' '}
            {formatDate(summary.asOf)} ({summary.timeZone}). Age counts whole days from the posted
            goods receipt date. {bandSetNote(summary.bandSet)}
          </p>
        </div>
      </div>
      <ul
        aria-label="Vehicles by ageing band"
        className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6"
      >
        {summary.bands.map((b) => (
          <li key={b.key}>
            <button
              type="button"
              className={`${tile} w-full ${activeBand === b.key ? 'border-primary bg-primary/5' : 'border-border/80'}`}
              aria-pressed={activeBand === b.key}
              onClick={() => {
                onBand(activeBand === b.key ? '' : b.key)
              }}
            >
              <span className="text-xs text-muted">{b.label}</span>
              <span className="text-lg font-semibold tabular-nums">{b.count}</span>
            </button>
          </li>
        ))}
        {summary.unbanded > 0 && (
          <li className={`${tile} border-border/80`}>
            <span className="text-xs text-muted">Aged, no band</span>
            <span className="text-lg font-semibold tabular-nums">{summary.unbanded}</span>
          </li>
        )}
        <li>
          <button
            type="button"
            className={`${tile} w-full ${activeOrigin === 'MISSING' ? 'border-warning bg-warning-subtle' : 'border-border/80'}`}
            aria-pressed={activeOrigin === 'MISSING'}
            onClick={onMissing}
          >
            <span className="text-xs text-muted">{NO_ORIGIN_TEXT}</span>
            <span
              className={`text-lg font-semibold tabular-nums ${summary.missingOrigin > 0 ? 'text-warning' : ''}`}
            >
              {summary.missingOrigin}
            </span>
          </button>
        </li>
      </ul>
      {summary.missingOrigin > 0 && (
        <ul
          aria-label="Why some vehicles have no origin date"
          className="mt-2 flex flex-wrap gap-2 text-xs"
        >
          {Object.entries(summary.missingByReason).map(([reason, n]) => (
            <li
              key={reason}
              className="rounded-full border border-border bg-surface-muted px-3 py-1"
            >
              {(missingReasonLabels as Record<string, string | undefined>)[reason] ?? reason}: {n}
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
