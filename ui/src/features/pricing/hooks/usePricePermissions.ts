import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import type { PriceType } from '../api/pricing'

/** What the server granted for prices (presentation only; the API checks again). */
export function usePricePerms() {
  const me = useMe()
  const p = me.data?.permissions ?? []
  const has = (k: string) => p.includes(k)
  return {
    meId: me.data?.user.id,
    viewSelling: has(Permission.PriceView),
    viewCost: has(Permission.CostPriceView),
    canDraft: (t: PriceType) =>
      has(t === 'COST' ? Permission.CostPriceEdit : Permission.PriceManage),
    manageLists: has(Permission.PriceManage) || has(Permission.CostPriceEdit),
    approve: has(Permission.PriceApprove),
  }
}

/** Whether to show price sections at all (any price view permission). */
export function useCanSeePrices(): boolean {
  const p = usePricePerms()
  return p.viewSelling || p.viewCost
}
