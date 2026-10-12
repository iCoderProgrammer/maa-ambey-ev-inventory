import { RefreshCw, WifiOff } from 'lucide-react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Table } from '@/components/ui/Table'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useHealth, useReadiness } from '../hooks/useSystemStatus'

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
          <RefreshCw className="size-3.5" aria-hidden="true" />
          Refresh
        </Button>
      }
    >
      {!online && (
        <div
          role="alert"
          className="mb-4 flex items-start gap-2 rounded-control border border-warning/30 bg-warning-subtle p-3 text-sm"
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
          className="mb-4 rounded-control border border-warning/30 bg-warning-subtle p-3 text-sm text-ink"
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
        <div className="flex flex-col gap-4">
          <dl className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-3">
            <div>
              <dt className="text-xs text-muted">API</dt>
              <dd className="mt-0.5">
                <Badge tone="success">
                  {health.data.status === 'ok' ? 'Online' : health.data.status}
                </Badge>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Version</dt>
              <dd className="mt-0.5 font-mono text-xs">{health.data.version}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Server time</dt>
              <dd className="mt-0.5">{formatTime(health.data.time)}</dd>
            </div>
          </dl>

          {ready.isPending ? (
            online && <LoadingState label="Checking dependencies…" />
          ) : checks.length > 0 ? (
            <Table
              caption="Dependency readiness"
              columns={[
                {
                  key: 'name',
                  header: 'Dependency',
                  render: (r) => <span className="capitalize">{r.name}</span>,
                },
                {
                  key: 'status',
                  header: 'Status',
                  render: (r) =>
                    r.up ? (
                      <Badge tone="success">Ready</Badge>
                    ) : (
                      <Badge tone="danger">Unavailable</Badge>
                    ),
                },
              ]}
              rows={checks}
              rowKey={(r) => r.name}
            />
          ) : (
            <ErrorState error={ready.error} title="Readiness check failed" onRetry={refresh} />
          )}
        </div>
      )}
    </Card>
  )
}
