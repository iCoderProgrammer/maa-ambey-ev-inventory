import type { ReactNode } from 'react'
import { cn } from '@/lib/utils/cn'

export interface CardProps {
  title?: string
  description?: string
  badge?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
  headerClassName?: string
  contentClassName?: string
}

export function Card({
  title,
  description,
  badge,
  actions,
  children,
  className,
  headerClassName,
  contentClassName,
}: CardProps) {
  return (
    <section
      className={cn(
        'rounded-card border border-border/80 bg-surface shadow-xs transition-[border-color,box-shadow] duration-150',
        className,
      )}
    >
      {(title || actions || badge) && (
        <header
          className={cn(
            'flex items-start justify-between gap-3 border-b border-border/80 px-5 py-4',
            headerClassName,
          )}
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              {title && (
                <h2 className="text-base font-semibold tracking-tight text-ink">{title}</h2>
              )}
              {badge}
            </div>
            {description && <p className="mt-0.5 text-sm text-muted">{description}</p>}
          </div>
          {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </header>
      )}
      <div className={cn('p-5', contentClassName)}>{children}</div>
    </section>
  )
}
