import { cn } from '@/lib/utils/cn'

/** Shared styling for text-like form controls (Input, Select). */
export const controlClass = cn(
  'h-10 w-full rounded-control border border-border bg-surface px-3 text-sm text-ink',
  'placeholder:text-muted/70 transition-[border-color,box-shadow] duration-150',
  'hover:border-border/90',
  'focus:outline-none focus:ring-2 focus:ring-primary/25 focus:border-primary',
  'disabled:cursor-not-allowed disabled:opacity-60 disabled:bg-surface-muted/50',
  'aria-invalid:border-danger aria-invalid:focus:ring-danger/25',
)
