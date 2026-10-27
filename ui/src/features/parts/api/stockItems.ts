import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { lifecycleStatusSchema, pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'
import { supplierStatusSchema } from '@/features/suppliers/api/suppliers'

/**
 * Spare part and accessory SKU master (E-03; docs/openapi, API_CONVENTIONS
 * "Spare part and accessory SKUs"). Master data only: no call here records
 * stock, a balance or a movement (E-04). Categories and SKUs are archived, never
 * deleted, and there is no reactivation. The SKU code and tracking mode are
 * fixed at creation. Supplier links come only with suppliers.view (otherwise the
 * server sends suppliers [] and suppliersHidden true).
 */

const stamp = z.iso.datetime({ offset: true })

export const itemStatusSchema = z.enum(['ACTIVE', 'ARCHIVED'])
export type ItemStatus = z.infer<typeof itemStatusSchema>
export const trackingModeSchema = z.enum(['QUANTITY', 'SERIAL'])
export type TrackingMode = z.infer<typeof trackingModeSchema>

/** The server's rules (ADR-021), mirrored for early feedback; the server checks again. */
export const SKU_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._/-]{0,39}$/
/** Unit code format only: no approved list of units exists yet (OQ-34). */
export const UOM_PATTERN = /^[A-Z][A-Z0-9]{0,9}$/
export const LEVEL_PATTERN = /^[0-9]{1,9}(\.[0-9]{1,3})?$/

/** The comparison form of a SKU code (trimmed, upper-cased). Separators are kept. */
export function normaliseSku(sku: string): string {
  return sku.trim().toUpperCase()
}

/** Whether a decimal string has no fractional part ("5", "5.000"). */
export function isWholeNumber(v: string): boolean {
  const i = v.indexOf('.')
  return i < 0 || /^0*$/.test(v.slice(i + 1))
}

/** "5.000" → "5", "2.500" → "2.5": the server's value, without trailing zeros. */
export function formatLevel(v: string | null): string | null {
  if (v === null) return null
  return v.includes('.') ? v.replace(/0+$/, '').replace(/\.$/, '') : v
}

