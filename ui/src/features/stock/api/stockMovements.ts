import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { unitStatusSchema } from '@/features/vehicles/api/vehicleUnits'

/**
 * Stock movements (E-01; docs/openapi, API_CONVENTIONS "Stock movements").
 * Every change of a vehicle unit's status is one append-only movement recorded
 * by the server; the client never sets a status or location itself. The API
 * offers only the moves no other workflow owns: putting an AVAILABLE unit on
 * display and taking it off again. PDI moves come with D-06, the RECEIPT
 * movement with receipt posting (D-07), and listing movements with E-02, so
 * there is no call for them here.
 */

const stamp = z.iso.datetime({ offset: true })

/** Movement types the server records (OpenAPI StockMovement.movementType). */
export const movementTypeSchema = z.enum([
  'RECEIPT',
  'PDI_PASS',
  'QUALITY_HOLD',
  'REJECT',
  'DISPLAY_ON',
  'DISPLAY_OFF',
])
export type MovementType = z.infer<typeof movementTypeSchema>

/** One ledger entry. A vehicle unit is one item: quantities are 0 or 1. */
export const stockMovementSchema = z.object({
  id: z.string(),
  itemType: z.literal('VEHICLE_UNIT'),
  vehicleUnitId: z.string(),
  movementType: movementTypeSchema,
  quantityBefore: z.number().int().min(0).max(1),
  quantityAfter: z.number().int().min(0).max(1),
  fromStatus: unitStatusSchema.nullable(),
  toStatus: unitStatusSchema,
  fromLocationId: z.string().nullable(),
  toLocationId: z.string().nullable(),
  sourceType: z.enum(['GOODS_RECEIPT', 'STAFF_ACTION']),
  sourceId: z.string().nullable(),
  sourceReference: z.string().nullable(),
  actorId: z.string(),
  note: z.string().nullable(),
  createdAt: stamp,
})
export type StockMovement = z.infer<typeof stockMovementSchema>

/** The unit as it is now (on a replay it may have moved on since the movement). */
export const stockUnitStateSchema = z.object({
  id: z.string(),
  unitNumber: z.string(),
  status: unitStatusSchema,
  locationId: z.string().nullable(),
  version: z.number().int().positive(),
})
export type StockUnitState = z.infer<typeof stockUnitStateSchema>

export const stockMoveResultSchema = z
  .object({ movement: stockMovementSchema, unit: stockUnitStateSchema })
  .refine((r) => r.movement.vehicleUnitId === r.unit.id, {
    message: 'the movement must belong to the unit returned with it',
    path: ['unit', 'id'],
  })
export type StockMoveResult = z.infer<typeof stockMoveResultSchema>

/** The two display moves and the lifecycle edge each records (System Architecture §6). */
export const DisplayMove = {
  PutOn: { path: 'put-on-display', movementType: 'DISPLAY_ON', from: 'AVAILABLE', to: 'DISPLAY' },
  TakeOff: {
    path: 'take-off-display',
    movementType: 'DISPLAY_OFF',
    from: 'DISPLAY',
    to: 'AVAILABLE',
  },
} as const
export type DisplayMoveKind = keyof typeof DisplayMove

/** Same rule as the server: 8–100 letters, digits, - or _. */
const idempotencyKeySchema = z.string().regex(/^[A-Za-z0-9_-]{8,100}$/)

/**
 * One logical display action: the unit, the version the user saw, an optional
 * note and the idempotency key that identifies the action. Create it once with
 * newDisplayAction when the user decides, and send that same object again to
 * retry; a new decision gets a new action (and so a new key).
 */
export interface DisplayAction {
  readonly kind: DisplayMoveKind
  readonly unitId: string
  readonly version: number
  readonly note: string | null
  readonly idempotencyKey: string
}

const displayActionSchema = z.object({
  kind: z.enum(['PutOn', 'TakeOff']),
  unitId: z.string().min(1),
  version: z.number().int().positive(),
  note: z.string().max(500).nullable(),
  idempotencyKey: idempotencyKeySchema,
})

/** Starts a new logical action with its own idempotency key (frozen, so it cannot drift). */
export function newDisplayAction(
  kind: DisplayMoveKind,
  unitId: string,
  version: number,
  note?: string | null,
): DisplayAction {
  const trimmed = note?.trim()
  return Object.freeze({
    kind,
    unitId,
    version,
    note: trimmed ? trimmed : null,
    idempotencyKey: crypto.randomUUID(),
  })
}

/**
 * Sends a display action. The server derives the movement type, the starting
 * state, the actor and the time; the body carries only the version and note.
 * A replay of the same action answers 200 with the first movement (and the
 * unit as it is now). The answer must be the movement this action asked for.
 */
export async function sendDisplayAction(action: DisplayAction): Promise<StockMoveResult> {
  const a = displayActionSchema.parse(action)
  const move = DisplayMove[a.kind]
  const expected = stockMoveResultSchema.refine(
    (r) => r.movement.movementType === move.movementType && r.movement.vehicleUnitId === a.unitId,
    { message: `expected a ${move.movementType} movement of this unit`, path: ['movement'] },
  )
  return apiRequest(`/vehicle-units/${encodeURIComponent(a.unitId)}/${move.path}`, expected, {
    method: 'POST',
    body: a.note === null ? { version: a.version } : { version: a.version, note: a.note },
    headers: { 'Idempotency-Key': a.idempotencyKey },
  })
}
