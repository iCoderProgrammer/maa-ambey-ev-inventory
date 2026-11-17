import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import {
  getAgeingConfiguration,
  getAgeingSummary,
  getVehicleUnit,
  listVehicleUnitMovements,
  listVehicleUnits,
  type AgeingSummaryParams,
  type VehicleUnitListParams,
  type VehicleUnitMovementParams,
} from '../api/vehicleUnits'

/**
 * Query keys and hooks for vehicle units (D-04). Read-only: there are no
 * mutations. Refused requests (403, 404, 422) are not retried (queryClient
 * defaults) and nothing falls back to a broader request.
 */
export const vehicleUnitKeys = {
  all: ['vehicle-units'] as const,
  list: (p: VehicleUnitListParams) => [...vehicleUnitKeys.all, 'list', listKey(p)] as const,
  detail: (id: string) => [...vehicleUnitKeys.all, 'detail', id] as const,
  movements: (id: string, p: VehicleUnitMovementParams) =>
    [...vehicleUnitKeys.all, 'movements', id, p] as const,
  ageingSummary: (p: AgeingSummaryParams) =>
    [...vehicleUnitKeys.all, 'ageing-summary', listKey(p)] as const,
  ageingConfiguration: ['stock-ageing', 'configuration'] as const,
}

/** The parameters that reach the server, so "" and undefined share one cache entry. */
function listKey<T extends object>(p: T): Partial<T> {
  return Object.fromEntries(
    Object.entries(p).filter(([, v]) => v !== undefined && v !== null && v !== ''),
  ) as Partial<T>
}

export function useVehicleUnitList(params: VehicleUnitListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: vehicleUnitKeys.list(params),
    queryFn: ({ signal }) => listVehicleUnits(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useVehicleUnit(id: string | undefined, { enabled = true } = {}) {
  return useQuery({
    queryKey: vehicleUnitKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getVehicleUnit(id ?? '', signal),
    enabled: enabled && Boolean(id),
  })
}

/** One page of a unit's stock movements (read-only ledger, D-08). */
export function useVehicleUnitMovements(
  id: string | undefined,
  params: VehicleUnitMovementParams,
  { enabled = true } = {},
) {
  return useQuery({
    queryKey: vehicleUnitKeys.movements(id ?? '', params),
    queryFn: ({ signal }) => listVehicleUnitMovements(id ?? '', params, signal),
    enabled: enabled && Boolean(id),
    placeholderData: keepPreviousData,
  })
}

/** Ageing counts of the units the filters select (E-07; server-calculated). */
export function useAgeingSummary(params: AgeingSummaryParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: vehicleUnitKeys.ageingSummary(params),
    queryFn: ({ signal }) => getAgeingSummary(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

/** The current ageing band configuration (E-07). */
export function useAgeingConfiguration({ enabled = true } = {}) {
  return useQuery({
    queryKey: vehicleUnitKeys.ageingConfiguration,
    queryFn: ({ signal }) => getAgeingConfiguration(signal),
    enabled,
  })
}

/** What the server granted for vehicle units (presentation only; the API checks again). */
export function useVehicleUnitPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    view: p.includes(Permission.VehicleUnitView),
    viewCost: p.includes(Permission.VehicleUnitCostView),
    viewPO: p.includes(Permission.PurchaseOrderView),
    viewGRN: p.includes(Permission.GoodsReceiptView),
    viewSuppliers: p.includes(Permission.SupplierView),
    viewCatalogue: p.includes(Permission.CatalogueView),
    attachmentsView: p.includes(Permission.AttachmentsView),
    pdiRecord: p.includes(Permission.PDIRecord),
  }
}
