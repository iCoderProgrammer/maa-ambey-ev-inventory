import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import {
  getPartsBalances,
  listPartsMovements,
  sendPartsAction,
  type PartsAction,
  type PartsHistoryParams,
} from '../api/partsStock'

/**
 * Query keys and hooks for parts stock (E-04). Nothing is changed in the cache
 * before the server confirms: after a receipt or issue the SKU's balances and
 * history are refetched; after INSUFFICIENT_STOCK or an unreadable 2xx answer the
 * balances are refetched too. Mutations are not retried automatically.
 */
export const partsStockKeys = {
  all: ['parts-stock'] as const,
  item: (itemId: string) => [...partsStockKeys.all, itemId] as const,
  balances: (itemId: string) => [...partsStockKeys.item(itemId), 'balances'] as const,
  history: (itemId: string, p: PartsHistoryParams) =>
    [...partsStockKeys.item(itemId), 'history', p] as const,
}

export function usePartsBalances(itemId: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: partsStockKeys.balances(itemId),
    queryFn: ({ signal }) => getPartsBalances(itemId, signal),
    enabled: enabled && Boolean(itemId),
  })
}

export function usePartsHistory(
  itemId: string,
  params: PartsHistoryParams,
  { enabled = true } = {},
) {
  return useQuery({
    queryKey: partsStockKeys.history(itemId, params),
    queryFn: ({ signal }) => listPartsMovements(itemId, params, signal),
    enabled: enabled && Boolean(itemId),
    placeholderData: keepPreviousData,
  })
}

/** Posts a frozen PartsAction; retrying means sending the same action again. */
export function usePartsPost(itemId: string) {
  const queryClient = useQueryClient()
  const refresh = () => queryClient.invalidateQueries({ queryKey: partsStockKeys.item(itemId) })
  return useMutation({
    mutationFn: (action: PartsAction) => sendPartsAction(action),
    onSuccess: refresh,
    onError: async (error) => {
      if (
        error instanceof ApiError &&
        (error.code === 'INSUFFICIENT_STOCK' || error.code === ClientErrorCode.InvalidResponse)
      ) {
        await refresh()
      }
    },
  })
}

/** What the server granted (presentation only; the API checks again). */
export function usePartsStockPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  const supplierView = p.includes(Permission.SupplierView)
  return {
    me,
    view: p.includes(Permission.PartsView),
    // A receipt names a supplier, so the server also needs suppliers.view.
    receive: p.includes(Permission.PartsReceive) && supplierView,
    receiveWithoutSuppliers: p.includes(Permission.PartsReceive) && !supplierView,
    issue: p.includes(Permission.PartsIssue),
    supplierView,
    /** The locations the caller may work at (GET /me; no location list API exists yet). */
    locations: me.data?.locationScope.locations ?? [],
  }
}
