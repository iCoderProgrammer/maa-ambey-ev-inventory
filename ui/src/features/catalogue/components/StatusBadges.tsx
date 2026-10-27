import { Archive, BadgeCheck, CircleDashed, ShieldAlert, ShieldCheck, ShieldX } from 'lucide-react'
import { Badge, type BadgeTone } from '@/components/ui/Badge'
import type { AuthorisationStatus, Confidence, LifecycleStatus } from '../api/contracts'
import { authorisationLabels, confidenceLabels, lifecycleLabels } from '../lib/presentation'

const icon = 'size-3'

const lifecycleTone: Record<LifecycleStatus, BadgeTone> = {
  PENDING_VERIFICATION: 'warning',
  ACTIVE: 'success',
  ARCHIVED: 'neutral',
}

export function LifecycleBadge({ status }: { status: LifecycleStatus }) {
  return (
    <Badge
      tone={lifecycleTone[status]}
      icon={
        status === 'ARCHIVED' ? (
          <Archive className={icon} aria-hidden="true" />
        ) : status === 'ACTIVE' ? (
          <BadgeCheck className={icon} aria-hidden="true" />
        ) : (
          <CircleDashed className={icon} aria-hidden="true" />
        )
      }
    >
      {lifecycleLabels[status]}
    </Badge>
  )
}

const authTone: Record<AuthorisationStatus, BadgeTone> = {
  UNVERIFIED: 'warning',
  AUTHORISED: 'success',
  NOT_AUTHORISED: 'danger',
  REVOKED: 'danger',
}

export function AuthorisationBadge({ status }: { status: AuthorisationStatus }) {
  const Icon =
    status === 'AUTHORISED' ? ShieldCheck : status === 'UNVERIFIED' ? ShieldAlert : ShieldX
  return (
    <Badge tone={authTone[status]} icon={<Icon className={icon} aria-hidden="true" />}>
      {authorisationLabels[status]}
    </Badge>
  )
}

const confidenceTone: Record<Confidence, BadgeTone> = {
  UNVERIFIED: 'warning',
  SUPPLIER_CONFIRMED: 'info',
  CLIENT_APPROVED: 'success',
}

export function ConfidenceBadge({ confidence }: { confidence: Confidence }) {
  return <Badge tone={confidenceTone[confidence]}>{confidenceLabels[confidence]}</Badge>
}

/**
 * Purchase/sales use as the server derives it (`operational`). Never computed
 * in the UI, so nothing unverified can appear operational.
 */
export function OperationalBadge({ operational }: { operational: boolean }) {
  return operational ? (
    <Badge tone="success" dot>
      Enabled for purchase &amp; sales
    </Badge>
  ) : (
    <Badge tone="neutral">Not enabled for purchase &amp; sales</Badge>
  )
}
