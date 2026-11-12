import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { formatINR } from '@/features/pricing/lib/format'
import type {
  VehicleUnit,
  VehicleUnitDocument,
  VehicleUnitMovementType,
  VehicleUnitSort,
  VehicleUnitStatus,
} from '../api/vehicleUnits'

/**
 * Saleable stock is AVAILABLE only (INV-006). Every other status — awaiting PDI,
 * quality or service hold, rejected, on display, reserved, sold, delivered,
 * returns — is never presented as saleable (INV-007, PO-006). Presentation only:
 * the server decides every reservation and sale.
 */
export function isSaleable(status: VehicleUnitStatus): boolean {
  return status === 'AVAILABLE'
}

/** Why a status is not saleable, in plain words. */
export const notSaleableReason: Partial<Record<VehicleUnitStatus, string>> = {
  RECEIVED_PENDING_PDI: 'Awaiting PDI — not saleable until it passes inspection.',
  QUALITY_HOLD: 'On quality hold — not saleable.',
  SERVICE_HOLD: 'On service hold — not saleable.',
  REJECTED: 'Rejected at PDI — not saleable.',
  DISPLAY: 'On display — not saleable until it is taken off display.',
  RESERVED: 'Reserved for a customer — not available to sell.',
  SOLD: 'Sold — not stock for sale.',
  DELIVERED: 'Delivered — not stock for sale.',
  ORDERED: 'Ordered — not in stock yet.',
  IN_TRANSIT: 'In transit — not in stock yet.',
  RETURN_PENDING_INSPECTION: 'Return awaiting inspection — not saleable.',
  RETURNED_TO_SUPPLIER: 'Returned to the supplier — not in stock.',
}

export const movementTypeLabels: Record<VehicleUnitMovementType, string> = {
  RECEIPT: 'Received into stock',
  PDI_PASS: 'Passed PDI',
  QUALITY_HOLD: 'Put on quality hold',
  REJECT: 'Rejected',
  DISPLAY_ON: 'Put on display',
  DISPLAY_OFF: 'Taken off display',
}

export const documentTypeLabels: Record<VehicleUnitDocument['documentType'], string> = {
  DELIVERY_CHALLAN: 'Delivery challan',
  SUPPLIER_INVOICE: 'Supplier invoice',
  PHOTO: 'Photo',
  OTHER: 'Other document',
  PDI_EVIDENCE: 'PDI evidence',
}

export const sortLabels: Record<VehicleUnitSort, string> = {
  '-createdAt': 'Newest registered',
  createdAt: 'Oldest registered',
  '-receivedDate': 'Received (newest)',
  receivedDate: 'Received (oldest)',
  unitNumber: 'Unit number (A–Z)',
  '-unitNumber': 'Unit number (Z–A)',
}

/** "0 days", "1 day", "12 days". */
export function ageText(days: number): string {
  return `${String(days)} ${days === 1 ? 'day' : 'days'}`
}

/** Shown instead of an amount when the cost exists but the caller may not see it. */
export const HIDDEN_COST_TEXT = 'Recorded (not visible to you)'

/**
 * The cost as text, or null when nothing should be shown. The server's
 * costHidden is authoritative; an amount appears only when the server sent one
 * and the caller holds vehicle_units.cost.view. Never "₹0" for a hidden cost.
 */
export function unitCostText(
  unit: Pick<VehicleUnit, 'unitCost' | 'costHidden'>,
  canViewCost: boolean,
): string | null {
  if (unit.costHidden) return HIDDEN_COST_TEXT
  if (!canViewCost || unit.unitCost === null) return null
  return formatINR(unit.unitCost)
}

export type VehicleUnitReadState =
  'signed-out' | 'forbidden' | 'not-found' | 'invalid-filter' | 'offline' | 'error'

export interface VehicleUnitReadError {
  state: VehicleUnitReadState
  message: string
  /** Server field errors for an invalid list request (q, status, the id filters, page, limit). */
  fields: { field: string; message: string }[]
  requestId?: string
  /** Whether trying the same request again may help. */
  retry: boolean
}

/**
 * Maps a failed vehicle unit read to a safe screen state. A 404 never says
 * whether the unit exists elsewhere (out-of-scope units answer 404 too), and an
 * out-of-scope location filter is shown as an invalid filter, not retried
 * without it.
 */
export function describeVehicleUnitReadError(error: unknown): VehicleUnitReadError {
  if (!(error instanceof ApiError)) {
    return {
      state: 'error',
      message: 'Something went wrong. Please try again.',
      fields: [],
      retry: true,
    }
  }
  const base = { requestId: error.requestId, fields: [] as VehicleUnitReadError['fields'] }
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
        message: 'You do not have permission to view vehicle units.',
        retry: false,
      }
    case 404:
      return {
        ...base,
        state: 'not-found',
        message: 'This vehicle unit was not found, or it is not available to you.',
        retry: false,
      }
    case 422:
      return {
        ...base,
        state: 'invalid-filter',
        message: 'Some search filters are not valid. Change them and try again.',
        fields: error.details,
        retry: false,
      }
  }
  return { ...base, state: 'error', message: error.message, retry: error.status >= 500 }
}
