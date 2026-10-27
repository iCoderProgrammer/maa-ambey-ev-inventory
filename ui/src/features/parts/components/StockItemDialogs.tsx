import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2 } from 'lucide-react'
import { useId, useState, type ReactNode } from 'react'
import {
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type UseFormRegister,
} from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { useBrands, useModels } from '@/features/catalogue/hooks/useCatalogue'
import { describeError, type ActionError } from '@/features/catalogue/lib/presentation'
import { useSupplierList } from '@/features/suppliers/hooks/useSuppliers'
import { ApiError } from '@/lib/api/client'
import {
  archiveCategory,
  archiveStockItem,
  createCategory,
  createStockItem,
  formatLevel,
  isWholeNumber,
  LEVEL_PATTERN,
  SKU_PATTERN,
  UOM_PATTERN,
  updateCategory,
  updateStockItem,
  type CompatibilityInput,
  type ItemCategory,
  type StockItemCreate,
  type StockItemDetail,
  type StockItemUpdate,
  type SupplierLinkInput,
  type TrackingMode,
} from '../api/stockItems'
import {
  useCategoryList,
  useRefreshStockItems,
  useStockItemMutation,
  useStockItemPerms,
} from '../hooks/useStockItems'
import { formPath, trackingModeHelp, trackingModeLabels } from '../lib/labels'

const opt = (max: number, label: string) =>
  z
    .string()
    .trim()
    .max(max, `${label}: at most ${String(max)} characters`)
const orNull = (v: string) => (v.trim() === '' ? null : v.trim())
const isTrackingMode = (v: string): v is TrackingMode => v === 'QUANTITY' || v === 'SERIAL'

/** Server field paths ("suppliers[1].supplierId") renamed to the form's ("suppliers.1.supplierId"). */
function withFormPaths(e: unknown): unknown {
  if (!(e instanceof ApiError)) return e
  return new ApiError(
    e.status,
    {
      code: e.code,
      message: e.message,
      requestId: e.requestId,
      details: e.details.map((d) => ({ ...d, field: formPath(d.field) })),
    },
    e.retryAfterSeconds,
  )
}

