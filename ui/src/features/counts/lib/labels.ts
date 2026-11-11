import type { BadgeTone } from '@/components/ui/Badge'
import {
  formatQuantity,
  type CountLine,
  type CountStatus,
  type LineOutcome,
} from '../api/stockCounts'

export const countStatusLabels: Record<CountStatus, string> = {
  DRAFT: 'Draft',
  SUBMITTED: 'Awaiting approval',
  APPROVED: 'Approved',
  REJECTED: 'Rejected',
}

export const countStatusTone: Record<CountStatus, BadgeTone> = {
  DRAFT: 'neutral',
  SUBMITTED: 'warning',
  APPROVED: 'success',
  REJECTED: 'danger',
}

/** What happened to a line, as the server reports it (never inferred from the variance). */
export const outcomeLabels: Record<LineOutcome, string> = {
  PENDING: 'Not decided yet',
  NO_CHANGE: 'No change',
  POSTED: 'Adjustment posted',
  DISCREPANCY_RECORDED: 'Finding recorded — vehicle not changed',
  NOT_POSTED: 'Not posted (rejected)',
}

export const outcomeTone: Record<LineOutcome, BadgeTone> = {
  PENDING: 'neutral',
  NO_CHANGE: 'neutral',
  POSTED: 'success',
  DISCREPANCY_RECORDED: 'warning',
  NOT_POSTED: 'neutral',
}

export function lineLabel(l: CountLine): string {
  if (l.itemType === 'VEHICLE_UNIT') return `Vehicle ${l.unitNumber ?? l.vehicleUnitId ?? ''}`
  return `${l.sku ?? ''}${l.itemName ? ` — ${l.itemName}` : ''}`
}

export function expectedText(l: CountLine): string {
  if (l.itemType === 'VEHICLE_UNIT') return `Here (${l.expectedStatus ?? '—'})`
  return `${formatQuantity(l.expectedQuantity)} ${l.uom ?? ''}`.trim()
}

export function observedText(l: CountLine): string {
  if (l.itemType === 'VEHICLE_UNIT') return l.found ? 'Found' : 'Not found'
  return `${formatQuantity(l.observedQuantity)} ${l.uom ?? ''}`.trim()
}

export function varianceText(l: CountLine): string {
  if (l.variance === '0.000') return 'None'
  if (l.itemType === 'VEHICLE_UNIT') return 'Missing'
  const v = formatQuantity(l.variance)
  return `${v.startsWith('-') ? '−' + v.slice(1) : '+' + v} ${l.uom ?? ''}`.trim()
}