export const categorySchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  status: itemStatusSchema,
  archivedBy: z.string().nullable(),
  archivedAt: stamp.nullable(),
  archiveReason: z.string().nullable(),
  activeItemCount: z.number().int().nonnegative(),
  createdBy: z.string(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type ItemCategory = z.infer<typeof categorySchema>

export const stockItemSchema = z.object({
  id: z.string(),
  sku: z.string(),
  skuNormalised: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  categoryId: z.string(),
  categoryName: z.string(),
  categoryStatus: itemStatusSchema,
  uom: z.string(),
  trackingMode: trackingModeSchema,
  /** Decimal string with 3 decimals, e.g. "5.000"; null when not set. */
  reorderLevel: z
    .string()
    .regex(/^\d+\.\d{3}$/)
    .nullable(),
  status: itemStatusSchema,
  archivedBy: z.string().nullable(),
  archivedAt: stamp.nullable(),
  archiveReason: z.string().nullable(),
  createdBy: z.string(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type StockItem = z.infer<typeof stockItemSchema>

export const supplierLinkSchema = z.object({
  id: z.string(),
  supplierId: z.string(),
  supplierName: z.string(),
  supplierStatus: supplierStatusSchema,
  supplierPartNumber: z.string().nullable(),
  createdBy: z.string(),
  createdAt: stamp,
  endedBy: z.string().nullable(),
  endedAt: stamp.nullable(),
})
export type StockItemSupplier = z.infer<typeof supplierLinkSchema>

export const compatibilitySchema = z.object({
  id: z.string(),
  brandId: z.string(),
  brandName: z.string(),
  brandStatus: lifecycleStatusSchema,
  /** Null: every model of the brand. */
  modelId: z.string().nullable(),
  modelName: z.string().nullable(),
})
export type StockItemCompatibility = z.infer<typeof compatibilitySchema>

export const stockItemDetailSchema = stockItemSchema
  .extend({
    suppliers: z.array(supplierLinkSchema),
    suppliersHidden: z.boolean(),
    compatibility: z.array(compatibilitySchema),
  })
  // Hidden suppliers carry no supplier data: never trust an answer that does.
  .refine((d) => !d.suppliersHidden || d.suppliers.length === 0, {
    message: 'hidden suppliers must not be listed',
    path: ['suppliers'],
  })
export type StockItemDetail = z.infer<typeof stockItemDetailSchema>

export interface CategoryListParams {
  q?: string
  status?: ItemStatus
  page?: number
  limit?: number
}

export interface StockItemListParams {
  /** SKU code (case-insensitive) or name. */
  q?: string
  status?: ItemStatus
  categoryId?: string
  trackingMode?: TrackingMode
  /** Items with a current link to the supplier; needs suppliers.view. */
  supplierId?: string
  /** Items with compatibility for the brand. */
  brandId?: string
  page?: number
  limit?: number
}

export interface SupplierLinkInput {
  supplierId: string
  supplierPartNumber?: string | null
}
export interface CompatibilityInput {
  brandId: string
  /** Omitted or null: every model of the brand. */
  modelId?: string | null
}

export interface StockItemCreate {
  sku: string
  name: string
  description?: string | null
  categoryId: string
  uom: string
  trackingMode: TrackingMode
  reorderLevel?: string | null
  suppliers?: SupplierLinkInput[]
  compatibility?: CompatibilityInput[]
}

/** Only fields sent change. suppliers / compatibility replace the current set. */
export interface StockItemUpdate {
  version: number
  name?: string
  description?: string | null
  categoryId?: string
  uom?: string
  reorderLevel?: string | null
  suppliers?: SupplierLinkInput[]
  compatibility?: CompatibilityInput[]
}

const id = (v: string) => encodeURIComponent(v)
const categoryPage = pageSchema(categorySchema)
const itemPage = pageSchema(stockItemSchema)

export function listCategories(
  params: CategoryListParams = {},
  signal?: AbortSignal,
): Promise<Page<ItemCategory>> {
  return apiRequest(`/stock-item-categories${queryString(params)}`, categoryPage, { signal })
}
export function createCategory(input: {
  name: string
  description?: string | null
}): Promise<ItemCategory> {
  return apiRequest('/stock-item-categories', categorySchema, { method: 'POST', body: input })
}
export function updateCategory(
  categoryId: string,
  input: { version: number; name?: string; description?: string | null },
): Promise<ItemCategory> {
  return apiRequest(`/stock-item-categories/${id(categoryId)}`, categorySchema, {
    method: 'PATCH',
    body: input,
  })
}
/** Refused (409 INVALID_STATE) while the category has active SKUs. */
export function archiveCategory(
  categoryId: string,
  input: { version: number; reason: string },
): Promise<ItemCategory> {
  return apiRequest(`/stock-item-categories/${id(categoryId)}/archive`, categorySchema, {
    method: 'POST',
    body: input,
  })
}

export function listStockItems(
  params: StockItemListParams = {},
  signal?: AbortSignal,
): Promise<Page<StockItem>> {
  return apiRequest(`/stock-items${queryString(params)}`, itemPage, { signal })
}
export function getStockItem(itemId: string, signal?: AbortSignal): Promise<StockItemDetail> {
  return apiRequest(`/stock-items/${id(itemId)}`, stockItemDetailSchema, { signal })
}
export function createStockItem(input: StockItemCreate): Promise<StockItemDetail> {
  return apiRequest('/stock-items', stockItemDetailSchema, { method: 'POST', body: input })
}
export function updateStockItem(itemId: string, input: StockItemUpdate): Promise<StockItemDetail> {
  return apiRequest(`/stock-items/${id(itemId)}`, stockItemDetailSchema, {
    method: 'PATCH',
    body: input,
  })
}
/** Keeps the SKU, its code, links and compatibility; it can no longer change. */
export function archiveStockItem(
  itemId: string,
  input: { version: number; reason: string },
): Promise<StockItemDetail> {
  return apiRequest(`/stock-items/${id(itemId)}/archive`, stockItemDetailSchema, {
    method: 'POST',
    body: input,
  })
}
