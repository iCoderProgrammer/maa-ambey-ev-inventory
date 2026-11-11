import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import {
  getStockCount,
  listStockCounts,
  recordCountLine,
  sendCountAction,
  submitStockCount,
  type CountAction,
  type CountListParams,
  type LineBody,
  type StockCount,
} from '../api/stockCounts'

/**
 * Query keys and hooks for stock counts (E-05). Nothing is changed in the cache
 * before the server confirms; every mutation, successful or not, refetches the
 * counts (a refusal may mean the count or the stock moved on). Mutations are not
 * retried automatically.
 */
export const countKeys = {
  all: ['stock-counts'] as const,
  list: (p: CountListParams) => [...countKeys.all, 'list', p] as const,
  detail: (id: string) => [...countKeys.all, 'detail', id] as const,
}

export function useStockCountList(params: CountListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: countKeys.list(params),
    queryFn: ({ signal }) => listStockCounts(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useStockCount(id: string) {
  return useQuery({
    queryKey: countKeys.detail(id),
    queryFn: ({ signal }) => getStockCount(id, signal),
    enabled: Boolean(id),
  })
}

function useCountMutation<V>(fn: (v: V) => Promise<StockCount>) {
  const queryClient = useQueryClient()
  const refresh = () => queryClient.invalidateQueries({ queryKey: countKeys.all })
  return useMutation({
    mutationFn: fn,
    onSuccess: async (c) => {
      queryClient.setQueryData(countKeys.detail(c.id), c)
      await refresh()
    },
    onError: refresh,
  })
}

export function useCountAction() {
  return useCountMutation((a: CountAction) => sendCountAction(a))
}

export function useRecordLine(countId: string) {
  return useCountMutation((body: LineBody) => recordCountLine(countId, body))
}

export function useSubmitCount(countId: string) {
  return useCountMutation((version: number) => submitStockCount(countId, version))
}

/** What the server granted (presentation only; the API checks every request again). */
export function useCountPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.AdjustmentView),
    create: p.includes(Permission.AdjustmentCreate),
    approve: p.includes(Permission.AdjustmentApprove),
    stockItems: p.includes(Permission.StockItemView),
    vehicleUnits: p.includes(Permission.VehicleUnitView),
    /** The locations the caller may count at (GET /me; no location list API exists yet). */
    locations: me.data?.locationScope.locations ?? [],
  }
}

/**
 * Who is known to have taken part (created, submitted, recorded a current line).
 * The server also remembers earlier recorders, so a 403 may still come back.
 */
export function tookPart(c: StockCount, userId: string | undefined): boolean {
  if (!userId) return false
  return (
    c.createdBy === userId ||
    c.submittedBy === userId ||
    c.lines.some((l) => l.recordedBy === userId)
  )
}
