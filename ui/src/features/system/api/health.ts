import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'

export const healthSchema = z.object({
  status: z.string(),
  service: z.string(),
  version: z.string(),
  time: z.string(),
})
export type Health = z.infer<typeof healthSchema>

export const readinessSchema = z.object({
  status: z.enum(['ready', 'not_ready']),
  checks: z.record(z.string(), z.string()),
})
export type Readiness = z.infer<typeof readinessSchema>

/** GET /health — liveness of the Go API process. */
export function getHealth(signal?: AbortSignal): Promise<Health> {
  return apiRequest('/health', healthSchema, { signal })
}

/**
 * GET /ready — dependency readiness (PostgreSQL reachable, schema migrated). A 503 carrying a readiness body is an
 * answer (status "not_ready"), returned as data; other failures throw ApiError.
 */
export function getReadiness(signal?: AbortSignal): Promise<Readiness> {
  return apiRequest('/ready', readinessSchema, { signal, acceptStatuses: [503] })
}
