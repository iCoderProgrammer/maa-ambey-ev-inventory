import { AlertTriangle, Inbox, Loader2, ShieldX } from 'lucide-react'
import type { ReactNode } from 'react'
import { ApiError } from '@/lib/api/client'
import { cn } from '@/lib/utils/cn'
import { Button } from './Button'

/** Smooth pulse placeholder skeleton for loading screens. */
export function Skeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn('animate-pulse rounded-control bg-surface-muted', className)}
      aria-hidden="true"
    />
  )
}

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div role="status" className="flex items-center gap-2.5 py-6 text-sm font-medium text-muted">
      <Loader2 className="size-4 animate-spin text-primary" aria-hidden="true" />
      <span>{label}</span>
    </div>
  )
}

export interface ErrorStateProps {
  error: unknown
  title?: string
  onRetry?: () => void
}

export interface AccessDeniedStateProps {
  /** What the user tried to open, e.g. "the role and permission matrix". */
  what?: string
  requestId?: string
}

/**
 * The server refused access (403) or the user lacks the permission. Says so plainly,
 * without naming internal permission keys or hinting at what exists behind it.
 */
export function AccessDeniedState({ what = 'this page', requestId }: AccessDeniedStateProps) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center gap-3 rounded-card border border-border bg-surface px-4 py-10 text-center"
    >
      <span
        className="grid size-12 place-items-center rounded-full bg-warning-subtle text-warning"
        aria-hidden="true"
      >
        <ShieldX className="size-6" />
      </span>
      <div className="flex flex-col items-center gap-1.5">
        <p className="text-base font-semibold tracking-tight text-ink">Access denied</p>
        <p className="max-w-md text-sm leading-relaxed text-muted">
          You do not have permission to view {what}. If you need it, ask the owner or your
          administrator to grant it.
        </p>
        {requestId && <p className="font-mono text-xs text-muted">Reference: {requestId}</p>}
      </div>
    </div>
  )
}

/**
 * Shows a safe message plus the request ID for support; never stack traces.
 * A 403 is shown as access denied (retrying would not help).
 */
export function ErrorState({ error, title = 'Something went wrong', onRetry }: ErrorStateProps) {
  if (error instanceof ApiError && error.status === 403 && error.code === 'FORBIDDEN') {
    return <AccessDeniedState requestId={error.requestId} />
  }
  const message =
    error instanceof ApiError ? error.message : 'An unexpected error occurred. Please try again.'
  const requestId = error instanceof ApiError ? error.requestId : undefined

  return (
    <div
      role="alert"
      className="flex flex-col gap-3 rounded-card border border-danger/30 bg-danger-subtle p-4 text-sm"
    >
      <div className="flex items-start gap-3 text-danger">
        <span className="grid size-7 shrink-0 place-items-center rounded-control bg-danger/10 text-danger">
          <AlertTriangle className="size-4" aria-hidden="true" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="font-semibold text-danger">{title}</p>
          <p className="mt-0.5 text-ink">{message}</p>
          {requestId && (
            <p className="mt-1.5 font-mono text-xs text-muted">
              Reference:{' '}
              <code className="rounded bg-black/5 px-1 py-0.5 font-semibold text-ink dark:bg-white/10">
                {requestId}
              </code>
            </p>
          )}
        </div>
      </div>
      {onRetry && (
        <div className="pt-1">
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
    <div className="flex flex-col items-center gap-3.5 px-4 py-10 text-center">
      <span
        className="grid size-13 place-items-center rounded-full bg-surface-muted text-muted shadow-xs ring-8 ring-surface-muted/50 transition-transform duration-200 hover:scale-105"
        aria-hidden="true"
      >
        {icon ?? <Inbox className="size-6 text-muted/80" />}
      </span>
      <div className="flex flex-col items-center gap-1.5">
        <p className="text-base font-semibold tracking-tight text-ink">{title}</p>
        {children && <div className="max-w-md text-sm leading-relaxed text-muted">{children}</div>}
      </div>
      {action && <div className="mt-1">{action}</div>}
    </div>
  )
}
