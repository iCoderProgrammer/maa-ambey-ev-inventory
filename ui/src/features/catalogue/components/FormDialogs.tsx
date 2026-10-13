import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import type { Brand, Variant, VariantColour, VehicleModel } from '../api/contracts'
import {
  useBrands,
  useCreateBrand,
  useCreateColour,
  useCreateModel,
  useCreateVariant,
  useModels,
  useRefreshCatalogue,
  useUpdateBrand,
  useUpdateItem,
} from '../hooks/useCatalogue'
import { describeError, type ActionError } from '../lib/presentation'
import {
  brandFormSchema,
  colourFormSchema,
  modelFormSchema,
  orNull,
  variantFormSchema,
  type BrandFormValues,
  type ColourFormValues,
  type ModelFormValues,
  type VariantFormValues,
} from '../lib/schemas'
import { ActionErrorBox } from './ActionErrorBox'

/** Options for a parent picker: never offer archived parents (the server refuses them too). */
const OPTION_LIMIT = 100

function Footer({
  formId,
  pending,
  submitLabel,
  onCancel,
}: {
  formId: string
  pending: boolean
  submitLabel: string
  onCancel: () => void
}) {
  return (
    <>
      <Button variant="secondary" onClick={onCancel} disabled={pending}>
        Cancel
      </Button>
      <Button type="submit" form={formId} loading={pending}>
        {submitLabel}
      </Button>
    </>
  )
}

function UnverifiedNote() {
  return (
    <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
      New records start <strong>Pending verification</strong> and <strong>Unverified</strong>. Enter
      only details from a supplier or client source; verification is recorded separately.
    </p>
  )
}

// ── Brand ───────────────────────────────────────────────────────────────────

function BrandFormDialogBody({
  open,
  onClose,
  brand,
  onSaved,
}: {
  open: boolean
  onClose: () => void
  /** Edit this brand; omitted = create. */
  brand?: Brand
  onSaved?: (brand: Brand) => void
}) {
  const formId = useId()
  const create = useCreateBrand()
  const update = useUpdateBrand()
  const refresh = useRefreshCatalogue()
  const [failure, setFailure] = useState<ActionError | null>(null)
  const form = useForm<BrandFormValues, unknown, BrandFormValues>({
    resolver: zodResolver(brandFormSchema),
    defaultValues: { name: brand?.name ?? '', slug: brand?.slug ?? '' },
  })
  const pending = create.isPending || update.isPending

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    try {
      const saved = brand
        ? await update.mutateAsync({
            id: brand.id,
            input: { version: brand.version, name: v.name, ...(v.slug ? { slug: v.slug } : {}) },
          })
        : await create.mutateAsync({ name: v.name, ...(v.slug ? { slug: v.slug } : {}) })
      onSaved?.(saved)
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['name', 'slug'] }))
    }
  })

  const errors = form.formState.errors
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={brand ? 'Edit brand' : 'Add brand'}
      description={
        brand
          ? 'Change the display name or URL slug. Authorisation and activation are separate steps.'
          : 'Brands are data: add any brand the showroom deals in. Adding a brand does not authorise it for purchase or sales.'
      }
      footer={
        <Footer
          formId={formId}
          pending={pending}
          submitLabel={brand ? 'Save brand' : 'Add brand'}
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
        {!brand && <UnverifiedNote />}
        <Field label="Brand name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <Field
          label="Slug"
          hint="Lowercase letters, digits and hyphens. Leave empty to derive it from the name."
          error={errors.slug?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('slug')} />}
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

// ── Model ───────────────────────────────────────────────────────────────────

/** Non-archived brands for a picker (the server refuses archived parents as well). */
function useBrandOptions(enabled: boolean) {
  const brands = useBrands({ limit: OPTION_LIMIT }, { enabled })
  const options = (brands.data?.items ?? [])
    .filter((b) => b.lifecycleStatus !== 'ARCHIVED')
    .map((b) => ({ value: b.id, label: b.name }))
  return { brands, options }
}

