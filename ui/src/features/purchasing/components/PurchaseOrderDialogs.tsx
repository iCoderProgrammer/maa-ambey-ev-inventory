import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { useBrands, useModels, useVariants } from '@/features/catalogue/hooks/useCatalogue'
import { lifecycleLabels, type ActionError } from '@/features/catalogue/lib/presentation'
import { usePriceLists } from '@/features/pricing/hooks/usePricing'
import { formatINR } from '@/features/pricing/lib/format'
import { useSupplierList } from '@/features/suppliers/hooks/useSuppliers'
import { supplierStatusLabels } from '@/features/suppliers/lib/labels'
import {
  addLine,
  approvePurchaseOrder,
  cancelPurchaseOrder,
  createPurchaseOrder,
  removeLine,
  returnPurchaseOrder,
  submitPurchaseOrder,
  updateLine,
  updatePOSettings,
  updatePurchaseOrder,
  type PODetail,
  type POLine,
  type POSettings,
} from '../api/purchaseOrders'
import { poKeys, usePOMutation, useRefreshPurchaseOrders } from '../hooks/usePurchaseOrders'
import { describePOError } from '../lib/labels'

const orNull = (v: string) => (v.trim() === '' ? null : v.trim())
const amountRe = /^\d{1,10}(\.\d{1,2})?$/
const amount = (label: string) =>
  z
    .string()
    .trim()
    .refine((v) => amountRe.test(v) && Number(v) > 0, {
      message: `${label}: a positive amount with up to 2 decimals, e.g. 1000.00`,
    })

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshPurchaseOrders()
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

function Note({ children }: { children: ReactNode }) {
  return (
    <p role="note" className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
      {children}
    </p>
  )
}

// ── Header: create / edit ───────────────────────────────────────────────────

const headerSchema = z.object({
  supplierId: z.string().min(1, 'Choose a supplier'),
  expectedDate: z
    .string()
    .refine((v) => v === '' || /^\d{4}-\d{2}-\d{2}$/.test(v), { message: 'Enter a date' }),
  notes: z.string().trim().max(1000, 'At most 1000 characters'),
})
type HeaderValues = z.input<typeof headerSchema>
const headerFields = ['supplierId', 'expectedDate', 'notes'] as const

/** Suppliers that may be chosen (not archived); needs suppliers.view. */
function useSupplierOptions(current?: { id: string; name: string }) {
  const me = useMe()
  const canList = (me.data?.permissions ?? []).includes(Permission.SupplierView)
  const q = useSupplierList({ limit: 100 }, { enabled: canList })
  const options = (q.data?.items ?? [])
    .filter((s) => s.status !== 'ARCHIVED')
    .map((s) => ({
      value: s.id,
      label:
        s.status === 'VERIFIED'
          ? s.legalName
          : `${s.legalName} (${supplierStatusLabels[s.status].toLowerCase()})`,
    }))
  if (current && !options.some((o) => o.value === current.id)) {
    options.unshift({ value: current.id, label: current.name })
  }
  return { canList, q, options }
}

