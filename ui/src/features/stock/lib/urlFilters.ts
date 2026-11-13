import { useCallback, useLayoutEffect, useRef } from 'react'
import { useSearchParams } from 'react-router'

/**
 * List filters kept in the URL (refresh and shared links show the same page),
 * for the movement timeline (E-02). Changing a filter returns to page 1.
 *
 * Unlike the shared catalogue helper (useListParams), every change is applied
 * on top of the most recent change, not on top of the last *rendered* URL: two
 * filters changed before React re-renders (a debounced search firing while a
 * select changes, or a multi-field reset) both survive. `set` is stable, so a
 * debounced caller can keep it without going stale.
 */
export function useUrlFilters<K extends string>(keys: readonly K[]) {
  const [params, setParams] = useSearchParams()
  const latest = useRef(params)
  // Back/forward or a link changes the URL from outside: follow it before the next user event.
  useLayoutEffect(() => {
    latest.current = params
  }, [params])
  const values = Object.fromEntries(keys.map((k) => [k, params.get(k) ?? ''])) as Record<K, string>
  const page = Math.max(1, Number.parseInt(params.get('page') ?? '1', 10) || 1)
  const set = useCallback(
    (changes: Partial<Record<K | 'page', string>>) => {
      const next = new URLSearchParams(latest.current)
      for (const [k, v] of Object.entries(changes) as [string, string][]) {
        if (v) next.set(k, v)
        else next.delete(k)
      }
      if (!('page' in changes)) next.delete('page')
      latest.current = next
      setParams(next, { replace: true })
    },
    [setParams],
  )
  return { values, page, set }
}

/**
 * A calendar day (YYYY-MM-DD, the user's own time zone) as the instant it
 * starts, for the API's RFC 3339 `from`/`to`. `to` uses the start of the day
 * after, because the API's `to` is exclusive. Anything else is sent as typed,
 * so the server reports it.
 */
export function dayStart(day: string, plusDays = 0): string | undefined {
  if (!day) return undefined
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day)
  if (!m) return day
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]) + plusDays)
  return Number.isNaN(d.getTime()) ? day : d.toISOString()
}
