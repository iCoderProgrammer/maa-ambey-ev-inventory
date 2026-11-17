import { z } from 'zod'
import { ApiError, apiRequest } from '@/lib/api/client'
import { approvalSchema, type ImportApproval } from './openingStock'

/**
 * Opening stock import APPROVAL and COMMIT (G-07; docs/openapi
 * `/opening-stock-imports/{id}/approval|commit`, API_CONVENTIONS "Opening stock
 * import commit", ADR-031). The server decides everything: it re-reads the file,
 * re-validates every line and creates the units, history, movements and audit in
 * one transaction, once. Nothing here computes eligibility or counts.
 */

const stamp = z.iso.datetime({ offset: true })

/** What the approver records; every confirmation is typed by the user. */
export interface ApprovalInput {
  physicalCountReference: string
  physicalCountDate: string
  mappingsReviewed: boolean
  duplicatesReviewed: boolean
  receivedDatesApproved: boolean
  note: string | null
}

export const committedUnitSchema = z.object({
  rowNumber: z.number().int().min(3),
  vehicleUnitId: z.string(),
  unitNumber: z.string(),
  status: z.string(),
  movementId: z.string(),
})
export type CommittedUnit = z.infer<typeof committedUnitSchema>

export const commitResultSchema = z
  .object({
    commitId: z.string(),
    reference: z.string(),
    batchId: z.string(),
    batchReference: z.string(),
    batchStatus: z.literal('COMMITTED'),
    unitsCreated: z.number().int().positive(),
    committedById: z.string(),
    committedByName: z.string().nullable(),
    committedAt: stamp,
    approvalId: z.string(),
    fileSha256: z.string().regex(/^[0-9a-f]{64}$/),
    templateVersion: z.string(),
    sourceType: z.literal('OPENING_STOCK'),
    movementType: z.literal('OPENING_BALANCE'),
    units: z.array(committedUnitSchema),
    replayed: z.boolean(),
  })
  .refine((c) => c.units.length === c.unitsCreated, {
    message: 'units do not match unitsCreated',
    path: ['units'],
  })
export type CommitResult = z.infer<typeof commitResultSchema>

const base = (id: string) => `/opening-stock-imports/${encodeURIComponent(id)}`

/** Records the batch's physical-count approval (stock.import.approve). Never sent automatically. */
export function approveImport(id: string, input: ApprovalInput): Promise<ImportApproval> {
  return apiRequest(`${base(id)}/approval`, approvalSchema, { method: 'POST', body: input })
}

/**
 * Commits the batch (stock.import.commit). The key belongs to this commit
 * attempt: a retry after an unknown outcome MUST reuse it (the server then
 * returns the first result with replayed: true and creates nothing new).
 */
export function commitImport(id: string, idempotencyKey: string): Promise<CommitResult> {
  return apiRequest(`${base(id)}/commit`, commitResultSchema, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}

/** The batch's commit, or null while it has none (404). Used to recover an unknown outcome. */
export async function getImportCommit(
  id: string,
  signal?: AbortSignal,
): Promise<CommitResult | null> {
  try {
    return await apiRequest(`${base(id)}/commit`, commitResultSchema, { signal })
  } catch (e) {
    if (e instanceof ApiError && e.status === 404 && e.code === 'NOT_FOUND') return null
    throw e
  }
}
