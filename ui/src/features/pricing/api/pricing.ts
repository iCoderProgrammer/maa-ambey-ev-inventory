import { z } from 'zod'
import { apiRequest, ApiError } from '@/lib/api/client'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'

/**
 * Effective-dated price history (C-04; docs/openapi, API_CONVENTIONS "Price
 * history"). Amounts are decimal strings ("125000.00"), never floats. The
 * server decides permissions (cost prices need cost permissions), approval
 * rules and overlaps; nothing here assumes success.
 */

export const priceTypeSchema = z.enum(['SELLING', 'COST'])
export type PriceType = z.infer<typeof priceTypeSchema>

export const priceStatusSchema = z.enum(['DRAFT', 'APPROVED', 'CANCELLED'])
export type PriceStatus = z.infer<typeof priceStatusSchema>

const stamp = z.iso.datetime({ offset: true })

export const priceListSchema = z.object({
  id: z.string(),
  brandId: z.string(),
  name: z.string(),
  currency: z.literal('INR'),
  lifecycleStatus: z.enum(['ACTIVE', 'ARCHIVED']),
  createdBy: z.string(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type PriceList = z.infer<typeof priceListSchema>

export const priceItemSchema = z.object({
  id: z.string(),
  priceListId: z.string(),
  brandId: z.string(),
  modelId: z.string(),
  variantId: z.string(),
  priceType: priceTypeSchema,
  amount: z.string().regex(/^\d+\.\d{2}$/),
  currency: z.literal('INR'),
  effectiveFrom: stamp,
  effectiveTo: stamp.nullable(),
  status: priceStatusSchema,
  /** Always null until an approved tax configuration exists (F-10, accountant). */
  taxConfigurationId: z.string().nullable(),
  sourceNote: z.string().nullable(),
  createdBy: z.string(),
  approvedBy: z.string().nullable(),
  approvedAt: stamp.nullable(),
  approvalReference: z.string().nullable(),
  supersedesId: z.string().nullable(),
  supersededById: z.string().nullable(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type PriceItem = z.infer<typeof priceItemSchema>

export interface PriceListParams {
  brandId?: string
  status?: 'ACTIVE' | 'ARCHIVED'
  page?: number
  limit?: number
}
export interface PriceItemParams {
  priceListId?: string
  brandId?: string
  variantId?: string
  priceType?: PriceType
  status?: PriceStatus
  page?: number
  limit?: number
}
export interface PriceItemCreate {
  priceListId: string
  variantId: string
  priceType: PriceType
  amount: string
  /** RFC 3339 with offset. */
  effectiveFrom: string
  effectiveTo?: string | null
  sourceNote?: string | null
  supersedesId?: string | null
}
export interface PriceItemUpdate {
  version: number
  amount?: string
  effectiveFrom?: string
  effectiveTo?: string | null
  sourceNote?: string | null
}

const listPage = pageSchema(priceListSchema)
const itemPage = pageSchema(priceItemSchema)
const id = (v: string) => encodeURIComponent(v)

/** GET /price-lists — needs a price or cost-price view permission. */
export function listPriceLists(
  params: PriceListParams = {},
  signal?: AbortSignal,
): Promise<Page<PriceList>> {
  return apiRequest(`/price-lists${queryString(params)}`, listPage, { signal })
}

/** POST /price-lists — needs price or cost-price management. */
export function createPriceList(input: { brandId: string; name: string }): Promise<PriceList> {
  return apiRequest('/price-lists', priceListSchema, { method: 'POST', body: input })
}

/** POST /price-lists/{id}/archive. */
export function archivePriceList(listId: string, version: number): Promise<PriceList> {
  return apiRequest(`/price-lists/${id(listId)}/archive`, priceListSchema, {
    method: 'POST',
    body: { version },
  })
}

/** GET /price-items — only types the caller may see; newest effective date first. */
export function listPriceItems(
  params: PriceItemParams = {},
  signal?: AbortSignal,
): Promise<Page<PriceItem>> {
  return apiRequest(`/price-items${queryString(params)}`, itemPage, { signal })
}

/** POST /price-items — drafts a price (SELLING: price.manage, COST: cost_price.edit). */
export function createPriceItem(input: PriceItemCreate): Promise<PriceItem> {
  return apiRequest('/price-items', priceItemSchema, { method: 'POST', body: input })
}

/** PATCH /price-items/{id} — drafts only. */
export function updatePriceItem(itemId: string, input: PriceItemUpdate): Promise<PriceItem> {
  return apiRequest(`/price-items/${id(itemId)}`, priceItemSchema, { method: 'PATCH', body: input })
}

/** POST /price-items/{id}/cancel — drafts only; kept as history. */
export function cancelPriceItem(itemId: string, version: number): Promise<PriceItem> {
  return apiRequest(`/price-items/${id(itemId)}/cancel`, priceItemSchema, {
    method: 'POST',
    body: { version },
  })
}

/** POST /price-items/{id}/approve — catalogue.price.approve, never the drafter. */
export function approvePriceItem(
  itemId: string,
  input: { version: number; approvalReference?: string },
): Promise<PriceItem> {
  return apiRequest(`/price-items/${id(itemId)}/approve`, priceItemSchema, {
    method: 'POST',
    body: input,
  })
}

/** GET /prices/lookup — the approved price valid at `at`; ApiError 404 PRICE_NOT_FOUND when none. */
export function lookupPrice(
  params: { priceListId: string; variantId: string; priceType: PriceType; at?: string },
  signal?: AbortSignal,
): Promise<PriceItem> {
  return apiRequest(`/prices/lookup${queryString(params)}`, priceItemSchema, { signal })
}

export const PriceErrorCode = {
  Overlap: 'PRICE_OVERLAP',
  ApprovalBlocked: 'APPROVAL_BLOCKED',
  SelfApproval: 'SELF_APPROVAL_NOT_ALLOWED',
  NotFound: 'PRICE_NOT_FOUND',
} as const

/** True when no approved price is valid at the requested time. */
export function isNoPrice(error: unknown): boolean {
  return error instanceof ApiError && error.code === PriceErrorCode.NotFound
}
