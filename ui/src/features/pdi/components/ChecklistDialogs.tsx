import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { useBrands } from '@/features/catalogue/hooks/useCatalogue'
import type { ChecklistItem } from '../api/pdi'
import { useCreateChecklistItem, useRetireChecklistItem } from '../hooks/usePdi'
import { describePdiError, type PdiError } from '../lib/pdiErrors'

function ErrorBox({ error }: { error: PdiError }) {
  return (
    <div
      role="alert"
      className="rounded-control border border-danger/30 bg-danger-subtle px-3 py-2 text-sm text-danger"
    >
      {error.message}
      {error.requestId && (
        <span className="ml-1 font-mono text-xs text-muted">Reference: {error.requestId}</span>
      )}
    </div>
  )
}

const createSchema = z.object({
  code: z
    .string()
    .trim()
    .toUpperCase()
    .regex(/^[A-Z][A-Z0-9_]{1,39}$/, '2–40 characters: a letter, then letters, digits or _.'),
  label: z.string().trim().min(1, 'Enter a label.').max(200, 'At most 200 characters.'),
  description: z.string().trim().max(1000, 'At most 1000 characters.'),
  brandId: z.string(),
  sortOrder: z
    .string()
    .trim()
    .regex(/^\d{0,4}$/, 'A whole number from 0 to 9999.'),
})
type CreateValues = z.infer<typeof createSchema>

/**
 * Adds a checklist item. What the client inspects is the client's to define:
 * nothing is pre-filled, and items are never edited (retire and create).
 */
export function CreateChecklistItemDialog({
  canListBrands,
  onClose,
}: {
  canListBrands: boolean
  onClose: () => void
}) {
  const create = useCreateChecklistItem()
  const brands = useBrands({ limit: 100 }, { enabled: canListBrands })
  const [error, setError] = useState<PdiError | null>(null)
  const form = useForm<z.input<typeof createSchema>, unknown, CreateValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { code: '', label: '', description: '', brandId: '', sortOrder: '' },
  })
  const e = form.formState.errors
  const submit = form.handleSubmit((v) => {
    setError(null)
    create.mutate(
      {
        code: v.code,
        label: v.label,
        ...(v.description ? { description: v.description } : {}),
        ...(v.brandId ? { brandId: v.brandId } : {}),
        ...(v.sortOrder ? { sortOrder: Number(v.sortOrder) } : {}),
      },
      {
        onSuccess: onClose,
        onError: (err) => {
          const d = describePdiError(err, false)
          for (const f of d.fields)
            if (['code', 'label', 'description', 'brandId', 'sortOrder'].includes(f.field))
              form.setError(f.field as keyof CreateValues, { message: f.message })
          setError(d)
        },
      },
    )
  })
  return (
    <Dialog
      open
      onClose={onClose}
      title="Add checklist item"
      description="The item applies to new inspections of every brand, or of one brand."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={create.isPending}>
            Cancel
          </Button>
          <Button type="submit" form="pdi-checklist-create" disabled={create.isPending}>
            {create.isPending ? 'Adding…' : 'Add item'}
          </Button>
        </>
      }
    >
      <form
        id="pdi-checklist-create"
        noValidate
        onSubmit={(ev) => void submit(ev)}
        className="flex flex-col gap-4"
      >
        <Field
          label="Code"
          required
          error={e.code?.message}
          hint="Stable key, e.g. BRAKES. Upper-cased."
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('code')} />}
        </Field>
        <Field label="Label" required error={e.label?.message}>
          {(p) => <Input {...p} {...form.register('label')} />}
        </Field>
        <Field
          label="Description"
          error={e.description?.message}
          hint="Optional guidance for inspectors."
        >
          {(p) => <Input {...p} {...form.register('description')} />}
        </Field>
        <Field
          label="Applies to"
          error={e.brandId?.message}
          hint={canListBrands ? undefined : 'Brand-specific items need catalogue view permission.'}
        >
          {(p) => (
            <Select
              {...p}
              options={[
                { value: '', label: 'All brands' },
                ...(brands.data?.items ?? []).map((b) => ({
                  value: b.id,
                  label: `${b.name} only`,
                })),
              ]}
              {...form.register('brandId')}
            />
          )}
        </Field>
        <Field
          label="Order"
          error={e.sortOrder?.message}
          hint="0–9999; lower comes first. Default 0."
        >
          {(p) => <Input {...p} inputMode="numeric" {...form.register('sortOrder')} />}
        </Field>
        {error && <ErrorBox error={error} />}
      </form>
    </Dialog>
  )
}

const retireSchema = z.object({
  reason: z.string().trim().min(1, 'Give a reason.').max(500, 'At most 500 characters.'),
})

/** Retires an item: it no longer applies to new inspections; recorded outcomes keep it. */
export function RetireChecklistItemDialog({
  item,
  onClose,
}: {
  item: ChecklistItem
  onClose: () => void
}) {
  const retire = useRetireChecklistItem()
  const [error, setError] = useState<PdiError | null>(null)
  const form = useForm<z.input<typeof retireSchema>, unknown, z.output<typeof retireSchema>>({
    resolver: zodResolver(retireSchema),
    defaultValues: { reason: '' },
  })
  const submit = form.handleSubmit((v) => {
    setError(null)
    retire.mutate(
      { id: item.id, version: item.version, reason: v.reason },
      {
        onSuccess: onClose,
        onError: (err) => {
          setError(describePdiError(err, false))
        },
      },
    )
  })
  return (
    <Dialog
      open
      onClose={onClose}
      title={`Retire ${item.code}?`}
      description="Retired items no longer apply to new inspections. Recorded outcomes keep them. This cannot be undone."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={retire.isPending}>
            Cancel
          </Button>
          <Button
            variant="danger"
            type="submit"
            form="pdi-checklist-retire"
            disabled={retire.isPending}
          >
            {retire.isPending ? 'Retiring…' : 'Retire item'}
          </Button>
        </>
      }
    >
      <form
        id="pdi-checklist-retire"
        noValidate
        onSubmit={(ev) => void submit(ev)}
        className="flex flex-col gap-4"
      >
        <Field label="Reason" required error={form.formState.errors.reason?.message}>
          {(p) => <Input {...p} {...form.register('reason')} />}
        </Field>
        {error && <ErrorBox error={error} />}
      </form>
    </Dialog>
  )
}
