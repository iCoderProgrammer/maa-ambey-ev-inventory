import type { DocumentType, LinkStatus, SupplierStatus } from '../api/suppliers'

export const documentTypeLabels: Record<DocumentType, string> = {
  GST_REGISTRATION: 'GST registration',
  AUTHORISATION: 'Authorisation',
  AGREEMENT: 'Agreement',
  OTHER: 'Other',
}

export const supplierStatusLabels: Record<SupplierStatus, string> = {
  PENDING_VERIFICATION: 'Pending verification',
  VERIFIED: 'Verified',
  ARCHIVED: 'Archived',
}

export const linkStatusLabels: Record<LinkStatus, string> = {
  UNVERIFIED: 'Unverified',
  VERIFIED: 'Verified',
  ENDED: 'Ended',
}
