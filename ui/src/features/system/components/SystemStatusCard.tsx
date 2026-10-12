import { RefreshCw, WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils/cn'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Table } from '@/components/ui/Table'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useHealth, useReadiness } from '../hooks/useSystemStatus'
import { checkLabel } from '../readinessChecks'

interface CheckRow {
  name: string
  up: boolean
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

/** Live status of the Go API and its dependencies. No mock data: every value comes from the API. */
export function SystemStatusCard() {
  const health = useHealth()
  const ready = useReadiness()
  const online = useOnlineStatus()
  const retrying =
    online &&
    ((!health.isError && health.failureCount > 0) || (!ready.isError && ready.failureCount > 0))

  const refresh = () => {
    void health.refetch()
    void ready.refetch()
  }

  // A failed check wins over cached data: showing the last "Ready" would be misleading.
  const checks: CheckRow[] =
    ready.data && !ready.isError
      ? Object.entries(ready.data.checks).map(([name, v]) => ({ name, up: v === 'up' }))
      : []

  return (
    <Card
      title="System status"
      actions={
        <Button
          variant="secondary"
          size="sm"
          onClick={refresh}
          disabled={!online || health.isFetching || ready.isFetching}
        >
          <RefreshCw
            className={cn('size-3.5', (health.isFetching || ready.isFetching) && 'animate-spin')}
            aria-hidden="true"
          />
          Refresh
        </Button>
      }
    >
      {!online && (
        <div
          role="alert"
          className="mb-4 flex items-start gap-2.5 rounded-control border border-warning/30 bg-warning-subtle p-3.5 text-sm"
        >
          <WifiOff className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <p className="text-ink">
            <span className="font-semibold">No network connection.</span>{' '}
            {health.data
              ? 'The status below may be out of date. '
              : 'The API status cannot be checked. '}
            It refreshes automatically when the connection returns.
          </p>
        </div>
      )}
      {retrying && (
        <p
          role="status"
          className="mb-4 rounded-control border border-warning/30 bg-warning-subtle p-3.5 text-sm text-ink"
        >
          <span className="font-semibold">The last check failed. Retrying…</span>{' '}
          {health.data ? 'Values below may be out of date.' : ''}
        </p>
      )}
      {health.isPending ? (
        online && <LoadingState label="Contacting API…" />
      ) : health.isError ? (
        <ErrorState error={health.error} title="API unreachable" onRetry={refresh} />
      ) : (
        <div className="flex flex-col gap-5">
          <dl className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-3">
            <div className="rounded-control border border-border/80 bg-surface-muted/40 p-3 shadow-2xs">
              <dt className="text-xs font-medium text-muted">API</dt>
              <dd className="mt-1.5 flex items-center">
                <Badge tone="success" dot pulse>
                  {health.data.status === 'ok' ? 'Online' : health.data.status}
                </Badge>
              </dd>
            </div>
            <div className="rounded-control border border-border/80 bg-surface-muted/40 p-3 shadow-2xs">
              <dt className="text-xs font-medium text-muted">Version</dt>
              <dd className="mt-1.5 font-mono text-xs font-semibold text-ink">
                {health.data.version}
              </dd>
            </div>
            <div className="rounded-control border border-border/80 bg-surface-muted/40 p-3 shadow-2xs">
              <dt className="text-xs font-medium text-muted">Server time</dt>
              <dd className="mt-1.5 text-xs font-medium text-ink">
                {formatTime(health.data.time)}
              </dd>
            </div>
          </dl>

          {ready.isPending ? (
            online && <LoadingState label="Checking dependencies…" />
          ) : checks.length > 0 ? (
            <div className="flex flex-col gap-2">
              <h3 className="text-xs font-semibold tracking-wider text-muted uppercase">
                Infrastructure Health
              </h3>
              <Table
                caption="Dependency readiness"
                columns={[
                  {
                    key: 'name',
                    header: 'Dependency',
                    render: (r) => (
                      <span className="font-medium text-ink">{checkLabel(r.name)}</span>
                    ),
                  },
                  {
                    key: 'status',
                    header: 'Status',
                    render: (r) =>
                      r.up ? (
                        <Badge tone="success" dot>
                          Ready
                        </Badge>
                      ) : (
                        <Badge tone="danger" dot>
                          Unavailable
                        </Badge>
                      ),
                  },
                ]}
                rows={checks}
                rowKey={(r) => r.name}
              />
              {/* Only when the database answered: an unreachable database also reports the schema down. */}
              {checks.some((r) => r.name === 'database' && r.up) &&
                checks.some((r) => r.name === 'migrations' && !r.up) && (
                  <p className="text-sm text-muted">
                    The database schema is not at the version this API expects (migrations pending
                    or a migration failed). An administrator must run the migrations; data is never
                    reset automatically.
                  </p>
                )}
            </div>
          ) : (
            <ErrorState error={ready.error} title="Readiness check failed" onRetry={refresh} />
          )}
        </div>
      )}
    </Card>
  )
}
