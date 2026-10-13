import { useSearchParams } from 'react-router'
import type { SelectOption } from '@/components/ui/Select'
import type { Confidence, LifecycleStatus } from '../api/contracts'
import { confidenceLabels, lifecycleLabels } from './presentation'

export const PAGE_SIZE = 25

export const lifecycleOptions: SelectOption[] = (
  Object.keys(lifecycleLabels) as LifecycleStatus[]
).map((v) => ({ value: v, label: lifecycleLabels[v] }))

export const confidenceOptions: SelectOption[] = (
  Object.keys(confidenceLabels) as Confidence[]
).map((v) => ({ value: v, label: confidenceLabels[v] }))

/**
 * List filters kept in the URL, so a refresh or shared link shows the same
 * page. Changing a filter returns to page 1.
 */
export function useListParams<K extends string>(keys: readonly K[]) {
  const [params, setParams] = useSearchParams()
  const values = Object.fromEntries(keys.map((k) => [k, params.get(k) ?? ''])) as Record<K, string>
  const page = Math.max(1, Number.parseInt(params.get('page') ?? '1', 10) || 1)
  const set = (changes: Partial<Record<K | 'page', string>>) => {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const [k, v] of Object.entries(changes) as [string, string][]) {
          if (v) next.set(k, v)
          else next.delete(k)
        }
        if (!('page' in changes)) next.delete('page')
        return next
      },
      { replace: true },
    )
  }
  return { values, page, set }
}
