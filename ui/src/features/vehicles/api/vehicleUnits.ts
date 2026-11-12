import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'

/**
 * Vehicle unit registry (D-04; docs/openapi, API_CONVENTIONS "Vehicle units").
 * Read-only: units are created only by receipt posting (D-07) and change only
 * through the stock movement service (E-01), so there is no create, edit or
 * delete call here. A goods receipt draft's captured units are not vehicle
 * units. The cost comes only with vehicle_units.cost.view: otherwise the server
 * sends unitCost/costCurrency null and costHidden true.
 */

const stamp = z.iso.datetime({ offset: true })

/** The approved lifecycle statuses (INV-005), as the server's OpenAPI enum lists them. */
export const unitStatusSchema = z.enum([
  'ORDERED',
  'IN_TRANSIT',
  'RECEIVED_PENDING_PDI',
  'QUALITY_HOLD',
  'AVAILABLE',
  'DISPLAY',
  'RESERVED',
  'SERVICE_HOLD',
  'SOLD',
  'DELIVERED',
  'REJECTED',
  'RETURN_PENDING_INSPECTION',
  'RETURNED_TO_SUPPLIER',
])
export type VehicleUnitStatus = z.infer<typeof unitStatusSchema>

/** Where a unit came from; GOODS_RECEIPT is the only source so far. */
export const unitSourceTypeSchema = z.enum(['GOODS_RECEIPT'])

export const vehicleUnitSchema = z
  .object({
    id: z.string(),
    unitNumber: z.string(),
    status: unitStatusSchema,
    brandId: z.string(),
    brandName: z.string(),
    modelId: z.string(),
    modelName: z.string(),
    variantId: z.string(),
    variantName: z.string(),
    /** Identifiers as captured; none is mandatory while the unit is pending PDI. */
    chassisNumber: z.string().nullable(),
    /** Trimmed, upper-cased: the form searches and duplicate checks compare. */
    chassisNormalised: z.string().nullable(),
    motorSerial: z.string().nullable(),
    batterySerial: z.string().nullable(),
    chargerSerial: z.string().nullable(),
    /** Null until receipts carry a location (OQ-24). */
    locationId: z.string().nullable(),
    locationCode: z.string().nullable(),
    locationName: z.string().nullable(),
    sourceType: unitSourceTypeSchema,
    supplierId: z.string().nullable(),
    supplierName: z.string().nullable(),
    receivedDate: z.iso.date(),
    /** Whole days since receivedDate (server date). Ageing policy/buckets are not defined here (E-07). */
    ageDays: z.number().int().min(0),
    /** Acquisition cost excluding tax (decimal string); null when hidden. */
    unitCost: z
      .string()
      .regex(/^\d+\.\d{2}$/)
      .nullable(),
    costCurrency: z.literal('INR').nullable(),
    costHidden: z.boolean(),
    version: z.number().int().positive(),
    createdAt: stamp,
    updatedAt: stamp,
  })
  // A hidden cost carries no amount: never trust one that does.
  .refine((u) => !u.costHidden || (u.unitCost === null && u.costCurrency === null), {
    message: 'a hidden cost must not carry an amount',
    path: ['unitCost'],
  })
export type VehicleUnit = z.infer<typeof vehicleUnitSchema>

export const unitSourceSchema = z.object({
  type: unitSourceTypeSchema,
  supplierId: z.string().nullable(),
  supplierName: z.string().nullable(),
  purchaseOrderId: z.string().nullable(),
  purchaseOrderNumber: z.string().nullable(),
  purchaseOrderLineId: z.string().nullable(),
  goodsReceiptId: z.string().nullable(),
  goodsReceiptNumber: z.string().nullable(),
  goodsReceiptLineId: z.string().nullable(),
  goodsReceiptUnitId: z.string().nullable(),
  supplierReference: z.string().nullable(),
  receivedDate: z.iso.date(),
})
export type VehicleUnitSource = z.infer<typeof unitSourceSchema>

/** One entry of the unit's append-only history, naming its source document. */
export const unitEventSchema = z.object({
  id: z.string(),
  action: z.enum(['REGISTERED']),
  toStatus: unitStatusSchema,
  sourceType: unitSourceTypeSchema,
  sourceId: z.string(),
  /** Human reference of the source document, e.g. the GRN number. */
  sourceReference: z.string(),
  actorId: z.string(),
  actorName: z.string().nullable(),
  note: z.string().nullable(),
  createdAt: stamp,
})
export type VehicleUnitEvent = z.infer<typeof unitEventSchema>

/**
 * Metadata of a private file about the unit (D-08): a document of its goods
 * receipt or evidence of one of its PDI inspections. The server lists only files
 * the caller may open; the file itself is fetched through the attachment API,
 * which checks again. No storage key or URL is sent.
 */
export const unitDocumentSchema = z.object({
  attachmentId: z.string(),
  sourceType: z.enum(['GOODS_RECEIPT', 'PDI_INSPECTION']),
  sourceId: z.string(),
  /** GRN or PDI inspection number. */
  sourceReference: z.string(),
  documentType: z.enum(['DELIVERY_CHALLAN', 'SUPPLIER_INVOICE', 'PHOTO', 'OTHER', 'PDI_EVIDENCE']),
  fileName: z.string(),
  contentType: z.string(),
  sizeBytes: z.number().int().positive(),
  createdAt: stamp,
})
export type VehicleUnitDocument = z.infer<typeof unitDocumentSchema>

