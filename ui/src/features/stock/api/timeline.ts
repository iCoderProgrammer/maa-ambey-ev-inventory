import { z } from 'zod'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'
import { unitStatusSchema } from '@/features/vehicles/api/vehicleUnits'
import { apiRequest } from '@/lib/api/client'

/**
 * Stock movement timeline (E-02; docs/openapi, API_CONVENTIONS "Stock movement
 * timeline", ADR-027). Read-only: the append-only ledger exactly as the server
 * recorded it, across vehicles and parts. There is no call that creates,
 * edits, reverses or deletes a movement. The server decides which rows the
 * caller may see (vehicle_units.view / stock.parts.view and location scope)
 * and hides sources the caller may not open (sourceHidden).
 */

const stamp = z.iso.datetime({ offset: true })
const decimal = z.string().regex(/^-?\d+(\.\d+)?$/)

/** Exactly the ledger's movement types (DB CHECK stock_movements_type_valid). */
export const timelineMovementTypeSchema = z.enum([
  'RECEIPT',
  'PDI_PASS',
  'QUALITY_HOLD',
  'REJECT',
  'DISPLAY_ON',
  'DISPLAY_OFF',
  'PARTS_RECEIPT',
  'PARTS_ISSUE',
  'COUNT_ADJUSTMENT',
])
export type TimelineMovementType = z.infer<typeof timelineMovementTypeSchema>

export const timelineSourceTypeSchema = z.enum([
  'GOODS_RECEIPT',
  'PDI_INSPECTION',
  'STAFF_ACTION',
  'SUPPLIER_DELIVERY',
  'STOCK_COUNT',
])
export type TimelineSourceType = z.infer<typeof timelineSourceTypeSchema>

export const timelineItemTypeSchema = z.enum(['VEHICLE_UNIT', 'STOCK_ITEM'])
export type TimelineItemType = z.infer<typeof timelineItemTypeSchema>

export const timelineMovementSchema = z
  .object({
    id: z.string(),
    itemType: timelineItemTypeSchema,
    movementType: timelineMovementTypeSchema,
    createdAt: stamp,
    vehicleUnit: z
      .object({ id: z.string(), unitNumber: z.string(), chassisNumber: z.string().nullable() })
      .nullable(),
    stockItem: z.object({ id: z.string(), sku: z.string(), name: z.string() }).nullable(),
    quantityBefore: decimal,
    quantityAfter: decimal,
    uom: z.string().nullable(),
    fromStatus: unitStatusSchema.nullable(),
    toStatus: unitStatusSchema.nullable(),
    fromLocationId: z.string().nullable(),
    fromLocationCode: z.string().nullable(),
    fromLocationName: z.string().nullable(),
    toLocationId: z.string().nullable(),
    toLocationCode: z.string().nullable(),
    toLocationName: z.string().nullable(),
    sourceType: timelineSourceTypeSchema,
    sourceId: z.string().nullable(),
    sourceReference: z.string().nullable(),
    sourceHidden: z.boolean(),
    supplierId: z.string().nullable(),
    supplierName: z.string().nullable(),
    reason: z.string().nullable(),
    note: z.string().nullable(),
    actorId: z.string(),
    actorName: z.string().nullable(),
  })
  // The item reference matches the item type, and a hidden source carries nothing of it.
  .refine((m) => (m.itemType === 'VEHICLE_UNIT') === (m.vehicleUnit !== null), {
    message: 'a vehicle movement names its unit, a parts movement does not',
    path: ['vehicleUnit'],
  })
  .refine((m) => (m.itemType === 'STOCK_ITEM') === (m.stockItem !== null), {
    message: 'a parts movement names its SKU, a vehicle movement does not',
    path: ['stockItem'],
  })
  .refine((m) => !m.sourceHidden || (m.sourceId === null && m.sourceReference === null), {
    message: 'a hidden source must not carry its reference',
    path: ['sourceReference'],
  })
export type TimelineMovement = z.infer<typeof timelineMovementSchema>

/** Query parameters (all optional; empty values are not sent). from/to are RFC 3339 instants. */
export interface TimelineParams {
  itemType?: TimelineItemType
  vehicleUnitId?: string
  stockItemId?: string
  q?: string
  movementType?: TimelineMovementType
  sourceType?: TimelineSourceType
  sourceReference?: string
  actorId?: string
  locationId?: string
  from?: string
  to?: string
  order?: 'asc' | 'desc'
  page?: number
  limit?: number
}

const timelinePage = pageSchema(timelineMovementSchema)
const base = '/stock-movements'

export function listStockMovements(
  params: TimelineParams = {},
  signal?: AbortSignal,
): Promise<Page<TimelineMovement>> {
  return apiRequest(`${base}${queryString(params)}`, timelinePage, { signal })
}

/** A movement the caller may not see answers 404, like one that does not exist. */
export function getStockMovement(id: string, signal?: AbortSignal): Promise<TimelineMovement> {
  return apiRequest(`${base}/${encodeURIComponent(id)}`, timelineMovementSchema, { signal })
}