function ModelFormDialogBody({
  open,
  onClose,
  model,
  fixedBrand,
  onSaved,
}: {
  open: boolean
  onClose: () => void
  model?: VehicleModel
  /** Create under this brand (no brand picker). */
  fixedBrand?: Brand
  onSaved?: (model: VehicleModel) => void
}) {
  const formId = useId()
  const create = useCreateModel()
  const update = useUpdateItem('models')
  const refresh = useRefreshCatalogue()
  const [failure, setFailure] = useState<ActionError | null>(null)
  const pickBrand = !model && !fixedBrand
  const { brands, options } = useBrandOptions(open && pickBrand)
  const defaults = (): ModelFormValues => ({
    brandId: model?.brandId ?? fixedBrand?.id ?? '',
    name: model?.name ?? '',
    internalCode: model?.internalCode ?? '',
    manufacturerCode: model?.manufacturerCode ?? '',
    vehicleType: model?.vehicleType ?? '',
  })
  const form = useForm<ModelFormValues, unknown, ModelFormValues>({
    resolver: zodResolver(modelFormSchema),
    defaultValues: defaults(),
  })
  const pending = create.isPending || update.isPending

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    try {
      const saved = model
        ? await update.mutateAsync({
            id: model.id,
            input: {
              version: model.version,
              name: v.name,
              internalCode: v.internalCode,
              manufacturerCode: orNull(v.manufacturerCode),
              vehicleType: orNull(v.vehicleType),
            },
          })
        : await create.mutateAsync({
            brandId: v.brandId,
            name: v.name,
            internalCode: v.internalCode,
            manufacturerCode: orNull(v.manufacturerCode),
            vehicleType: orNull(v.vehicleType),
          })
      onSaved?.(saved)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['brandId', 'name', 'internalCode', 'manufacturerCode', 'vehicleType'],
        }),
      )
    }
  })

  const errors = form.formState.errors
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={model ? 'Edit model' : 'Add model'}
      description={
        model
          ? 'Only pending models can be edited. Editing a verified model returns it to Unverified.'
          : fixedBrand
            ? `New model for ${fixedBrand.name}.`
            : 'The model belongs to one brand for good; it cannot move to another brand later.'
      }
      footer={
        <Footer
          formId={formId}
          pending={pending}
          submitLabel={model ? 'Save model' : 'Add model'}
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
        {!model && <UnverifiedNote />}
        {model?.confidence && model.confidence !== 'UNVERIFIED' && (
          <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
            This model is verified. Saving changes clears its verification; it must be verified
            again.
          </p>
        )}
        {pickBrand && (
          <Field
            label="Brand"
            required
            hint={
              brands.isPending
                ? 'Loading brands…'
                : brands.isError
                  ? 'Could not load brands.'
                  : undefined
            }
            error={errors.brandId?.message}
          >
            {(p) => (
              <Select
                {...p}
                options={options}
                placeholder="Choose a brand"
                {...form.register('brandId')}
              />
            )}
          </Field>
        )}
        <Field label="Model name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Internal code"
            required
            hint="Unique within the brand; stored upper-case."
            error={errors.internalCode?.message}
          >
            {(p) => <Input {...p} autoComplete="off" {...form.register('internalCode')} />}
          </Field>
          <Field
            label="Manufacturer code"
            hint="Only if the supplier documents one."
            error={errors.manufacturerCode?.message}
          >
            {(p) => <Input {...p} autoComplete="off" {...form.register('manufacturerCode')} />}
          </Field>
        </div>
        <Field
          label="Vehicle type"
          hint="Optional upper-case classification, e.g. SCOOTER. The value list is not confirmed yet (OQ-11)."
          error={errors.vehicleType?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('vehicleType')} />}
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

// ── Variant ─────────────────────────────────────────────────────────────────

