import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { getStockMovement, listStockMovements, type TimelineParams } from '../api/timeline'

/**
 * Stock movement timeline queries (E-02). Read-only: there is no mutation, and
 * nothing here writes to the ledger. Refused requests (403, 404, 422) are not
 * retried (queryClient defaults).
 */
export const timelineKeys = {
  all: ['stock-movements'] as const,
  list: (p: TimelineParams) => [...timelineKeys.all, 'list', sent(p)] as const,
  detail: (id: string) => [...timelineKeys.all, 'detail', id] as const,
}

/** Only the parameters that reach the server, so "" and undefined share one cache entry. */
function sent(p: TimelineParams): TimelineParams {
  return Object.fromEntries(
    Object.entries(p).filter(([, v]) => v !== undefined && v !== null && v !== ''),
  )
}

export function useStockMovementTimeline(params: TimelineParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: timelineKeys.list(params),
    queryFn: ({ signal }) => listStockMovements(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useStockMovement(id: string | undefined, { enabled = true } = {}) {
  return useQuery({
    queryKey: timelineKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getStockMovement(id ?? '', signal),
    enabled: enabled && Boolean(id),
  })
}

/**
 * What the server granted (presentation only; the API decides every row and
 * shapes every source again). The timeline needs vehicle_units.view or
 * stock.parts.view; the link flags only decide whether a related screen is
 * offered — the server still hides sources the caller may not open.
 */
export function useTimelinePerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  const vehicles = p.includes(Permission.VehicleUnitView)
  const parts = p.includes(Permission.PartsView)
  return {
    me,
    meId: me.data?.user.id,
    view: vehicles || parts,
    vehicles,
    parts,
    openGRN: p.includes(Permission.GoodsReceiptView),
    openCount: p.includes(Permission.AdjustmentView),
    openSKU: p.includes(Permission.StockItemView),
    openSupplier: p.includes(Permission.SupplierView),
  }
}
