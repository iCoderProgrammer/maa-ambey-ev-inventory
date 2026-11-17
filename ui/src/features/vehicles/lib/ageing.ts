import type { MissingOriginReason, StockAgeing } from '../api/vehicleUnits'
import { ageText } from './presentation'

/**
 * Stock ageing labels (E-07). Only presentation: ages, bands and reasons come
 * from the server (ADR-029) and are never recomputed in the browser.
 */
export const missingReasonLabels: Record<MissingOriginReason, string> = {
  UNSUPPORTED_SOURCE: 'No approved source',
  GRN_NOT_POSTED: 'Goods receipt not posted',
  GRN_DATE_MISMATCH: 'Date does not match goods receipt',
  OPENING_NOT_APPROVED: 'Opening-stock date not approved',
  NO_ORIGIN_DATE: 'No origin date',
  FUTURE_ORIGIN_DATE: 'Origin date after today',
}

export const originSourceLabels = { GRN: 'Goods receipt', OPENING_STOCK: 'Opening stock' } as const

export const NO_ORIGIN_TEXT = 'No reliable origin date'

/** "12 days" for an aged unit; the missing-origin label otherwise (never "0 days"). */
export function stockAgeText(a: StockAgeing): string {
  return a.status === 'AGED' && a.ageDays !== null ? ageText(a.ageDays) : NO_ORIGIN_TEXT
}

export const originStatusLabels = {
  KNOWN: 'Has origin date',
  MISSING: 'No reliable origin date',
} as const

/** "Recommended bands (proposed, awaiting approval)" etc. */
export function bandSetNote(
  set: { status: 'PROPOSED' | 'APPROVED'; version: number } | null,
): string {
  if (!set) return 'No ageing bands are configured: ages are shown without bands.'
  return set.status === 'APPROVED'
    ? `Ageing bands version ${String(set.version)} (approved).`
    : `Ageing bands version ${String(set.version)} are proposed and awaiting owner approval.`
}
