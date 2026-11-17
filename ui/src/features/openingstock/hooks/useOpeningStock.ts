import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import {
  getImport,
  getSpecification,
  listImportRows,
  listImports,
  uploadImport,
  type RowParams,
} from '../api/openingStock'
import { approveImport, commitImport, getImportCommit, type ApprovalInput } from '../api/commit'
import { getPreflight } from '../api/preflight'

/** Opening stock import preview queries (G-06). The only write is the upload, which creates no stock. */
export const openingStockKeys = {
  all: ['opening-stock-imports'] as const,
  spec: () => [...openingStockKeys.all, 'specification'] as const,
  list: (page: number) => [...openingStockKeys.all, 'list', page] as const,
  batch: (id: string) => [...openingStockKeys.all, 'batch', id] as const,
  rows: (id: string, p: RowParams) => [...openingStockKeys.all, 'rows', id, p] as const,
  commit: (id: string) => [...openingStockKeys.all, 'commit', id] as const,
  preflight: (id: string) => [...openingStockKeys.all, 'preflight', id] as const,
}

export function useImportSpecification({ enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.spec(),
    queryFn: ({ signal }) => getSpecification(signal),
    enabled,
  })
}

export function useImportList(page: number, { enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.list(page),
    queryFn: ({ signal }) => listImports({ page, limit: 10 }, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useImportBatch(id: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.batch(id),
    queryFn: ({ signal }) => getImport(id, signal),
    enabled,
  })
}

export function useImportRows(id: string, params: RowParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.rows(id, params),
    queryFn: ({ signal }) => listImportRows(id, params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useUploadImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { file: File; key: string; onProgress?: (f: number) => void }) =>
      uploadImport(v.file, v.key, { onProgress: v.onProgress }),
    onSuccess: (b) => {
      qc.setQueryData(openingStockKeys.batch(b.id), b)
      void qc.invalidateQueries({ queryKey: [...openingStockKeys.all, 'list'] })
    },
  })
}

/** The batch's commit (G-07), null while uncommitted. The source of truth after any unknown outcome. */
export function useImportCommit(id: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.commit(id),
    queryFn: ({ signal }) => getImportCommit(id, signal),
    enabled,
    retry: false,
  })
}

/**
 * The read-only reconciliation preflight (H-08). Runs each time the page opens
 * and when the user asks; never refreshed in the background (it reads the
 * stored file and the current database each time).
 */
export function usePreflight(id: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: openingStockKeys.preflight(id),
    queryFn: ({ signal }) => getPreflight(id, signal),
    enabled,
    refetchOnWindowFocus: false,
    retry: false,
  })
}

/** Records the approval (G-07); the batch is re-read from the server afterwards. */
export function useApproveImport(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ApprovalInput) => approveImport(id, input),
    retry: false,
    onSettled: () => qc.invalidateQueries({ queryKey: openingStockKeys.batch(id) }),
  })
}

/** Commits the batch with the caller's key (G-07). Never retried automatically. */
export function useCommitImport(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => commitImport(id, key),
    retry: false,
    onSuccess: (c) => {
      qc.setQueryData(openingStockKeys.commit(id), c)
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: openingStockKeys.batch(id) })
      void qc.invalidateQueries({ queryKey: [...openingStockKeys.all, 'list'] })
    },
  })
}

/**
 * Presentation only: the API checks every permission on every request. Batches
 * and rows are readable with preview, approve or commit (G-07); upload needs preview.
 */
export function useImportPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  const preview = p.includes(Permission.ImportPreview)
  const approve = p.includes(Permission.ImportApprove)
  const commit = p.includes(Permission.ImportCommit)
  return { me, preview, approve, commit, view: preview || approve || commit }
}
