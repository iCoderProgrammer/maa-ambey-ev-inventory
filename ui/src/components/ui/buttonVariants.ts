import { cn } from '@/lib/utils/cn'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'icon'

const variants: Record<ButtonVariant, string> = {
  primary:
    'bg-primary text-primary-fg shadow-xs hover:bg-primary-hover hover:shadow-sm active:scale-[0.98]',
  secondary:
    'border border-border bg-surface text-ink shadow-xs hover:bg-surface-muted hover:border-border/90 active:scale-[0.98]',
  ghost: 'text-ink hover:bg-surface-muted active:scale-[0.98]',
  danger: 'bg-danger text-white shadow-xs hover:opacity-90 active:scale-[0.98]',
}

const sizes: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-xs font-medium',
  md: 'h-10 px-4 text-sm font-medium',
  /** Square icon-only button; always pass aria-label. */
  icon: 'size-9 text-sm',
}

const base = [
  'inline-flex cursor-pointer items-center justify-center gap-2 rounded-control select-none',
  'transition-[color,background-color,border-color,opacity,box-shadow,transform] duration-150',
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 focus-visible:ring-offset-1',
  'disabled:pointer-events-none disabled:opacity-50',
]

/** Button styling for other elements, e.g. a router <Link> that navigates. */
export function buttonVariants({
  variant = 'primary',
  size = 'md',
  className,
}: { variant?: ButtonVariant; size?: ButtonSize; className?: string } = {}): string {
  return cn(base, variants[variant], sizes[size], className)
}
