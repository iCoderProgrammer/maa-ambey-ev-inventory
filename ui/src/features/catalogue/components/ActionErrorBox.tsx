import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { checkLabel, type ActionError } from '../lib/presentation'

/** The server's refusal, shown where the action was attempted. Nothing changed. */
export function ActionErrorBox({ error, onReload }: { error: ActionError; onReload?: () => void }) {
  return (
    <div
      role="alert"
      className="flex flex-col gap-2 rounded-control border border-danger/30 bg-danger-subtle p-3 text-sm"
    >
      <div className="flex items-start gap-2 text-ink">
        <AlertTriangle className="mt-0.5 size-4 shrink-0 text-danger" aria-hidden="true" />
        <p>{error.message}</p>
      </div>
      {error.blockers && error.blockers.length > 0 && (
        <ul className="ml-6 list-disc space-y-1 text-ink">
          {error.blockers.map((b) => (
            <li key={b.key}>
              <span className="font-medium">{checkLabel(b.key)}</span>
              <span className="text-muted"> — {b.message}</span>
            </li>
          ))}
        </ul>
      )}
      {error.stale && onReload && (
        <div>
          <Button variant="secondary" size="sm" onClick={onReload}>
            <RefreshCw className="size-3.5" aria-hidden="true" />
            Reload latest version
          </Button>
        </div>
      )}
      {error.requestId && (
        <p className="font-mono text-xs text-muted">Reference: {error.requestId}</p>
      )}
    </div>
  )
}
