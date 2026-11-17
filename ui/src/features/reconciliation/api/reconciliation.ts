import { z } from 'zod'
import { queryString } from '@/features/catalogue/api/catalogue'
import { apiRequest } from '@/lib/api/client'

/**
 * Stock ledger reconciliation (E-06; docs/openapi `GET /stock-reconciliation`,
 * API_CONVENTIONS "Stock ledger reconciliation", ADR-028). Read-only: every call
 * runs the comparison on the server in one read-only database snapshot and
 * returns what disagrees. There is no call that repairs, adjusts or stores
 * anything; corrections go through the approved stock count flow.
 */

const stamp = z.iso.datetime({ offset: true })

/** Exactly the server's discrepancy types (stock.ReconcileTypes). */
export const discrepancyTypeSchema = z.enum([
  'VEHICLE_NO_MOVEMENTS',
  'VEHICLE_MISSING_RECEIPT',
  'VEHICLE_REPEATED_RECEIPT',
  'VEHICLE_STATUS_MISMATCH',
  'VEHICLE_LOCATION_MISMATCH',
  'VEHICLE_NOT_HELD',
  'VEHICLE_BROKEN_CHAIN',
  'PARTS_BALANCE_MISMATCH',
  'PARTS_BALANCE_MISSING',
  'PARTS_NEGATIVE_BALANCE',
  'PARTS_BROKEN_CHAIN',
])
export type DiscrepancyType = z.infer<typeof discrepancyTypeSchema>

export const severitySchema = z.enum(['HIGH', 'MEDIUM'])
export type Severity = z.infer<typeof severitySchema>

export const reconcileItemTypeSchema = z.enum(['VEHICLE_UNIT', 'STOCK_ITEM'])
export type ReconcileItemType = z.infer<typeof reconcileItemTypeSchema>

export const discrepancySchema = z
  .object({
    reference: z.string(),
    itemType: reconcileItemTypeSchema,
    type: discrepancyTypeSchema,
    severity: severitySchema,
    vehicleUnit: z
      .object({ id: z.string(), unitNumber: z.string(), chassisNumber: z.string().nullable() })
      .nullable(),
    stockItem: z
      .object({ id: z.string(), sku: z.string(), name: z.string(), uom: z.string() })
      .nullable(),
    brandId: z.string().nullable(),
    brandName: z.string().nullable(),
    locationId: z.string().nullable(),
    locationCode: z.string().nullable(),
    locationName: z.string().nullable(),
    expected: z.string().nullable(),
    actual: z.string().nullable(),
    explanation: z.string(),
    movementIds: z.array(z.string()),
    detectedAt: stamp,
  })
  .refine((d) => (d.itemType === 'VEHICLE_UNIT') === (d.vehicleUnit !== null), {
    message: 'a vehicle discrepancy names its unit, a parts discrepancy does not',
    path: ['vehicleUnit'],
  })
  .refine((d) => (d.itemType === 'STOCK_ITEM') === (d.stockItem !== null), {
    message: 'a parts discrepancy names its SKU, a vehicle discrepancy does not',
    path: ['stockItem'],
  })
export type Discrepancy = z.infer<typeof discrepancySchema>

const count = z.number().int().nonnegative()

export const reconcileReportSchema = z.object({
  runId: z.string(),
  status: z.literal('COMPLETED'),
  runAt: stamp,
  completedAt: stamp,
  durationMs: count,
  coverage: z.object({
    vehicleUnits: z.boolean(),
    parts: z.boolean(),
    allLocations: z.boolean(),
    locationIds: z.array(z.string()),
  }),
  summary: z.object({
    vehicleUnitsChecked: count,
    partBalancesChecked: count,
    recordsChecked: count,
    movementsChecked: count,
    discrepancies: count,
    byType: z.record(z.string(), count),
    bySeverity: z.record(z.string(), count),
  }),
  items: z.array(discrepancySchema),
  page: z.number().int().positive(),
  limit: z.number().int().positive(),
  total: count,
})
export type ReconcileReport = z.infer<typeof reconcileReportSchema>

/** Filters and paging (all optional; empty values are not sent). The summary always covers the whole run. */
export interface ReconcileParams {
  itemType?: ReconcileItemType
  type?: DiscrepancyType
  severity?: Severity
  locationId?: string
  page?: number
  limit?: number
}

/** Runs the reconciliation now (a GET: it reads and never changes anything). */
export function runReconciliation(
  params: ReconcileParams = {},
  signal?: AbortSignal,
): Promise<ReconcileReport> {
  return apiRequest(`/stock-reconciliation${queryString(params)}`, reconcileReportSchema, {
    signal,
  })
}
