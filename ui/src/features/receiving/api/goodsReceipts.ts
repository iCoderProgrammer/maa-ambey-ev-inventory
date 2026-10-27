import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'
import { poStatusSchema } from '@/features/purchasing/api/purchaseOrders'

/**
 * Goods receipt drafts (D-03; docs/openapi, API_CONVENTIONS "Goods receipts").
 * GRN numbers, ordered/posted/available quantities, excess, warnings and the
 * caller's actions all come from the server; the screen never computes them.
 * A draft changes no stock: posting is a separate step (D-07).
 */

const stamp = z.iso.datetime({ offset: true })

export const grnStatusSchema = z.enum(['DRAFT', 'POSTED', 'CANCELLED'])
export type GRNStatus = z.infer<typeof grnStatusSchema>

export const unitConditionSchema = z.enum(['ACCEPTED', 'REJECTED'])
export type UnitCondition = z.infer<typeof unitConditionSchema>

/** Proposal pending confirmation (OQ-25); the server checks the same list. */
export const documentTypes = ['DELIVERY_CHALLAN', 'SUPPLIER_INVOICE', 'PHOTO', 'OTHER'] as const
export type GRNDocumentType = (typeof documentTypes)[number]

export const lineWarningSchema = z.enum([
  'OVER_RECEIPT',
  'OVER_RECEIPT_NOT_AUTHORISED',
  'PARTIAL_DELIVERY',
  'IDENTIFIERS_INCOMPLETE',
  'REJECTION_NOT_EXPLAINED',
])
export type GRNLineWarning = z.infer<typeof lineWarningSchema>

export const unitWarningSchema = z.enum([
  'CHASSIS_MISSING',
  'DUPLICATE_MOTOR_SERIAL',
  'DUPLICATE_BATTERY_SERIAL',
  'DUPLICATE_CHARGER_SERIAL',
])
export type GRNUnitWarning = z.infer<typeof unitWarningSchema>

