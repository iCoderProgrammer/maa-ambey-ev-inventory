import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'
import { unitStatusSchema } from '@/features/vehicles/api/vehicleUnits'

/**
 * PDI checklist and disposition (D-06; docs/openapi, API_CONVENTIONS "PDI
 * checklist and disposition"). An inspection is recorded complete in one
 * request; the server applies the disposition through the stock movement
 * ledger and answers with the inspection and the unit as it is now. Nothing
 * here decides a status: the client sends outcomes, notes, evidence and the
 * chosen disposition, and the server checks every rule again.
 */

const stamp = z.iso.datetime({ offset: true })

export const dispositionSchema = z.enum(['PASS', 'HOLD', 'REJECT'])
export type Disposition = z.infer<typeof dispositionSchema>

export const outcomeSchema = z.enum(['PASS', 'FAIL'])
export type Outcome = z.infer<typeof outcomeSchema>

export const resultSchema = z.enum(['ALL_PASSED', 'HAS_FAILURES', 'NOT_CHECKED'])
export type InspectionResult = z.infer<typeof resultSchema>

export const checklistStatusSchema = z.enum(['ACTIVE', 'RETIRED'])

export const checklistItemSchema = z.object({
  id: z.string(),
  code: z.string(),
  label: z.string(),
  description: z.string().nullable(),
  /** Null = applies to units of every brand. */
  brandId: z.string().nullable(),
  brandName: z.string().nullable(),
  sortOrder: z.number().int(),
  status: checklistStatusSchema,
  version: z.number().int().positive(),
  createdBy: z.string(),
  createdAt: stamp,
  retiredBy: z.string().nullable(),
  retiredAt: stamp.nullable(),
  retireReason: z.string().nullable(),
})
export type ChecklistItem = z.infer<typeof checklistItemSchema>

export const inspectionSchema = z.object({
  id: z.string(),
  inspectionNumber: z.string(),
  vehicleUnitId: z.string(),
  unitNumber: z.string(),
  disposition: dispositionSchema,
  result: resultSchema,
  fromStatus: unitStatusSchema,
  toStatus: unitStatusSchema,
  locationId: z.string().nullable(),
  notes: z.string().nullable(),
  movementId: z.string(),
  inspectedBy: z.string(),
  inspectedByName: z.string().nullable(),
  inspectedAt: stamp,
  itemCount: z.number().int().min(0),
  failedCount: z.number().int().min(0),
  evidenceCount: z.number().int().min(0),
})
export type Inspection = z.infer<typeof inspectionSchema>

export const itemOutcomeSchema = z.object({
  lineNo: z.number().int().positive(),
  checklistItemId: z.string(),
  code: z.string(),
  label: z.string(),
  outcome: outcomeSchema,
  note: z.string().nullable(),
})
export type ItemOutcome = z.infer<typeof itemOutcomeSchema>

export const evidenceSchema = z.object({
  attachmentId: z.string(),
  originalName: z.string(),
  contentType: z.string(),
  sizeBytes: z.number().int().positive(),
})
export type Evidence = z.infer<typeof evidenceSchema>

export const inspectionDetailSchema = inspectionSchema
  .extend({
    items: z.array(itemOutcomeSchema),
    evidence: z.array(evidenceSchema),
    evidenceHidden: z.boolean(),
  })
  // Hidden evidence carries no file details: never trust an answer that does.
  .refine((d) => !d.evidenceHidden || d.evidence.length === 0, {
    message: 'hidden evidence must not list files',
    path: ['evidence'],
  })
export type InspectionDetail = z.infer<typeof inspectionDetailSchema>

export const unitStateSchema = z.object({
  id: z.string(),
  unitNumber: z.string(),
  status: unitStatusSchema,
  brandId: z.string(),
  locationId: z.string().nullable(),
  version: z.number().int().positive(),
  /** AVAILABLE only (INV-006/007): held, rejected and pending units are not saleable. */
  saleable: z.boolean(),
})
export type PdiUnitState = z.infer<typeof unitStateSchema>

export const overviewSchema = z.object({
  unit: unitStateSchema,
  /** What the unit's status admits; empty = no inspection can be recorded now. */
  allowedDispositions: z.array(dispositionSchema),
  /** The caller holds vehicle_units.pdi.record (the server checks again). */
  canRecord: z.boolean(),
  /** Active items that apply to the unit, in order; empty = not configured. */
  checklist: z.array(checklistItemSchema),
  inspections: z.array(inspectionSchema),
})
export type PdiOverview = z.infer<typeof overviewSchema>

export const recordResultSchema = z
  .object({ inspection: inspectionDetailSchema, unit: unitStateSchema })
  .refine((r) => r.inspection.vehicleUnitId === r.unit.id, {
    message: 'the inspection must belong to the unit returned with it',
    path: ['unit', 'id'],
  })
