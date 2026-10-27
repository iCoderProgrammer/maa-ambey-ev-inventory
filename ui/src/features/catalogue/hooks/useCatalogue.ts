import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  activateBrand,
  activateItem,
  archiveBrand,
  archiveItem,
  createBrand,
  createColour,
  createModel,
  createVariant,
  getBrand,
  getBrandReadiness,
  getItem,
  listBrands,
  listColours,
  listModels,
  listVariants,
  recordBrandAuthorisation,
  updateBrand,
  updateItem,
  verifyItem,
  type ArchiveInput,
  type BrandAuthorisation,
  type BrandListParams,
  type BrandUpdate,
  type CatalogueItemKind,
  type ColourListParams,
  type ModelListParams,
  type UpdateByKind,
  type VariantListParams,
  type VerifyInput,
} from '../api/catalogue'

/**
 * TanStack Query hooks for the catalogue API (C-02). Screens arrive in C-03.
 * Mutations change the cache only after the server confirms; nothing is
 * updated optimistically, because the server may refuse (permissions,
 * relationships, lifecycle, stale version).
 */

export const catalogueKeys = {
  all: ['catalogue'] as const,
  brands: (params: BrandListParams = {}) =>
    [...catalogueKeys.all, 'brands', 'list', params] as const,
  brand: (id: string) => [...catalogueKeys.all, 'brands', 'detail', id] as const,
  readiness: (id: string) => [...catalogueKeys.all, 'brands', 'readiness', id] as const,
  list: (kind: CatalogueItemKind, params: object = {}) =>
    [...catalogueKeys.all, kind, 'list', params] as const,
  item: (kind: CatalogueItemKind, id: string) =>
    [...catalogueKeys.all, kind, 'detail', id] as const,
}

interface Enabled {
  /** Pass false when the user lacks catalogue.view (from /me) to avoid a request bound to fail. */
  enabled?: boolean
}

export function useBrands(params: BrandListParams = {}, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.brands(params),
    queryFn: ({ signal }) => listBrands(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useBrand(id: string | undefined, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.brand(id ?? ''),
    queryFn: ({ signal }) => getBrand(id ?? '', signal),
    enabled: enabled && !!id,
  })
}

export function useBrandReadiness(id: string | undefined, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.readiness(id ?? ''),
    queryFn: ({ signal }) => getBrandReadiness(id ?? '', signal),
    enabled: enabled && !!id,
  })
}

export function useModels(params: ModelListParams = {}, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.list('models', params),
    queryFn: ({ signal }) => listModels(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useVariants(params: VariantListParams = {}, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.list('variants', params),
    queryFn: ({ signal }) => listVariants(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useColours(params: ColourListParams = {}, { enabled = true }: Enabled = {}) {
  return useQuery({
    queryKey: catalogueKeys.list('colours', params),
    queryFn: ({ signal }) => listColours(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useCatalogueItem<K extends CatalogueItemKind>(
  kind: K,
  id: string | undefined,
  { enabled = true }: Enabled = {},
) {
  return useQuery({
    queryKey: catalogueKeys.item(kind, id ?? ''),
    queryFn: ({ signal }) => getItem(kind, id ?? '', signal),
    enabled: enabled && !!id,
  })
}

/**
 * After a confirmed change: store the returned record and refetch everything
 * else in the catalogue (lists, readiness), since one change can affect others
 * (e.g. activating a variant changes its brand's readiness).
 */
function useSettled() {
  const queryClient = useQueryClient()
  return {
    brand: (brand: { id: string }) => {
      queryClient.setQueryData(catalogueKeys.brand(brand.id), brand)
      return queryClient.invalidateQueries({
        queryKey: catalogueKeys.all,
        predicate: (q) => q.queryKey.join('/') !== catalogueKeys.brand(brand.id).join('/'),
      })
    },
    item: (kind: CatalogueItemKind, item: { id: string }) => {
      queryClient.setQueryData(catalogueKeys.item(kind, item.id), item)
      return queryClient.invalidateQueries({
        queryKey: catalogueKeys.all,
        predicate: (q) => q.queryKey.join('/') !== catalogueKeys.item(kind, item.id).join('/'),
      })
    },
  }
}

export function useCreateBrand() {
  const settled = useSettled()
  return useMutation({ mutationFn: createBrand, onSuccess: settled.brand })
}

export function useUpdateBrand() {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: BrandUpdate }) => updateBrand(vars.id, vars.input),
    onSuccess: settled.brand,
  })
}

export function useRecordBrandAuthorisation() {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: BrandAuthorisation }) =>
      recordBrandAuthorisation(vars.id, vars.input),
    onSuccess: settled.brand,
  })
}

export function useActivateBrand() {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; version: number }) => activateBrand(vars.id, vars.version),
    onSuccess: settled.brand,
  })
}

export function useArchiveBrand() {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: ArchiveInput }) => archiveBrand(vars.id, vars.input),
    onSuccess: settled.brand,
  })
}

export function useCreateModel() {
  const settled = useSettled()
  return useMutation({ mutationFn: createModel, onSuccess: (m) => settled.item('models', m) })
}

export function useCreateVariant() {
  const settled = useSettled()
  return useMutation({ mutationFn: createVariant, onSuccess: (v) => settled.item('variants', v) })
}

export function useCreateColour() {
  const settled = useSettled()
  return useMutation({ mutationFn: createColour, onSuccess: (c) => settled.item('colours', c) })
}

export function useUpdateItem<K extends CatalogueItemKind>(kind: K) {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: UpdateByKind[K] }) =>
      updateItem(kind, vars.id, vars.input),
    onSuccess: (item) => settled.item(kind, item),
  })
}

export function useVerifyItem(kind: CatalogueItemKind) {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: VerifyInput }) => verifyItem(kind, vars.id, vars.input),
    onSuccess: (item) => settled.item(kind, item),
  })
}

export function useActivateItem(kind: CatalogueItemKind) {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; version: number }) =>
      activateItem(kind, vars.id, vars.version),
    onSuccess: (item) => settled.item(kind, item),
  })
}

export function useArchiveItem(kind: CatalogueItemKind) {
  const settled = useSettled()
  return useMutation({
    mutationFn: (vars: { id: string; input: ArchiveInput }) =>
      archiveItem(kind, vars.id, vars.input),
    onSuccess: (item) => settled.item(kind, item),
  })
}

/** Refetches every catalogue query, e.g. after a stale-version conflict. */
export function useRefreshCatalogue() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: catalogueKeys.all })
}
