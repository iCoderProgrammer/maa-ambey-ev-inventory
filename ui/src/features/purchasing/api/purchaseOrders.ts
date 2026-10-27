import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { lifecycleStatusSchema, pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'
import { supplierStatusSchema } from '@/features/suppliers/api/suppliers'

/**
 * Purchase orders (D-02; docs/openapi, API_CONVENTIONS "Purchase orders").
 * Numbers, totals, remaining quantities, warnings, prerequisites and the
 * caller's possible actions all come from the server; the screen never
 * computes them. Amounts are decimal strings in INR, excluding tax.
 */

const stamp = z.iso.datetime({ offset: true })

export const poStatusSchema = z.enum([
  'DRAFT',
  'SUBMITTED',
  'APPROVED',
  'PARTIALLY_RECEIVED',
  'RECEIVED',
  'CLOSED',
  'CANCELLED',
])
export type POStatus = z.infer<typeof poStatusSchema>

export const lineWarningSchema = z.enum(['PRICE_VARIANCE', 'PARTIAL_DELIVERY', 'OVER_RECEIPT'])
export type LineWarning = z.infer<typeof lineWarningSchema>

export const purchaseOrderSchema = z.object({
  id: z.string(),
  number: z.string(),
  supplierId: z.string(),
  supplierName: z.string(),
  supplierStatus: supplierStatusSchema,
  status: poStatusSchema,
  expectedDate: z.iso.date().nullable(),
  notes: z.string().nullable(),
  currency: z.string(),
  createdBy: z.string(),
  createdByName: z.string(),
  submittedBy: z.string().nullable(),
  submittedAt: stamp.nullable(),
  approvalBasis: z.enum(['APPROVER', 'BELOW_THRESHOLD']).nullable(),
  approvedBy: z.string().nullable(),
  approvedAt: stamp.nullable(),
  approvalNote: z.string().nullable(),
  thresholdApplied: z.string().nullable(),
  cancelledBy: z.string().nullable(),
  cancelledAt: stamp.nullable(),
  cancelReason: z.string().nullable(),
  lineCount: z.number().int().nonnegative(),
  totalQuantity: z.number().int().nonnegative(),
  receivedQuantity: z.number().int().nonnegative(),
  remainingQuantity: z.number().int().nonnegative(),
  totalAmount: z.string(),
  brandNames: z.array(z.string()),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type PurchaseOrder = z.infer<typeof purchaseOrderSchema>

export const lineSchema = z.object({
  id: z.string(),
  lineNo: z.number().int().positive(),
  brandId: z.string(),
  brandName: z.string(),
  brandStatus: lifecycleStatusSchema,
  modelId: z.string(),
  modelName: z.string(),
  modelStatus: lifecycleStatusSchema,
  variantId: z.string(),
  variantName: z.string(),
  variantStatus: lifecycleStatusSchema,
  quantity: z.number().int().positive(),
  unitCost: z.string(),
  lineTotal: z.string(),
  receivedQuantity: z.number().int().nonnegative(),
  remainingQuantity: z.number().int().nonnegative(),
  referencePriceListId: z.string().nullable(),
  referencePriceItemId: z.string().nullable(),
  referenceUnitCost: z.string().nullable(),
  referenceCapturedAt: stamp.nullable(),
  /** A snapshot exists but the caller may not see catalogue cost prices. */
  referenceHidden: z.boolean(),
  costVariance: z.string().nullable(),
  note: z.string().nullable(),
  warnings: z.array(lineWarningSchema),
  createdAt: stamp,
  updatedAt: stamp,
})
export type POLine = z.infer<typeof lineSchema>

export const eventSchema = z.object({
  id: z.string(),
  action: z.enum([
    'CREATED',
    'SUBMITTED',
    'APPROVED',
    'APPROVAL_NOT_REQUIRED',
    'RETURNED',
    'CANCELLED',
  ]),
  fromStatus: z.string().nullable(),
  toStatus: z.string(),
  actorId: z.string().nullable(),
  actorName: z.string().nullable(),
  note: z.string().nullable(),
  createdAt: stamp,
})
export type POEvent = z.infer<typeof eventSchema>

export const checkSchema = z.object({
  key: z.string(),
  message: z.string(),
  lineId: z.string().nullable(),
})
export type POCheck = z.infer<typeof checkSchema>

export const actionsSchema = z.object({
  edit: z.boolean(),
  submit: z.boolean(),
  approve: z.boolean(),
  return: z.boolean(),
  cancel: z.boolean(),
})
export type POActions = z.infer<typeof actionsSchema>

export const detailSchema = purchaseOrderSchema.extend({
  lines: z.array(lineSchema),
  events: z.array(eventSchema),
  blockers: z.array(checkSchema),
  approvalRequired: z.boolean().nullable(),
  approvalReason: z.string().nullable(),
  actions: actionsSchema,
})
export type PODetail = z.infer<typeof detailSchema>

export const settingsSchema = z.object({
  approvalThreshold: z.string().nullable(),
  firstTimeSupplierNeedsApproval: z.boolean(),
  approvalReference: z.string().nullable(),
  updatedBy: z.string().nullable(),
  version: z.number().int().positive(),
  updatedAt: stamp.nullable(),
})
export type POSettings = z.infer<typeof settingsSchema>

export interface POListParams {
  q?: string
  status?: POStatus
  supplierId?: string
  brandId?: string
  page?: number
  limit?: number
}

export interface LineFields {
  brandId: string
  modelId: string
  variantId: string
  quantity: number
  unitCost: string
  priceListId?: string | null
  note?: string | null
}

export interface HeaderFields {
  supplierId?: string
  expectedDate?: string | null
  notes?: string | null
}

const page = pageSchema(purchaseOrderSchema)
const id = (v: string) => encodeURIComponent(v)
const base = '/purchase-orders'

export function listPurchaseOrders(
  params: POListParams = {},
  signal?: AbortSignal,
): Promise<Page<PurchaseOrder>> {
  return apiRequest(`${base}${queryString(params)}`, page, { signal })
}
export function getPurchaseOrder(poId: string, signal?: AbortSignal): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}`, detailSchema, { signal })
}
/**
 * Creates a draft. The idempotency key makes a repeated click or retry return
 * the PO created first instead of a second one.
 */
export function createPurchaseOrder(
  input: { supplierId: string; expectedDate?: string | null; notes?: string | null },
  idempotencyKey: string,
): Promise<PODetail> {
  return apiRequest(base, detailSchema, {
    method: 'POST',
    body: input,
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}
export function updatePurchaseOrder(
  poId: string,
  input: HeaderFields & { version: number },
): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}`, detailSchema, { method: 'PATCH', body: input })
}
export function addLine(poId: string, input: LineFields & { version: number }): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/lines`, detailSchema, { method: 'POST', body: input })
}
export function updateLine(
  poId: string,
  lineId: string,
  input: {
    version: number
    quantity?: number
    unitCost?: string
    priceListId?: string | null
    note?: string | null
  },
): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/lines/${id(lineId)}`, detailSchema, {
    method: 'PATCH',
    body: input,
  })
}
export function removeLine(poId: string, lineId: string, version: number): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/lines/${id(lineId)}/remove`, detailSchema, {
    method: 'POST',
    body: { version },
  })
}
export function submitPurchaseOrder(poId: string, version: number): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/submit`, detailSchema, {
    method: 'POST',
    body: { version },
  })
}
export function approvePurchaseOrder(
  poId: string,
  input: { version: number; note?: string | null },
): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/approve`, detailSchema, { method: 'POST', body: input })
}
export function returnPurchaseOrder(
  poId: string,
  input: { version: number; reason: string },
): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/return`, detailSchema, { method: 'POST', body: input })
}
export function cancelPurchaseOrder(
  poId: string,
  input: { version: number; reason: string },
): Promise<PODetail> {
  return apiRequest(`${base}/${id(poId)}/cancel`, detailSchema, { method: 'POST', body: input })
}
export function getPOSettings(signal?: AbortSignal): Promise<POSettings> {
  return apiRequest('/purchase-order-settings', settingsSchema, { signal })
}
export function updatePOSettings(input: {
  version: number
  approvalThreshold: string | null
  firstTimeSupplierNeedsApproval: boolean
  approvalReference: string | null
}): Promise<POSettings> {
  return apiRequest('/purchase-order-settings', settingsSchema, { method: 'PUT', body: input })
}
