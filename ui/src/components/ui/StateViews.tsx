import { AlertTriangle, Inbox, Loader2 } from 'lucide-react'
import type { ReactNode } from 'react'
import { ApiError } from '@/lib/api/client'
import { Button } from './Button'

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div role="status" className="flex items-center gap-2 py-6 text-sm text-muted">
      <Loader2 className="size-4 animate-spin" aria-hidden="true" />
      {label}
    </div>
  )
}

export interface ErrorStateProps {
  error: unknown
  title?: string
  onRetry?: () => void
}

/** Shows a safe message plus the request ID for support; never stack traces. */
export function ErrorState({ error, title = 'Something went wrong', onRetry }: ErrorStateProps) {
  const message =
    error instanceof ApiError ? error.message : 'An unexpected error occurred. Please try again.'
  const requestId = error instanceof ApiError ? error.requestId : undefined

  return (
    <div
      role="alert"
      className="flex flex-col gap-3 rounded-control border border-danger/30 bg-danger-subtle p-4 text-sm"
    >
      <div className="flex items-start gap-2 text-danger">
        <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <div>
          <p className="font-semibold">{title}</p>
          <p className="text-ink">{message}</p>
          {requestId && (
            <p className="mt-1 text-xs text-muted">
              Reference: <code>{requestId}</code>
            </p>
          )}
        </div>
      </div>
      {onRetry && (
        <div>
          <Button variant="secondary" size="sm" onClick={onRetry}>
            Try again
          </Button>
        </div>
      )}
    </div>
  )
}

export interface EmptyStateProps {
  title: string
  children?: ReactNode
  /** Defaults to an inbox icon. */
  icon?: ReactNode
  /** A relevant, permitted next action (button or link). */
  action?: ReactNode
}

/** Explains why there is nothing here and, where allowed, what to do next. */
export function EmptyState({ title, children, icon, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center gap-3 px-4 py-10 text-center">
      <span
        className="grid size-12 place-items-center rounded-full bg-surface-muted text-muted ring-8 ring-surface-muted/40"
        aria-hidden="true"
      >
        {icon ?? <Inbox className="size-6" />}
      </span>
      <div className="flex flex-col items-center gap-1">
        <p className="font-semibold text-ink">{title}</p>
        {children && <div className="max-w-md text-sm text-muted">{children}</div>}
      </div>
      {action && <div className="mt-1">{action}</div>}
    </div>
  )
}
