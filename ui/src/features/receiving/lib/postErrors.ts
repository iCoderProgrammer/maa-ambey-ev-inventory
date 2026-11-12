import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { VehicleErrorCode } from '@/features/vehicles/lib/identifiers'
import { fieldLabel } from './labels'

export type PostFailure =
  | 'blocked'
  | 'duplicate'
  | 'version'
  | 'state'
  | 'key-reused'
  | 'location'
  | 'signed-out'
  | 'forbidden'
  | 'not-found'
  | 'uncertain'
  | 'error'

export interface PostError {
  kind: PostFailure
  message: string
  /** What the server said blocks the posting, labelled for people. */
  blockers: { key: string; message: string }[]
  requestId?: string
  /**
   * Unknown outcome (network failure, timeout, 5xx, unexpected answer): the
   * receipt is re-read first; sending the same action again is safe (same key).
   */
  uncertain: boolean
}

/**
 * Maps a failed posting to what the dialog shows (API_CONVENTIONS "Posting a
 * goods receipt"). Never claims success; every refusal says nothing changed.
 */
export function describePostError(error: unknown, lineNos?: number[]): PostError {
  const none = { blockers: [], uncertain: false }
  if (!(error instanceof ApiError))
    return { ...none, kind: 'error', message: 'Something went wrong. Nothing was posted.' }
  const base = { ...none, requestId: error.requestId }
  const list = () =>
    error.details.map((d) => ({ key: fieldLabel(d.field, lineNos), message: d.message }))
  if (
    error.code === ClientErrorCode.Network ||
    error.code === ClientErrorCode.InvalidResponse ||
    error.code === 'REQUEST_TIMEOUT' ||
    error.status >= 500
  ) {
    return {
      ...base,
      kind: 'uncertain',
      uncertain: true,
      message:
        'The server did not confirm the posting. It may or may not have been posted. The receipt has been checked again below; if it is still a draft, send the same request again — it will never be posted twice.',
    }
  }
  switch (error.code) {
    case 'POSTING_BLOCKED':
      return {
        ...base,
        kind: 'blocked',
        blockers: list(),
        message:
          'This receipt cannot be posted yet. Nothing was changed. Fix these and save the draft first:',
      }
    case 'DUPLICATE':
      return {
        ...base,
        kind: 'duplicate',
        blockers: list(),
        message:
          'Some chassis/frame numbers are already on another goods receipt. Nothing was posted:',
      }
    case VehicleErrorCode.DuplicateIdentifier:
      return {
        ...base,
        kind: 'duplicate',
        blockers: list().map((b) => ({ ...b, message: 'already registered to another vehicle' })),
        message:
          'A chassis/frame number is already registered to another vehicle. Nothing was posted:',
      }
    case 'VERSION_CONFLICT':
      return {
        ...base,
        kind: 'version',
        message:
          'This receipt was changed by someone else. It has been reloaded: review it and confirm again. Nothing was posted.',
      }
    case 'INVALID_STATE':
      return { ...base, kind: 'state', message: `${error.message} The receipt has been reloaded.` }
    case 'IDEMPOTENCY_KEY_REUSED':
      return {
        ...base,
        kind: 'key-reused',
        message:
          'This request could not be matched to the earlier attempt. Nothing was posted. The receipt has been reloaded: check it and confirm again.',
      }
  }
  switch (error.status) {
    case 401:
      return {
        ...base,
        kind: 'signed-out',
        message: 'Your session has ended. Sign in again. Nothing was posted.',
      }
    case 403:
      return {
        ...base,
        kind: 'forbidden',
        message: 'You do not have permission to post goods receipts. Nothing was posted.',
      }
    case 404:
      return {
        ...base,
        kind: 'not-found',
        message: 'This goods receipt was not found, or it is not available to you.',
      }
    case 422:
      return {
        ...base,
        kind: error.details.some((d) => d.field === 'locationId') ? 'location' : 'error',
        blockers: error.details.map((d) => ({
          key: d.field === 'locationId' ? 'Receiving location' : d.field,
          message: d.message,
        })),
        message: 'The server refused the request. Nothing was posted:',
      }
  }
  return { ...base, kind: 'error', message: error.message }
}
