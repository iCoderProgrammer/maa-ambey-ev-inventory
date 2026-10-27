import { ApiError, ClientErrorCode } from '@/lib/api/client'

export type PartsFailure =
  | 'insufficient'
  | 'serial'
  | 'archived'
  | 'key-reused'
  | 'signed-out'
  | 'forbidden'
  | 'not-found'
  | 'invalid-input'
  | 'uncertain'
  | 'error'

export interface PartsError {
  kind: PartsFailure
  message: string
  /** Server field errors (locationId, quantity, uom, supplierId, sourceReference, reason, note). */
  fields: { field: string; message: string }[]
  requestId?: string
  /**
   * The outcome is unknown: sending the same request again (same values, so the
   * same Idempotency-Key) either records it once or returns what was recorded.
   */
  retrySame: boolean
}

/** Maps a failed receipt/issue to what the dialog shows. Unknown outcomes are never called failures. */
export function describePartsError(error: unknown): PartsError {
  const none = { fields: [] as PartsError['fields'], retrySame: false }
  if (!(error instanceof ApiError))
    return { ...none, kind: 'error', message: 'Something went wrong. Nothing was changed.' }
  const base = { ...none, requestId: error.requestId }
  if (
    error.code === ClientErrorCode.Network ||
    error.status >= 500 ||
    error.code === 'REQUEST_TIMEOUT'
  ) {
    return {
      ...base,
      kind: 'uncertain',
      message:
        'The server did not confirm this. It may or may not have been recorded: send it again unchanged to find out — it will not be counted twice.',
      retrySame: true,
    }
  }
  if (error.code === ClientErrorCode.InvalidResponse) {
    return {
      ...base,
      kind: 'uncertain',
      message:
        'The server answered, but not as expected. Check the balance and history before trying again.',
    }
  }
  switch (error.code) {
    case 'INSUFFICIENT_STOCK':
      return {
        ...base,
        kind: 'insufficient',
        message:
          'There is not enough stock at this location for this issue. The balance has been refreshed. Nothing was changed.',
        fields: [{ field: 'quantity', message: 'More than the available balance' }],
      }
    case 'TRACKING_NOT_SUPPORTED':
      return {
        ...base,
        kind: 'serial',
        message:
          'Serial-tracked items cannot be received or issued yet: there is no serial register. Nothing was changed.',
      }
    case 'INVALID_STATE':
      return { ...base, kind: 'archived', message: error.message }
    case 'IDEMPOTENCY_KEY_REUSED':
      return {
        ...base,
        kind: 'key-reused',
        message:
          'This request could not be matched to the earlier attempt. Nothing was changed. Close this form and start again.',
      }
  }
  switch (error.status) {
    case 401:
      return { ...base, kind: 'signed-out', message: 'Your session has ended. Sign in again.' }
    case 403:
      return {
        ...base,
        kind: 'forbidden',
        message: 'You do not have permission to do this. Nothing was changed.',
      }
    case 404:
      return {
        ...base,
        kind: 'not-found',
        message: 'This SKU was not found, or it is not available to you.',
      }
    case 422:
      return {
        ...base,
        kind: 'invalid-input',
        message: 'Check the highlighted fields. Nothing was changed.',
        fields: error.details,
      }
  }
  return { ...base, kind: 'error', message: error.message }
}
