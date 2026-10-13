import type { Readiness } from './api/health'

/**
 * Display names for the dependency checks reported by GET /api/v1/ready.
 * Unknown checks fall back to their raw name so new backend checks still show up.
 */
const CHECK_LABELS: Record<string, string> = {
  database: 'Database',
  migrations: 'Database schema',
}

export function checkLabel(name: string): string {
  return CHECK_LABELS[name] ?? name
}

/**
 * Short reason for a "not_ready" answer, most fundamental failure first: when the
 * database is unreachable its schema cannot be checked either.
 */
export function notReadyReason(checks: Readiness['checks']): string {
  if (checks.database === 'down') return 'Database unavailable'
  if (checks.migrations === 'down') return 'Schema not ready'
  return 'Not ready'
}