export const vehicleUnitDetailSchema = vehicleUnitSchema
  .and(
    z.object({
      source: unitSourceSchema,
      events: z.array(unitEventSchema),
      documents: z.array(unitDocumentSchema),
      /** True when the caller may not open attachments at all. */
      documentsHidden: z.boolean(),
    }),
  )
  // Hidden documents carry no file details: never trust an answer that does.
  .refine((d) => !d.documentsHidden || d.documents.length === 0, {
    message: 'hidden documents must not list files',
    path: ['documents'],
  })
export type VehicleUnitDetail = z.infer<typeof vehicleUnitDetailSchema>

/** GET /vehicle-units query parameters (all optional; limit 1–100, default 25). */
export interface VehicleUnitListParams {
  /** Unit number or any identifier; case-insensitive, part of a value is enough. */
  q?: string
  status?: VehicleUnitStatus
  brandId?: string
  variantId?: string
  supplierId?: string
  purchaseOrderId?: string
  goodsReceiptId?: string
  /** Must be in the caller's location scope, or the server answers 422 on locationId. */
  locationId?: string
  /** true: AVAILABLE units only (saleable, INV-006); false: every other status (D-06). */
  saleable?: boolean
  /** D-08 filters: model, inclusive received dates (YYYY-MM-DD), inclusive age in days. */
  modelId?: string
  receivedFrom?: string
  receivedTo?: string
  /** Whole days; a non-numeric value is sent as typed so the server reports it (422). */
  minAgeDays?: number | string
  maxAgeDays?: number | string
  sort?: VehicleUnitSort
  page?: number
  limit?: number
}

/** The server's list orders; every one ends with the unit ID, so pages are stable. */
export const vehicleUnitSorts = [
  '-createdAt',
  'createdAt',
  '-receivedDate',
  'receivedDate',
  'unitNumber',
  '-unitNumber',
] as const
export type VehicleUnitSort = (typeof vehicleUnitSorts)[number]

const unitPage = pageSchema(vehicleUnitSchema)
const base = '/vehicle-units'

export function listVehicleUnits(
  params: VehicleUnitListParams = {},
  signal?: AbortSignal,
): Promise<Page<VehicleUnit>> {
  return apiRequest(`${base}${queryString(params)}`, unitPage, { signal })
}

/** A unit outside the caller's location scope answers 404, like a unit that does not exist. */
export function getVehicleUnit(unitId: string, signal?: AbortSignal): Promise<VehicleUnitDetail> {
  return apiRequest(`${base}/${encodeURIComponent(unitId)}`, vehicleUnitDetailSchema, { signal })
}

/** Movement types of a vehicle unit's ledger entries (E-01; PDI moves D-06). */
export const unitMovementTypeSchema = z.enum([
  'RECEIPT',
  'PDI_PASS',
  'QUALITY_HOLD',
  'REJECT',
  'DISPLAY_ON',
  'DISPLAY_OFF',
])
export type VehicleUnitMovementType = z.infer<typeof unitMovementTypeSchema>

/**
 * One read-only entry of the unit's stock movement ledger (D-08,
 * GET /vehicle-units/{id}/stock-movements). Location code and name are null for
 * a location outside the caller's scope. No cost.
 */
export const unitMovementSchema = z.object({
  id: z.string(),
  movementType: unitMovementTypeSchema,
  fromStatus: unitStatusSchema.nullable(),
  toStatus: unitStatusSchema,
  fromLocationId: z.string().nullable(),
  fromLocationCode: z.string().nullable(),
  fromLocationName: z.string().nullable(),
  toLocationId: z.string().nullable(),
  toLocationCode: z.string().nullable(),
  toLocationName: z.string().nullable(),
  sourceType: z.enum(['GOODS_RECEIPT', 'PDI_INSPECTION', 'STAFF_ACTION']),
  sourceId: z.string().nullable(),
  sourceReference: z.string().nullable(),
  actorId: z.string(),
  actorName: z.string().nullable(),
  note: z.string().nullable(),
  createdAt: stamp,
})
export type VehicleUnitMovement = z.infer<typeof unitMovementSchema>

export interface VehicleUnitMovementParams {
  /** asc (default): oldest first, in ledger order; desc: newest first. */
  order?: 'asc' | 'desc'
  page?: number
  /** 1–100, default 50. */
  limit?: number
}

const movementPage = pageSchema(unitMovementSchema)

/** A unit outside the caller's scope answers 404. */
export function listVehicleUnitMovements(
  unitId: string,
  params: VehicleUnitMovementParams = {},
  signal?: AbortSignal,
): Promise<Page<VehicleUnitMovement>> {
  return apiRequest(
    `${base}/${encodeURIComponent(unitId)}/stock-movements${queryString(params)}`,
    movementPage,
    { signal },
  )
}
