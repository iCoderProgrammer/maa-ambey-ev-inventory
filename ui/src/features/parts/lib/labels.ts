import type { ItemStatus, TrackingMode } from '../api/stockItems'

export const itemStatusLabels: Record<ItemStatus, string> = {
  ACTIVE: 'Active',
  ARCHIVED: 'Archived',
}

export const trackingModeLabels: Record<TrackingMode, string> = {
  QUANTITY: 'Quantity',
  SERIAL: 'Serial',
}

/** What each tracking mode means for future stock records (E-04). */
export const trackingModeHelp: Record<TrackingMode, string> = {
  QUANTITY: 'Counted in its unit of measure (e.g. 12 PCS).',
  SERIAL: 'Each piece will be identified by its own serial number.',
}

/** Field paths the server may name, in the form's own names ("suppliers[1].supplierId" → "suppliers.1.supplierId"). */
export function formPath(field: string): string {
  return field.replace(/\[(\d+)\]/g, '.$1')
}
