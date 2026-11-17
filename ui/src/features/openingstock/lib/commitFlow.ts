import { ApiError, ClientErrorCode } from '@/lib/api/client'
import type { ImportBatch } from '../api/openingStock'

/**
 * G-07 presentation logic. Every value comes from the server; these helpers only
 * explain it. The server re-checks everything at approval and at commit.
 */

export interface Blocker {
  key: string
  text: string
}

/**
 * Why the server would refuse this batch as it stands (from its own summary,
 * expiry and status). Shown so the user knows what to correct; it never enables
 * anything the server would not accept. approvedStatuses: null when not known to
 * this user (the specification needs stock.import.preview).
 */
export function batchBlockers(b: ImportBatch, approvedStatuses: string[] | null): Blocker[] {
  const s = b.summary
  const out: Blocker[] = []
  if (b.status === 'REJECTED')
    out.push({
      key: 'rejected',
      text: 'The file could not be read as the template. Upload a corrected file.',
    })
  if (b.templateVersion !== null && b.templateVersion !== 'OS-IMPORT-1' && b.status !== 'REJECTED')
    out.push({
      key: 'template',
      text: `Template ${b.templateVersion} is not the current template.`,
    })
  if (b.expired)
    out.push({
      key: 'expired',
      text: 'This preview has expired. Upload the file again for a new preview.',
    })
  if (b.status === 'PREVIEWED' && s.totalRows === 0)
    out.push({ key: 'empty', text: 'The preview has no lines.' })
  if (s.errorRows > 0)
    out.push({
      key: 'errors',
      text: `${String(s.errorRows)} line(s) have errors (duplicates, unknown catalogue or location, invalid values). Correct them in the file and upload it again.`,
    })
  if (s.blockedRows > 0)
    out.push({
      key: 'blocked',
      text: `${String(s.blockedRows)} line(s) map to a brand, model, variant or colour that is not verified and active. They cannot become stock until the catalogue is approved.`,
    })
  if (s.warningRows > 0)
    out.push({
      key: 'warnings',
      text: `${String(s.warningRows)} line(s) have warnings. No approved rule accepts warnings at commit (OQ-49): correct them and upload the file again.`,
    })
  if (approvedStatuses !== null && approvedStatuses.length === 0)
    out.push({
      key: 'statuses',
      text: 'No stock status is approved for opening stock yet (OQ-47), so the server refuses every commit.',
    })
  return out
}

const unknownOutcome = (e: ApiError) =>
  e.code === ClientErrorCode.Network ||
  e.code === ClientErrorCode.InvalidResponse ||
  e.status >= 500

export type CommitFailure =
  /** The request may or may not have committed: ask GET …/commit before anything else. */
  | { kind: 'unknown'; message: string }
  /** Committed by another request or user: show that commit. */
  | { kind: 'already-committed'; message: string }
  /** The key already committed another batch: a new attempt needs a new key. */
  | { kind: 'key-reused'; message: string }
  | { kind: 'rows-blocked'; message: string; lines: { field: string; message: string }[] }
  | { kind: 'forbidden' | 'signed-out' | 'not-found' | 'refused'; message: string; code?: string }

/** Guidance shown with each refusal code (the server's message is shown too). */
export const commitGuidance: Record<string, string> = {
  IMPORT_APPROVAL_REQUIRED: 'Record the physical-count approval first.',
  IMPORT_PREVIEW_EXPIRED: 'Upload the file again; the new preview needs its own approval.',
  OPENING_STATUSES_NOT_APPROVED:
    'The owner has not approved any opening-stock status (OQ-47). Nothing can be committed until then.',
  IMPORT_SOURCE_INVALID:
    'The stored file no longer matches its preview. Upload the original file again.',
  INVALID_STATE: 'This batch cannot be committed. Upload a corrected file.',
  DUPLICATE_IDENTIFIER:
    'A chassis/frame number was registered by someone else meanwhile. Correct the file and upload it again.',
}

export function describeCommitError(e: unknown): CommitFailure {
  if (!(e instanceof ApiError))
    return { kind: 'unknown', message: 'The outcome of the commit is not known yet.' }
  if (unknownOutcome(e))
    return {
      kind: 'unknown',
      message:
        'The server did not confirm the commit (connection lost or server error). Its outcome is not known yet.',
    }
  switch (e.code) {
    case 'IMPORT_ALREADY_COMMITTED':
      return { kind: 'already-committed', message: e.message }
    case 'IDEMPOTENCY_KEY_REUSED':
      return { kind: 'key-reused', message: e.message }
    case 'IMPORT_ROWS_BLOCKED':
      return { kind: 'rows-blocked', message: e.message, lines: e.details }
  }
  if (e.status === 401)
    return {
      kind: 'signed-out',
      message: 'Your session has ended. Sign in again, then reopen this batch.',
    }
  if (e.status === 403)
    return {
      kind: 'forbidden',
      message:
        'You do not have permission to commit this batch (stock.import.commit, and every line’s location in your scope).',
    }
  if (e.status === 404) return { kind: 'not-found', message: 'This import batch was not found.' }
  return { kind: 'refused', message: e.message, code: e.code }
}

/** Splits "rows[5].chassis_number" into its line and column. */
export function lineOf(field: string): { line: number | null; column: string | null } {
  const m = /^rows\[(\d+)\](?:\.(.+))?$/.exec(field)
  return m ? { line: Number(m[1]), column: m[2] ?? null } : { line: null, column: null }
}

/** A new key for one commit attempt (kept for every retry of that attempt). */
export function newCommitKey(): string {
  return `osc-${crypto.randomUUID()}`
}
