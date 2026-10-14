import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { describeError, type ActionError } from '@/features/catalogue/lib/presentation'
import type { PriceItem, PriceList, PriceType } from '../api/pricing'
import {
  useApprovePriceItem,
  useArchivePriceList,
  useCancelPriceItem,
  useCreatePriceItem,
  useCreatePriceList,
  useRefreshPricing,
  useUpdatePriceItem,
} from '../hooks/usePricing'
import {
  formatINR,
  formatInstant,
  localInputToRFC3339,
  nextHourInput,
  toLocalInput,
} from '../lib/format'

const typeLabel: Record<PriceType, string> = { SELLING: 'selling price', COST: 'cost price' }

/** The snapshot rule, repeated wherever a price is approved (SAL-003). */
export function SnapshotNotice() {
  return (
    <p role="note" className="rounded-control bg-info-subtle px-3 py-2 text-xs text-ink">
      Approving or superseding a price never changes existing quotations or reservations: they keep
      the price snapshot taken when they were made. Approved prices cannot be edited later.
    </p>
  )
}

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshPricing()
  return { failure, setFailure, refresh }
}

function Footer({
  formId,
  pending,
  label,
  onCancel,
  danger,
}: {
  formId: string
  pending: boolean
  label: string
  onCancel: () => void
  danger?: boolean
}) {
  return (
    <>
      <Button variant="secondary" onClick={onCancel} disabled={pending}>
        Cancel
      </Button>
      <Button type="submit" form={formId} loading={pending} variant={danger ? 'danger' : 'primary'}>
        {label}
      </Button>
    </>
  )
}

// ── Draft / edit / supersede ────────────────────────────────────────────────

const priceFormSchema = z
  .object({
    amount: z
      .string()
      .trim()
      .regex(/^\d{1,10}(\.\d{1,2})?$/, 'Enter an amount like 125000 or 125000.50')
      .refine((v) => Number(v) > 0, 'Amount must be greater than zero'),
    effectiveFrom: z.string().min(1, 'Start is required'),
    effectiveTo: z.string(),
    sourceNote: z.string().trim().max(500, 'At most 500 characters'),
  })
  .refine((v) => v.effectiveTo === '' || v.effectiveTo > v.effectiveFrom, {
    message: 'End must be after the start',
    path: ['effectiveTo'],
  })
type PriceFormValues = z.input<typeof priceFormSchema>

export interface PriceFormTarget {
  list: PriceList
  variantId: string
  priceType: PriceType
  /** Edit this draft. */
  draft?: PriceItem
  /** Draft a successor of this approved price. */
  supersedes?: PriceItem
}

