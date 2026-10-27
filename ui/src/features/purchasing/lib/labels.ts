import type { FieldValues, UseFormSetError } from 'react-hook-form'
import { ApiError } from '@/lib/api/client'
import { describeError, type ActionError } from '@/features/catalogue/lib/presentation'
import type { LineWarning, POEvent, POStatus } from '../api/purchaseOrders'

/** Labels for the value sets the database defines (migration 000011 CHECKs). */
export const poStatusLabels: Record<POStatus, string> = {
  DRAFT: 'Draft',
  SUBMITTED: 'Awaiting approval',
  APPROVED: 'Approved',
  PARTIALLY_RECEIVED: 'Partially received',
  RECEIVED: 'Received',
  CLOSED: 'Closed',
  CANCELLED: 'Cancelled',
}

export const eventLabels: Record<POEvent['action'], string> = {
  CREATED: 'Created',
  SUBMITTED: 'Submitted',
  APPROVED: 'Approved',
  APPROVAL_NOT_REQUIRED: 'Approved without approver (within threshold)',
  RETURNED: 'Returned to draft',
  CANCELLED: 'Cancelled',
}

export const warningLabels: Record<LineWarning, string> = {
  PRICE_VARIANCE: 'Agreed cost differs from the catalogue cost',
  PARTIAL_DELIVERY: 'Partly delivered',
  OVER_RECEIPT: 'More received than ordered',
}

/**
 * describeError for purchase orders: field problems go on the form; PO
 * refusals get their own wording. Blocker keys keep the server's messages.
 */
export function describePOError<T extends FieldValues>(
  error: unknown,
  fields?: { setError: UseFormSetError<T>; names: readonly string[] },
): ActionError {
  const base = describeError(error, fields)
  if (!(error instanceof ApiError)) return base
  const blockers = error.details.map((d) => ({ key: d.field, message: d.message }))
  switch (error.code) {
    case 'SUBMISSION_BLOCKED':
      return {
        ...base,
        message: 'This purchase order cannot be submitted yet. Nothing was changed:',
        blockers,
      }
    case 'APPROVAL_BLOCKED':
      return {
        ...base,
        message: 'This purchase order cannot be approved yet. Nothing was changed:',
        blockers,
      }
    case 'SELF_APPROVAL_NOT_ALLOWED':
      return {
        ...base,
        message:
          'You created or submitted this purchase order, so another authorised person must approve it.',
      }
    case 'DUPLICATE':
      return {
        ...base,
        message: 'This variant is already on another line. Change that line’s quantity instead.',
      }
    case 'INVALID_STATE':
      return { ...base, stale: true, message: `${error.message} Reload to see its current status.` }
    default:
      return base
  }
}
