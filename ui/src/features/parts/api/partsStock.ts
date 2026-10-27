import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { queryString } from '@/features/catalogue/api/catalogue'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'

/**
 * Parts receipts and issues (E-04; docs/openapi, API_CONVENTIONS "Parts receipts
 * and issues"). Balances are the server's: computed from receipts and issues,
 * never sent or set by the client. Every post is one ledger movement, needs an
 * Idempotency-Key and changes stock only after the server confirms. SERIAL SKUs
 * are refused by the server (TRACKING_NOT_SUPPORTED); no unit conversion.
 */

const stamp = z.iso.datetime({ offset: true })

/** A decimal string as thousandths (exact, no floating point): "2.5" → 2500n. */
export function thousandths(v: string): bigint {
  const [whole = '0', frac = ''] = v.split('.')
  return BigInt(whole) * 1000n + BigInt((frac + '000').slice(0, 3))
}
/** Quantities as the server sends them: decimal strings with 3 decimals. */
const qty = z.string().regex(/^\d+\.\d{3}$/)

export const partsBalanceSchema = z.object({
  stockItemId: z.string(),
  sku: z.string(),
  locationId: z.string(),
  locationCode: z.string(),
  locationName: z.string(),
  uom: z.string(),
  quantity: qty,
  version: z.number().int().positive(),
  updatedAt: stamp,
})
export type PartsBalance = z.infer<typeof partsBalanceSchema>

export const partsMovementTypeSchema = z.enum(['PARTS_RECEIPT', 'PARTS_ISSUE'])
export type PartsMovementType = z.infer<typeof partsMovementTypeSchema>

export const partsMovementSchema = z
  .object({
    id: z.string(),
    stockItemId: z.string(),
    sku: z.string(),
    movementType: partsMovementTypeSchema,
    locationId: z.string(),
    locationCode: z.string(),
    locationName: z.string(),
    uom: z.string(),
    quantity: qty,
    quantityBefore: qty,
    quantityAfter: qty,
    sourceType: z.enum(['SUPPLIER_DELIVERY', 'STAFF_ACTION']),
    /** Null without suppliers.view. */
    supplierId: z.string().nullable(),
    supplierName: z.string().nullable(),
    sourceReference: z.string().nullable(),
    reason: z.string().nullable(),
    note: z.string().nullable(),
    actorId: z.string(),
    actorName: z.string().nullable(),
    createdAt: stamp,
  })
  // A receipt adds and an issue takes away; nothing below zero.
  .refine(
    (m) =>
      m.movementType === 'PARTS_RECEIPT'
        ? thousandths(m.quantityAfter) > thousandths(m.quantityBefore)
        : thousandths(m.quantityAfter) < thousandths(m.quantityBefore),
    { message: 'the movement direction does not match its type', path: ['quantityAfter'] },
  )
export type PartsMovement = z.infer<typeof partsMovementSchema>

export const partsResultSchema = z
  .object({ movement: partsMovementSchema, balance: partsBalanceSchema })
  // The balance returned is the one the movement changed.
  .refine(
    (r) =>
      r.balance.stockItemId === r.movement.stockItemId &&
      r.balance.locationId === r.movement.locationId,
    { message: 'the balance must belong to the movement', path: ['balance'] },
  )
export type PartsResult = z.infer<typeof partsResultSchema>

/** Same rule as the server: > 0, up to 9 whole digits and 3 decimals, no sign or exponent. */
export const QUANTITY_PATTERN = /^[0-9]{1,9}(\.[0-9]{1,3})?$/

/** Whether a quantity string is valid and more than zero (checked as text, no floating point). */
export function isValidQuantity(v: string): boolean {
  return QUANTITY_PATTERN.test(v) && /[1-9]/.test(v)
}

/** "15.000" → "15", "2.500" → "2.5": the server's value without trailing zeros. */
export function formatQuantity(v: string): string {
  return v.includes('.') ? v.replace(/0+$/, '').replace(/\.$/, '') : v
}

export interface PartsHistoryParams {
  locationId?: string
  type?: PartsMovementType
  page?: number
  limit?: number
}

const id = (v: string) => encodeURIComponent(v)
const historyPage = pageSchema(partsMovementSchema)

/** The SKU's balances at the caller's locations (a location never stocked has no row). */
export function getPartsBalances(itemId: string, signal?: AbortSignal): Promise<PartsBalance[]> {
  return apiRequest(`/stock-items/${id(itemId)}/balances`, z.array(partsBalanceSchema), { signal })
}

/** The SKU's receipts and issues, newest first. */
export function listPartsMovements(
  itemId: string,
  params: PartsHistoryParams = {},
  signal?: AbortSignal,
): Promise<Page<PartsMovement>> {
  return apiRequest(
    `/stock-items/${id(itemId)}/stock-movements${queryString(params)}`,
    historyPage,
    {
      signal,
    },
  )
}

export interface ReceiptBody {
  locationId: string
  quantity: string
  uom: string
  supplierId: string
  sourceReference: string
  note?: string
}
export interface IssueBody {
  locationId: string
  quantity: string
  uom: string
  reason: string
  sourceReference?: string
  note?: string
}

/**
 * One logical receipt or issue: the SKU, the exact body and the idempotency key
 * that identifies it. Made once per decision (newPartsAction) and frozen; a
 * retry sends the same action again. Different values are a new action.
 */
export type PartsAction = Readonly<
  | { kind: 'receipt'; itemId: string; body: Readonly<ReceiptBody>; idempotencyKey: string }
  | { kind: 'issue'; itemId: string; body: Readonly<IssueBody>; idempotencyKey: string }
>

export function newPartsAction(kind: 'receipt', itemId: string, body: ReceiptBody): PartsAction
export function newPartsAction(kind: 'issue', itemId: string, body: IssueBody): PartsAction
export function newPartsAction(
  kind: 'receipt' | 'issue',
  itemId: string,
  body: ReceiptBody | IssueBody,
): PartsAction {
  return Object.freeze({
    kind,
    itemId,
    body: Object.freeze({ ...body }),
    idempotencyKey: crypto.randomUUID(),
  }) as PartsAction
}

/** Whether two bodies are the same request (same fields and values). */
export function sameBody(a: object, b: object): boolean {
  const norm = (o: object) =>
    JSON.stringify(
      Object.entries(o)
        .filter(([, v]) => v !== undefined)
        .sort(([x], [y]) => x.localeCompare(y)),
    )
  return norm(a) === norm(b)
}

const keyPattern = /^[A-Za-z0-9_-]{8,100}$/

/**
 * Sends an action. The answer must be the movement this action asked for (type,
 * SKU, location). A replay answers 200 with the first movement.
 */
export async function sendPartsAction(action: PartsAction): Promise<PartsResult> {
  if (!keyPattern.test(action.idempotencyKey) || !action.itemId) {
    throw new Error('a parts action needs its SKU and a valid idempotency key')
  }
  const type = action.kind === 'receipt' ? 'PARTS_RECEIPT' : 'PARTS_ISSUE'
  const expected = partsResultSchema.refine(
    (r) =>
      r.movement.movementType === type &&
      r.movement.stockItemId === action.itemId &&
      r.movement.locationId === action.body.locationId,
    { message: `expected the ${type} of this request`, path: ['movement'] },
  )
  return apiRequest(
    `/stock-items/${id(action.itemId)}/${action.kind === 'receipt' ? 'receipts' : 'issues'}`,
    expected,
    {
      method: 'POST',
      body: action.body,
      headers: { 'Idempotency-Key': action.idempotencyKey },
    },
  )
}
