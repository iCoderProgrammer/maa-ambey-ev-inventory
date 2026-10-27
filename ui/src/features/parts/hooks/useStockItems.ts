import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import {
  getStockItem,
  listCategories,
  listStockItems,
  type CategoryListParams,
  type StockItemListParams,
} from '../api/stockItems'

/**
 * Query keys and hooks for the SKU master (E-03). Mutations refresh every
 * category and SKU query only after the server confirms; nothing is changed in
 * the cache before that.
 */
export const stockItemKeys = {
  all: ['stock-items'] as const,
  categories: (p: CategoryListParams) => [...stockItemKeys.all, 'categories', p] as const,
  list: (p: StockItemListParams) => [...stockItemKeys.all, 'list', p] as const,
  detail: (id: string) => [...stockItemKeys.all, 'detail', id] as const,
}

export function useCategoryList(params: CategoryListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: stockItemKeys.categories(params),
    queryFn: ({ signal }) => listCategories(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useStockItemList(params: StockItemListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: stockItemKeys.list(params),
    queryFn: ({ signal }) => listStockItems(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useStockItem(id: string | undefined) {
  return useQuery({
    queryKey: stockItemKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getStockItem(id ?? '', signal),
    enabled: Boolean(id),
  })
}

export function useStockItemMutation<V, R>(fn: (v: V) => Promise<R>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: stockItemKeys.all }),
  })
}

export function useRefreshStockItems() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: stockItemKeys.all })
}

/**
 * What the server granted (presentation only; the API checks again). Supplier
 * links need suppliers.view to see or set; compatibility needs catalogue.view to set.
 */
export function useStockItemPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.StockItemView),
    manage: p.includes(Permission.StockItemManage),
    archive: p.includes(Permission.StockItemArchive),
    suppliers: p.includes(Permission.SupplierView),
    catalogue: p.includes(Permission.CatalogueView),
  }
}
