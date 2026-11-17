import { unitStatusLabels } from '@/features/pdi/lib/labels'
import { movementTypeLabels as vehicleMovementLabels } from '@/features/vehicles/lib/presentation'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import type {
  TimelineItemType,
  TimelineMovement,
  TimelineMovementType,
  TimelineSourceType,
} from '../api/timeline'

/** Readable event names; the vehicle ones are the D-08 labels, so both screens say the same. */
export const timelineTypeLabels: Record<TimelineMovementType, string> = {
  ...vehicleMovementLabels,
  PARTS_RECEIPT: 'Parts received',
  PARTS_ISSUE: 'Parts issued',
  COUNT_ADJUSTMENT: 'Count adjustment',
  OPENING_BALANCE: 'Opening balance',
}

export const sourceTypeLabels: Record<TimelineSourceType, string> = {
  GOODS_RECEIPT: 'Goods receipt',
  PDI_INSPECTION: 'PDI inspection',
  STAFF_ACTION: 'Staff action',
  SUPPLIER_DELIVERY: 'Supplier delivery',
  STOCK_COUNT: 'Stock count',
  OPENING_STOCK: 'Opening stock import',
}

export const itemTypeLabels: Record<TimelineItemType, string> = {
  VEHICLE_UNIT: 'Vehicles',
  STOCK_ITEM: 'Parts & accessories',
}

export const HIDDEN_SOURCE_TEXT = 'Not visible to you'
export const HIDDEN_LOCATION_TEXT = 'A location not visible to you'

/** "SR1 — Showroom", the ID-only form for a location outside the scope, or "No location". */
export function locationLabel(id: string | null, code: string | null, name: string | null): string {
  if (!id) return 'No location'
  if (!code) return HIDDEN_LOCATION_TEXT
  return name ? `${code} — ${name}` : code
}

/** The status change of a vehicle movement ("Not in stock → Awaiting PDI"); null for parts. */
export function statusChange(m: TimelineMovement): { from: string; to: string } | null {
  if (m.itemType !== 'VEHICLE_UNIT' || !m.toStatus) return null
  return {
    from: m.fromStatus ? unitStatusLabels[m.fromStatus] : 'Not in stock',
    to: unitStatusLabels[m.toStatus],
  }
}

/** "10 → 15 PCS" for parts; the server's decimal strings with trailing zeros removed. */
export function quantityChange(m: TimelineMovement): string | null {
  if (m.itemType !== 'STOCK_ITEM') return null
  const trim = (v: string) => v.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
  return `${trim(m.quantityBefore)} → ${trim(m.quantityAfter)}${m.uom ? ` ${m.uom}` : ''}`
}

/**
 * A movement time in the user's own time zone (the API sends UTC, RFC 3339), with
 * seconds so movements of the same minute can be told apart. Invalid input is shown as sent.
 */
export function movementTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeStyle: 'medium' }).format(d)
}

/** The user's UTC offset, e.g. "UTC+05:30", for the time zone note. */
export function utcOffsetLabel(date = new Date()): string {
  const minutes = -date.getTimezoneOffset()
  const sign = minutes >= 0 ? '+' : '-'
  const abs = Math.abs(minutes)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `UTC${sign}${pad(Math.floor(abs / 60))}:${pad(abs % 60)}`
}

export type TimelineReadState =
  'signed-out' | 'forbidden' | 'not-found' | 'invalid-filter' | 'offline' | 'error'

export interface TimelineReadError {
  state: TimelineReadState
  message: string
  fields: { field: string; message: string }[]
  requestId?: string
  retry: boolean
}

/** Maps a failed timeline read to the state to show. A 404 never says whether the movement exists. */
export function describeTimelineError(error: unknown): TimelineReadError {
  if (!(error instanceof ApiError)) {
    return {
      state: 'error',
      message: 'Something went wrong. Please try again.',
      fields: [],
      retry: true,
    }
  }
  const base = { requestId: error.requestId, fields: [] as TimelineReadError['fields'] }
  if (error.code === ClientErrorCode.Network) {
    return {
      ...base,
      state: 'offline',
      message: 'Cannot reach the server. Check your connection and try again.',
      retry: true,
    }
  }
  switch (error.status) {
    case 401:
      return {
        ...base,
        state: 'signed-out',
        message: 'Your session has ended. Sign in again.',
        retry: false,
      }
    case 403:
      return {
        ...base,
        state: 'forbidden',
        message: 'You do not have permission to view stock movements.',
        retry: false,
      }
    case 404:
      return {
        ...base,
        state: 'not-found',
        message: 'This stock movement was not found, or it is not available to you.',
        retry: false,
      }
    case 422:
      return {
        ...base,
        state: 'invalid-filter',
        message: 'Some filters are not valid.',
        fields: error.details,
        retry: false,
      }
  }
  return { ...base, state: 'error', message: error.message, retry: true }
}