export type RecordResult = z.infer<typeof recordResultSchema>

/** The only edge each disposition may take (System Architecture §6; OQ-31 keeps the rest closed). */
export const dispositionEdges: Record<Disposition, { from: string; to: string }> = {
  PASS: { from: 'RECEIVED_PENDING_PDI', to: 'AVAILABLE' },
  HOLD: { from: 'RECEIVED_PENDING_PDI', to: 'QUALITY_HOLD' },
  REJECT: { from: 'QUALITY_HOLD', to: 'REJECTED' },
}

const id = (s: string) => encodeURIComponent(s)

export function getPdiOverview(unitId: string, signal?: AbortSignal): Promise<PdiOverview> {
  return apiRequest(
    `/vehicle-units/${id(unitId)}/pdi`,
    overviewSchema.refine((o) => o.unit.id === unitId, { message: 'expected the requested unit' }),
    { signal },
  )
}

export function getInspection(
  inspectionId: string,
  signal?: AbortSignal,
): Promise<InspectionDetail> {
  return apiRequest(`/pdi-inspections/${id(inspectionId)}`, inspectionDetailSchema, { signal })
}

export interface ChecklistListParams {
  status?: 'ACTIVE' | 'RETIRED'
  /** Items that apply to this brand (every-brand items and its own). */
  brandId?: string
  page?: number
  limit?: number
}

const checklistPage = pageSchema(checklistItemSchema)

export function listChecklist(
  params: ChecklistListParams = {},
  signal?: AbortSignal,
): Promise<Page<ChecklistItem>> {
  return apiRequest(`/pdi-checklist-items${queryString(params)}`, checklistPage, { signal })
}

export interface ChecklistCreateBody {
  code: string
  label: string
  description?: string
  brandId?: string
  sortOrder?: number
}

export function createChecklistItem(body: ChecklistCreateBody): Promise<ChecklistItem> {
  return apiRequest(
    '/pdi-checklist-items',
    checklistItemSchema.refine((c) => c.status === 'ACTIVE', {
      message: 'expected an active item',
    }),
    { method: 'POST', body },
  )
}

export function retireChecklistItem(
  itemId: string,
  body: { version: number; reason: string },
): Promise<ChecklistItem> {
  return apiRequest(
    `/pdi-checklist-items/${id(itemId)}/retire`,
    checklistItemSchema.refine((c) => c.id === itemId && c.status === 'RETIRED', {
      message: 'expected the retired item',
    }),
    { method: 'POST', body },
  )
}

export interface InspectionBody {
  version: number
  disposition: Disposition
  items: { checklistItemId: string; outcome: Outcome; note?: string }[]
  notes?: string
  attachmentIds: string[]
}

/**
 * One inspection with its Idempotency-Key, made once per confirmation and
 * frozen; a retry after an unknown outcome sends the same action, so the
 * server records it at most once.
 */
export interface InspectionAction {
  readonly unitId: string
  readonly body: Readonly<InspectionBody>
  readonly idempotencyKey: string
}

export function newInspectionAction(unitId: string, body: InspectionBody): InspectionAction {
  return Object.freeze({
    unitId,
    body: Object.freeze({
      ...body,
      items: body.items.map((i) => Object.freeze({ ...i })),
      attachmentIds: [...body.attachmentIds],
    }),
    idempotencyKey: crypto.randomUUID(),
  })
}

/** Whether a new attempt is the same request as an earlier action. */
export function sameInspection(a: InspectionAction, unitId: string, body: InspectionBody): boolean {
  const norm = (b: InspectionBody) =>
    JSON.stringify({
      ...b,
      notes: b.notes ?? null,
      items: [...b.items]
        .map((i) => ({ ...i, note: i.note ?? null }))
        .sort((x, y) => x.checklistItemId.localeCompare(y.checklistItemId)),
      attachmentIds: [...b.attachmentIds].sort(),
    })
  return a.unitId === unitId && norm(a.body) === norm(body)
}

/** POST /vehicle-units/{id}/pdi; the answer must be this disposition's inspection of this unit. */
export function sendInspection(action: InspectionAction): Promise<RecordResult> {
  const edge = dispositionEdges[action.body.disposition]
  return apiRequest(
    `/vehicle-units/${id(action.unitId)}/pdi`,
    recordResultSchema.refine(
      (r) =>
        r.unit.id === action.unitId &&
        r.inspection.disposition === action.body.disposition &&
        r.inspection.toStatus === edge.to,
      { message: 'expected the recorded inspection' },
    ),
    {
      method: 'POST',
      body: action.body,
      headers: { 'Idempotency-Key': action.idempotencyKey },
    },
  )
}
