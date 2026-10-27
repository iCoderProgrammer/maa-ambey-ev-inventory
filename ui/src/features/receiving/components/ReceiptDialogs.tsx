import { zodResolver } from '@hookform/resolvers/zod'
import { PackageCheck } from 'lucide-react'
import { useId, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { AttachmentUploader } from '@/features/attachments/components/AttachmentUploader'
import { useAttachmentLimits, useMyAttachments } from '@/features/attachments/hooks/useAttachments'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { SearchBox } from '@/features/catalogue/components/ListControls'
import {
  describeError,
  formatDate,
  todayISO,
  type ActionError,
} from '@/features/catalogue/lib/presentation'
import { poStatusLabels } from '@/features/purchasing/lib/labels'
import { cn } from '@/lib/utils/cn'
import {
  addGoodsReceiptDocument,
  cancelGoodsReceipt,
  createGoodsReceipt,
  documentTypes,
  type GRNDetail,
} from '../api/goodsReceipts'
import {
  useGRNMutation,
  useReceivablePurchaseOrders,
  useRefreshGoodsReceipts,
} from '../hooks/useGoodsReceipts'
import { describeGRNError, documentTypeLabels } from '../lib/labels'

const orNull = (v: string) => (v.trim() === '' ? null : v.trim())

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshGoodsReceipts()
  return { failure, setFailure, refresh }
}

