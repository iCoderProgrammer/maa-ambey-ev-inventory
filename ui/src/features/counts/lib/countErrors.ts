import { ApiError, ClientErrorCode } from '@/lib/api/client'

export type CountFailure =
  | 'stale'
  | 'self'
  | 'version'
  | 'state'
  | 'serial'
  | 'key-reused'
  | 'signed-out'
  | 'forbidden'
  | 'not-found'
  | 'invalid-input'
  | 'uncertain'
  | 'error'

export interface CountError {
  kind: CountFailure
  message: string
  /** Server field errors (422). */
  fields: { field: string; message: string }[]
  /** STOCK_CHANGED_SINCE_COUNT: the lines whose stock moved after they were counted. */
  staleLineIds: string[]
  requestId?: string
  /**
   * Unknown outcome of a keyed request (create/approve/reject): sending the same
   * action again (same Idempotency-Key) either does it once or returns what was done.
   */
  retrySame: boolean
}

/**
 * Maps a failed count request to what the screen shows. `keyed`: the request
 * carried an Idempotency-Key, so an unknown outcome can be resolved by sending
 * it again unchanged; otherwise the count must be reloaded and checked.
 */
export function describeCountError(error: unknown, keyed: boolean): CountError {
  const none = { fields: [], staleLineIds: [], retrySame: false }
  if (!(error instanceof ApiError))
    return { ...none, kind: 'error', message: 'Something went wrong. Nothing was changed.' }
  const base = { ...none, requestId: error.requestId }
  if (
    error.code === ClientErrorCode.Network ||
    error.status >= 500 ||
    error.code === 'REQUEST_TIMEOUT'
  ) {
    return keyed
      ? {
          ...base,
          kind: 'uncertain',
          retrySame: true,
          message:
            'The server did not confirm this. It may or may not have been done: send it again unchanged to find out — it will not be done twice.',
        }
      : {
          ...base,
          kind: 'uncertain',
          message:
            'The server did not confirm this. It may or may not have been saved: the count has been reloaded — check it before trying again.',
        }
  }
  if (error.code === ClientErrorCode.InvalidResponse) {
    return {
      ...base,
      kind: 'uncertain',
      message:
        'The server answered, but not as expected. The count has been reloaded: check its status before doing anything else.',
    }
  }
  switch (error.code) {
    case 'STOCK_CHANGED_SINCE_COUNT':
      return {
        ...base,
        kind: 'stale',
        message:
          'Stock moved after some lines were counted, so the count no longer matches the stock record. Nothing was changed.',
        staleLineIds: error.details
          .filter((d) => d.field.startsWith('lines.'))
          .map((d) => d.field.slice('lines.'.length)),
      }
    case 'SELF_APPROVAL_NOT_ALLOWED':
      return {
        ...base,
        kind: 'self',
        message:
          'You took part in this count (created, counted or submitted it), so you cannot decide it. Someone else with approval permission must. Nothing was changed.',
      }
    case 'VERSION_CONFLICT':
      return {
        ...base,
        kind: 'version',
        message:
          'This count was changed by someone else. It has been reloaded: check it and try again. Nothing was changed.',
      }
    case 'INVALID_STATE':
      return { ...base, kind: 'state', message: error.message }
    case 'TRACKING_NOT_SUPPORTED':
      return {
        ...base,
        kind: 'serial',
        message:
          'Serial-tracked items cannot be counted by quantity yet: there is no serial register. Nothing was changed.',
      }
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
        message: 'This count was not found, or it is not available to you.',
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
