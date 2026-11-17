import { z } from 'zod'
import { queryString } from '@/features/catalogue/api/catalogue'
import { apiFetchBlob, apiRequest, apiUpload, buildUrl } from '@/lib/api/client'

/**
 * Opening stock import PREVIEW (G-06; docs/openapi `/opening-stock-imports`,
 * API_CONVENTIONS "Opening stock import preview", ADR-030). The server stores
 * the uploaded CSV privately and validates every row; nothing here creates,
 * changes or commits stock. There is no commit call here: approving and
 * committing a preview is the separate G-07 workflow (./commit.ts).
 */

const stamp = z.iso.datetime({ offset: true })
const count = z.number().int().min(0)

export const rowStatusSchema = z.enum(['VALID', 'WARNING', 'ERROR', 'BLOCKED_UNVERIFIED_MAPPING'])
export type RowStatus = z.infer<typeof rowStatusSchema>

export const issueSchema = z.object({
  field: z.string(),
  code: z.string(),
  severity: z.enum(['ERROR', 'BLOCKED', 'WARNING']),
  message: z.string(),
  suggestion: z.string().optional(),
})
export type RowIssue = z.infer<typeof issueSchema>

export const fileIssueSchema = z.object({
  code: z.string(),
  line: z.number().int().min(0),
  column: z.string(),
  message: z.string(),
})
export type FileIssue = z.infer<typeof fileIssueSchema>

export const columnSchema = z.object({
  name: z.string(),
  required: z.boolean(),
  maxLength: z.number().int().positive(),
  format: z.string(),
  description: z.string(),
})
export type TemplateColumn = z.infer<typeof columnSchema>

export const specificationSchema = z.object({
  templateVersion: z.string(),
  format: z.string(),
  firstLine: z.string(),
  maxBytes: z.number().int().positive(),
  maxRows: z.number().int().positive(),
  previewTtlHours: z.number().int().positive(),
  columns: z.array(columnSchema),
  stockStatuses: z.array(z.string()),
  /** Statuses a commit accepts (OQ-47, G-07). Empty: the server refuses every commit. */
  approvedOpeningStatuses: z.array(z.string()),
  businessDate: z.iso.date(),
  timeZone: z.string(),
  notes: z.array(z.string()),
})
export type Specification = z.infer<typeof specificationSchema>

export const summarySchema = z
  .object({
    totalRows: count,
    validRows: count,
    warningRows: count,
    errorRows: count,
    blockedRows: count,
    canProceedRows: count,
    duplicateRows: count,
    unresolvedRows: count,
    skippedBlankRows: count,
  })
  .refine((s) => s.totalRows === s.validRows + s.warningRows + s.errorRows + s.blockedRows, {
    message: 'row counts do not add up',
    path: ['totalRows'],
  })
export type ImportSummary = z.infer<typeof summarySchema>

/** The recorded physical-count approval of a batch (G-07; set by the server only). */
const batchApprovalSchema = z.object({
  id: z.string(),
  approvedById: z.string(),
  approvedByName: z.string().nullable(),
  approvedAt: stamp,
  physicalCountReference: z.string(),
  physicalCountDate: z.iso.date(),
  mappingsReviewed: z.boolean(),
  duplicatesReviewed: z.boolean(),
  receivedDatesApproved: z.boolean(),
  note: z.string().nullable(),
})
export type ImportApproval = z.infer<typeof batchApprovalSchema>
export { batchApprovalSchema as approvalSchema }

/** The batch's commit as shown on the batch (G-07); once set the batch is committed for good. */
const batchCommittedSchema = z.object({
  id: z.string(),
  reference: z.string(),
  committedById: z.string(),
  committedByName: z.string().nullable(),
  committedAt: stamp,
  unitsCreated: z.number().int().positive(),
})

export const batchSchema = z
  .object({
    id: z.string(),
    reference: z.string(),
    status: z.enum(['PREVIEWED', 'REJECTED']),
    templateVersion: z.string().nullable(),
    fileName: z.string(),
    fileSha256: z.string().regex(/^[0-9a-f]{64}$/),
    fileSizeBytes: z.number().int().positive(),
    fileAttachmentId: z.string(),
    uploadedById: z.string(),
    uploadedByName: z.string().nullable(),
    uploadedAt: stamp,
    businessDate: z.iso.date(),
    expiresAt: stamp,
    expired: z.boolean(),
    summary: summarySchema,
    fileIssues: z.array(fileIssueSchema),
    // A preview never offers a commit (G-07 is a separate, authorised workflow).
    commitAvailable: z.literal(false),
    approval: batchApprovalSchema.nullable(),
    commit: batchCommittedSchema.nullable(),
  })
  .refine((b) => b.status !== 'REJECTED' || b.summary.totalRows === 0, {
    message: 'a rejected file has no rows',
    path: ['summary'],
  })
export type ImportBatch = z.infer<typeof batchSchema>

const nullableId = z.string().nullable()
export const rowSchema = z.object({
  rowNumber: z.number().int().min(3),
  status: rowStatusSchema,
  canProceed: z.boolean(),
  duplicate: z.boolean(),
  unresolved: z.boolean(),
  source: z.record(z.string(), z.string()),
  normalised: z.object({
    brandId: nullableId,
    modelId: nullableId,
    variantId: nullableId,
    colourId: nullableId,
    locationId: nullableId,
    supplierId: nullableId,
    chassisNumber: nullableId,
    chassisNormalised: nullableId,
    motorNormalised: nullableId,
    batteryNormalised: nullableId,
    chargerNormalised: nullableId,
    stockStatus: nullableId,
    receivedDate: z.iso.date().nullable(),
    acquisitionCost: nullableId,
  }),
  resolved: z.record(z.string(), z.unknown()),
  issues: z.array(issueSchema),
})
export type ImportRow = z.infer<typeof rowSchema>

const pageOf = <T extends z.ZodType>(item: T) =>
  z.object({
    items: z.array(item),
    page: z.number().int().positive(),
    limit: z.number().int().positive(),
    total: count,
  })

export interface RowParams {
  status?: RowStatus
  duplicate?: boolean
  unresolved?: boolean
  page?: number
  limit?: number
}

const base = '/opening-stock-imports'

export function getSpecification(signal?: AbortSignal): Promise<Specification> {
  return apiRequest(`${base}/specification`, specificationSchema, { signal })
}

/** The template file (marker + header, no data rows). */
export function downloadTemplate(signal?: AbortSignal): Promise<Blob> {
  return apiFetchBlob(buildUrl(`${base}/template`), signal)
}

/**
 * POST the filled template. The same idempotencyKey with the same file returns
 * the first preview (safe retry after a lost response); a new file needs a new key.
 */
export function uploadImport(
  file: File,
  idempotencyKey: string,
  options: { signal?: AbortSignal; onProgress?: (fraction: number) => void } = {},
): Promise<ImportBatch> {
  const form = new FormData()
  form.append('file', file, file.name)
  return apiUpload(base, batchSchema, form, {
    ...options,
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}

export function listImports(params: { page?: number; limit?: number } = {}, signal?: AbortSignal) {
  return apiRequest(`${base}${queryString(params)}`, pageOf(batchSchema), { signal })
}

export function getImport(id: string, signal?: AbortSignal): Promise<ImportBatch> {
  return apiRequest(`${base}/${encodeURIComponent(id)}`, batchSchema, { signal })
}

export function listImportRows(id: string, params: RowParams = {}, signal?: AbortSignal) {
  return apiRequest(
    `${base}/${encodeURIComponent(id)}/rows${queryString(params)}`,
    pageOf(rowSchema),
    { signal },
  )
}
