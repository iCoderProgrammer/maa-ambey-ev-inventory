import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { getVehicleUnit, listVehicleUnits, type VehicleUnitListParams } from '../api/vehicleUnits'

/**
 * Query keys and hooks for vehicle units (D-04). Read-only: there are no
 * mutations. Refused requests (403, 404, 422) are not retried (queryClient
 * defaults) and nothing falls back to a broader request.
 */
export const vehicleUnitKeys = {
  all: ['vehicle-units'] as const,
  list: (p: VehicleUnitListParams) => [...vehicleUnitKeys.all, 'list', listKey(p)] as const,
  detail: (id: string) => [...vehicleUnitKeys.all, 'detail', id] as const,
}

/** The parameters that reach the server, so "" and undefined share one cache entry. */
function listKey(p: VehicleUnitListParams): VehicleUnitListParams {
  return Object.fromEntries(
    Object.entries(p).filter(([, v]) => v !== undefined && v !== null && v !== ''),
  )
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

/** What the server granted for vehicle units (presentation only; the API checks again). */
export function useVehicleUnitPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    view: p.includes(Permission.VehicleUnitView),
    viewCost: p.includes(Permission.VehicleUnitCostView),
  }
}
