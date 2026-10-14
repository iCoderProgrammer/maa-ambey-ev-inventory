import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { catalogueKeys } from '@/features/catalogue/hooks/useCatalogue'
import {
  getSupplier,
  listBrandLinks,
  listSuppliers,
  type SupplierListParams,
} from '../api/suppliers'

/** Query keys and hooks for the supplier master (D-01). */
export const supplierKeys = {
  all: ['suppliers'] as const,
  list: (p: SupplierListParams) => [...supplierKeys.all, 'list', p] as const,
  detail: (id: string) => [...supplierKeys.all, 'detail', id] as const,
  links: (p: object) => [...supplierKeys.all, 'links', p] as const,
}

export function useSupplierList(params: SupplierListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: supplierKeys.list(params),
    queryFn: ({ signal }) => listSuppliers(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useSupplier(id: string | undefined) {
  return useQuery({
    queryKey: supplierKeys.detail(id ?? ''),
    queryFn: ({ signal }) => getSupplier(id ?? '', signal),
    enabled: Boolean(id),
  })
}

export function useBrandLinks(
  params: { brandId?: string; supplierId?: string },
  { enabled = true } = {},
) {
  return useQuery({
    queryKey: supplierKeys.links(params),
    queryFn: ({ signal }) => listBrandLinks(params, signal),
    enabled,
  })
}

/**
 * A mutation that refreshes supplier data (and catalogue readiness, which reads
 * supplier relationships) only after the server confirms.
 */
export function useSupplierMutation<V, R>(fn: (v: V) => Promise<R>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: supplierKeys.all }),
        queryClient.invalidateQueries({ queryKey: catalogueKeys.all }),
      ]),
  })
}

export function useRefreshSuppliers() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: supplierKeys.all })
}
