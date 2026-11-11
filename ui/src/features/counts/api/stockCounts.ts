import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { queryString } from '@/features/catalogue/api/catalogue'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'

/**
 * Stock counts and adjustment approval (E-05; docs/openapi, API_CONVENTIONS
 * "Stock counts and adjustment approval"). The server reads the expected
 * quantity/status, computes the variance, decides who may approve and posts
 * the adjustments; nothing here computes or sends any of that. Approval posts
 * one COUNT_ADJUSTMENT per parts line with a variance; vehicle discrepancies
 * are recorded findings only (OQ-39). No approval threshold exists (OQ-40).
 */

const stamp = z.iso.datetime({ offset: true })
/** Decimal strings with 3 decimals as the server sends them. */
const qty = z.string().regex(/^\d+\.\d{3}$/)
const signedQty = z.string().regex(/^-?\d+\.\d{3}$/)

export const countStatusSchema = z.enum(['DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED'])
export type CountStatus = z.infer<typeof countStatusSchema>

export const lineOutcomeSchema = z.enum([
  'PENDING',
  'NO_CHANGE',
  'POSTED',
  'DISCREPANCY_RECORDED',
  'NOT_POSTED',
])
export type LineOutcome = z.infer<typeof lineOutcomeSchema>

export const countLineSchema = z
  .object({
    id: z.string(),
    itemType: z.enum(['STOCK_ITEM', 'VEHICLE_UNIT']),
    stockItemId: z.string().nullable(),
    sku: z.string().nullable(),
    itemName: z.string().nullable(),
    uom: z.string().nullable(),
    vehicleUnitId: z.string().nullable(),
    unitNumber: z.string().nullable(),
    expectedStatus: z.string().nullable(),
    found: z.boolean().nullable(),
    expectedQuantity: qty,
    observedQuantity: qty,
    variance: signedQty,
    reason: z.string().nullable(),
    stockChanged: z.boolean(),
    outcome: lineOutcomeSchema,
    postedMovementId: z.string().nullable(),
    recordedBy: z.string(),
    recordedByName: z.string().nullable(),
    recordedAt: stamp,
  })
  // A posted adjustment is always a parts line with its movement; vehicles never post.
  .refine(
    (l) =>
      l.outcome === 'POSTED'
        ? l.itemType === 'STOCK_ITEM' && l.postedMovementId !== null
        : l.postedMovementId === null,
    { message: 'a posted outcome needs its parts movement', path: ['postedMovementId'] },
  )
export type CountLine = z.infer<typeof countLineSchema>

export const stockCountSchema = z.object({
  id: z.string(),
  countNumber: z.string(),
  locationId: z.string(),
  locationCode: z.string(),
  locationName: z.string(),
  status: countStatusSchema,
  approvalPolicy: z.string(),
  note: z.string().nullable(),
  createdBy: z.string(),
  createdByName: z.string().nullable(),
  createdAt: stamp,
  submittedBy: z.string().nullable(),
  submittedByName: z.string().nullable(),
  submittedAt: stamp.nullable(),
  decidedBy: z.string().nullable(),
  decidedByName: z.string().nullable(),
  decidedAt: stamp.nullable(),
  decisionNote: z.string().nullable(),
  lineCount: z.number().int().nonnegative(),
  varianceLines: z.number().int().nonnegative(),
  version: z.number().int().positive(),
  updatedAt: stamp,
  lines: z.array(countLineSchema),
})
export type StockCount = z.infer<typeof stockCountSchema>

export interface CountListParams {
  status?: CountStatus
  locationId?: string
  page?: number
  limit?: number
}

const id = (v: string) => encodeURIComponent(v)
const countPage = pageSchema(stockCountSchema)

export function listStockCounts(
  params: CountListParams = {},
  signal?: AbortSignal,
): Promise<Page<StockCount>> {
  return apiRequest(`/stock-counts${queryString(params)}`, countPage, { signal })
}