function PriceFormDialogBody({
  onClose,
  target,
}: {
  open: boolean
  onClose: () => void
  target: PriceFormTarget
}) {
  const formId = useId()
  const create = useCreatePriceItem()
  const update = useUpdatePriceItem()
  const { failure, setFailure, refresh } = useFailure()
  const { draft, supersedes } = target
  const form = useForm<PriceFormValues, unknown, PriceFormValues>({
    resolver: zodResolver(priceFormSchema),
    defaultValues: {
      amount: draft?.amount ?? '',
      effectiveFrom: draft ? toLocalInput(draft.effectiveFrom) : nextHourInput(),
      effectiveTo: draft?.effectiveTo ? toLocalInput(draft.effectiveTo) : '',
      sourceNote: draft?.sourceNote ?? '',
    },
  })
  const pending = create.isPending || update.isPending

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    const from = localInputToRFC3339(v.effectiveFrom)
    const to = v.effectiveTo ? localInputToRFC3339(v.effectiveTo) : null
    const note = v.sourceNote.trim() || null
    try {
      if (draft) {
        await update.mutateAsync({
          id: draft.id,
          input: {
            version: draft.version,
            amount: v.amount,
            effectiveFrom: from,
            effectiveTo: to,
            sourceNote: note,
          },
        })
      } else {
        await create.mutateAsync({
          priceListId: target.list.id,
          variantId: target.variantId,
          priceType: target.priceType,
          amount: v.amount,
          effectiveFrom: from,
          effectiveTo: to,
          sourceNote: note,
          supersedesId: supersedes?.id ?? null,
        })
      }
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['amount', 'effectiveFrom', 'effectiveTo', 'sourceNote'],
        }),
      )
    }
  })

  const errors = form.formState.errors
  const title = draft
    ? 'Edit draft price'
    : supersedes
      ? 'Draft a replacement price'
      : `Draft a ${typeLabel[target.priceType]}`
  return (
    <Dialog
      open
      onClose={onClose}
      title={title}
      description={`Price list “${target.list.name}”. Amounts are in Indian rupees (INR).`}
      footer={
        <Footer
          formId={formId}
          pending={pending}
          label={draft ? 'Save draft' : 'Save as draft'}
          onCancel={onClose}
        />
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
          Enter only a price from an approved source (e.g. the owner’s price circular). A draft is
          not used anywhere until another authorised person approves it.
        </p>
        {supersedes && (
          <p className="rounded-control bg-info-subtle px-3 py-2 text-xs text-ink">
            Replaces the approved price of {formatINR(supersedes.amount)} (from{' '}
            {formatInstant(supersedes.effectiveFrom)}). When this draft is approved, the old price
            ends at this draft’s start; the old price itself is never changed or deleted.
          </p>
        )}
        <Field label="Amount (₹)" required hint="Up to 2 decimals." error={errors.amount?.message}>
          {(p) => (
            <Input {...p} inputMode="decimal" autoComplete="off" {...form.register('amount')} />
          )}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Effective from"
            required
            hint="Your local time. Approval needs a start that is not in the past."
            error={errors.effectiveFrom?.message}
          >
            {(p) => <Input {...p} type="datetime-local" {...form.register('effectiveFrom')} />}
          </Field>
          <Field
            label="Effective until"
            hint="Optional; leave empty for open-ended."
            error={errors.effectiveTo?.message}
          >
            {(p) => <Input {...p} type="datetime-local" {...form.register('effectiveTo')} />}
          </Field>
        </div>
        <Field
          label="Source note"
          hint="Where the price comes from, e.g. circular number."
          error={errors.sourceNote?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('sourceNote')} />}
        </Field>
        <p className="text-xs text-muted">
          Tax treatment: not set. No approved tax configuration exists yet (accountant sign-off), so
          none can be attached.
        </p>
        {failure && (
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        )}
      </form>
    </Dialog>
  )
}

export function PriceFormDialog(props: {
  open: boolean
  onClose: () => void
  target: PriceFormTarget
}) {
  return props.open ? <PriceFormDialogBody {...props} /> : null
}

// ── Approve ─────────────────────────────────────────────────────────────────

const approveSchema = z.object({
  approvalReference: z.string().trim().max(200, 'At most 200 characters'),
})
type ApproveValues = z.input<typeof approveSchema>

function ApproveDialogBody({
  onClose,
  item,
}: {
  open: boolean
  onClose: () => void
  item: PriceItem
}) {
  const formId = useId()
  const approve = useApprovePriceItem()
  const { failure, setFailure, refresh } = useFailure()
  const form = useForm<ApproveValues, unknown, ApproveValues>({
    resolver: zodResolver(approveSchema),
    defaultValues: { approvalReference: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (approve.isPending) return
    setFailure(null)
    try {
      await approve.mutateAsync({
        id: item.id,
        version: item.version,
        approvalReference: v.approvalReference.trim() || undefined,
      })
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['approvalReference'] }))
    }
  })
  return (
    <Dialog
      open
      onClose={onClose}
      title="Approve price"
      description={`${formatINR(item.amount)} ${typeLabel[item.priceType]}, effective ${formatInstant(item.effectiveFrom)}${
        item.effectiveTo ? ` until ${formatInstant(item.effectiveTo)}` : ' (open-ended)'
      }.`}
      footer={
        <Footer
          formId={formId}
          pending={approve.isPending}
          label="Approve price"
          onCancel={onClose}
        />
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-3 text-sm"
      >
        <ul className="ml-5 list-disc space-y-1 text-ink">
          <li>You are recorded as the approver. You cannot approve a price you drafted.</li>
          <li>
            It cannot overlap another approved price of this list, variant and type
            {item.supersedesId ? '' : ' (supersede that price instead)'}.
          </li>
          {item.supersedesId && <li>The price it replaces will end at this price’s start.</li>}
        </ul>
        <SnapshotNotice />
        <Field
          label="Approval reference"
          hint="Optional, e.g. the owner’s signed price circular number."
          error={form.formState.errors.approvalReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('approvalReference')} />}
        </Field>
        {failure && (
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        )}
      </form>
    </Dialog>
  )
}