function FormDialog({
  title,
  description,
  onClose,
  formId,
  pending,
  submitLabel,
  danger,
  children,
  failure,
  onReload,
}: {
  title: string
  description?: string
  onClose: () => void
  formId: string
  pending: boolean
  submitLabel: string
  danger?: boolean
  children: ReactNode
  failure: ActionError | null
  onReload: () => void
}) {
  return (
    <Dialog
      open
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={pending}>
            Cancel
          </Button>
          <Button
            type="submit"
            form={formId}
            loading={pending}
            variant={danger ? 'danger' : 'primary'}
          >
            {submitLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {children}
        {failure && <ActionErrorBox error={failure} onReload={onReload} />}
      </div>
    </Dialog>
  )
}

// ── New receipt: choose an approved PO ──────────────────────────────────────

const newSchema = z.object({
  purchaseOrderId: z.string().min(1, 'Choose the purchase order the goods arrived for'),
  receivedDate: z
    .string()
    .refine((v) => /^\d{4}-\d{2}-\d{2}$/.test(v), { message: 'Enter the date the goods arrived' })
    .refine((v) => v <= todayISO(), { message: 'Cannot be in the future' }),
  supplierReference: z.string().trim().max(100, 'At most 100 characters'),
  notes: z.string().trim().max(1000, 'At most 1000 characters'),
})
type NewValues = z.input<typeof newSchema>
const newFields = ['purchaseOrderId', 'receivedDate', 'supplierReference', 'notes'] as const

function NewReceiptBody({
  onClose,
  onSaved,
  purchaseOrderId,
}: {
  onClose: () => void
  onSaved: (d: GRNDetail) => void
  purchaseOrderId?: string
}) {
  const formId = useId()
  // One key per dialog: a double click or retry returns the receipt created first.
  const [idemKey] = useState(() => crypto.randomUUID())
  const [q, setQ] = useState('')
  const pos = useReceivablePurchaseOrders(q)
  const { failure, setFailure, refresh } = useFailure()
  const save = useGRNMutation((v: NewValues) =>
    createGoodsReceipt(
      {
        purchaseOrderId: v.purchaseOrderId,
        receivedDate: v.receivedDate,
        supplierReference: orNull(v.supplierReference),
        notes: orNull(v.notes),
      },
      idemKey,
    ),
  )
  const form = useForm<NewValues, unknown, NewValues>({
    resolver: zodResolver(newSchema),
    defaultValues: {
      purchaseOrderId: purchaseOrderId ?? '',
      receivedDate: todayISO(),
      supplierReference: '',
      notes: '',
    },
  })
  const chosen = useWatch({ control: form.control, name: 'purchaseOrderId' })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      const d = await save.mutateAsync(v)
      onSaved(d)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: newFields }))
    }
  })

  let list: ReactNode
  if (pos.isPending) list = <LoadingState label="Loading approved purchase orders…" />
  else if (pos.isError && !pos.data) {
    list = (
      <ErrorState
        error={pos.error}
        title="Could not load purchase orders"
        onRetry={() => void pos.refetch()}
      />
    )
  } else if (pos.data.items.length === 0) {
    list = (
      <EmptyState
        title={q ? 'No approved purchase order matches' : 'No purchase order is open for receipts'}
        icon={<PackageCheck className="size-6 text-muted/80" />}
      >
        Goods can be received only against an approved purchase order.
      </EmptyState>
    )
  } else {
    list = (
      <fieldset aria-describedby={errors.purchaseOrderId ? `${formId}-po-error` : undefined}>
        <legend className="mb-2 text-sm font-medium text-ink">
          Purchase order
          <span className="ml-0.5 text-danger" aria-hidden="true">
            *
          </span>
        </legend>
        <ul className="flex max-h-72 flex-col gap-2 overflow-y-auto">
          {pos.data.items.map((po) => (
            <li key={po.id}>
              <label
                className={cn(
                  'flex cursor-pointer items-start gap-3 rounded-control border px-3 py-2 text-sm',
                  chosen === po.id ? 'border-primary bg-primary/5' : 'border-border',
                )}
              >
                <input
                  type="radio"
                  value={po.id}
                  className="mt-1"
                  {...form.register('purchaseOrderId')}
                />
                <span className="flex min-w-0 flex-col">
                  <span className="font-medium text-ink">
                    <span className="font-mono">{po.number}</span> · {po.supplierName}
                  </span>
                  <span className="text-xs text-muted">
                    {poStatusLabels[po.status]} · expected {formatDate(po.expectedDate)} ·{' '}
                    {po.brandNames.join(', ')}
                  </span>
                  <span className="text-xs text-ink">
                    {po.availableQuantity} of {po.orderedQuantity} units still to receive
                    {po.openDraftCount > 0 &&
                      ` · ${String(po.openDraftCount)} open draft${po.openDraftCount === 1 ? '' : 's'} hold ${String(po.draftQuantity)}`}
                  </span>
                </span>
              </label>
            </li>
          ))}
        </ul>
        {errors.purchaseOrderId && (
          <p
            id={`${formId}-po-error`}
            role="alert"
            className="mt-1 text-xs font-medium text-danger"
          >
            {errors.purchaseOrderId.message}
          </p>
        )}
      </fieldset>
    )
  }

  return (
    <FormDialog
      title="New goods receipt"
      description="Starts a draft against an approved purchase order. A draft does not change stock."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Start draft"
      failure={failure}
      onReload={() => {
        void refresh()
      }}
    >
      <SearchBox label="Search PO number or supplier" value={q} onSearch={setQ} />
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        {list}
        <Field
          label="Date received"
          required
          error={errors.receivedDate?.message}
          hint="The day the goods arrived."
        >
          {(p) => <Input {...p} type="date" max={todayISO()} {...form.register('receivedDate')} />}
        </Field>
        <Field
          label="Supplier delivery reference"
          error={errors.supplierReference?.message}
          hint="Delivery challan or invoice number as printed."
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('supplierReference')} />}
        </Field>
        <Field label="Notes" error={errors.notes?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('notes')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function NewReceiptDialog(props: {
  open: boolean
  onClose: () => void
  onSaved: (d: GRNDetail) => void
  purchaseOrderId?: string
}) {
  return props.open ? <NewReceiptBody {...props} /> : null
}

// ── Cancel draft ────────────────────────────────────────────────────────────

const reasonSchema = z.object({
  reason: z.string().trim().min(1, 'A reason is required').max(500, 'At most 500 characters'),
})
type ReasonValues = z.input<typeof reasonSchema>

export function CancelReceiptDialog({ grn, onClose }: { grn: GRNDetail; onClose: () => void }) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const m = useGRNMutation((v: ReasonValues) =>
    cancelGoodsReceipt(grn.id, { version: grn.version, reason: v.reason.trim() }),
  )
  const form = useForm<ReasonValues, unknown, ReasonValues>({
    resolver: zodResolver(reasonSchema),
    defaultValues: { reason: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (m.isPending) return
    setFailure(null)
    try {
      await m.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeGRNError(e))
    }
  })
  return (
    <FormDialog
      title={`Cancel ${grn.number}?`}
      description="The draft stays as history and cannot be reopened. Its quantities no longer count against the purchase order. Nothing in stock changes."
      onClose={onClose}
      formId={formId}
      pending={m.isPending}
      submitLabel="Cancel draft"
      danger
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form id={formId} noValidate onSubmit={(e) => void onSubmit(e)}>
        <Field label="Reason" required error={form.formState.errors.reason?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

// ── Attach document ─────────────────────────────────────────────────────────

const docSchema = z.object({
  attachmentId: z.string().min(1, 'Upload a file or choose one of your uploads'),
  documentType: z.enum(documentTypes),
  note: z.string().trim().max(300, 'At most 300 characters'),
})
type DocValues = z.input<typeof docSchema>

function DocumentBody({ grn, onClose }: { grn: GRNDetail; onClose: () => void }) {
  const formId = useId()
  const me = useMe()
  const canView = me.data?.permissions.includes(Permission.AttachmentsView) ?? false
  const mine = useMyAttachments(canView)
  const limits = useAttachmentLimits(true)
  const { failure, setFailure, refresh } = useFailure()
  const save = useGRNMutation((v: DocValues) =>
    addGoodsReceiptDocument(grn.id, {
      attachmentId: v.attachmentId,
      documentType: v.documentType,
      note: orNull(v.note),
    }),
  )
  const form = useForm<DocValues, unknown, DocValues>({
    resolver: zodResolver(docSchema),
    defaultValues: { attachmentId: '', documentType: 'DELIVERY_CHALLAN', note: '' },
  })
  const attachmentId = useWatch({ control: form.control, name: 'attachmentId' })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['attachmentId', 'documentType', 'note'],
        }),
      )
    }
  })
  return (
    <FormDialog
      title="Attach document"
      description="Delivery challan, supplier invoice or photos. Files stay private: only people allowed to view goods receipts and documents can open them. Attaching is saved at once and cannot be undone."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Attach document"
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field label="Document type" required error={errors.documentType?.message}>
          {(p) => (
            <Select
              {...p}
              options={documentTypes.map((t) => ({ value: t, label: documentTypeLabels[t] }))}
              {...form.register('documentType')}
            />
          )}
        </Field>
        {canView && (
          <Field
            label="Your uploads"
            hint={mine.isPending ? 'Loading your uploads…' : undefined}
            error={errors.attachmentId?.message}
          >
            {(p) => (
              <Select
                {...p}
                options={(mine.data ?? []).map((a) => ({ value: a.id, label: a.fileName }))}
                placeholder="Choose a file"
                value={attachmentId}
                onChange={(e) => {
                  form.setValue('attachmentId', e.target.value, {
                    shouldValidate: form.formState.isSubmitted,
                  })
                }}
              />
            )}
          </Field>
        )}
        {limits.isError && (
          <p role="alert" className="text-xs text-danger">
            Could not load the upload limits: {describeError(limits.error).message}
          </p>
        )}
        {limits.data && (
          <div className="rounded-control border border-border p-3">
            <p className="mb-2 text-sm font-medium text-ink">Upload a new file</p>
            <AttachmentUploader
              limits={limits.data}
              onUploaded={(a) => {
                form.setValue('attachmentId', a.id, { shouldValidate: true })
              }}
            />
          </div>
        )}
        {!canView && errors.attachmentId && (
          <p role="alert" className="text-xs font-medium text-danger">
            {errors.attachmentId.message}
          </p>
        )}
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function DocumentDialog(props: { open: boolean; grn: GRNDetail; onClose: () => void }) {
  return props.open ? <DocumentBody grn={props.grn} onClose={props.onClose} /> : null
}
