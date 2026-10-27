import type { PriceItem } from '../api/pricing'

const inr = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' })

/** "125000.00" → "₹1,25,000.00" (display only; amounts stay strings). */
export function formatINR(amount: string): string {
  return inr.format(Number(amount))
}

export type PricePeriod = 'DRAFT' | 'CANCELLED' | 'CURRENT' | 'FUTURE' | 'PAST'

/** Where a price stands at `now`, from server data (display only). */
export function pricePeriod(item: PriceItem, now: Date = new Date()): PricePeriod {
  if (item.status === 'DRAFT') return 'DRAFT'
  if (item.status === 'CANCELLED') return 'CANCELLED'
  if (new Date(item.effectiveFrom) > now) return 'FUTURE'
  if (item.effectiveTo && new Date(item.effectiveTo) <= now) return 'PAST'
  return 'CURRENT'
}

/** datetime-local value ("2026-11-01T09:30") → RFC 3339 instant in the user's zone. */
export function localInputToRFC3339(value: string): string {
  return new Date(value).toISOString()
}

/** RFC 3339 → datetime-local value in the user's zone. */
export function toLocalInput(value: string | Date): string {
  const d = new Date(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** A sensible default start: the next full hour from now. */
export function nextHourInput(now: Date = new Date()): string {
  const d = new Date(now)
  d.setMinutes(0, 0, 0)
  d.setHours(d.getHours() + 1)
  return toLocalInput(d)
}

export function formatInstant(value: string | null): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value),
  )
}
