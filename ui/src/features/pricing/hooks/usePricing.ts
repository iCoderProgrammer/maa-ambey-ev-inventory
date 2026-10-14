import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { catalogueKeys } from '@/features/catalogue/hooks/useCatalogue'
import {
  approvePriceItem,
  archivePriceList,
  cancelPriceItem,
  createPriceItem,
  createPriceList,
  listPriceItems,
  listPriceLists,
  lookupPrice,
  updatePriceItem,
  type PriceItemParams,
  type PriceItemUpdate,
  type PriceListParams,
  type PriceType,
} from '../api/pricing'

/**
 * TanStack Query hooks for price history (C-04). Mutations update the cache
 * only after the server confirms, then refetch prices and catalogue readiness
 * (an approved price can satisfy a brand's approved_prices check).
 */
export const pricingKeys = {
  all: ['pricing'] as const,
  lists: (params: PriceListParams) => [...pricingKeys.all, 'lists', params] as const,
  items: (params: PriceItemParams) => [...pricingKeys.all, 'items', params] as const,
  lookup: (params: object) => [...pricingKeys.all, 'lookup', params] as const,
}

interface Enabled {
  enabled?: boolean
}

export function usePriceLists(params: PriceListParams, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: pricingKeys.lists(params),
    queryFn: ({ signal }) => listPriceLists(params, signal),
    enabled,
  })
}

export function usePriceItems(params: PriceItemParams, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: pricingKeys.items(params),
    queryFn: ({ signal }) => listPriceItems(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function usePriceLookup(
  params: { priceListId: string; variantId: string; priceType: PriceType; at?: string },
  { enabled = true }: Enabled = {},
) {
  return useQuery({
    queryKey: pricingKeys.lookup(params),
    queryFn: ({ signal }) => lookupPrice(params, signal),
    enabled,
    staleTime: 0,
  })
}

function useRefresh() {
  const queryClient = useQueryClient()
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: pricingKeys.all }),
      queryClient.invalidateQueries({ queryKey: catalogueKeys.all }),
    ])
}

/** Refetches all price data, e.g. after a stale-version conflict. */
export function useRefreshPricing() {
  return useRefresh()
}

export function useCreatePriceList() {
  const refresh = useRefresh()
  return useMutation({ mutationFn: createPriceList, onSuccess: refresh })
}

export function useArchivePriceList() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: (v: { id: string; version: number }) => archivePriceList(v.id, v.version),
    onSuccess: refresh,
  })
}

export function useCreatePriceItem() {
  const refresh = useRefresh()
  return useMutation({ mutationFn: createPriceItem, onSuccess: refresh })
}

export function useUpdatePriceItem() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: (v: { id: string; input: PriceItemUpdate }) => updatePriceItem(v.id, v.input),
    onSuccess: refresh,
  })
}

export function useCancelPriceItem() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: (v: { id: string; version: number }) => cancelPriceItem(v.id, v.version),
    onSuccess: refresh,
  })
}

export function useApprovePriceItem() {
  const refresh = useRefresh()
  return useMutation({
    mutationFn: (v: { id: string; version: number; approvalReference?: string }) =>
      approvePriceItem(v.id, { version: v.version, approvalReference: v.approvalReference }),
    onSuccess: refresh,
  })
}
