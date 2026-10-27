import { ApiError, ClientErrorCode } from '@/lib/api/client'

export type DisplayMoveFailure =
  | 'signed-out'
  | 'forbidden'
  | 'not-found'
  | 'stale-version'
  | 'invalid-state'
  | 'key-reused'
  | 'invalid-input'
  | 'uncertain'
  | 'error'

export interface DisplayMoveError {
  kind: DisplayMoveFailure
  message: string
  /** Server field errors (version, note, Idempotency-Key). */
  fields: { field: string; message: string }[]
  requestId?: string
  /**
   * Whether sending the same action again (same Idempotency-Key and body) is the
   * right next step: the outcome is unknown, and a repeat either applies it once
   * or returns the movement already recorded.
   */
  retrySameAction: boolean
  /** Whether the user must reload the unit and decide again (a new action and key). */
  reload: boolean
}

/**
 * Maps a failed display move to what the caller should show and do. A 404
 * never says whether the unit exists elsewhere (out-of-scope units answer 404
 * too). Nothing here claims the move failed when the outcome is unknown.
 */
export function describeDisplayMoveError(error: unknown): DisplayMoveError {
  const none = { fields: [] as DisplayMoveError['fields'], retrySameAction: false, reload: false }
  if (!(error instanceof ApiError)) {
    return { ...none, kind: 'error', message: 'Something went wrong. Nothing was changed.' }
  }
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
        'The server did not confirm the change. It may or may not have been made: try the same action again to find out.',
      retrySameAction: true,
    }
  }
  if (error.code === ClientErrorCode.InvalidResponse) {
    return {
      ...base,
      kind: 'uncertain',
      message:
        'The server answered, but not as expected. Reload the vehicle unit to see its status.',
      reload: true,
    }
  }
  switch (error.code) {
    case 'VERSION_CONFLICT':
      return {
        ...base,
        kind: 'stale-version',
        message: 'This vehicle unit was changed by someone else. Reload it and try again.',
        reload: true,
      }
    case 'INVALID_STATE':
      return {
        ...base,
        kind: 'invalid-state',
        message: error.message,
        reload: true,
      }
    case 'IDEMPOTENCY_KEY_REUSED':
      return {
        ...base,
        kind: 'key-reused',
        message: 'This request could not be matched to the earlier one. Start the action again.',
        reload: true,
      }
  }
  switch (error.status) {
    case 401:
      return { ...base, kind: 'signed-out', message: 'Your session has ended. Sign in again.' }
    case 403:
      return {
        ...base,
        kind: 'forbidden',
        message: 'You do not have permission to change the display status of vehicle units.',
      }
    case 404:
      return {
        ...base,
        kind: 'not-found',
        message: 'This vehicle unit was not found, or it is not available to you.',
      }
    case 422:
      return {
        ...base,
        kind: 'invalid-input',
        message: 'Check the highlighted fields.',
        fields: error.details,
      }
  }
  return { ...base, kind: 'error', message: error.message }
}
