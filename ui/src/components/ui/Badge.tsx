import type { ReactNode } from 'react'
import { cn } from '@/lib/utils/cn'

export type BadgeTone = 'success' | 'warning' | 'danger' | 'neutral' | 'info'

const tones: Record<BadgeTone, string> = {
  success: 'bg-success-subtle text-success ring-success/20',
  warning: 'bg-warning-subtle text-warning ring-warning/20',
  danger: 'bg-danger-subtle text-danger ring-danger/20',
  neutral: 'bg-neutral-subtle text-neutral ring-neutral/20',
  info: 'bg-info-subtle text-info ring-info/20',
}

export interface BadgeProps {
  tone?: BadgeTone
  /** Optional icon; status must never be conveyed by colour alone, so always pass text. */
  icon?: ReactNode
  children: ReactNode
  className?: string
}

export function Badge({ tone = 'neutral', icon, children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset',
        tones[tone],
        className,
      )}
    >
      {icon}
      {children}
    </span>
  )
}
