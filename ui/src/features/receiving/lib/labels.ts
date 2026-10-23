import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '@/lib/api/client'
import { describeError, type ActionError } from '@/features/catalogue/lib/presentation'
import type {
  GRNDocumentType,
  GRNEvent,
  GRNLineWarning,
  GRNStatus,
  GRNUnitWarning,
  UnitCondition,
} from '../api/goodsReceipts'

/** Labels for the value sets the database defines (migration 000012 CHECKs). */
export const grnStatusLabels: Record<GRNStatus, string> = {
  DRAFT: 'Draft',
  POSTED: 'Posted',
  CANCELLED: 'Cancelled',
}

export const conditionLabels: Record<UnitCondition, string> = {
  ACCEPTED: 'Accepted',
  REJECTED: 'Rejected',
}

export const documentTypeLabels: Record<GRNDocumentType, string> = {
  DELIVERY_CHALLAN: 'Delivery challan',
  SUPPLIER_INVOICE: 'Supplier invoice',
  PHOTO: 'Photo',
  OTHER: 'Other',
}

export const eventLabels: Record<GRNEvent['action'], string> = {
  CREATED: 'Draft started',
  CANCELLED: 'Cancelled',
  POSTED: 'Posted',
}

export const lineWarningLabels: Record<GRNLineWarning, string> = {
  OVER_RECEIPT: 'Over-receipt (authorised)',
  OVER_RECEIPT_NOT_AUTHORISED: 'Over-receipt not authorised',
  PARTIAL_DELIVERY: 'Partial delivery',
  IDENTIFIERS_INCOMPLETE: 'Identifiers incomplete',
  REJECTION_NOT_EXPLAINED: 'Rejection without a note',
}

export const unitWarningLabels: Record<GRNUnitWarning, string> = {
  CHASSIS_MISSING: 'No chassis/frame number',
  DUPLICATE_MOTOR_SERIAL: 'Motor serial repeated',
  DUPLICATE_BATTERY_SERIAL: 'Battery serial repeated',
  DUPLICATE_CHARGER_SERIAL: 'Charger serial repeated',
}

/** "lines[0].units[1].chassisNumber" (server) → "lines.0.units.1.chassisNumber" (form). */
export function formPath(field: string): string {
  return field.replace(/\[(\d+)\]/g, '.$1')
}

/** "lines[0].units[1].chassisNumber" → "Line 1, unit 2: chassis/frame number". */
export function fieldLabel(field: string, lineNos?: number[]): string {
  const m = /^lines\[(\d+)\](?:\.units\[(\d+)\])?\.?(.*)$/.exec(field)
  if (!m) return field
  const index = Number(m[1])
  const parts = [`Line ${String(lineNos?.[index] ?? index + 1)}`]
  if (m[2] !== undefined) parts.push(`unit ${String(Number(m[2]) + 1)}`)
  const names: Record<string, string> = {
    receivedQuantity: 'received',
    acceptedQuantity: 'accepted',
    rejectedQuantity: 'rejected',
    chassisNumber: 'chassis/frame number',
    motorSerial: 'motor serial',
    batterySerial: 'battery serial',
    chargerSerial: 'charger serial',
    overReceiptReason: 'over-receipt reason',
    discrepancyNote: 'discrepancy note',
    units: 'units',
    purchaseOrderLineId: 'PO line',
  }
  return `${parts.join(', ')}${m[3] ? `: ${names[m[3]] ?? m[3]}` : ''}`
}

/** Error codes the receipt API adds (API_CONVENTIONS "Goods receipts"). */
export const GRNErrorCode = {
  OverReceiptNeedsOverride: 'OVER_RECEIPT_REQUIRES_OVERRIDE',
  OverReceiptForbidden: 'OVER_RECEIPT_NOT_AUTHORISED',
} as const

/**
 * describeError for goods receipts. Field problems (validation, duplicate
 * identifiers, over-receipt) are put on the matching form fields and also
 * listed, so nothing the server said is hidden. Draft values stay in the form.
 */
export function describeGRNError<T extends FieldValues>(
  error: unknown,
  fields?: { setError: UseFormSetError<T>; isField: (path: string) => boolean },
  lineNos?: number[],
): ActionError {
  if (!(error instanceof ApiError)) return describeError(error)
  const listed = ['VALIDATION_FAILED', 'DUPLICATE', ...Object.values(GRNErrorCode)]
  if (!listed.includes(error.code)) {
    const base = describeError(error)
    if (error.code === 'INVALID_STATE') {
      return { ...base, stale: true, message: `${error.message} Reload to see its current status.` }
    }
    return base
  }
  for (const d of error.details) {
    const path = formPath(d.field)
    if (fields?.isField(path)) fields.setError(path as Path<T>, { message: d.message })
  }
  const blockers = error.details.map((d) => ({
    key: fieldLabel(d.field, lineNos),
    message: d.message,
  }))
  const requestId = error.requestId
  switch (error.code) {
    case GRNErrorCode.OverReceiptNeedsOverride:
      return {
        requestId,
        blockers,
        message:
          'More is received than the purchase order has left. Someone allowed to authorise over-receipt must enter a reason. Nothing was saved; your entries are kept:',
      }
    case GRNErrorCode.OverReceiptForbidden:
      return {
        requestId,
        blockers,
        message:
          'You may not authorise over-receipt. Remove the reason or the extra units, or ask an authorised person. Nothing was saved:',
      }
    case 'DUPLICATE':
      return {
        requestId,
        blockers,
        message: 'Some identifiers are already on another goods receipt. Nothing was saved:',
      }
    default:
      return {
        requestId,
        blockers,
        message: 'Check the highlighted entries. Nothing was saved; your entries are kept:',
      }
  }
}
