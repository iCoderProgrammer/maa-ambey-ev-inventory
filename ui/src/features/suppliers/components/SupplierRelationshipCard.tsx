import { Link } from 'react-router'
import { Card } from '@/components/ui/Card'
import { ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Permission } from '@/features/access/api/access'
import { usePermission } from '@/features/access/hooks/useAccess'
import { useBrandLinks } from '../hooks/useSuppliers'
import { LinkStatusBadge } from './SupplierBadges'

/**
 * Supplier relationships of a brand (D-01). Users who may view suppliers see
 * the real relationships; others only see whether the server counts one as
 * current. Nothing is assumed.
 */
export function SupplierRelationshipCard({
  brandId,
  linkMet,
}: {
  brandId: string
  linkMet: boolean
}) {
  const canView = usePermission(Permission.SupplierView).allowed
  const links = useBrandLinks({ brandId }, { enabled: canView })
  return (
    <Card
      title="Supplier relationship"
      description="Verified supplier ↔ brand relationships (Suppliers)."
    >
      {!canView ? (
        <p className="text-sm text-muted">
          {linkMet
            ? 'A verified, current supplier relationship is recorded.'
            : 'No verified, current supplier relationship is recorded. No supplier is shown or assumed for this brand.'}
        </p>
      ) : links.isPending ? (
        <LoadingState label="Loading supplier relationships…" />
      ) : links.isError ? (
        <ErrorState
          error={links.error}
          title="Could not load supplier relationships"
          onRetry={() => void links.refetch()}
        />
      ) : links.data.length === 0 ? (
        <p className="text-sm text-muted">
          No supplier relationship is recorded for this brand. No supplier is shown or assumed.
        </p>
      ) : (
        <ul
          className="flex flex-col divide-y divide-border rounded-control border border-border"
          aria-label="Supplier relationships"
        >
          {links.data.map((l) => (
            <li
              key={l.id}
              className="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5 text-sm"
            >
              <Link
                to={`/procurement/suppliers/${l.supplierId}`}
                className="font-medium text-primary hover:underline"
              >
                {l.supplierName}
              </Link>
              <LinkStatusBadge link={l} />
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}
