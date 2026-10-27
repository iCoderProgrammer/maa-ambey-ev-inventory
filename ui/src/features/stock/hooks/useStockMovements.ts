import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { vehicleUnitKeys } from '@/features/vehicles/hooks/useVehicleUnits'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { sendDisplayAction, type DisplayAction } from '../api/stockMovements'

/**
 * Display moves (E-01). The variables are a DisplayAction made once per user
 * decision (newDisplayAction), so calling mutate again with the same action
 * retries with the same Idempotency-Key and body; the hook never makes a key.
 * Mutations are not retried automatically (queryClient defaults). Nothing is
 * changed in the cache before the server confirms: on success every vehicle unit
 * list and detail is refetched from the server; on a stale version, a wrong
 * state or an unreadable 2xx answer they are refetched too, so the screen shows
 * the real state. Other failures leave the cache as it was.
 */
export function useDisplayMove() {
  const queryClient = useQueryClient()
  const refresh = () => queryClient.invalidateQueries({ queryKey: vehicleUnitKeys.all })
  return useMutation({
    mutationFn: (action: DisplayAction) => sendDisplayAction(action),
    onSuccess: refresh,
    onError: async (error) => {
      if (
        error instanceof ApiError &&
        (error.code === 'VERSION_CONFLICT' ||
          error.code === 'INVALID_STATE' ||
          error.code === ClientErrorCode.InvalidResponse)
      ) {
        await refresh()
      }
    },
  })
}

/** Whether the server granted the display moves (presentation only; the API checks again). */
export function useStockMovementPerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  return { me, manageDisplay: p.includes(Permission.VehicleUnitDisplayManage) }
}
