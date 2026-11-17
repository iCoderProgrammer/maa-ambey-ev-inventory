import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { approvalSchema, issueSchema, rowStatusSchema, summarySchema } from './openingStock'

/**
 * Opening stock RECONCILIATION PREFLIGHT (H-08; docs/openapi
 * `GET /opening-stock-imports/{id}/preflight`, ADR-032). Read only and safe to
 * repeat: the server re-checks the batch exactly as a commit would and reports
 * the result. It never approves, commits or changes anything, and this module
 * has no write call. Counts and categories come from the server only.
 */

const stamp = z.iso.datetime({ offset: true })
const count = z.number().int().min(0)

export const preflightCategorySchema = z.enum([
  'WARNING',
  'BLOCKING_ERROR',
  'UNVERIFIED',
  'MANUAL_RECONCILIATION',
])
export type PreflightCategory = z.infer<typeof preflightCategorySchema>

export const preflightSourceSchema = z.object({
  status: z.enum(['VERIFIED', 'INVALID', 'NOT_CHECKED']),
  fileAvailable: z.boolean(),
  checksumMatches: z.boolean(),
  sizeMatches: z.boolean(),
  templateCompatible: z.boolean(),
  matchesStagedRows: z.boolean(),
  message: z.string(),
})

export const preflightSummarySchema = z
  .object({
    totalRows: count,
    validRows: count,
    warningRows: count,
    blockingErrorRows: count,
    unverifiedRows: count,
    manualReconciliationRows: count,
    duplicateRows: count,
    missingChassisRows: count,
    changedSincePreviewRows: count,
  })
  .refine(
    (s) =>
      s.totalRows ===
      s.validRows +
        s.warningRows +
        s.blockingErrorRows +
        s.unverifiedRows +
        s.manualReconciliationRows,
    { message: 'row counts do not add up', path: ['totalRows'] },
  )
export type PreflightSummary = z.infer<typeof preflightSummarySchema>

export const preflightIssueCountSchema = z.object({
  code: z.string(),
  severity: z.enum(['ERROR', 'BLOCKED', 'WARNING']),
  rows: z.number().int().positive(),
})

export const preflightRowSchema = z.object({
  rowNumber: z.number().int().min(3),
  category: preflightCategorySchema,
  previewStatus: rowStatusSchema,
  currentStatus: rowStatusSchema,
  changedSincePreview: z.boolean(),
  source: z.record(z.string(), z.string()),
  issues: z.array(issueSchema),
})
export type PreflightRow = z.infer<typeof preflightRowSchema>

export const preflightQuantitySchema = z.object({
  locationCode: z.string(),
  brand: z.string(),
  modelCode: z.string(),
  variantCode: z.string(),
  stockStatus: z.string(),
  fileRows: z.number().int().positive(),
  validRows: count,
  // The system holds no counted quantities (H-08): never filled in by the client.
  physicalCount: z.number().int().min(0).nullable(),
  difference: z.number().int().nullable(),
})
export type PreflightQuantity = z.infer<typeof preflightQuantitySchema>

export const preflightBlockerSchema = z.object({
  code: z.string(),
  message: z.string(),
  decision: z.string(),
})
export type PreflightBlocker = z.infer<typeof preflightBlockerSchema>

const committedSchema = z.object({
  id: z.string(),
  reference: z.string(),
  committedById: z.string(),
  committedByName: z.string().nullable(),
  committedAt: stamp,
  unitsCreated: z.number().int().positive(),
})

export const preflightSchema = z
  .object({
    batchId: z.string(),
    batchReference: z.string(),
    batchStatus: z.enum(['PREVIEWED', 'REJECTED']),
    fileName: z.string(),
    fileSha256: z.string().regex(/^[0-9a-f]{64}$/),
    fileSizeBytes: z.number().int().positive(),
    templateVersion: z.string().nullable(),
    uploadedByName: z.string().nullable(),
    uploadedAt: stamp,
    expiresAt: stamp,
    expired: z.boolean(),
    checkedAt: stamp,
    businessDate: z.iso.date(),
    // The server states that nothing was written; anything else is refused.
    nonMutating: z.literal(true),
    rowsChecked: z.boolean(),
    source: preflightSourceSchema,
    summary: preflightSummarySchema,
    previewSummary: summarySchema,
    issueCounts: z.array(preflightIssueCountSchema),
    quantities: z.array(preflightQuantitySchema),
    physicalCount: z.object({
      status: z.enum(['NOT_RECORDED', 'REFERENCE_ONLY']),
      reference: z.string().nullable(),
      countDate: z.iso.date().nullable(),
      message: z.string(),
    }),
    approvedOpeningStatuses: z.array(z.string()),
    approval: approvalSchema.nullable(),
    commit: committedSchema.nullable(),
    blockers: z.array(preflightBlockerSchema),
    exceptions: z.array(preflightRowSchema),
  })
  .refine((p) => p.rowsChecked || p.summary.totalRows === 0, {
    message: 'rows counted without being checked',
    path: ['summary'],
  })
  .refine((p) => p.exceptions.length === p.summary.totalRows - p.summary.validRows, {
    message: 'exceptions do not match the summary',
    path: ['exceptions'],
  })
export type Preflight = z.infer<typeof preflightSchema>

/** Runs the read-only preflight of a batch (stock.import.preview, approve or commit). */
export function getPreflight(id: string, signal?: AbortSignal): Promise<Preflight> {
  return apiRequest(`/opening-stock-imports/${encodeURIComponent(id)}/preflight`, preflightSchema, {
    signal,
  })
}

/** Neutralises spreadsheet formulas (=, +, -, @, tab, CR) and quotes a CSV cell. */
export function csvCell(v: string | number): string {
  let s = String(v)
  if (/^[=+\-@\t\r]/.test(s)) s = `'${s}`
  return /[",\n\r]/.test(s) ? `"${s.replaceAll('"', '""')}"` : s
}

/**
 * The exceptions report as CSV, built only from the server's answer: one line
 * per issue of every non-valid row. Nothing is computed or corrected here.
 */
export function exceptionsCsv(p: Preflight): string {
  const head = [
    'batch_reference',
    'line',
    'category',
    'preview_status',
    'current_status',
    'changed_since_preview',
    'chassis_number',
    'field',
    'code',
    'severity',
    'message',
  ]
  const lines = [head.join(',')]
  for (const r of p.exceptions) {
    const issues =
      r.issues.length > 0 ? r.issues : [{ field: '', code: '', severity: '', message: '' }]
    for (const i of issues)
      lines.push(
        [
          p.batchReference,
          r.rowNumber,
          r.category,
          r.previewStatus,
          r.currentStatus,
          r.changedSincePreview ? 'yes' : 'no',
          r.source.chassis_number ?? '',
          i.field,
          i.code,
          i.severity,
          i.message,
        ]
          .map(csvCell)
          .join(','),
      )
  }
  return `${lines.join('\r\n')}\r\n`
}