function VariantFormDialogBody({
  open,
  onClose,
  variant,
  fixedModel,
  onSaved,
}: {
  open: boolean
  onClose: () => void
  variant?: Variant
  /** Create under this model (no pickers). */
  fixedModel?: VehicleModel
  onSaved?: (variant: Variant) => void
}) {
  const formId = useId()
  const create = useCreateVariant()
  const update = useUpdateItem('variants')
  const refresh = useRefreshCatalogue()
  const [failure, setFailure] = useState<ActionError | null>(null)
  const pick = !variant && !fixedModel
  const { brands, options: brandOptions } = useBrandOptions(open && pick)
  const defaults = (): VariantFormValues => ({
    brandId: variant?.brandId ?? fixedModel?.brandId ?? '',
    modelId: variant?.modelId ?? fixedModel?.id ?? '',
    name: variant?.name ?? '',
    variantCode: variant?.variantCode ?? '',
    manufacturerCode: variant?.manufacturerCode ?? '',
  })
  const form = useForm<VariantFormValues, unknown, VariantFormValues>({
    resolver: zodResolver(variantFormSchema),
    defaultValues: defaults(),
  })

  // Brand → model: the model list holds only the chosen brand's models, so a
  // cross-brand pair cannot be picked. The server derives the brand anyway.
  const brandId = useWatch({ control: form.control, name: 'brandId' })
  const models = useModels(
    { brandId, limit: OPTION_LIMIT },
    { enabled: open && pick && brandId !== '' },
  )
  const modelOptions = (models.data?.items ?? [])
    .filter((m) => m.brandId === brandId && m.lifecycleStatus !== 'ARCHIVED')
    .map((m) => ({ value: m.id, label: `${m.name} (${m.internalCode})` }))
  const pending = create.isPending || update.isPending

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    try {
      const saved = variant
        ? await update.mutateAsync({
            id: variant.id,
            input: {
              version: variant.version,
              name: v.name,
              variantCode: v.variantCode,
              manufacturerCode: orNull(v.manufacturerCode),
            },
          })
        : await create.mutateAsync({
            modelId: v.modelId,
            name: v.name,
            variantCode: v.variantCode,
            manufacturerCode: orNull(v.manufacturerCode),
          })
      onSaved?.(saved)
      onClose()
    } catch (e) {
      setFailure(
        describeError(e, {
          setError: form.setError,
          names: ['brandId', 'modelId', 'name', 'variantCode', 'manufacturerCode'],
        }),
      )
    }
  })

  const errors = form.formState.errors
  const brandField = form.register('brandId')
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={variant ? 'Edit variant' : 'Add variant'}
      description={
        variant
          ? 'Only pending variants can be edited. Editing a verified variant returns it to Unverified.'
          : fixedModel
            ? `New variant of ${fixedModel.name}.`
            : 'Choose the brand first: only that brand’s models are offered.'
      }
      footer={
        <Footer
          formId={formId}
          pending={pending}
          submitLabel={variant ? 'Save variant' : 'Add variant'}
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
        {!variant && <UnverifiedNote />}
        {variant?.confidence && variant.confidence !== 'UNVERIFIED' && (
          <p className="rounded-control bg-warning-subtle px-3 py-2 text-xs text-ink">
            This variant is verified. Saving changes clears its verification; it must be verified
            again.
          </p>
        )}
        {pick && (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label="Brand"
              required
              hint={brands.isPending ? 'Loading brands…' : undefined}
              error={errors.brandId?.message}
            >
              {(p) => (
                <Select
                  {...p}
                  options={brandOptions}
                  placeholder="Choose a brand"
                  {...brandField}
                  onChange={(e) => {
                    void brandField.onChange(e)
                    form.setValue('modelId', '')
                  }}
                />
              )}
            </Field>
            <Field
              label="Model"
              required
              hint={
                brandId === ''
                  ? 'Choose a brand first.'
                  : models.isPending
                    ? 'Loading this brand’s models…'
                    : modelOptions.length === 0
                      ? 'This brand has no models you can add variants to.'
                      : undefined
              }
              error={errors.modelId?.message}
            >
              {(p) => (
                <Select
                  {...p}
                  options={modelOptions}
                  placeholder="Choose a model"
                  disabled={brandId === '' || models.isPending}
                  {...form.register('modelId')}
                />
              )}
            </Field>
          </div>
        )}
        <Field label="Variant name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Variant code"
            required
            hint="Unique within the model; stored upper-case."
            error={errors.variantCode?.message}
          >
            {(p) => <Input {...p} autoComplete="off" {...form.register('variantCode')} />}
          </Field>
          <Field
            label="Manufacturer code"
            hint="Only if the supplier documents one."
            error={errors.manufacturerCode?.message}
          >
            {(p) => <Input {...p} autoComplete="off" {...form.register('manufacturerCode')} />}
          </Field>
        </div>
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

// ── Colour ──────────────────────────────────────────────────────────────────

function ColourFormDialogBody({
  open,
  onClose,
  colour,
  variant,
}: {
  open: boolean
  onClose: () => void
  colour?: VariantColour
  /** Parent variant (create). */
  variant: Variant
}) {
  const formId = useId()
  const create = useCreateColour()
  const update = useUpdateItem('colours')
  const refresh = useRefreshCatalogue()
  const [failure, setFailure] = useState<ActionError | null>(null)
  const form = useForm<ColourFormValues, unknown, ColourFormValues>({
    resolver: zodResolver(colourFormSchema),
    defaultValues: { name: colour?.name ?? '', colourCode: colour?.colourCode ?? '' },
  })
  const pending = create.isPending || update.isPending

  const onSubmit = form.handleSubmit(async (v) => {
    if (pending) return
    setFailure(null)
    try {
      if (colour) {
        await update.mutateAsync({
          id: colour.id,
          input: { version: colour.version, name: v.name, colourCode: orNull(v.colourCode) },
        })
      } else {
        await create.mutateAsync({
          variantId: variant.id,
          name: v.name,
          colourCode: orNull(v.colourCode),
        })
      }
      onClose()
    } catch (e) {
      setFailure(describeError(e, { setError: form.setError, names: ['name', 'colourCode'] }))
    }
  })

  const errors = form.formState.errors
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={colour ? 'Edit colour' : 'Add colour'}
      description={
        colour ? 'Only pending colours can be edited.' : `New colour for ${variant.name}.`
      }
      footer={
        <Footer
          formId={formId}
          pending={pending}
          submitLabel={colour ? 'Save colour' : 'Add colour'}
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
        {!colour && <UnverifiedNote />}
        <Field label="Colour name" required error={errors.name?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('name')} />}
        </Field>
        <Field
          label="Colour code"
          hint="Manufacturer colour code, if documented."
          error={errors.colourCode?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('colourCode')} />}
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

/** Mounted only while open, so every opening starts from fresh values. */
export function BrandFormDialog(props: Parameters<typeof BrandFormDialogBody>[0]) {
  return props.open ? <BrandFormDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function ModelFormDialog(props: Parameters<typeof ModelFormDialogBody>[0]) {
  return props.open ? <ModelFormDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function VariantFormDialog(props: Parameters<typeof VariantFormDialogBody>[0]) {
  return props.open ? <VariantFormDialogBody {...props} /> : null
}

/** Mounted only while open, so every opening starts from fresh values. */
export function ColourFormDialog(props: Parameters<typeof ColourFormDialogBody>[0]) {
  return props.open ? <ColourFormDialogBody {...props} /> : null
}
