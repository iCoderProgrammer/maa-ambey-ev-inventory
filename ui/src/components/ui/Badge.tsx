import type { ReactNode } from 'react'
import { cn } from '@/lib/utils/cn'

export type BadgeTone = 'success' | 'warning' | 'danger' | 'neutral' | 'info'

const tones: Record<BadgeTone, string> = {
  success: 'bg-success-subtle text-success ring-success/25',
  warning: 'bg-warning-subtle text-warning ring-warning/25',
  danger: 'bg-danger-subtle text-danger ring-danger/25',
  neutral: 'bg-neutral-subtle text-neutral ring-neutral/25',
  info: 'bg-info-subtle text-info ring-info/25',
}

const dotColors: Record<BadgeTone, string> = {
  success: 'bg-success',
  warning: 'bg-warning',
  danger: 'bg-danger',
  neutral: 'bg-neutral',
  info: 'bg-info',
}

export interface BadgeProps {
  tone?: BadgeTone
  /** Optional icon; status must never be conveyed by colour alone, so always pass text. */
  icon?: ReactNode
  /** Shows a subtle status indicator dot. */
  dot?: boolean
  /** Adds a radar-like pulse effect to the dot for live status. */
  pulse?: boolean
  children: ReactNode
  className?: string
}

export function Badge({
  tone = 'neutral',
  icon,
  dot = false,
  pulse = false,
  children,
  className,
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium tracking-wide whitespace-nowrap ring-1 ring-inset transition-colors duration-150',
        tones[tone],
        className,
      )}
    >
      {dot && (
        <span className="relative flex size-1.5 shrink-0" aria-hidden="true">
          {pulse && (
            <span
              className={cn(
                'absolute inline-flex size-full animate-ping rounded-full opacity-75',
                dotColors[tone],
              )}
            />
          )}
          <span className={cn('relative inline-flex size-1.5 rounded-full', dotColors[tone])} />
        </span>
      )}
      {icon}
      <span>{children}</span>
    </span>
  )
}
