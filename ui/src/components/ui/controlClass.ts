import { cn } from '@/lib/utils/cn'

/** Shared styling for text-like form controls (Input, Select). */
export const controlClass = cn(
  'h-10 w-full rounded-control border border-border bg-surface px-3 text-sm text-ink',
  'placeholder:text-muted disabled:cursor-not-allowed disabled:opacity-60',
  'aria-invalid:border-danger',
)
