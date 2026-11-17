import { unitStatusLabels } from '@/features/pdi/lib/labels'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import type { DiscrepancyType, ReconcileItemType, Severity } from '../api/reconciliation'

export const discrepancyTypeLabels: Record<DiscrepancyType, string> = {
  VEHICLE_NO_MOVEMENTS: 'Vehicle without movements',
  VEHICLE_MISSING_RECEIPT: 'Vehicle receipt missing',
  VEHICLE_REPEATED_RECEIPT: 'Vehicle received twice',
  VEHICLE_STATUS_MISMATCH: 'Vehicle status mismatch',
  VEHICLE_LOCATION_MISMATCH: 'Vehicle location mismatch',
  VEHICLE_NOT_HELD: 'Vehicle not held',
  VEHICLE_BROKEN_CHAIN: 'Vehicle history gap',
  PARTS_BALANCE_MISMATCH: 'Parts balance mismatch',
  PARTS_BALANCE_MISSING: 'Parts balance missing',
  PARTS_NEGATIVE_BALANCE: 'Negative parts balance',
  PARTS_BROKEN_CHAIN: 'Parts history gap',
}

export const severityLabels: Record<Severity, string> = { HIGH: 'High', MEDIUM: 'Medium' }

export const itemTypeLabels: Record<ReconcileItemType, string> = {
  VEHICLE_UNIT: 'Vehicles',
  STOCK_ITEM: 'Parts & accessories',
}

/** A vehicle status code as its readable label with the code ("Available (AVAILABLE)"); anything else as sent. */
export function valueLabel(value: string | null): string {
  if (value === null) return '—'
  const label = (unitStatusLabels as Record<string, string | undefined>)[value]
  return label ? `${label} (${value})` : value
}

/** The run time in the user's time zone, with seconds. */
export function runTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeStyle: 'medium' }).format(d)
}

export type ReconcileReadState = 'signed-out' | 'forbidden' | 'invalid-filter' | 'offline' | 'error'

export interface ReconcileReadError {
  state: ReconcileReadState
  message: string
  fields: { field: string; message: string }[]
  requestId?: string
  retry: boolean
}

/** Maps a failed run to the state to show. A failed run never shows partial results. */
export function describeReconcileError(error: unknown): ReconcileReadError {
  if (!(error instanceof ApiError)) {
    return {
      state: 'error',
      message: 'Something went wrong. Please try again.',
      fields: [],
      retry: true,
    }
  }
  const base = { requestId: error.requestId, fields: [] as ReconcileReadError['fields'] }
  if (error.code === ClientErrorCode.Network) {
    return {
      ...base,
      state: 'offline',
      message: 'Cannot reach the server. Check your connection and run the reconciliation again.',
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
        message: 'You do not have permission to run the stock reconciliation.',
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
  return {
    ...base,
    state: 'error',
    message:
      error.status >= 500
        ? 'The reconciliation could not be completed. Nothing was changed. Try again.'
        : error.message,
    retry: true,
  }
}
