import type { BadgeTone } from '@/components/ui/Badge'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import type { RowStatus } from '../api/openingStock'

export const rowStatusLabels: Record<RowStatus, string> = {
  VALID: 'Valid',
  WARNING: 'Warning',
  ERROR: 'Error',
  BLOCKED_UNVERIFIED_MAPPING: 'Blocked — unverified catalogue',
}

export const rowStatusTones: Record<RowStatus, BadgeTone> = {
  VALID: 'success',
  WARNING: 'warning',
  ERROR: 'danger',
  BLOCKED_UNVERIFIED_MAPPING: 'info',
}

export const severityTones = { ERROR: 'danger', BLOCKED: 'info', WARNING: 'warning' } as const

export const PREVIEW_ONLY_TEXT =
  'Preview completed. No inventory records have been created or modified.'

export function formatBytes(n: number): string {
  if (n < 1024) return `${String(n)} B`
  if (n < 1 << 20) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1 << 20)).toFixed(1)} MB`
}

export interface UploadFailure {
  kind:
    'type' | 'size' | 'invalid' | 'forbidden' | 'signed-out' | 'key-reused' | 'offline' | 'error'
  message: string
  retrySame: boolean
  requestId?: string
}

/** Maps a failed upload. A lost response may be retried with the same key: it never creates a second preview. */
export function describeUploadError(error: unknown): UploadFailure {
  if (!(error instanceof ApiError))
    return { kind: 'error', message: 'Something went wrong. Try again.', retrySame: true }
  const base = { requestId: error.requestId }
  if (error.code === ClientErrorCode.Network)
    return {
      ...base,
      kind: 'offline',
      message:
        'Cannot reach the server. Check your connection and try again — the same upload will not be counted twice.',
      retrySame: true,
    }
  switch (error.status) {
    case 401:
      return {
        ...base,
        kind: 'signed-out',
        message: 'Your session has ended. Sign in again.',
        retrySame: false,
      }
    case 403:
      return {
        ...base,
        kind: 'forbidden',
        message: 'You do not have permission to upload opening stock files.',
        retrySame: false,
      }
    case 409:
      return { ...base, kind: 'key-reused', message: error.message, retrySame: false }
    case 413:
      return { ...base, kind: 'size', message: error.message, retrySame: false }
    case 415:
      return { ...base, kind: 'type', message: error.message, retrySame: false }
    case 422:
      return { ...base, kind: 'invalid', message: error.message, retrySame: false }
  }
  return { ...base, kind: 'error', message: error.message, retrySame: error.status >= 500 }
}

// ── H-08 reconciliation preflight ─────────────────────────────────────────

export const preflightCategoryLabels = {
  WARNING: 'Warning — not accepted',
  BLOCKING_ERROR: 'Blocking error',
  UNVERIFIED: 'Unverified catalogue',
  MANUAL_RECONCILIATION: 'Needs manual reconciliation',
} as const

export const preflightCategoryTones = {
  WARNING: 'warning',
  BLOCKING_ERROR: 'danger',
  UNVERIFIED: 'info',
  MANUAL_RECONCILIATION: 'warning',
} as const satisfies Record<keyof typeof preflightCategoryLabels, BadgeTone>

export const PREFLIGHT_NO_CHANGE_TEXT =
  'Read-only check. No inventory records were created or changed, and nothing was approved or committed.'
