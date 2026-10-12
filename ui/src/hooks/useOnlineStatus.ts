import { onlineManager } from '@tanstack/react-query'
import { useSyncExternalStore } from 'react'

/**
 * Browser connectivity as TanStack Query sees it. While offline, queries pause instead
 * of failing, so screens use this to explain the wait rather than spin indefinitely.
 */
export function useOnlineStatus(): boolean {
  return useSyncExternalStore(
    (onChange) => onlineManager.subscribe(onChange),
    () => onlineManager.isOnline(),
    () => true,
  )
}
