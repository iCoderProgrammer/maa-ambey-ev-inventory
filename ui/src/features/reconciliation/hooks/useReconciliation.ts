import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { runReconciliation, type ReconcileParams } from '../api/reconciliation'

/**
 * Reconciliation runs (E-06). A run starts only when the user asks for it
 * (`enabled`), and is never repeated in the background: no refetch on focus,
 * reconnect or remount (staleTime Infinity) — "Run again" and changed filters
 * start new runs. Nothing here writes. Refused requests (403, 422) are not
 * retried (queryClient defaults).
 */
export const reconciliationKeys = {
  all: ['stock-reconciliation'] as const,
  run: (p: ReconcileParams) => [...reconciliationKeys.all, sent(p)] as const,
}

function sent(p: ReconcileParams): ReconcileParams {
  return Object.fromEntries(
    Object.entries(p).filter(([, v]) => v !== undefined && v !== null && v !== ''),
  )
}

export function useReconciliation(params: ReconcileParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: reconciliationKeys.run(params),
    queryFn: ({ signal }) => runReconciliation(params, signal),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })
}

/**
 * What the server granted (presentation only; the API checks every run and
 * decides every row). The report needs stock.reconciliation.view and
 * vehicle_units.view or stock.parts.view; link flags only decide whether a
 * related screen is offered.
 */
export function useReconciliationPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  const vehicles = p.includes(Permission.VehicleUnitView)
  const parts = p.includes(Permission.PartsView)
  return {
    me,
    view: p.includes(Permission.ReconciliationView) && (vehicles || parts),
    vehicles,
    parts,
    openSKU: p.includes(Permission.StockItemView),
    openMovement: vehicles || parts,
  }
}