function useFailure() {
  const [failure, setFailure] = useState<ActionError | null>(null)
  const refresh = useRefreshStockItems()
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

// ── SKU create / edit ───────────────────────────────────────────────────────

function itemSchema(editing: boolean) {
  return z
    .object({
      sku: z.string().trim(),
      name: z.string().trim().min(1, 'Name is required').max(200, 'At most 200 characters'),
      description: opt(1000, 'Description'),
      categoryId: z.string().min(1, 'Category is required'),
      uom: z
        .string()
        .trim()
        .transform((v) => v.toUpperCase())
        .refine((v) => UOM_PATTERN.test(v), {
          message: 'Enter a unit code: 1–10 letters or digits, starting with a letter (e.g. PCS)',
        }),
      // Empty until chosen, so a plain string here; only the two modes pass.
      trackingMode: z.string().refine((v) => isTrackingMode(v), {
        message: 'Choose how the item is tracked',
      }),
      reorderLevel: z.string().trim(),
      suppliers: z.array(
        z.object({
          supplierId: z.string().min(1, 'Choose a supplier'),
          supplierPartNumber: opt(60, 'Supplier part number'),
        }),
      ),
      compatibility: z.array(
        z.object({ brandId: z.string().min(1, 'Choose a brand'), modelId: z.string() }),
      ),
    })
    .superRefine((v, ctx) => {
      if (!editing) {
        if (v.sku === '')
          ctx.addIssue({ code: 'custom', path: ['sku'], message: 'SKU is required' })
        else if (!SKU_PATTERN.test(v.sku))
          ctx.addIssue({
            code: 'custom',
            path: ['sku'],
            message: "1–40 letters, digits, '.', '_', '/' or '-' — no spaces",
          })
      }
      if (v.reorderLevel !== '') {
        if (!LEVEL_PATTERN.test(v.reorderLevel))
          ctx.addIssue({
            code: 'custom',
            path: ['reorderLevel'],
            message: 'Enter a number from 0 with up to 3 decimals (e.g. 5 or 2.5)',
          })
        else if (v.trackingMode === 'SERIAL' && !isWholeNumber(v.reorderLevel))
          ctx.addIssue({
            code: 'custom',
            path: ['reorderLevel'],
            message: 'Serial-tracked items count whole pieces: enter a whole number',
          })
      }
      const seen = new Set<string>()
      v.suppliers.forEach((s, i) => {
        if (s.supplierId && seen.has(s.supplierId))
          ctx.addIssue({
            code: 'custom',
            path: ['suppliers', i, 'supplierId'],
            message: 'Listed twice',
          })
        seen.add(s.supplierId)
      })
      const whole = new Set(v.compatibility.filter((c) => !c.modelId).map((c) => c.brandId))
      const keys = new Set<string>()
      v.compatibility.forEach((c, i) => {
        const key = `${c.brandId}/${c.modelId}`
        if (keys.has(key))
          ctx.addIssue({
            code: 'custom',
            path: ['compatibility', i, 'brandId'],
            message: 'Listed twice',
          })
        else if (c.modelId && whole.has(c.brandId))
          ctx.addIssue({
            code: 'custom',
            path: ['compatibility', i, 'modelId'],
            message: 'Already covered: the whole brand is listed',
          })
        keys.add(key)
      })
    })
}
type ItemValues = z.input<ReturnType<typeof itemSchema>>

function fieldNames(v: Pick<ItemValues, 'suppliers' | 'compatibility'>): string[] {
  return [
    'sku',
    'name',
    'description',
    'categoryId',
    'uom',
    'trackingMode',
    'reorderLevel',
    ...v.suppliers.flatMap((_, i) => [
      `suppliers.${String(i)}.supplierId`,
      `suppliers.${String(i)}.supplierPartNumber`,
    ]),
    ...v.compatibility.flatMap((_, i) => [
      `compatibility.${String(i)}.brandId`,
      `compatibility.${String(i)}.modelId`,
    ]),
  ]
}

const toSuppliers = (v: ItemValues): SupplierLinkInput[] =>
  v.suppliers.map((s) => ({
    supplierId: s.supplierId,
    supplierPartNumber: orNull(s.supplierPartNumber),
  }))
const toCompatibility = (v: ItemValues): CompatibilityInput[] =>
  v.compatibility.map((c) => ({ brandId: c.brandId, modelId: c.modelId || null }))
const sameSet = (a: unknown[], b: unknown[]) =>
  JSON.stringify(a.map((x) => JSON.stringify(x)).sort()) ===
  JSON.stringify(b.map((x) => JSON.stringify(x)).sort())

function initialValues(item?: StockItemDetail): ItemValues {
  return {
    sku: item?.sku ?? '',
    name: item?.name ?? '',
    description: item?.description ?? '',
    categoryId: item?.categoryId ?? '',
    uom: item?.uom ?? '',
    trackingMode: item?.trackingMode ?? '',
    reorderLevel: formatLevel(item?.reorderLevel ?? null) ?? '',
    suppliers: (item?.suppliers ?? [])
      .filter((s) => s.endedAt === null)
      .map((s) => ({ supplierId: s.supplierId, supplierPartNumber: s.supplierPartNumber ?? '' })),
    compatibility: (item?.compatibility ?? []).map((c) => ({
      brandId: c.brandId,
      modelId: c.modelId ?? '',
    })),
  }
}

/** Only what changed, so a PATCH never resends (or needs permission for) untouched parts. */
function changes(
  item: StockItemDetail,
  v: ItemValues,
  perms: { suppliers: boolean; catalogue: boolean },
): Omit<StockItemUpdate, 'version'> {
  const before = initialValues(item)
  const out: Omit<StockItemUpdate, 'version'> = {}
  if (v.name.trim() !== item.name) out.name = v.name.trim()
  if (orNull(v.description) !== item.description) out.description = orNull(v.description)
  if (v.categoryId !== item.categoryId) out.categoryId = v.categoryId
  if (v.uom.trim().toUpperCase() !== item.uom) out.uom = v.uom.trim().toUpperCase()
  if (orNull(v.reorderLevel) !== orNull(before.reorderLevel))
    out.reorderLevel = orNull(v.reorderLevel)
  if (perms.suppliers && !item.suppliersHidden && !sameSet(toSuppliers(v), toSuppliers(before)))
    out.suppliers = toSuppliers(v)
  if (perms.catalogue && !sameSet(toCompatibility(v), toCompatibility(before)))
    out.compatibility = toCompatibility(v)
  return out
}

function ItemFormBody({
  onClose,
  item,
  onSaved,
}: {
  onClose: () => void
  item?: StockItemDetail
  onSaved?: (d: StockItemDetail) => void
}) {
  const formId = useId()
  const editing = Boolean(item)
  const perms = useStockItemPerms()
  const canSuppliers = perms.suppliers && !item?.suppliersHidden
  const { failure, setFailure, refresh } = useFailure()
  const categories = useCategoryList({ status: 'ACTIVE', limit: 100 })
  const save = useStockItemMutation((v: ItemValues) => {
    if (item) return updateStockItem(item.id, { version: item.version, ...changes(item, v, perms) })
    const body: StockItemCreate = {
      sku: v.sku.trim(),
      name: v.name.trim(),
      description: orNull(v.description),
      categoryId: v.categoryId,
      uom: v.uom.trim().toUpperCase(),
      trackingMode: v.trackingMode as StockItemCreate['trackingMode'],
      reorderLevel: orNull(v.reorderLevel),
    }
    if (canSuppliers && v.suppliers.length > 0) body.suppliers = toSuppliers(v)
    if (perms.catalogue && v.compatibility.length > 0) body.compatibility = toCompatibility(v)
    return createStockItem(body)
  })
  const form = useForm<ItemValues, unknown, ItemValues>({
    resolver: zodResolver(itemSchema(editing)),
    defaultValues: initialValues(item),
  })
  const { control, register } = form
  const errors = form.formState.errors
  const tracking = useWatch({ control, name: 'trackingMode' })
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    if (item && Object.keys(changes(item, v, perms)).length === 0) {
      setFailure({ message: 'Nothing has changed. Edit a field, or cancel.' })
      return
    }
    try {
      const d = await save.mutateAsync(v)
      onSaved?.(d)
      onClose()
    } catch (e) {
      setFailure(describeError(withFormPaths(e), { setError: form.setError, names: fieldNames(v) }))
    }
  })
  const categoryOptions = (categories.data?.items ?? []).map((c) => ({
    value: c.id,
    label: c.name,
  }))
  if (item && !categoryOptions.some((o) => o.value === item.categoryId))
    categoryOptions.unshift({ value: item.categoryId, label: item.categoryName })

  return (
    <FormDialog
      title={editing ? 'Edit SKU' : 'Add SKU'}
      description={
        editing
          ? 'The SKU code and tracking mode cannot be changed after creation.'
          : 'A part or accessory record. Creating it records no stock.'
      }
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={editing ? 'Save SKU' : 'Add SKU'}
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
        <div className="grid gap-4 sm:grid-cols-2">
          {item ? (
            <ReadOnly label="SKU code" value={item.sku} hint="Fixed at creation." mono />
          ) : (
            <Field
              label="SKU code"
              required
              hint="Letters, digits, . _ / - (no spaces). Compared without case: BRK-01 and brk-01 are the same SKU; BRK-01 and BRK.01 are not."
              error={errors.sku?.message}
            >
              {(p) => (
                <Input {...p} autoComplete="off" className="font-mono" {...register('sku')} />
              )}
            </Field>
          )}
          <Field label="Name" required error={errors.name?.message}>
            {(p) => <Input {...p} autoComplete="off" {...register('name')} />}
          </Field>
        </div>
        <Field label="Description" error={errors.description?.message}>
          {(p) => <Input {...p} autoComplete="off" {...register('description')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Category"
            required
            hint={
              categories.isError
                ? 'Categories could not be loaded.'
                : categoryOptions.length === 0 && !categories.isPending
                  ? 'Add a category first.'
                  : undefined
            }
            error={errors.categoryId?.message}
          >
            {(p) => (
              <Select
                {...p}
                // Remount once the options arrive: a native select can only show a value it has an
                // option for, and the form writes its stored value in when the field (re)attaches.
                key={categories.isSuccess ? 'ready' : 'loading'}
                options={categoryOptions}
                placeholder={categories.isPending ? 'Loading…' : 'Choose a category'}
                {...register('categoryId')}
              />
            )}
          </Field>
          <Field
            label="Unit of measure"
            required
            hint="A unit code such as the supplier’s paperwork uses (e.g. PCS, SET). No approved list yet."
            error={errors.uom?.message}
          >
            {(p) => <Input {...p} autoComplete="off" className="uppercase" {...register('uom')} />}
          </Field>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {item ? (
            <ReadOnly
              label="Tracking mode"
              value={trackingModeLabels[item.trackingMode]}
              hint={`${trackingModeHelp[item.trackingMode]} Fixed at creation: stock records depend on it.`}
            />
          ) : (
            <Field
              label="Tracking mode"
              required
              hint={
                isTrackingMode(tracking)
                  ? `${trackingModeHelp[tracking]} Cannot be changed later.`
                  : 'Cannot be changed later.'
              }
              error={errors.trackingMode?.message}
            >
              {(p) => (
                <Select
                  {...p}
                  options={(['QUANTITY', 'SERIAL'] as const).map((m) => ({
                    value: m,
                    label: trackingModeLabels[m],
                  }))}
                  placeholder="Choose a tracking mode"
                  {...register('trackingMode')}
                />
              )}
            </Field>
          )}
          <Field
            label="Reorder level"
            hint={
              (item?.trackingMode ?? tracking) === 'SERIAL'
                ? 'Whole pieces. Optional.'
                : 'In the unit of measure, up to 3 decimals. Optional.'
            }
            error={errors.reorderLevel?.message}
          >
            {(p) => (
              <Input {...p} inputMode="decimal" autoComplete="off" {...register('reorderLevel')} />
            )}
          </Field>
        </div>
        {canSuppliers ? (
          <SupplierRows control={control} register={register} errors={errors} item={item} />
        ) : (
          <p className="rounded-control bg-neutral-subtle px-3 py-2 text-xs text-muted">
            Suppliers are not shown: choosing or seeing suppliers needs permission to view
            suppliers.
            {editing && ' Saving leaves the supplier links unchanged.'}
          </p>
        )}
        {perms.catalogue ? (
          <CompatibilityRows control={control} register={register} errors={errors} />
        ) : (
          <p className="rounded-control bg-neutral-subtle px-3 py-2 text-xs text-muted">
            Compatibility needs permission to view the catalogue.
            {editing && ' Saving leaves it unchanged.'}
          </p>
        )}
      </form>
    </FormDialog>
  )
}

function ReadOnly({
  label,
  value,
  hint,
  mono,
}: {
  label: string
  value: string
  hint: string
  mono?: boolean
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-ink">{label}</span>
      <span
        className={`rounded-control border border-border bg-neutral-subtle px-3 py-2 text-sm ${mono ? 'font-mono' : ''}`}
        aria-label={`${label} (cannot be changed)`}
      >
        {value}
      </span>
      <span className="text-xs text-muted">{hint}</span>
    </div>
  )
}

type Errors = ReturnType<typeof useForm<ItemValues>>['formState']['errors']

function SupplierRows({
  control,
  register,
  errors,
  item,
}: {
  control: Control<ItemValues>
  register: UseFormRegister<ItemValues>
  errors: Errors
  item?: StockItemDetail
}) {
  const { fields, append, remove } = useFieldArray({ control, name: 'suppliers' })
  const suppliers = useSupplierList({ limit: 100 })
  const current = new Set(
    (item?.suppliers ?? []).filter((s) => s.endedAt === null).map((s) => s.supplierId),
  )
  // Archived suppliers cannot be newly linked; one already linked stays listed so the form can show it.
  const options = (suppliers.data?.items ?? [])
    .filter((s) => s.status !== 'ARCHIVED' || current.has(s.id))
    .map((s) => ({
      value: s.id,
      label:
        s.legalName +
        (s.status === 'PENDING_VERIFICATION'
          ? ' (pending verification)'
          : s.status === 'ARCHIVED'
            ? ' (archived)'
            : ''),
    }))
  return (
    <fieldset className="flex flex-col gap-3 rounded-card border border-border p-3">
      <legend className="px-1 text-sm font-medium text-ink">Suppliers</legend>
      <p className="text-xs text-muted">
        Who supplies this item. A removed supplier is kept as history.
        {suppliers.isError && ' Suppliers could not be loaded.'}
      </p>
      {fields.map((f, i) => (
        <div key={f.id} className="grid items-end gap-2 sm:grid-cols-[1fr_12rem_auto]">
          <Field
            label={`Supplier ${String(i + 1)}`}
            error={errors.suppliers?.[i]?.supplierId?.message}
          >
            {(p) => (
              <Select
                {...p}
                key={suppliers.isSuccess ? 'ready' : 'loading'}
                options={options}
                placeholder={suppliers.isPending ? 'Loading…' : 'Choose a supplier'}
                {...register(`suppliers.${i}.supplierId`)}
              />
            )}
          </Field>
          <Field
            label={`Supplier part number ${String(i + 1)}`}
            error={errors.suppliers?.[i]?.supplierPartNumber?.message}
          >
            {(p) => (
              <Input {...p} autoComplete="off" {...register(`suppliers.${i}.supplierPartNumber`)} />
            )}
          </Field>
          <Button
            variant="ghost"
            size="sm"
            aria-label={`Remove supplier ${String(i + 1)}`}
            onClick={() => {
              remove(i)
            }}
          >
            <Trash2 className="size-3.5" aria-hidden="true" />
          </Button>
        </div>
      ))}
      <div>
        <Button
          variant="secondary"
          size="sm"
          disabled={fields.length >= 20}
          onClick={() => {
            append({ supplierId: '', supplierPartNumber: '' })
          }}
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add supplier
        </Button>
      </div>
    </fieldset>
  )
}

function CompatibilityRows({
  control,
  register,
  errors,
}: {
  control: Control<ItemValues>
  register: UseFormRegister<ItemValues>
  errors: Errors
}) {
  const { fields, append, remove } = useFieldArray({ control, name: 'compatibility' })
  const brands = useBrands({ limit: 100 })
  return (
    <fieldset className="flex flex-col gap-3 rounded-card border border-border p-3">
      <legend className="px-1 text-sm font-medium text-ink">Compatibility</legend>
      <p className="text-xs text-muted">
        Optional. A whole brand, or one model of it. Staff-entered information, not an approved
        manufacturer fitment.
        {brands.isError && ' Brands could not be loaded.'}
      </p>
      {fields.map((f, i) => (
        <CompatibilityRow
          key={f.id}
          index={i}
          control={control}
          register={register}
          errors={errors}
          brands={(brands.data?.items ?? [])
            .filter((b) => b.lifecycleStatus !== 'ARCHIVED')
            .map((b) => ({ value: b.id, label: b.name }))}
          brandsPending={brands.isPending}
          onRemove={() => {
            remove(i)
          }}
        />
      ))}
      <div>
        <Button
          variant="secondary"
          size="sm"
          disabled={fields.length >= 50}
          onClick={() => {
            append({ brandId: '', modelId: '' })
          }}
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add compatibility
        </Button>
      </div>
    </fieldset>
  )
}

function CompatibilityRow({
  index,
  control,
  register,
  errors,
  brands,
  brandsPending,
  onRemove,
}: {
  index: number
  control: Control<ItemValues>
  register: UseFormRegister<ItemValues>
  errors: Errors
  brands: { value: string; label: string }[]
  brandsPending: boolean
  onRemove: () => void
}) {
  const brandId = useWatch({ control, name: `compatibility.${index}.brandId` })
  // Models come from the catalogue for the chosen brand only, so a model of another brand cannot be picked.
  const models = useModels({ brandId, limit: 100 }, { enabled: Boolean(brandId) })
  const n = String(index + 1)
  return (
    <div className="grid items-end gap-2 sm:grid-cols-[1fr_1fr_auto]">
      <Field label={`Brand ${n}`} error={errors.compatibility?.[index]?.brandId?.message}>
        {(p) => (
          <Select
            {...p}
            key={brandsPending ? 'loading' : 'ready'}
            options={brands}
            placeholder={brandsPending ? 'Loading…' : 'Choose a brand'}
            {...register(`compatibility.${index}.brandId`)}
          />
        )}
      </Field>
      <Field
        label={`Model ${n}`}
        hint={models.isError ? 'Models could not be loaded.' : undefined}
        error={errors.compatibility?.[index]?.modelId?.message}
      >
        {(p) => (
          <Select
            {...p}
            key={`${brandId}-${models.isSuccess ? 'ready' : 'loading'}`}
            disabled={!brandId}
            options={(models.data?.items ?? [])
              .filter((m) => m.lifecycleStatus !== 'ARCHIVED' && m.brandId === brandId)
              .map((m) => ({ value: m.id, label: m.name }))}
            placeholder="All models of the brand"
            {...register(`compatibility.${index}.modelId`)}
          />
        )}
      </Field>
      <Button variant="ghost" size="sm" aria-label={`Remove compatibility ${n}`} onClick={onRemove}>
        <Trash2 className="size-3.5" aria-hidden="true" />
      </Button>
    </div>
  )
}

export function StockItemFormDialog(props: {
  open: boolean
  onClose: () => void
  item?: StockItemDetail
  onSaved?: (d: StockItemDetail) => void
}) {
  return props.open ? <ItemFormBody {...props} /> : null
}

// ── Archive (SKU and category) ──────────────────────────────────────────────

const reasonSchema = z.object({
  reason: z.string().trim().min(1, 'Give a reason').max(300, 'At most 300 characters'),
})
type ReasonValues = z.input<typeof reasonSchema>

function ArchiveBody({
  title,
  description,
  points,
  submitLabel,
  onClose,
  run,
}: {
  title: string
  description: string
  points: string[]
  submitLabel: string
  onClose: () => void
  run: (reason: string) => Promise<unknown>
}) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useStockItemMutation((v: ReasonValues) => run(v.reason.trim()))
  const form = useForm<ReasonValues, unknown, ReasonValues>({
    resolver: zodResolver(reasonSchema),
    defaultValues: { reason: '' },
  })
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['reason'] }))
    }
  })
  return (
    <FormDialog
      title={title}
      description={description}
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={submitLabel}
      danger
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
        className="flex flex-col gap-3 text-sm"
      >
        <ul className="ml-5 list-disc space-y-1 text-ink">
          {points.map((p) => (
            <li key={p}>{p}</li>
          ))}
        </ul>
        <Field
          label="Reason"
          required
          hint="Kept with the record and in the audit log."
          error={form.formState.errors.reason?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function ArchiveStockItemDialog({
  open,
  onClose,
  item,
}: {
  open: boolean
  onClose: () => void
  item: StockItemDetail
}) {
  if (!open) return null
  return (
    <ArchiveBody
      title="Archive SKU"
      description={`“${item.sku}” will be archived. Nothing is deleted.`}
      points={[
        'Its code stays taken, and its suppliers and compatibility stay as history.',
        'It can no longer be changed or chosen for new records. There is no reactivation.',
      ]}
      submitLabel="Archive SKU"
      onClose={onClose}
      run={(reason) => archiveStockItem(item.id, { version: item.version, reason })}
    />
  )
}

export function ArchiveCategoryDialog({
  open,
  onClose,
  category,
}: {
  open: boolean
  onClose: () => void
  category: ItemCategory
}) {
  if (!open) return null
  return (
    <ArchiveBody
      title="Archive category"
      description={`“${category.name}” will be archived. Nothing is deleted.`}
      points={[
        'Only a category without active SKUs can be archived: move or archive them first.',
        'Its archived SKUs keep it; no new SKU can be added to it. There is no reactivation.',
      ]}
      submitLabel="Archive category"
      onClose={onClose}
      run={(reason) => archiveCategory(category.id, { version: category.version, reason })}
    />
  )
}

// ── Category create / edit ──────────────────────────────────────────────────

const categorySchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100, 'At most 100 characters'),
  description: opt(500, 'Description'),
})
type CategoryValues = z.input<typeof categorySchema>

