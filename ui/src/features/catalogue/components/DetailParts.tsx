import {
  Archive,
  BadgeCheck,
  CheckCircle2,
  CircleDashed,
  Clock,
  FileCheck2,
  Pencil,
  Power,
  XCircle,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import type { CatalogueItemKind } from '../api/catalogue'
import type { BrandReadiness, LifecycleStatus, Provenance } from '../api/contracts'
import {
  checkLabel,
  confidenceLabels,
  formatDate,
  formatDateTime,
  personLabel,
  readinessSummary,
} from '../lib/presentation'
import { ActivateItemDialog, ArchiveDialog, VerifyDialog, type ItemRef } from './ActionDialogs'

export function DefinitionList({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
      {items.map(([term, value]) => (
        <div key={term} className="flex min-w-0 flex-col gap-0.5">
          <dt className="text-xs font-medium text-muted">{term}</dt>
          <dd className="break-words text-ink">{value}</dd>
        </div>
      ))}
    </dl>
  )
}

export function ArchivedBanner({ what }: { what: string }) {
  return (
    <p
      role="status"
      className="mb-4 flex items-start gap-2 rounded-card border border-border bg-neutral-subtle px-4 py-3 text-sm text-ink"
    >
      <Archive className="mt-0.5 size-4 shrink-0 text-neutral" aria-hidden="true" />
      This {what} is archived. It is kept for history and cannot be changed or used for new records.
    </p>
  )
}

/** Where the data came from and who checked it (BR-004). */
export function ProvenanceCard({ provenance }: { provenance: Provenance }) {
  const me = useMe()
  const unverified = provenance.confidence === 'UNVERIFIED'
  return (
    <Card title="Source and verification" description="Recorded when the details were confirmed.">
      {unverified ? (
        <p role="note" className="rounded-control bg-warning-subtle px-3 py-2 text-sm text-ink">
          <strong>Unverified.</strong> No supplier or client confirmation has been recorded. Do not
          present these details as fact.
        </p>
      ) : (
        <DefinitionList
          items={[
            ['Confidence', confidenceLabels[provenance.confidence]],
            [
              'Source link',
              provenance.sourceUrl ? (
                <a
                  href={provenance.sourceUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline"
                >
                  {provenance.sourceUrl}
                </a>
              ) : (
                '—'
              ),
            ],
            ['Source document', provenance.sourceDocumentId ? 'Private document attached' : '—'],
            ['Source date', formatDate(provenance.sourceDate)],
            ['Verified by', personLabel(provenance.verifiedBy, me.data?.user.id)],
            ['Verified at', formatDateTime(provenance.verifiedAt)],
          ]}
        />
      )}
    </Card>
  )
}

/**
 * Edit / verify / activate / archive for a model, variant or colour. Buttons
 * appear only with the permission and in a state the server allows; the server
 * checks both again.
 */
export function ItemActions({
  kind,
  item,
  status,
  confidence,
  parentNote,
  childrenNote,
  onEdit,
  compact,
}: {
  kind: CatalogueItemKind
  item: ItemRef
  status: LifecycleStatus
  confidence: string
  parentNote?: string
  childrenNote?: string
  onEdit?: () => void
  compact?: boolean
}) {
  const me = useMe()
  const perms = me.data?.permissions ?? []
  const [open, setOpen] = useState<'verify' | 'activate' | 'archive' | null>(null)
  const close = () => {
    setOpen(null)
  }
  const pending = status === 'PENDING_VERIFICATION'
  const size = compact ? 'sm' : 'md'
  const buttons: ReactNode[] = []
  if (pending && onEdit && perms.includes(Permission.CatalogueManage)) {
    buttons.push(
      <Button key="edit" variant="secondary" size={size} onClick={onEdit}>
        <Pencil className="size-3.5" aria-hidden="true" />
        Edit
      </Button>,
    )
  }
  if (pending && perms.includes(Permission.CatalogueVerify)) {
    buttons.push(
      <Button
        key="verify"
        variant="secondary"
        size={size}
        onClick={() => {
          setOpen('verify')
        }}
      >
        <FileCheck2 className="size-3.5" aria-hidden="true" />
        {confidence === 'UNVERIFIED' ? 'Verify' : 'Re-verify'}
      </Button>,
    )
  }
  if (pending && perms.includes(Permission.CatalogueActivate)) {
    buttons.push(
      <Button
        key="activate"
        size={size}
        onClick={() => {
          setOpen('activate')
        }}
      >
        <Power className="size-3.5" aria-hidden="true" />
        Activate
      </Button>,
    )
  }
  if (status !== 'ARCHIVED' && perms.includes(Permission.CatalogueArchive)) {
    buttons.push(
      <Button
        key="archive"
        variant="ghost"
        size={size}
        onClick={() => {
          setOpen('archive')
        }}
      >
        <Archive className="size-3.5" aria-hidden="true" />
        Archive
      </Button>,
    )
  }
  if (buttons.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-2">
      {buttons}
      <VerifyDialog open={open === 'verify'} onClose={close} kind={kind} item={item} />
      <ActivateItemDialog
        open={open === 'activate'}
        onClose={close}
        kind={kind}
        item={item}
        parentNote={parentNote}
      />
      <ArchiveDialog
        open={open === 'archive'}
        onClose={close}
        kind={kind}
        item={item}
        childrenNote={childrenNote}
      />
    </div>
  )
}

/** The server's brand activation checklist, shown as-is. */
export function ReadinessList({ readiness }: { readiness: BrandReadiness }) {
  const s = readinessSummary(readiness.checks)
  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-muted">
        {s.met} of {readiness.checks.length} requirements met
        {s.pending > 0 && ` · ${String(s.pending)} depend on features not built yet`}
      </p>
      <ul className="flex flex-col divide-y divide-border rounded-control border border-border">
        {readiness.checks.map((c) => {
          const Icon = c.met ? CheckCircle2 : c.pending ? Clock : XCircle
          const tone = c.met ? 'text-success' : c.pending ? 'text-muted' : 'text-warning'
          return (
            <li key={c.key} className="flex items-start gap-3 px-3 py-2.5 text-sm">
              <Icon className={`mt-0.5 size-4 shrink-0 ${tone}`} aria-hidden="true" />
              <div className="min-w-0">
                <p className="font-medium text-ink">
                  {checkLabel(c.key)}
                  <span className="sr-only">
                    {c.met ? ' — met' : c.pending ? ' — not available yet' : ' — not met'}
                  </span>
                </p>
                {!c.met && <p className="text-xs text-muted">{c.message}</p>}
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

/** Small "what's missing" list for the specs/prices that the catalogue cannot hold yet. */
export function DataGapsCard() {
  return (
    <Card
      title="Specifications, prices and warranty"
      description="What the catalogue can and cannot tell you yet."
    >
      <ul className="flex flex-col gap-2 text-sm text-ink">
        <li className="flex items-start gap-2">
          <CircleDashed className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <span>
            <strong>Specifications</strong> (battery, motor, range…) have no verified values yet. Do
            not quote specifications from this record.
          </span>
        </li>
        <li className="flex items-start gap-2">
          <CircleDashed className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <span>
            <strong>Prices</strong> come only from approved, effective-dated price history (shown to
            authorised roles). No price is shown or implied without an approved one.
          </span>
        </li>
        <li className="flex items-start gap-2">
          <CircleDashed className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <span>
            <strong>Warranty terms</strong> have not been supplied by the client yet.
          </span>
        </li>
      </ul>
    </Card>
  )
}

export function VerifiedMark({ verified }: { verified: boolean }) {
  return verified ? (
    <BadgeCheck className="size-4 text-success" aria-label="Verified" />
  ) : (
    <CircleDashed className="size-4 text-warning" aria-label="Unverified" />
  )
}
