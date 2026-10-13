import { CircleAlert, CircleCheck, Loader2, WifiOff } from 'lucide-react'
import { Badge } from '@/components/ui/Badge'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useHealth, useReadiness } from '../hooks/useSystemStatus'
import { notReadyReason } from '../readinessChecks'

/** Compact header pill showing live API, database and schema status from the real Go API. */
export function ApiStatusIndicator() {
  const health = useHealth()
  const ready = useReadiness()
  const online = useOnlineStatus()

  // Checked first: while the browser is offline, queries pause and cached data may be stale.
  if (!online) {
    return (
      <Badge tone="danger" icon={<WifiOff className="size-3" aria-hidden="true" />}>
        Offline
      </Badge>
    )
  }
  if (health.isPending) {
    return (
      <Badge tone="neutral" icon={<Loader2 className="size-3 animate-spin" aria-hidden="true" />}>
        Checking API…
      </Badge>
    )
  }
  // A check failed and a retry is pending (or paused while the tab is hidden): do not keep
  // showing the last good result as if it were current.
  if (!health.isError && !ready.isError && (health.failureCount > 0 || ready.failureCount > 0)) {
    return (
      <Badge tone="warning" icon={<Loader2 className="size-3 animate-spin" aria-hidden="true" />}>
        Reconnecting…
      </Badge>
    )
  }
  if (health.isError) {
    return (
      <Badge tone="danger" icon={<CircleAlert className="size-3" aria-hidden="true" />}>
        API offline
      </Badge>
    )
  }
  if (ready.isError) {
    // The readiness check itself failed, so database status is unknown (not "down").
    return (
      <Badge tone="warning" icon={<CircleAlert className="size-3" aria-hidden="true" />}>
        Status unknown
      </Badge>
    )
  }
  if (ready.data?.status === 'not_ready') {
    return (
      <Badge tone="warning" icon={<CircleAlert className="size-3" aria-hidden="true" />}>
        {notReadyReason(ready.data.checks)}
      </Badge>
    )
  }
  return (
    <Badge
      tone="success"
      dot
      pulse
      icon={<CircleCheck className="size-3 text-success" aria-hidden="true" />}
    >
      <span className="hidden sm:inline">API </span>online
    </Badge>
  )
}
