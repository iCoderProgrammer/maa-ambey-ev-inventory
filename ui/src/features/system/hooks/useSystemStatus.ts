import { useQuery } from '@tanstack/react-query'
import { getHealth, getReadiness } from '../api/health'

export const systemKeys = {
  all: ['system'] as const,
  health: () => [...systemKeys.all, 'health'] as const,
  readiness: () => [...systemKeys.all, 'readiness'] as const,
}

const POLL_MS = 30_000

export function useHealth() {
  return useQuery({
    queryKey: systemKeys.health(),
    queryFn: ({ signal }) => getHealth(signal),
    refetchInterval: POLL_MS,
  })
}

export function useReadiness() {
  return useQuery({
    queryKey: systemKeys.readiness(),
    queryFn: ({ signal }) => getReadiness(signal),
    refetchInterval: POLL_MS,
  })
}
