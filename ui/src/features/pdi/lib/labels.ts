import type { BadgeTone } from '@/components/ui/Badge'
import type { VehicleUnitStatus } from '@/features/vehicles/api/vehicleUnits'
import type { Disposition, InspectionResult, Outcome } from '../api/pdi'

export const dispositionLabels: Record<Disposition, string> = {
  PASS: 'Pass',
  HOLD: 'Hold',
  REJECT: 'Reject',
}

/** PASS success, HOLD warning, REJECT error — always shown with the text label. */
export const dispositionTone: Record<Disposition, BadgeTone> = {
  PASS: 'success',
  HOLD: 'warning',
  REJECT: 'danger',
}

/** What each disposition does to the unit (the server applies it; nothing else is possible). */
export const dispositionEffect: Record<Disposition, string> = {
  PASS: 'The vehicle passes PDI and becomes available (saleable) stock.',
  HOLD: 'The vehicle goes on quality hold. It is not saleable and cannot be reserved or sold.',
  REJECT:
    'The vehicle is rejected. It is not saleable and cannot be reserved or sold. This cannot be undone here.',
}

export const confirmVerb: Record<Disposition, string> = {
  PASS: 'Record pass',
  HOLD: 'Put on hold',
  REJECT: 'Reject vehicle',
}

export const resultLabels: Record<InspectionResult, string> = {
  ALL_PASSED: 'All items passed',
  HAS_FAILURES: 'Items failed',
  NOT_CHECKED: 'No checklist configured',
}

export const outcomeLabels: Record<Outcome, string> = { PASS: 'Pass', FAIL: 'Fail' }

export const unitStatusLabels: Record<VehicleUnitStatus, string> = {
  ORDERED: 'Ordered',
  IN_TRANSIT: 'In transit',
  RECEIVED_PENDING_PDI: 'Awaiting PDI',
  QUALITY_HOLD: 'Quality hold',
  AVAILABLE: 'Available',
  DISPLAY: 'On display',
  RESERVED: 'Reserved',
  SERVICE_HOLD: 'Service hold',
  SOLD: 'Sold',
  DELIVERED: 'Delivered',
  REJECTED: 'Rejected',
  RETURN_PENDING_INSPECTION: 'Return pending inspection',
  RETURNED_TO_SUPPLIER: 'Returned to supplier',
}

export function unitStatusTone(status: VehicleUnitStatus): BadgeTone {
  switch (status) {
    case 'AVAILABLE':
      return 'success'
    case 'RECEIVED_PENDING_PDI':
    case 'QUALITY_HOLD':
    case 'SERVICE_HOLD':
      return 'warning'
    case 'REJECTED':
      return 'danger'
    case 'DISPLAY':
    case 'RESERVED':
      return 'info'
    default:
      return 'neutral'
  }
}