function CategoryFormBody({ onClose, category }: { onClose: () => void; category?: ItemCategory }) {
  const formId = useId()
  const { failure, setFailure, refresh } = useFailure()
  const save = useStockItemMutation((v: CategoryValues) => {
    if (!category)
      return createCategory({ name: v.name.trim(), description: orNull(v.description) })
    const body: { version: number; name?: string; description?: string | null } = {
      version: category.version,
    }
    if (v.name.trim() !== category.name) body.name = v.name.trim()
    if (orNull(v.description) !== category.description) body.description = orNull(v.description)
    return updateCategory(category.id, body)
  })
  const form = useForm<CategoryValues, unknown, CategoryValues>({
    resolver: zodResolver(categorySchema),
    defaultValues: { name: category?.name ?? '', description: category?.description ?? '' },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (save.isPending) return
    setFailure(null)
    if (
      category &&
      v.name.trim() === category.name &&
      orNull(v.description) === category.description
    ) {
      setFailure({ message: 'Nothing has changed. Edit a field, or cancel.' })
      return
    }
    try {
      await save.mutateAsync(v)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['name', 'description'] }))
    }
  })
  return (
    <FormDialog
      title={category ? 'Edit category' : 'Add category'}
      description="Categories group parts and accessories. Names are unique, ignoring case."
      onClose={onClose}
      formId={formId}
      pending={save.isPending}
      submitLabel={category ? 'Save category' : 'Add category'}
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
        <Field label="Name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <Field label="Description" error={errors.description?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('description')} />}
        </Field>
      </form>
    </FormDialog>
  )
}

export function CategoryFormDialog(props: {
  open: boolean
  onClose: () => void
  category?: ItemCategory
}) {
  return props.open ? <CategoryFormBody {...props} /> : null
}