function HeaderFormBody({
  onClose,
  po,
  onSaved,
}: {
  onClose: () => void
  po?: PODetail
  onSaved?: (d: PODetail) => void
}) {
  const formId = useId()
  // One key per dialog: a double click or retry returns the PO created first.
  const [idemKey] = useState(() => crypto.randomUUID())
  const { failure, setFailure, refresh } = useFailure()
  const suppliers = useSupplierOptions(
    po ? { id: po.supplierId, name: po.supplierName } : undefined,
  )
  const save = usePOMutation((v: HeaderValues) => {
    const body = { expectedDate: orNull(v.expectedDate), notes: orNull(v.notes) }
    if (!po) return createPurchaseOrder({ supplierId: v.supplierId, ...body }, idemKey)
    return updatePurchaseOrder(po.id, {
      version: po.version,
      ...(v.supplierId !== po.supplierId ? { supplierId: v.supplierId } : {}),
      ...body,
    })
  })
  const form = useForm<HeaderValues, unknown, HeaderValues>({
    resolver: zodResolver(headerSchema),
    defaultValues: {
      supplierId: po?.supplierId ?? '',
      expectedDate: po?.expectedDate ?? '',
      notes: po?.notes ?? '',
    },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      const d = await save.mutateAsync(v)
      onSaved?.(d)
      onClose()
    } catch (e) {
      setFailure(describePOError(e, { setError: form.setError, names: headerFields }))
    }
  })
  return (
    <FormDialog
      title={po ? `Edit ${po.number}` : 'New purchase order'}
      description={
        po
          ? 'Drafts only. Lines are edited on the purchase order page.'
          : 'Creates a draft with the next PO number. Add lines, then submit it for approval.'
      }
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={po ? 'Save changes' : 'Create draft'}
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
        {!suppliers.canList ? (
          <Note>
            Choosing a supplier needs permission to view suppliers. Ask the owner to grant it.
          </Note>
        ) : (
          <Field
            label="Supplier"
            required
            error={errors.supplierId?.message}
            hint="A supplier must be verified, with a current verified relationship to each brand, before the PO can be submitted."
          >
            {(p) => (
              <Select
                {...p}
                placeholder={suppliers.q.isPending ? 'Loading suppliers…' : 'Choose a supplier'}
                options={suppliers.options}
                {...form.register('supplierId')}
              />
            )}
          </Field>
        )}
        <Field
          label="Expected delivery date"
          hint="Needed before submitting."
          error={errors.expectedDate?.message}
        >
          {(p) => <Input {...p} type="date" {...form.register('expectedDate')} />}
        </Field>
        <Field label="Notes" error={errors.notes?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('notes')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function HeaderFormDialog(props: {
  open: boolean
  onClose: () => void
  po?: PODetail
  onSaved?: (d: PODetail) => void
}) {
  return props.open ? <HeaderFormBody {...props} /> : null
}

// ── Lines ───────────────────────────────────────────────────────────────────

const lineSchema = z.object({
  brandId: z.string().min(1, 'Choose a brand'),
  modelId: z.string().min(1, 'Choose a model'),
  variantId: z.string().min(1, 'Choose a variant'),
  quantity: z
    .string()
    .trim()
    .refine((v) => /^\d+$/.test(v) && Number(v) >= 1 && Number(v) <= 10000, {
      message: 'A whole number from 1 to 10000',
    }),
  unitCost: amount('Agreed unit cost'),
  priceListId: z.string(),
  note: z.string().trim().max(300, 'At most 300 characters'),
})
type LineValues = z.input<typeof lineSchema>
const lineFields = [
  'brandId',
  'modelId',
  'variantId',
  'quantity',
  'unitCost',
  'priceListId',
  'note',
] as const

const itemLabel = (name: string, status: keyof typeof lifecycleLabels) =>
  status === 'ACTIVE' ? name : `${name} (${lifecycleLabels[status].toLowerCase()})`

function LineFormBody({ onClose, po, line }: { onClose: () => void; po: PODetail; line?: POLine }) {
  const formId = useId()
  const me = useMe()
  const perms = me.data?.permissions ?? []
  const canCatalogue = perms.includes(Permission.CatalogueView)
  const canCost = perms.includes(Permission.CostPriceView)
  const { failure, setFailure, refresh } = useFailure()
  const form = useForm<LineValues, unknown, LineValues>({
    resolver: zodResolver(lineSchema),
    defaultValues: {
      brandId: line?.brandId ?? '',
      modelId: line?.modelId ?? '',
      variantId: line?.variantId ?? '',
      quantity: line ? String(line.quantity) : '1',
      unitCost: line?.unitCost ?? '',
      priceListId: line?.referencePriceListId ?? '',
      note: line?.note ?? '',
    },
  })
  const [brandId, modelId] = useWatch({ control: form.control, name: ['brandId', 'modelId'] })
  const editing = Boolean(line)
  const brands = useBrands({ limit: 100 }, { enabled: canCatalogue && !editing })
  const models = useModels(
    { brandId, limit: 100 },
    { enabled: canCatalogue && !editing && brandId !== '' },
  )
  const variants = useVariants(
    { brandId, modelId, limit: 100 },
    { enabled: canCatalogue && !editing && modelId !== '' },
  )
  const lists = usePriceLists(
    { brandId, status: 'ACTIVE', limit: 100 },
    { enabled: canCost && brandId !== '' },
  )
  const notArchived = <T extends { lifecycleStatus: keyof typeof lifecycleLabels }>(items: T[]) =>
    items.filter((i) => i.lifecycleStatus !== 'ARCHIVED')

  const save = usePOMutation((v: LineValues) => {
    const priceListId = canCost ? orNull(v.priceListId) : undefined
    if (!line) {
      return addLine(po.id, {
        version: po.version,
        brandId: v.brandId,
        modelId: v.modelId,
        variantId: v.variantId,
        quantity: Number(v.quantity),
        unitCost: v.unitCost.trim(),
        note: orNull(v.note),
        ...(priceListId ? { priceListId } : {}),
      })
    }
    // Only a changed list retakes the snapshot; otherwise the old one is kept.
    const listChanged = canCost && (priceListId ?? null) !== line.referencePriceListId
    return updateLine(po.id, line.id, {
      version: po.version,
      quantity: Number(v.quantity),
      unitCost: v.unitCost.trim(),
      note: orNull(v.note),
      ...(listChanged ? { priceListId: priceListId ?? null } : {}),
    })
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describePOError(e, { setError: form.setError, names: lineFields }))
    }
  })
  return (
    <FormDialog
      title={line ? `Edit line ${String(line.lineNo)}` : 'Add line'}
      description={`${po.number} · one line per variant. The agreed unit cost comes from the supplier agreement (INR, excluding tax).`}
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={line ? 'Save line' : 'Add line'}
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
        {line ? (
          <p className="text-sm text-ink">
            <span className="font-medium">{line.brandName}</span> · {line.modelName} ·{' '}
            {line.variantName}
          </p>
        ) : !canCatalogue ? (
          <Note>Choosing a variant needs permission to view the catalogue.</Note>
        ) : (
          <>
            <Field label="Brand" required error={errors.brandId?.message}>
              {(p) => (
                <Select
                  {...p}
                  placeholder={brands.isPending ? 'Loading brands…' : 'Choose a brand'}
                  options={notArchived(brands.data?.items ?? []).map((b) => ({
                    value: b.id,
                    label: b.operational ? b.name : `${b.name} (not activated for purchase)`,
                  }))}
                  {...form.register('brandId', {
                    onChange: () => {
                      form.setValue('modelId', '')
                      form.setValue('variantId', '')
                      form.setValue('priceListId', '')
                    },
                  })}
                />
              )}
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Model" required error={errors.modelId?.message}>
                {(p) => (
                  <Select
                    {...p}
                    disabled={brandId === ''}
                    placeholder={brandId === '' ? 'Choose a brand first' : 'Choose a model'}
                    options={notArchived(models.data?.items ?? []).map((m) => ({
                      value: m.id,
                      label: itemLabel(m.name, m.lifecycleStatus),
                    }))}
                    {...form.register('modelId', {
                      onChange: () => {
                        form.setValue('variantId', '')
                      },
                    })}
                  />
                )}
              </Field>
              <Field label="Variant" required error={errors.variantId?.message}>
                {(p) => (
                  <Select
                    {...p}
                    disabled={modelId === ''}
                    placeholder={modelId === '' ? 'Choose a model first' : 'Choose a variant'}
                    options={notArchived(variants.data?.items ?? []).map((v) => ({
                      value: v.id,
                      label: itemLabel(v.name, v.lifecycleStatus),
                    }))}
                    {...form.register('variantId')}
                  />
                )}
              </Field>
            </div>
            {brands.data && brands.data.items.length === 0 && (
              <Note>No brands in the catalogue yet.</Note>
            )}
          </>
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Quantity" required error={errors.quantity?.message}>
            {(p) => <Input {...p} inputMode="numeric" {...form.register('quantity')} />}
          </Field>
          <Field label="Agreed unit cost (₹)" required error={errors.unitCost?.message}>
            {(p) => (
              <Input {...p} inputMode="decimal" autoComplete="off" {...form.register('unitCost')} />
            )}
          </Field>
        </div>
        {canCost && (
          <Field
            label="Catalogue cost reference"
            hint="Optional. Snapshots the approved COST price valid now from this list; later price changes do not change this PO."
            error={errors.priceListId?.message}
          >
            {(p) => (
              <Select
                {...p}
                disabled={brandId === ''}
                placeholder="No catalogue reference"
                options={(lists.data?.items ?? []).map((l) => ({ value: l.id, label: l.name }))}
                {...form.register('priceListId')}
              />
            )}
          </Field>
        )}
        {line?.referenceUnitCost && (
          <p className="text-xs text-muted">
            Current snapshot: {formatINR(line.referenceUnitCost)} per unit. Choose the list again to
            take a fresh snapshot, or “No catalogue reference” to remove it.
          </p>
        )}
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function LineFormDialog(props: {
  open: boolean
  onClose: () => void
  po: PODetail
  line?: POLine
}) {
  return props.open ? <LineFormBody {...props} /> : null
}

// ── Confirmations ───────────────────────────────────────────────────────────

function ConfirmBody({
  title,
  description,
  confirmLabel,
  danger,
  onClose,
  run,
  children,
}: {
  title: string
  description: string
  confirmLabel: string
  danger?: boolean
  onClose: () => void
  run: () => Promise<PODetail>
  children?: ReactNode
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const m = usePOMutation(run)
  return (
    <FormDialog
      title={title}
      description={description}
      onClose={onClose}
      formId={formId}
      pending={m.isPending}
      submitLabel={confirmLabel}
      danger={danger}
      failure={failure}
      onReload={() => {
        void refresh()
        onClose()
      }}
    >
      <form
        id={formId}
        onSubmit={(e) => {
          e.preventDefault()
          if (m.isPending) return
          setFailure(null)
          m.mutateAsync(undefined).then(onClose, (err: unknown) => {
            setFailure(describePOError(err))
          })
        }}
      >
        {children}
      </form>
    </FormDialog>
  )
}

export function RemoveLineDialog({
  po,
  line,
  onClose,
}: {
  po: PODetail
  line: POLine
  onClose: () => void
}) {
  return (
    <ConfirmBody
      title={`Remove line ${String(line.lineNo)}?`}
      description={`${line.brandName} · ${line.modelName} · ${line.variantName} × ${String(line.quantity)}. Only draft lines can be removed; the change is audited.`}
      confirmLabel="Remove line"
      danger
      onClose={onClose}
      run={() => removeLine(po.id, line.id, po.version)}
    />
  )
}

export function SubmitDialog({ po, onClose }: { po: PODetail; onClose: () => void }) {
  return (
    <ConfirmBody
      title={`Submit ${po.number}?`}
      description="The server checks the supplier, brands and catalogue again. Lines can no longer be changed after submission."
      confirmLabel="Submit purchase order"
      onClose={onClose}
      run={() => submitPurchaseOrder(po.id, po.version)}
    >
      <dl className="grid grid-cols-2 gap-2 text-sm">
        <dt className="text-muted">Total (excl. tax)</dt>
        <dd className="text-right font-medium">{formatINR(po.totalAmount)}</dd>
        <dt className="text-muted">Units</dt>
        <dd className="text-right">{po.totalQuantity}</dd>
      </dl>
      {po.approvalReason && <p className="mt-3 text-sm text-ink">{po.approvalReason}</p>}
    </ConfirmBody>
  )
}

const noteSchema = z.object({ text: z.string().trim().max(500, 'At most 500 characters') })
const reasonSchema = z.object({
  text: z.string().trim().min(1, 'A reason is required').max(500, 'At most 500 characters'),
})
type TextValues = z.input<typeof noteSchema>

function TextBody({
  title,
  description,
  label,
  required,
  confirmLabel,
  danger,
  onClose,
  run,
  children,
}: {
  title: string
  description: string
  label: string
  required?: boolean
  confirmLabel: string
  danger?: boolean
  onClose: () => void
  run: (text: string) => Promise<PODetail>
  children?: ReactNode
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const m = usePOMutation((v: TextValues) => run(v.text.trim()))
  const form = useForm<TextValues, unknown, TextValues>({
    resolver: zodResolver(required ? reasonSchema : noteSchema),
    defaultValues: { text: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (m.isPending) return
    setFailure(null)
    try {
      await m.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describePOError(e))
    }
  })
  return (
    <FormDialog
      title={title}
      description={description}
      onClose={onClose}
      formId={formId}
      pending={m.isPending}
      submitLabel={confirmLabel}
      danger={danger}
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
        {children}
        <Field label={label} required={required} error={form.formState.errors.text?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('text')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function ApproveDialog({ po, onClose }: { po: PODetail; onClose: () => void }) {
  return (
    <TextBody
      title={`Approve ${po.number}?`}
      description="The server checks the supplier, brands and catalogue again. You cannot approve a purchase order you created or submitted."
      label="Approval note"
      confirmLabel="Approve purchase order"
      onClose={onClose}
      run={(note) => approvePurchaseOrder(po.id, { version: po.version, note: note || null })}
    >
      <p className="text-sm text-ink">
        {po.supplierName} · {po.totalQuantity} units · {formatINR(po.totalAmount)} (excl. tax)
      </p>
    </TextBody>
  )
}

export function ReturnDialog({ po, onClose }: { po: PODetail; onClose: () => void }) {
  return (
    <TextBody
      title={`Return ${po.number} to draft?`}
      description="The creator can then change it and submit it again. The reason is kept in the history."
      label="Reason"
      required
      confirmLabel="Return to draft"
      onClose={onClose}
      run={(reason) => returnPurchaseOrder(po.id, { version: po.version, reason })}
    />
  )
}

export function CancelDialog({ po, onClose }: { po: PODetail; onClose: () => void }) {
  return (
    <TextBody
      title={`Cancel ${po.number}?`}
      description="The purchase order stays as history and cannot be reopened. Only possible while nothing has been received."
      label="Reason"
      required
      confirmLabel="Cancel purchase order"
      danger
      onClose={onClose}
      run={(reason) => cancelPurchaseOrder(po.id, { version: po.version, reason })}
    />
  )
}

// ── Approval settings ───────────────────────────────────────────────────────

const settingsSchema = z
  .object({
    approvalThreshold: z
      .string()
      .trim()
      .refine((v) => v === '' || (amountRe.test(v) && Number(v) > 0), {
        message: 'A positive amount with up to 2 decimals, or empty for “not configured”',
      }),
    firstTimeSupplierNeedsApproval: z.boolean(),
    approvalReference: z.string().trim().max(200, 'At most 200 characters'),
  })
  .refine((v) => v.approvalThreshold === '' || v.approvalReference !== '', {
    path: ['approvalReference'],
    message: 'Record the owner’s sign-off reference for a threshold',
  })
type SettingsValues = z.input<typeof settingsSchema>
const settingsFields = [
  'approvalThreshold',
  'firstTimeSupplierNeedsApproval',
  'approvalReference',
] as const

function SettingsBody({ settings, onClose }: { settings: POSettings; onClose: () => void }) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const queryClient = useQueryClient()
  const save = useMutation({
    mutationFn: (v: SettingsValues) =>
      updatePOSettings({
        version: settings.version,
        approvalThreshold: orNull(v.approvalThreshold),
        firstTimeSupplierNeedsApproval: v.firstTimeSupplierNeedsApproval,
        approvalReference: orNull(v.approvalReference),
      }),
    onSuccess: (s) => {
      queryClient.setQueryData(poKeys.settings(), s)
      return queryClient.invalidateQueries({ queryKey: poKeys.all })
    },
  })
  const form = useForm<SettingsValues, unknown, SettingsValues>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      approvalThreshold: settings.approvalThreshold ?? '',
      firstTimeSupplierNeedsApproval: settings.firstTimeSupplierNeedsApproval,
      approvalReference: settings.approvalReference ?? '',
    },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describePOError(e, { setError: form.setError, names: settingsFields }))
    }
  })
  return (
    <FormDialog
      title="Approval settings"
      description="Only record values the owner has signed off. Without a threshold every purchase order needs an approver."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel="Save settings"
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
        <Field
          label="Approval threshold (₹, excl. tax)"
          hint="POs with a total up to this amount need no approver. Leave empty for “not configured”."
          error={errors.approvalThreshold?.message}
        >
          {(p) => (
            <Input
              {...p}
              inputMode="decimal"
              autoComplete="off"
              {...form.register('approvalThreshold')}
            />
          )}
        </Field>
        <label className="flex items-center gap-2 text-sm text-ink">
          <input
            type="checkbox"
            className="size-4"
            {...form.register('firstTimeSupplierNeedsApproval')}
          />
          A supplier’s first purchase order always needs an approver
        </label>
        <Field
          label="Owner sign-off reference"
          required={false}
          error={errors.approvalReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('approvalReference')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function SettingsDialog(props: {
  open: boolean
  settings: POSettings | undefined
  onClose: () => void
}) {
  return props.open && props.settings ? (
    <SettingsBody settings={props.settings} onClose={props.onClose} />
  ) : null
}
