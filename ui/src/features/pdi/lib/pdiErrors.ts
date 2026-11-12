import { ApiError, ClientErrorCode } from '@/lib/api/client'

export type PdiFailure =
  | 'version'
  | 'state'
  | 'not-configured'
  | 'key-reused'
  | 'duplicate'
  | 'signed-out'
  | 'forbidden'
  | 'not-found'
  | 'invalid-input'
  | 'uncertain'
  | 'error'

export interface PdiError {
  kind: PdiFailure
  message: string
  /** Server field errors (422 VALIDATION_FAILED, 409 DUPLICATE). */
  fields: { field: string; message: string }[]
  requestId?: string
  /**
   * Unknown outcome of a keyed request (recording an inspection): sending the
   * same action again (same Idempotency-Key) records it once or returns what was recorded.
   */
  retrySame: boolean
  /** The unit (or checklist) changed on the server: the form keeps its entries; reload before retrying. */
  reload: boolean
}

/**
 * Maps a failed PDI request to what the screen shows. `keyed`: the request
 * carried an Idempotency-Key. Never claims success; a refusal says nothing was saved.
 */
export function describePdiError(error: unknown, keyed: boolean): PdiError {
  const none = { fields: [], retrySame: false, reload: false }
  if (!(error instanceof ApiError))
    return { ...none, kind: 'error', message: 'Something went wrong. Nothing was saved.' }
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
            'The server did not confirm this inspection. It may or may not have been recorded: send it again unchanged to find out — it will not be recorded twice.',
        }
      : {
          ...base,
          kind: 'uncertain',
          message:
            'The server did not confirm this. It may or may not have been saved: reload and check before trying again.',
        }
  }
  if (error.code === ClientErrorCode.InvalidResponse) {
    return {
      ...base,
      kind: 'uncertain',
      reload: true,
      message:
        'The server answered, but not as expected. The vehicle has been reloaded: check its status and history before doing anything else.',
    }
  }
  switch (error.code) {
    case 'VERSION_CONFLICT':
      return {
        ...base,
        kind: 'version',
        reload: true,
        message:
          'This vehicle (or checklist item) was changed by someone else — for example another inspection was recorded. It has been reloaded: check it before trying again. Nothing was saved.',
      }
    case 'INVALID_STATE':
      return { ...base, kind: 'state', reload: true, message: error.message }
    case 'PDI_CHECKLIST_NOT_CONFIGURED':
      return {
        ...base,
        kind: 'not-configured',
        reload: true,
        message:
          'No PDI checklist applies to this vehicle yet, so it cannot pass PDI. Someone with checklist permission must configure it first. Nothing was saved.',
      }
    case 'IDEMPOTENCY_KEY_REUSED':
      return {
        ...base,
        kind: 'key-reused',
        message:
          'This request could not be matched to the earlier attempt. Nothing was saved. Close the confirmation and submit again.',
      }
    case 'DUPLICATE':
      return { ...base, kind: 'duplicate', message: error.message, fields: error.details }
  }
  switch (error.status) {
    case 401:
      return { ...base, kind: 'signed-out', message: 'Your session has ended. Sign in again.' }
    case 403:
      return {
        ...base,
        kind: 'forbidden',
        message: 'You do not have permission to do this. Nothing was saved.',
      }
    case 404:
      return {
        ...base,
        kind: 'not-found',
        message: 'This vehicle or record was not found, or it is not available to you.',
      }
    case 422:
      return {
        ...base,
        kind: 'invalid-input',
        message: 'The server refused some entries. Check the highlighted items. Nothing was saved.',
        fields: error.details,
      }
  }
  return { ...base, kind: 'error', message: error.message }
}

/** A readable explanation for a server field path (items[2].note → "Item 3 note"). */
export function fieldLabel(field: string, itemLabels: string[]): string {
  const item = /^items\[(\d+)\]\.(\w+)$/.exec(field)
  if (item) {
    const name = itemLabels[Number(item[1])] ?? `Item ${String(Number(item[1]) + 1)}`
    const part = item[2] === 'note' ? 'note' : item[2] === 'outcome' ? 'outcome' : 'item'
    return `${name} — ${part}`
  }
  const att = /^attachmentIds\[(\d+)\]$/.exec(field)
  if (att) return `Evidence file ${String(Number(att[1]) + 1)}`
  const names: Record<string, string> = {
    items: 'Checklist',
    notes: 'Defect notes',
    attachmentIds: 'Evidence',
    disposition: 'Disposition',
    version: 'Vehicle version',
    'Idempotency-Key': 'Request key',
  }
  return names[field] ?? field
}