export function ApproveDialog(props: { open: boolean; onClose: () => void; item: PriceItem }) {
  return props.open ? <ApproveDialogBody {...props} /> : null
}

// ── Cancel draft ────────────────────────────────────────────────────────────

function CancelDraftBody({
  onClose,
  item,
}: {
  open: boolean
  onClose: () => void
  item: PriceItem
}) {
  const cancel = useCancelPriceItem()
  const { failure, setFailure, refresh } = useFailure()
  return (
    <Dialog
      open
      onClose={onClose}
      title="Cancel draft price"
      description={`The draft of ${formatINR(item.amount)} is kept as cancelled history and can no longer be approved.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={cancel.isPending}>
            Keep draft
          </Button>
          <Button
            variant="danger"
            loading={cancel.isPending}
            onClick={() => {
              setFailure(null)
              cancel.mutate(
                { id: item.id, version: item.version },
                {
                  onSuccess: onClose,
                  onError: (e) => {
                    setFailure(describeError(e))
                  },
                },
              )
            }}
          >
            Cancel draft
          </Button>
        </>
      }
    >
      {failure && (
        <ActionErrorBox
          error={failure}
          onReload={() => {
            void refresh()
            onClose()
          }}
        />
      )}
    </Dialog>
  )
}

export function CancelDraftDialog(props: { open: boolean; onClose: () => void; item: PriceItem }) {
  return props.open ? <CancelDraftBody {...props} /> : null
}

// ── Price lists ─────────────────────────────────────────────────────────────

const listSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(120, 'At most 120 characters'),
})
type ListValues = z.input<typeof listSchema>

function CreateListBody({
  onClose,
  brandId,
  brandName,
}: {
  open: boolean
  onClose: () => void
  brandId: string
  brandName: string
}) {
  const formId = useId()
  const create = useCreatePriceList()
  const { failure, setFailure } = useFailure()
  const form = useForm<ListValues, unknown, ListValues>({
    resolver: zodResolver(listSchema),
    defaultValues: { name: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (create.isPending) return
    setFailure(null)
    try {
      await create.mutateAsync({ brandId, name: v.name.trim() })
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['name'] }))
    }
  })
  return (
    <Dialog
      open
      onClose={onClose}
      title="Add price list"
      description={`A named list of prices for ${brandName}, in Indian rupees.`}
      footer={
        <Footer
          formId={formId}
          pending={create.isPending}
          label="Add price list"
          onCancel={onClose}
        />
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-3"
      >
        <Field
          label="Name"
          required
          hint="e.g. the showroom’s price list."
          error={form.formState.errors.name?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        {failure && <ActionErrorBox error={failure} />}
      </form>
    </Dialog>
  )
}

export function CreatePriceListDialog(props: {
  open: boolean
  onClose: () => void
  brandId: string
  brandName: string
}) {
  return props.open ? <CreateListBody {...props} /> : null
}

function ArchiveListBody({
  onClose,
  list,
}: {
  open: boolean
  onClose: () => void
  list: PriceList
}) {
  const archive = useArchivePriceList()
  const { failure, setFailure, refresh } = useFailure()
  return (
    <Dialog
      open
      onClose={onClose}
      title="Archive price list"
      description={`“${list.name}” will be archived. Nothing is deleted.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={archive.isPending}>
            Cancel
          </Button>
          <Button
            variant="danger"
            loading={archive.isPending}
            onClick={() => {
              setFailure(null)
              archive.mutate(
                { id: list.id, version: list.version },
                {
                  onSuccess: onClose,
                  onError: (e) => {
                    setFailure(describeError(e))
                  },
                },
              )
            }}
          >
            Archive price list
          </Button>
        </>
      }
    >
      <ul className="ml-5 list-disc space-y-1 text-sm text-ink">
        <li>No new prices can be drafted or approved on it.</li>
        <li>Its approved prices stay as history and still answer price lookups for past dates.</li>
      </ul>
      {failure && (
        <div className="mt-3">
          <ActionErrorBox
            error={failure}
            onReload={() => {
              void refresh()
              onClose()
            }}
          />
        </div>
      )}
    </Dialog>
  )
}

export function ArchivePriceListDialog(props: {
  open: boolean
  onClose: () => void
  list: PriceList
}) {
  return props.open ? <ArchiveListBody {...props} /> : null
}
