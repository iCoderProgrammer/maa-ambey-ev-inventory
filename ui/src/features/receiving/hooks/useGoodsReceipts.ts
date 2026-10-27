import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { poKeys } from '@/features/purchasing/hooks/usePurchaseOrders'
import {
  getGoodsReceipt,
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
