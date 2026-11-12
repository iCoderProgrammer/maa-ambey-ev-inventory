import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { pdiKeys } from '@/features/pdi/hooks/usePdi'
import { poKeys } from '@/features/purchasing/hooks/usePurchaseOrders'
import { vehicleUnitKeys } from '@/features/vehicles/hooks/useVehicleUnits'
import {
  getGoodsReceipt,
  postGoodsReceipt,
  type PostAction,
  listGoodsReceipts,
  listReceivablePurchaseOrders,
  type GRNDetail,
  type GRNListParams,
} from '../api/goodsReceipts'

/** Query keys and hooks for goods receipts (D-03). */
export const grnKeys = {
  all: ['goods-receipts'] as const,
  list: (p: GRNListParams) => [...grnKeys.all, 'list', p] as const,
  detail: (id: string) => [...grnKeys.all, 'detail', id] as const,
  receivable: (p: { q?: string }) => [...grnKeys.all, 'receivable', p] as const,
}

export function useGoodsReceiptList(params: GRNListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: grnKeys.list(params),
    queryFn: ({ signal }) => listGoodsReceipts(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useGoodsReceipt(id: string | undefined) {
  return useQuery({
    queryKey: grnKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getGoodsReceipt(id ?? '', signal),
    enabled: Boolean(id),
  })
}

export function useReceivablePurchaseOrders(q: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: grnKeys.receivable({ q }),
    queryFn: ({ signal }) => listReceivablePurchaseOrders({ q: q || undefined, limit: 25 }, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

/**
 * A receipt mutation: the server's answer (the whole receipt) replaces the
 * cached detail only after it confirms; lists and receivable POs refresh (other
 * drafts change what is available), and so do purchase orders.
 */
export function useGRNMutation<V>(fn: (v: V) => Promise<GRNDetail>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: async (d) => {
      queryClient.setQueryData(grnKeys.detail(d.id), d)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [...grnKeys.all, 'list'] }),
        queryClient.invalidateQueries({ queryKey: [...grnKeys.all, 'receivable'] }),
        queryClient.invalidateQueries({ queryKey: poKeys.all }),
      ])
    },
  })
}

export function useRefreshGoodsReceipts() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: grnKeys.all })
}

/**
 * Posting (D-07). Nothing in the cache changes before the server confirms: on
 * success the posted receipt replaces the cached detail and every query the
 * posting affects refreshes (receipts, purchase orders, vehicle units, PDI
 * queues). On any failure the receipt refreshes too, because it may have moved
 * on (posted by someone else, changed, cancelled). Not retried automatically.
 */
export function usePostGoodsReceipt() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (a: PostAction) => postGoodsReceipt(a),
    onSuccess: async (r) => {
      queryClient.setQueryData(grnKeys.detail(r.receipt.id), r.receipt)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [...grnKeys.all, 'list'] }),
        queryClient.invalidateQueries({ queryKey: [...grnKeys.all, 'receivable'] }),
        queryClient.invalidateQueries({ queryKey: poKeys.all }),
        queryClient.invalidateQueries({ queryKey: vehicleUnitKeys.all }),
        queryClient.invalidateQueries({ queryKey: pdiKeys.all }),
      ])
    },
    onError: (_e, a) => queryClient.invalidateQueries({ queryKey: grnKeys.detail(a.grnId) }),
  })
}
