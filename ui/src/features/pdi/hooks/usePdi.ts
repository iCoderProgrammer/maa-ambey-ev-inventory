import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { vehicleUnitKeys } from '@/features/vehicles/hooks/useVehicleUnits'
import {
  createChecklistItem,
  getInspection,
  getPdiOverview,
  listChecklist,
  retireChecklistItem,
  sendInspection,
  type ChecklistCreateBody,
  type ChecklistListParams,
  type InspectionAction,
} from '../api/pdi'

/**
 * Query keys and hooks for PDI (D-06). Nothing in the cache changes before the
 * server confirms: a recorded inspection refreshes the unit's PDI overview and
 * every vehicle unit query (status and saleability changed on the server), and
 * a refusal refreshes them too (the unit may have moved on). Mutations are not
 * retried automatically.
 */
export const pdiKeys = {
  all: ['pdi'] as const,
  overview: (unitId: string) => [...pdiKeys.all, 'overview', unitId] as const,
  inspection: (id: string) => [...pdiKeys.all, 'inspection', id] as const,
  checklist: (p: ChecklistListParams) => [...pdiKeys.all, 'checklist', p] as const,
}

export function usePdiOverview(unitId: string) {
  return useQuery({
    queryKey: pdiKeys.overview(unitId),
    queryFn: ({ signal }) => getPdiOverview(unitId, signal),
    enabled: Boolean(unitId),
  })
}

export function useInspection(id: string | null) {
  return useQuery({
    queryKey: pdiKeys.inspection(id ?? ''),
    queryFn: ({ signal }) => getInspection(id ?? '', signal),
    enabled: Boolean(id),
  })
}

export function useChecklist(params: ChecklistListParams, { enabled = true } = {}) {
  return useQuery({
    queryKey: pdiKeys.checklist(params),
    queryFn: ({ signal }) => listChecklist(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useRecordInspection() {
  const queryClient = useQueryClient()
  const refresh = (unitId: string) =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: pdiKeys.overview(unitId) }),
      queryClient.invalidateQueries({ queryKey: vehicleUnitKeys.all }),
    ])
  return useMutation({
    mutationFn: (a: InspectionAction) => sendInspection(a),
    onSuccess: async (r, a) => {
      queryClient.setQueryData(pdiKeys.inspection(r.inspection.id), r.inspection)
      await refresh(a.unitId)
    },
    onError: (_e, a) => refresh(a.unitId),
  })
}

function useChecklistMutation<V>(fn: (v: V) => Promise<unknown>) {
  const queryClient = useQueryClient()
  const refresh = () => queryClient.invalidateQueries({ queryKey: pdiKeys.all })
  return useMutation({ mutationFn: fn, onSuccess: refresh, onError: refresh })
}

export function useCreateChecklistItem() {
  return useChecklistMutation((b: ChecklistCreateBody) => createChecklistItem(b))
}

export function useRetireChecklistItem() {
  return useChecklistMutation((v: { id: string; version: number; reason: string }) =>
    retireChecklistItem(v.id, { version: v.version, reason: v.reason }),
  )
}

/** What the server granted (presentation only; the API checks every request again). */
export function usePdiPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return {
    me,
    meId: me.data?.user.id,
    view: p.includes(Permission.VehicleUnitView),
    record: p.includes(Permission.PDIRecord),
    manage: p.includes(Permission.PDIChecklistManage),
    upload: p.includes(Permission.AttachmentsUpload),
    attachmentsView: p.includes(Permission.AttachmentsView),
    catalogueView: p.includes(Permission.CatalogueView),
  }
}