export const receiptSchema = z.object({
  id: z.string(),
  number: z.string(),
  status: grnStatusSchema,
  purchaseOrderId: z.string(),
  purchaseOrderNumber: z.string(),
  purchaseOrderStatus: poStatusSchema,
  supplierId: z.string(),
  supplierName: z.string(),
  receivedDate: z.iso.date(),
  supplierReference: z.string().nullable(),
  notes: z.string().nullable(),
  createdBy: z.string(),
  createdByName: z.string(),
  cancelledBy: z.string().nullable(),
  cancelledAt: stamp.nullable(),
  cancelReason: z.string().nullable(),
  postedBy: z.string().nullable(),
  postedAt: stamp.nullable(),
  lineCount: z.number().int().nonnegative(),
  receivedQuantity: z.number().int().nonnegative(),
  acceptedQuantity: z.number().int().nonnegative(),
  rejectedQuantity: z.number().int().nonnegative(),
  unitCount: z.number().int().nonnegative(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type GoodsReceipt = z.infer<typeof receiptSchema>

export const unitSchema = z.object({
  id: z.string(),
  unitNo: z.number().int().positive(),
  condition: unitConditionSchema,
  chassisNumber: z.string().nullable(),
  chassisNormalised: z.string().nullable(),
  motorSerial: z.string().nullable(),
  batterySerial: z.string().nullable(),
  chargerSerial: z.string().nullable(),
  note: z.string().nullable(),
  warnings: z.array(unitWarningSchema),
})
export type GRNUnit = z.infer<typeof unitSchema>

export const overReceiptSchema = z.object({
  quantity: z.number().int().positive(),
  reason: z.string(),
  authorisedBy: z.string(),
  authorisedByName: z.string(),
  authorisedAt: stamp,
})

export const lineSchema = z.object({
  /** Null: nothing of this PO line is on the receipt. */
  id: z.string().nullable(),
  purchaseOrderLineId: z.string(),
  lineNo: z.number().int().positive(),
  brandId: z.string(),
  brandName: z.string(),
  modelName: z.string(),
  variantName: z.string(),
  orderedQuantity: z.number().int().nonnegative(),
  postedQuantity: z.number().int().nonnegative(),
  otherDraftsQuantity: z.number().int().nonnegative(),
  /** ordered − posted − other drafts: what this receipt may take without an override. */
  availableQuantity: z.number().int().nonnegative(),
  receivedQuantity: z.number().int().nonnegative(),
  acceptedQuantity: z.number().int().nonnegative(),
  rejectedQuantity: z.number().int().nonnegative(),
  excessQuantity: z.number().int().nonnegative(),
  overReceipt: overReceiptSchema.nullable(),
  discrepancyNote: z.string().nullable(),
  units: z.array(unitSchema),
  warnings: z.array(lineWarningSchema),
})
export type GRNLine = z.infer<typeof lineSchema>

export const documentSchema = z.object({
  id: z.string(),
  attachmentId: z.string(),
  documentType: z.enum(documentTypes),
  note: z.string().nullable(),
  fileName: z.string(),
  contentType: z.string(),
  addedBy: z.string(),
  addedByName: z.string(),
  createdAt: stamp,
})
export type GRNDocument = z.infer<typeof documentSchema>

export const eventSchema = z.object({
  id: z.string(),
  action: z.enum(['CREATED', 'CANCELLED', 'POSTED']),
  fromStatus: z.string().nullable(),
  toStatus: z.string(),
  actorId: z.string().nullable(),
  actorName: z.string().nullable(),
  note: z.string().nullable(),
  createdAt: stamp,
})
export type GRNEvent = z.infer<typeof eventSchema>

export const actionsSchema = z.object({
  edit: z.boolean(),
  cancel: z.boolean(),
  addDocument: z.boolean(),
  authoriseOverReceipt: z.boolean(),
  /** Posting changes stock and is a separate step; always false until it exists. */
  post: z.boolean(),
})

export const detailSchema = receiptSchema.extend({
  purchaseOrderReceivable: z.boolean(),
  lines: z.array(lineSchema),
  documents: z.array(documentSchema),
  events: z.array(eventSchema),
  stockPosted: z.boolean(),
  actions: actionsSchema,
})
export type GRNDetail = z.infer<typeof detailSchema>

export const receivablePOSchema = z.object({
  id: z.string(),
  number: z.string(),
  status: poStatusSchema,
  supplierId: z.string(),
  supplierName: z.string(),
  expectedDate: z.iso.date().nullable(),
  orderedQuantity: z.number().int().nonnegative(),
  postedQuantity: z.number().int().nonnegative(),
  draftQuantity: z.number().int().nonnegative(),
  availableQuantity: z.number().int().nonnegative(),
  openDraftCount: z.number().int().nonnegative(),
  brandNames: z.array(z.string()),
})
export type ReceivablePO = z.infer<typeof receivablePOSchema>

export interface GRNListParams {
  q?: string
  status?: GRNStatus
  purchaseOrderId?: string
  page?: number
  limit?: number
}

export interface UnitFields {
  condition: UnitCondition
  chassisNumber: string | null
  motorSerial: string | null
  batterySerial: string | null
  chargerSerial: string | null
  note: string | null
}

export interface LineFields {
  purchaseOrderLineId: string
  receivedQuantity: number
  acceptedQuantity: number
  rejectedQuantity: number
  discrepancyNote: string | null
  overReceiptReason: string | null
  units: UnitFields[]
}

export interface SaveFields {
  version: number
  receivedDate: string
  supplierReference: string | null
  notes: string | null
  lines: LineFields[]
}

const page = pageSchema(receiptSchema)
const receivablePage = pageSchema(receivablePOSchema)
const id = (v: string) => encodeURIComponent(v)
const base = '/goods-receipts'

export function listGoodsReceipts(
  params: GRNListParams = {},
  signal?: AbortSignal,
): Promise<Page<GoodsReceipt>> {
  return apiRequest(`${base}${queryString(params)}`, page, { signal })
}
export function listReceivablePurchaseOrders(
  params: { q?: string; page?: number; limit?: number } = {},
  signal?: AbortSignal,
): Promise<Page<ReceivablePO>> {
  return apiRequest(`/receivable-purchase-orders${queryString(params)}`, receivablePage, { signal })
}
export function getGoodsReceipt(grnId: string, signal?: AbortSignal): Promise<GRNDetail> {
  return apiRequest(`${base}/${id(grnId)}`, detailSchema, { signal })
}
/**
 * Starts a draft. The idempotency key makes a repeated click or retry return
 * the receipt created first instead of a second one.
 */
export function createGoodsReceipt(
  input: {
    purchaseOrderId: string
    receivedDate: string
    supplierReference?: string | null
    notes?: string | null
  },
  idempotencyKey: string,
): Promise<GRNDetail> {
  return apiRequest(base, detailSchema, {
    method: 'POST',
    body: input,
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}
/** Replaces the draft's header, lines and units (all or nothing). Changes no stock. */
export function saveGoodsReceipt(grnId: string, input: SaveFields): Promise<GRNDetail> {
  return apiRequest(`${base}/${id(grnId)}`, detailSchema, { method: 'PUT', body: input })
}
export function cancelGoodsReceipt(
  grnId: string,
  input: { version: number; reason: string },
): Promise<GRNDetail> {
  return apiRequest(`${base}/${id(grnId)}/cancel`, detailSchema, { method: 'POST', body: input })
}
export function addGoodsReceiptDocument(
  grnId: string,
  input: { attachmentId: string; documentType: GRNDocumentType; note?: string | null },
): Promise<GRNDetail> {
  return apiRequest(`${base}/${id(grnId)}/documents`, detailSchema, { method: 'POST', body: input })
}
