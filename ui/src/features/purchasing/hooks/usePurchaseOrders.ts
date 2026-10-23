import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getPOSettings,
  getPurchaseOrder,
  listPurchaseOrders,
  type PODetail,
  type POListParams,
} from '../api/purchaseOrders'

/** Query keys and hooks for purchase orders (D-02). */
export const poKeys = {
  all: ['purchase-orders'] as const,
  list: (p: POListParams) => [...poKeys.all, 'list', p] as const,
  detail: (id: string) => [...poKeys.all, 'detail', id] as const,
  settings: () => [...poKeys.all, 'settings'] as const,
}

export function usePurchaseOrderList(params: POListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: poKeys.list(params),
    queryFn: ({ signal }) => listPurchaseOrders(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function usePurchaseOrder(id: string | undefined) {
  return useQuery({
    queryKey: poKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getPurchaseOrder(id ?? '', signal),
    enabled: Boolean(id),
  })
}

export function usePOSettings({ enabled = true } = {}) {
  return useQuery({
    queryKey: poKeys.settings(),
    queryFn: ({ signal }) => getPOSettings(signal),
    enabled,
  })
}

/**
 * A PO mutation: the server's answer (the whole PO) replaces the cached detail
 * only after it confirms, then lists refresh.
 */
export function usePOMutation<V>(fn: (v: V) => Promise<PODetail>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (d) => {
      queryClient.setQueryData(poKeys.detail(d.id), d)
      return queryClient.invalidateQueries({ queryKey: [...poKeys.all, 'list'] })
    },
  })
}

export function useRefreshPurchaseOrders() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: poKeys.all })
}
