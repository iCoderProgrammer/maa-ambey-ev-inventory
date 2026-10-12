import { cn } from '@/lib/utils/cn'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'icon'

const variants: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-primary-fg shadow-xs hover:bg-primary-hover',
  secondary: 'border border-border bg-surface text-ink shadow-xs hover:bg-surface-muted',
  ghost: 'text-ink hover:bg-surface-muted',
  danger: 'bg-danger text-white shadow-xs hover:opacity-90',
}

const sizes: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-sm',
  md: 'h-10 px-4 text-sm',
  /** Square icon-only button; always pass aria-label. */
  icon: 'size-9 text-sm',
}

const base = [
  'inline-flex cursor-pointer items-center justify-center gap-2 rounded-control font-medium select-none',
  'transition-[color,background-color,border-color,opacity,transform] duration-150',
  'active:translate-y-px disabled:pointer-events-none disabled:opacity-50',
]

/** Button styling for other elements, e.g. a router <Link> that navigates. */
export function buttonVariants({
  variant = 'primary',
  size = 'md',
  className,
}: { variant?: ButtonVariant; size?: ButtonSize; className?: string } = {}): string {
  return cn(base, variants[variant], sizes[size], className)
}
