import { Badge, type BadgeTone } from '@/components/ui/Badge'
import type { BrandLink, SupplierStatus } from '../api/suppliers'
import { linkStatusLabels, supplierStatusLabels } from '../lib/labels'

const supplierTone: Record<SupplierStatus, BadgeTone> = {
  PENDING_VERIFICATION: 'warning',
  VERIFIED: 'success',
  ARCHIVED: 'neutral',
}

export function SupplierStatusBadge({ status }: { status: SupplierStatus }) {
  return <Badge tone={supplierTone[status]}>{supplierStatusLabels[status]}</Badge>
}

/** Relationship status; "Current" only when the server says so. */
export function LinkStatusBadge({ link }: { link: BrandLink }) {
  if (link.current) {
    return (
      <Badge tone="success" dot>
        Verified · current
      </Badge>
    )
  }
  const tone: BadgeTone =
    link.status === 'UNVERIFIED' ? 'warning' : link.status === 'ENDED' ? 'neutral' : 'info'
  const suffix = link.status === 'VERIFIED' ? ' · not current' : ''
  return <Badge tone={tone}>{linkStatusLabels[link.status] + suffix}</Badge>
}