/** A count outside the caller's location scope answers 404, like one that does not exist. */
export function getStockCount(countId: string, signal?: AbortSignal): Promise<StockCount> {
  return apiRequest(`/stock-counts/${id(countId)}`, stockCountSchema, { signal })
}

/** One counted item: a part (observed quantity in the SKU's unit) or a vehicle unit (found or not). */
export type LineBody =
  | { version: number; stockItemId: string; observedQuantity: string; uom: string; reason?: string }
  | { version: number; vehicleUnitId: string; found: boolean; reason?: string }

/** Records or re-records a line on a draft. No Idempotency-Key: the version guards repeats. */
export function recordCountLine(countId: string, body: LineBody): Promise<StockCount> {
  return apiRequest(
    `/stock-counts/${id(countId)}/lines`,
    stockCountSchema.refine((c) => c.id === countId && c.status === 'DRAFT', {
      message: 'expected this draft count',
    }),
    { method: 'POST', body },
  )
}

/** DRAFT -> SUBMITTED; changes no stock. */
export function submitStockCount(countId: string, version: number): Promise<StockCount> {
  return apiRequest(
    `/stock-counts/${id(countId)}/submit`,
    stockCountSchema.refine((c) => c.id === countId && c.status === 'SUBMITTED', {
      message: 'expected the submitted count',
    }),
    { method: 'POST', body: { version } },
  )
}

/** Same rule as the server: 0 or more, up to 9 whole digits and 3 decimals. */
export const OBSERVED_PATTERN = /^[0-9]{1,9}(\.[0-9]{1,3})?$/

/** "15.000" → "15"; "-2.500" → "-2.5". */
export function formatQuantity(v: string): string {
  return v.includes('.') ? v.replace(/0+$/, '').replace(/\.$/, '') : v
}

/**
 * One logical create/approve/reject with its Idempotency-Key, made once per
 * decision and frozen; a retry after an unknown outcome sends the same action.
 */
export type CountAction = Readonly<
  | {
      kind: 'create'
      body: Readonly<{ locationId: string; note?: string }>
      idempotencyKey: string
    }
  | {
      kind: 'approve'
      countId: string
      body: Readonly<{ version: number; note?: string }>
      idempotencyKey: string
    }
  | {
      kind: 'reject'
      countId: string
      body: Readonly<{ version: number; reason: string }>
      idempotencyKey: string
    }
>

type Distribute<T> = T extends unknown ? Omit<T, 'idempotencyKey'> : never

export function newCountAction(a: Distribute<CountAction>): CountAction {
  return Object.freeze({
    ...a,
    body: Object.freeze({ ...a.body }),
    idempotencyKey: crypto.randomUUID(),
  }) as CountAction
}

/** Whether a new attempt is the same request as an earlier action (same kind, count and body). */
export function sameAction(a: CountAction, b: Distribute<CountAction>): boolean {
  const norm = (o: object) =>
    JSON.stringify(
      Object.entries(o)
        .filter(([, v]) => v !== undefined)
        .sort(([x], [y]) => x.localeCompare(y)),
    )
  const countOf = (x: object) => ('countId' in x ? x.countId : '')
  return a.kind === b.kind && countOf(a) === countOf(b) && norm(a.body) === norm(b.body)
}

/** Sends an action; the answer must be the count in the state the action asked for. */
export function sendCountAction(action: CountAction): Promise<StockCount> {
  const headers = { 'Idempotency-Key': action.idempotencyKey }
  if (action.kind === 'create') {
    return apiRequest(
      '/stock-counts',
      stockCountSchema.refine(
        (c) => c.status === 'DRAFT' && c.locationId === action.body.locationId,
        {
          message: 'expected the new draft count',
        },
      ),
      { method: 'POST', body: action.body, headers },
    )
  }
  const status = action.kind === 'approve' ? 'APPROVED' : 'REJECTED'
  return apiRequest(
    `/stock-counts/${id(action.countId)}/${action.kind}`,
    stockCountSchema.refine((c) => c.id === action.countId && c.status === status, {
      message: `expected the ${status.toLowerCase()} count`,
    }),
    { method: 'POST', body: action.body, headers },
  )
}
