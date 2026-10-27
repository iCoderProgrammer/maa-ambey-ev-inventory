import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, Check, Lock, Plus, Save, Trash2 } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import {
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type FieldErrors,
  type UseFormRegister,
} from 'react-hook-form'
import { z } from 'zod'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { ActionErrorBox } from '@/features/catalogue/components/ActionErrorBox'
import { formatDateTime, todayISO, type ActionError } from '@/features/catalogue/lib/presentation'
import { normaliseChassis } from '@/features/vehicles/lib/identifiers'
import { ApiError } from '@/lib/api/client'
import { cn } from '@/lib/utils/cn'
import {
  getGoodsReceipt,
  saveGoodsReceipt,
  type GRNDetail,
  type GRNLine,
} from '../api/goodsReceipts'
import { grnKeys, useGRNMutation } from '../hooks/useGoodsReceipts'
import {
  conditionLabels,
  describeGRNError,
  formPath,
  lineWarningLabels,
  unitWarningLabels,
} from '../lib/labels'

/**
 * The draft receipt stepper (Design Specification §4.5): delivered lines →
 * unit identifiers → supplier documents → review. "Save draft" sends the whole
 * draft and changes no stock; "Post receipt" is a separate step that does not
 * exist yet. Entries stay in the form when the server refuses a save.
 */

// Printable ASCII only, as the server requires (normalised form is the same everywhere).
const identifierRe = /^[\x20-\x7e]*$/
const identifier = z
  .string()
  .trim()
  .max(50, 'At most 50 characters')
  .refine((v) => identifierRe.test(v), {
    message: 'Letters, digits, spaces and punctuation only',
  })
const quantity = z
  .string()
  .trim()
  .refine((v) => v === '' || (/^\d+$/.test(v) && Number(v) <= 10000), {
    message: 'A whole number from 0 to 10000',
  })

const unitSchema = z.object({
  condition: z.enum(['ACCEPTED', 'REJECTED']),
  chassisNumber: identifier,
  motorSerial: identifier,
  batterySerial: identifier,
  chargerSerial: identifier,
  note: z.string().trim().max(300, 'At most 300 characters'),
})

const lineSchema = z.object({
  purchaseOrderLineId: z.string(),
  receivedQuantity: quantity,
  rejectedQuantity: quantity,
  discrepancyNote: z.string().trim().max(500, 'At most 500 characters'),
  overReceiptReason: z.string().trim().max(500, 'At most 500 characters'),
  units: z.array(unitSchema),
})

const num = (v: string) => (v.trim() === '' ? 0 : Number(v))

const formSchema = z
  .object({
    receivedDate: z
      .string()
      .refine((v) => /^\d{4}-\d{2}-\d{2}$/.test(v), { message: 'Enter the date the goods arrived' })
      .refine((v) => v <= todayISO(), { message: 'Cannot be in the future' }),
    supplierReference: z.string().trim().max(100, 'At most 100 characters'),
    notes: z.string().trim().max(1000, 'At most 1000 characters'),
    lines: z.array(lineSchema),
  })
  .superRefine((v, ctx) => {
    const seen = new Map<string, string>()
    v.lines.forEach((l, i) => {
      const received = num(l.receivedQuantity)
      const rejected = num(l.rejectedQuantity)
      if (rejected > received) {
        ctx.addIssue({
          code: 'custom',
          path: ['lines', i, 'rejectedQuantity'],
          message: 'Cannot be more than received',
        })
      }
      if (l.units.length > received) {
        ctx.addIssue({
          code: 'custom',
          path: ['lines', i, 'receivedQuantity'],
          message: `${String(l.units.length)} units are captured: receive at least that many, or remove units`,
        })
      }
      const rejectedUnits = l.units.filter((u) => u.condition === 'REJECTED').length
      if (rejectedUnits > rejected) {
        ctx.addIssue({
          code: 'custom',
          path: ['lines', i, 'rejectedQuantity'],
          message: `${String(rejectedUnits)} units are marked rejected`,
        })
      }
      if (l.units.length - rejectedUnits > received - rejected) {
        ctx.addIssue({
          code: 'custom',
          path: ['lines', i, 'receivedQuantity'],
          message: 'More units are marked accepted than the accepted quantity',
        })
      }
      l.units.forEach((u, j) => {
        const n = normaliseChassis(u.chassisNumber)
        if (!n) return
        const prev = seen.get(n)
        if (prev) {
          ctx.addIssue({
            code: 'custom',
            path: ['lines', i, 'units', j, 'chassisNumber'],
            message: `Same chassis/frame number as ${prev}`,
          })
        } else {
          seen.set(n, `line ${String(i + 1)}, unit ${String(j + 1)}`)
        }
      })
    })
  })
type FormValues = z.input<typeof formSchema>
type UnitValues = FormValues['lines'][number]['units'][number]

const blankUnit = (condition: UnitValues['condition'] = 'ACCEPTED'): UnitValues => ({
  condition,
  chassisNumber: '',
  motorSerial: '',
  batterySerial: '',
  chargerSerial: '',
  note: '',
})

function toValues(grn: GRNDetail): FormValues {
  const text = (v: string | null) => v ?? ''
  return {
    receivedDate: grn.receivedDate,
    supplierReference: text(grn.supplierReference),
    notes: text(grn.notes),
    lines: grn.lines.map((l) => ({
      purchaseOrderLineId: l.purchaseOrderLineId,
      receivedQuantity: l.receivedQuantity ? String(l.receivedQuantity) : '',
      rejectedQuantity: l.rejectedQuantity ? String(l.rejectedQuantity) : '',
      discrepancyNote: text(l.discrepancyNote),
      // An earlier authorisation is kept by the server; a new reason re-authorises.
      overReceiptReason: '',
      units: l.units.map((u) => ({
        condition: u.condition,
        chassisNumber: text(u.chassisNumber),
        motorSerial: text(u.motorSerial),
        batterySerial: text(u.batterySerial),
        chargerSerial: text(u.chargerSerial),
        note: text(u.note),
      })),
    })),
  }
}

const orNull = (v: string) => (v.trim() === '' ? null : v.trim())

/** Every PO line is sent (the server drops empty ones), so error paths match form rows. */
function toSave(version: number, v: FormValues) {
  return {
    version,
    receivedDate: v.receivedDate,
    supplierReference: orNull(v.supplierReference),
    notes: orNull(v.notes),
    lines: v.lines.map((l) => {
      const received = num(l.receivedQuantity)
      const rejected = num(l.rejectedQuantity)
      return {
        purchaseOrderLineId: l.purchaseOrderLineId,
        receivedQuantity: received,
        acceptedQuantity: Math.max(received - rejected, 0),
        rejectedQuantity: rejected,
        discrepancyNote: orNull(l.discrepancyNote),
        overReceiptReason: orNull(l.overReceiptReason),
        units: l.units.map((u) => ({
          condition: u.condition,
          chassisNumber: orNull(u.chassisNumber),
          motorSerial: orNull(u.motorSerial),
          batterySerial: orNull(u.batterySerial),
          chargerSerial: orNull(u.chargerSerial),
          note: orNull(u.note),
        })),
      }
    }),
  }
}

const fieldPathRe =
  /^(receivedDate|supplierReference|notes)$|^lines\.\d+\.(receivedQuantity|rejectedQuantity|discrepancyNote|overReceiptReason)$|^lines\.\d+\.units\.\d+\.(condition|chassisNumber|motorSerial|batterySerial|chargerSerial|note)$/

const steps = [
  { key: 'lines', label: 'Delivered lines' },
  { key: 'units', label: 'Unit identifiers' },
  { key: 'documents', label: 'Supplier documents' },
  { key: 'review', label: 'Review and save' },
] as const
type StepKey = (typeof steps)[number]['key']

/** Over-receipt as the screen can tell from the last loaded availability (the server decides). */
function excessOf(line: GRNLine, received: number) {
  return Math.max(received - line.availableQuantity, 0)
}
function overrideCovers(line: GRNLine, received: number) {
  return line.overReceipt !== null && line.overReceipt.quantity >= received
}

export function ReceiptEditor({ grn, documents }: { grn: GRNDetail; documents: ReactNode }) {
  const [step, setStep] = useState<StepKey>('lines')
  const [failure, setFailure] = useState<ActionError | null>(null)
  const [savedAt, setSavedAt] = useState<string | null>(null)
  // The version the form's values came from: a save never overwrites changes it has not seen.
  const [base, setBase] = useState(grn.version)
  const queryClient = useQueryClient()
  const form = useForm<FormValues, unknown, FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: toValues(grn),
  })
  const save = useGRNMutation((v: FormValues) => saveGoodsReceipt(grn.id, toSave(base, v)))
  const dirty = form.formState.isDirty
  // Someone else saved meanwhile: saving would be refused (stale version); offer a reload.
  const ownSave = save.data?.version === grn.version
  const changedElsewhere = grn.version !== base && !ownSave
  const lineNos = grn.lines.map((l) => l.lineNo)
  const errors = form.formState.errors

  /** A server field path this form can show: a known field on a line and unit that exist here. */
  const isField = (path: string) => {
    if (!fieldPathRe.test(path)) return false
    if (!path.startsWith('lines.')) return true
    const m = /^lines\.(0|[1-9]\d*)(?:\.units\.(0|[1-9]\d*))?\./.exec(path)
    if (!m) return false
    const l = form.getValues('lines')[Number(m[1])]
    return l !== undefined && (m[2] === undefined || Number(m[2]) < l.units.length)
  }
  const goToErrors = (paths: string[]) => {
    if (paths.some((p) => /\.units(\.|$)/.test(p))) setStep('units')
    else if (paths.length > 0) setStep('lines')
  }
  const onSubmit = form.handleSubmit(
    async (v) => {
      if (save.isPending) return
      setFailure(null)
      try {
        const saved = await save.mutateAsync(v)
        form.reset(toValues(saved))
        setBase(saved.version)
        setSavedAt(saved.updatedAt)
      } catch (e) {
        // Nothing is retried: a refusal waits for the user to correct and save again.
        const f = describeGRNError(e, { setError: form.setError, isField }, lineNos)
        setFailure(f)
        if (e instanceof ApiError)
          goToErrors(e.details.map((d) => formPath(d.field)).filter(isField))
      }
    },
    (errs) => {
      setFailure({ message: 'Check the highlighted entries. Nothing was saved.' })
      goToErrors(
        Object.keys(errs.lines ?? {}).flatMap((i) => {
          const e = errs.lines?.[Number(i)]
          return e?.units ? [`lines.${i}.units.`] : [`lines.${i}`]
        }),
      )
    },
  )

  const move = (by: number) => {
    const next = steps[steps.findIndex((s) => s.key === step) + by]
    if (next) setStep(next.key)
  }
  /** Discards this screen's unsaved entries and loads the latest saved draft. */
  const reload = () => {
    getGoodsReceipt(grn.id)
      .then((d) => {
        queryClient.setQueryData(grnKeys.detail(d.id), d)
        form.reset(toValues(d))
        setBase(d.version)
        setFailure(null)
      })
      .catch((e: unknown) => {
        setFailure(describeGRNError(e))
      })
  }

  return (
    <form noValidate onSubmit={(e) => void onSubmit(e)} aria-label={`Draft ${grn.number}`}>
      <nav aria-label="Receipt steps" className="mb-4">
        <ol className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {steps.map((s, i) => (
            <li key={s.key}>
              <button
                type="button"
                onClick={() => {
                  setStep(s.key)
                }}
                aria-current={step === s.key ? 'step' : undefined}
                className={cn(
                  'flex w-full items-center gap-2 rounded-control border px-3 py-2 text-left text-sm',
                  step === s.key
                    ? 'border-primary bg-primary/5 font-medium text-ink'
                    : 'border-border text-muted hover:text-ink',
                )}
              >
                <span
                  className={cn(
                    'grid size-5 shrink-0 place-items-center rounded-full text-xs',
                    step === s.key ? 'bg-primary text-primary-fg' : 'bg-surface-muted',
                  )}
                  aria-hidden="true"
                >
                  {i + 1}
                </span>
                {s.label}
              </button>
            </li>
          ))}
        </ol>
      </nav>

      {changedElsewhere && (
        <div
          role="alert"
          className="mb-4 flex flex-col gap-2 rounded-card border border-warning/30 bg-warning-subtle px-4 py-3 text-sm text-ink sm:flex-row sm:items-center sm:justify-between"
        >
          <p>
            Someone else saved this draft after you started editing. Saving now would be refused;
            reload to see their version (your unsaved entries here will be replaced).
          </p>
          <Button type="button" size="sm" variant="secondary" onClick={reload}>
            Reload latest version
          </Button>
        </div>
      )}
      {step === 'lines' && (
        <LinesStep grn={grn} control={form.control} register={form.register} errors={errors} />
      )}
      {step === 'units' && (
        <UnitsStep grn={grn} control={form.control} register={form.register} errors={errors} />
      )}
      {step === 'documents' && documents}
      {step === 'review' && <ReviewStep grn={grn} control={form.control} />}

      <div className="sticky bottom-0 z-10 mt-6 flex flex-col gap-3 rounded-card border border-border/80 bg-surface p-4 shadow-xs">
        {failure && <ActionErrorBox error={failure} onReload={reload} />}
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-muted" aria-live="polite">
            {dirty
              ? 'Unsaved changes. Saving a draft does not change stock.'
              : savedAt
                ? `Draft saved ${formatDateTime(savedAt)}. Stock is unchanged.`
                : 'Saving a draft does not change stock.'}
          </p>
          <div className="flex flex-wrap gap-2">
            {step !== 'lines' && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  move(-1)
                }}
              >
                Back
              </Button>
            )}
            {step !== 'review' && (
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  move(1)
                }}
              >
                Next
              </Button>
            )}
            <Button type="submit" loading={save.isPending}>
              <Save className="size-4" aria-hidden="true" />
              Save draft
            </Button>
          </div>
        </div>
      </div>
    </form>
  )
}

interface StepProps {
  grn: GRNDetail
  control: Control<FormValues, unknown, FormValues>
  register: UseFormRegister<FormValues>
  errors: FieldErrors<FormValues>
}

// ── Step 1: delivered lines ─────────────────────────────────────────────────

function LinesStep({ grn, control, register, errors }: StepProps) {
  return (
    <div className="flex flex-col gap-4">
      <Card title="Delivery" description="As printed on the supplier's paperwork.">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Date received" required error={errors.receivedDate?.message}>
            {(p) => <Input {...p} type="date" max={todayISO()} {...register('receivedDate')} />}
          </Field>
          <Field
            label="Supplier delivery reference"
            hint="Delivery challan or invoice number."
            error={errors.supplierReference?.message}
          >
            {(p) => <Input {...p} autoComplete="off" {...register('supplierReference')} />}
          </Field>
          <div className="sm:col-span-2">
            <Field label="Notes on the delivery" error={errors.notes?.message}>
              {(p) => <Input {...p} autoComplete="off" {...register('notes')} />}
            </Field>
          </div>
        </div>
      </Card>
      <Card
        title="Delivered lines"
        description="Enter what arrived for each line of the purchase order. Leave a line empty if nothing of it came. Partial deliveries are fine."
      >
        <ul className="flex flex-col divide-y divide-border">
          {grn.lines.map((line, i) => (
            <LineRow
              key={line.purchaseOrderLineId}
              index={i}
              line={line}
              grn={grn}
              control={control}
              register={register}
              errors={errors}
            />
          ))}
        </ul>
      </Card>
    </div>
  )
}

function Quantities({ line }: { line: GRNLine }) {
  return (
    <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-4">
      {(
        [
          ['Ordered', line.orderedQuantity],
          ['Posted earlier', line.postedQuantity],
          ['On other drafts', line.otherDraftsQuantity],
          ['Left to receive', line.availableQuantity],
        ] as const
      ).map(([k, v]) => (
        <div key={k} className="flex justify-between gap-2 sm:flex-col sm:justify-start">
          <dt className="text-muted">{k}</dt>
          <dd className="font-medium tabular-nums text-ink">{v}</dd>
        </div>
      ))}
    </dl>
  )
}

function LineRow({
  index,
  line,
  grn,
  control,
  register,
  errors,
}: StepProps & { index: number; line: GRNLine }) {
  const values = useWatch({ control, name: `lines.${index}` })
  const received = num(values.receivedQuantity)
  const rejected = num(values.rejectedQuantity)
  const excess = excessOf(line, received)
  const covered = overrideCovers(line, received)
  const e = errors.lines?.[index]
  return (
    <li className="flex flex-col gap-3 py-4 first:pt-0 last:pb-0">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <p className="font-medium text-ink">
          {line.lineNo}. {line.brandName} · {line.modelName} · {line.variantName}
        </p>
        <span className="flex flex-wrap gap-1">
          {line.warnings.map((w) => (
            <Badge key={w} tone={w === 'OVER_RECEIPT_NOT_AUTHORISED' ? 'danger' : 'warning'}>
              {lineWarningLabels[w]}
            </Badge>
          ))}
        </span>
      </div>
      <Quantities line={line} />
      <div className="grid gap-3 sm:grid-cols-3">
        <Field
          label={`Received (line ${String(line.lineNo)})`}
          error={e?.receivedQuantity?.message}
        >
          {(p) => (
            <Input
              {...p}
              inputMode="numeric"
              autoComplete="off"
              placeholder="0"
              {...register(`lines.${index}.receivedQuantity`)}
            />
          )}
        </Field>
        <Field
          label={`Rejected (line ${String(line.lineNo)})`}
          error={e?.rejectedQuantity?.message}
        >
          {(p) => (
            <Input
              {...p}
              inputMode="numeric"
              autoComplete="off"
              placeholder="0"
              {...register(`lines.${index}.rejectedQuantity`)}
            />
          )}
        </Field>
        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-ink">Accepted</span>
          <span className="flex h-9 items-center text-sm tabular-nums text-ink" aria-live="polite">
            {Math.max(received - rejected, 0)}
          </span>
        </div>
      </div>
      {excess > 0 && (
        <div
          role="alert"
          className={cn(
            'flex flex-col gap-2 rounded-control border px-3 py-2 text-sm',
            covered ? 'border-warning/30 bg-warning-subtle' : 'border-danger/30 bg-danger-subtle',
          )}
        >
          <p className="flex items-start gap-2 text-ink">
            <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
            <span>
              <strong>Over-receipt:</strong> {received} received but only {line.availableQuantity}{' '}
              left on this line ({excess} extra).{' '}
              {covered && line.overReceipt
                ? `Authorised by ${line.overReceipt.authorisedByName} on ${formatDateTime(line.overReceipt.authorisedAt)} for up to ${String(line.overReceipt.quantity)}: ${line.overReceipt.reason}`
                : grn.actions.authoriseOverReceipt
                  ? 'Enter the reason to authorise the extra units.'
                  : 'Saving will be refused until someone allowed to authorise over-receipt approves it with a reason.'}
            </span>
          </p>
          {grn.actions.authoriseOverReceipt && (
            <Field
              label={covered ? 'New over-receipt reason (re-authorises)' : 'Over-receipt reason'}
              required={!covered}
              error={e?.overReceiptReason?.message}
            >
              {(p) => (
                <Input
                  {...p}
                  autoComplete="off"
                  {...register(`lines.${index}.overReceiptReason`)}
                />
              )}
            </Field>
          )}
        </div>
      )}
      <Field
        label={`Discrepancy note (line ${String(line.lineNo)})`}
        hint="Damage, shortages, wrong items or anything else that differs from the order."
        error={e?.discrepancyNote?.message}
      >
        {(p) => <Input {...p} autoComplete="off" {...register(`lines.${index}.discrepancyNote`)} />}
      </Field>
    </li>
  )
}

// ── Step 2: unit identifiers ────────────────────────────────────────────────

function UnitsStep({ grn, control, register, errors }: StepProps) {
  const lines = useWatch({ control, name: 'lines' })
  const active = grn.lines
    .map((line, i) => ({ line, i }))
    .filter(({ i }) => {
      const l = lines[i]
      return l !== undefined && (num(l.receivedQuantity) > 0 || l.units.length > 0)
    })
  return (
    <Card
      title="Unit identifiers"
      description="Capture each scooter as it is unloaded: chassis/frame number and, where the paperwork or label shows them, motor, battery and charger serials. Identifier rules per brand are not configured yet, so every identifier is optional in a draft."
    >
      {active.length === 0 ? (
        <p className="text-sm text-muted">Enter received quantities in step 1 first.</p>
      ) : (
        <div className="flex flex-col gap-6">
          {active.map(({ line, i }) => (
            <LineUnits
              key={line.purchaseOrderLineId}
              index={i}
              line={line}
              grn={grn}
              control={control}
              register={register}
              errors={errors}
            />
          ))}
        </div>
      )}
    </Card>
  )
}

const identifierFields = [
  ['chassisNumber', 'Chassis/frame number'],
  ['motorSerial', 'Motor serial'],
  ['batterySerial', 'Battery serial'],
  ['chargerSerial', 'Charger serial'],
] as const

function LineUnits({
  index,
  line,
  control,
  register,
  errors,
}: StepProps & { index: number; line: GRNLine }) {
  const { fields, append, remove } = useFieldArray({ control, name: `lines.${index}.units` })
  const values = useWatch({ control, name: `lines.${index}` })
  const received = num(values.receivedQuantity)
  const rejected = num(values.rejectedQuantity)
  const [focusLast, setFocusLast] = useState(false)
  const missing = Math.max(received - fields.length, 0)
  const saved = new Map(line.units.map((u) => [u.unitNo, u]))
  const e = errors.lines?.[index]
  const addOne = () => {
    const rejectedUnits = values.units.filter((u) => u.condition === 'REJECTED').length
    append(
      blankUnit(
        rejectedUnits < rejected && fields.length >= received - rejected ? 'REJECTED' : 'ACCEPTED',
      ),
      {
        shouldFocus: false,
      },
    )
    setFocusLast(true)
  }
  return (
    <section aria-labelledby={`units-${line.purchaseOrderLineId}`} className="flex flex-col gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h3 id={`units-${line.purchaseOrderLineId}`} className="font-medium text-ink">
          {line.lineNo}. {line.brandName} · {line.modelName} · {line.variantName}
          <span className="ml-2 text-xs font-normal text-muted">
            {fields.length} of {received} units captured
          </span>
        </h3>
        <Button
          type="button"
          size="sm"
          variant="secondary"
          onClick={addOne}
          disabled={fields.length >= received}
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add unit
        </Button>
      </div>
      {missing > 0 && (
        <p className="text-xs text-muted">
          {missing} more {missing === 1 ? 'unit' : 'units'} to capture before posting.
        </p>
      )}
      {e?.units?.message && (
        <p role="alert" className="text-xs font-medium text-danger">
          {e.units.message}
        </p>
      )}
      <ol className="flex flex-col gap-3">
        {fields.map((f, j) => {
          const ue = e?.units?.[j]
          const prev = saved.get(j + 1)
          return (
            <li key={f.id} className="rounded-control border border-border p-3">
              <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm font-medium text-ink">Unit {j + 1}</span>
                <span className="flex flex-wrap items-center gap-1">
                  {prev?.warnings.map((w) => (
                    <Badge key={w} tone="warning">
                      {unitWarningLabels[w]}
                    </Badge>
                  ))}
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      remove(j)
                    }}
                    aria-label={`Remove unit ${String(j + 1)} of line ${String(line.lineNo)}`}
                  >
                    <Trash2 className="size-3.5" aria-hidden="true" />
                    Remove
                  </Button>
                </span>
              </div>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <Field
                  label={`Condition (line ${String(line.lineNo)}, unit ${String(j + 1)})`}
                  error={ue?.condition?.message}
                >
                  {(p) => (
                    <Select
                      {...p}
                      options={(['ACCEPTED', 'REJECTED'] as const).map((c) => ({
                        value: c,
                        label: conditionLabels[c],
                      }))}
                      {...register(`lines.${index}.units.${j}.condition`)}
                    />
                  )}
                </Field>
                {identifierFields.map(([name, label]) => {
                  const r = register(`lines.${index}.units.${j}.${name}`)
                  return (
                    <Field
                      key={name}
                      label={`${label} (line ${String(line.lineNo)}, unit ${String(j + 1)})`}
                      error={ue?.[name]?.message}
                    >
                      {(p) => (
                        <Input
                          {...p}
                          autoComplete="off"
                          spellCheck={false}
                          autoCapitalize="characters"
                          {...r}
                          ref={(el) => {
                            r.ref(el)
                            if (
                              el &&
                              focusLast &&
                              name === 'chassisNumber' &&
                              j === fields.length - 1
                            ) {
                              el.focus()
                              setFocusLast(false)
                            }
                          }}
                        />
                      )}
                    </Field>
                  )
                })}
                <Field
                  label={`Note (line ${String(line.lineNo)}, unit ${String(j + 1)})`}
                  error={ue?.note?.message}
                >
                  {(p) => (
                    <Input
                      {...p}
                      autoComplete="off"
                      {...register(`lines.${index}.units.${j}.note`)}
                    />
                  )}
                </Field>
              </div>
            </li>
          )
        })}
      </ol>
    </section>
  )
}

// ── Step 4: review ──────────────────────────────────────────────────────────

function ReviewStep({ grn, control }: { grn: GRNDetail; control: StepProps['control'] }) {
  const lines = useWatch({ control, name: 'lines' })
  const rows = grn.lines
    .map((line, i) => {
      const l = lines[i]
      return {
        line,
        received: num(l?.receivedQuantity ?? ''),
        rejected: num(l?.rejectedQuantity ?? ''),
        units: l?.units ?? [],
        reason: l?.overReceiptReason.trim() ?? '',
      }
    })
    .filter((r) => r.received > 0)
  const total = rows.reduce((s, r) => s + r.received, 0)
  const rejected = rows.reduce((s, r) => s + r.rejected, 0)
  const captured = rows.reduce((s, r) => s + r.units.length, 0)
  const withChassis = rows.reduce(
    (s, r) => s + r.units.filter((u) => u.chassisNumber.trim()).length,
    0,
  )
  const unauthorised = rows.filter(
    (r) => excessOf(r.line, r.received) > 0 && !overrideCovers(r.line, r.received) && !r.reason,
  )
  return (
    <div className="flex flex-col gap-4">
      <Card
        title="Review"
        description="What this draft will record. The server checks everything again when you save."
      >
        <dl className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          {(
            [
              ['Received', total],
              ['Accepted', total - rejected],
              ['Rejected', rejected],
              ['Units with chassis/frame number', `${String(withChassis)} of ${String(total)}`],
            ] as const
          ).map(([k, v]) => (
            <div key={k} className="flex flex-col">
              <dt className="text-xs text-muted">{k}</dt>
              <dd className="font-medium tabular-nums text-ink">{v}</dd>
            </div>
          ))}
        </dl>
        <ul className="mt-4 flex flex-col gap-1 text-sm">
          {rows.length === 0 && <li className="text-muted">Nothing entered yet.</li>}
          {rows.map((r) => (
            <li key={r.line.purchaseOrderLineId} className="flex flex-wrap items-center gap-2">
              <span>
                Line {r.line.lineNo} · {r.line.variantName}: {r.received} received (
                {r.received - r.rejected} accepted, {r.rejected} rejected) · {r.units.length}{' '}
                captured
              </span>
              {excessOf(r.line, r.received) > 0 && <Badge tone="warning">Over-receipt</Badge>}
              {r.received < r.line.availableQuantity && <Badge tone="info">Partial delivery</Badge>}
            </li>
          ))}
        </ul>
        {captured < total && (
          <p className="mt-3 text-xs text-muted">
            {total - captured} units still have no identifiers. That is allowed in a draft; posting
            will need them.
          </p>
        )}
        {unauthorised.length > 0 && (
          <p role="alert" className="mt-3 text-sm text-danger">
            Lines {unauthorised.map((r) => r.line.lineNo).join(', ')} receive more than the purchase
            order has left without an authorisation. Saving will be refused.
          </p>
        )}
      </Card>
      <Card title="Post receipt" description="Posting is a separate step from saving a draft.">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="flex items-start gap-2 text-sm text-ink">
            <Lock className="mt-0.5 size-4 shrink-0 text-muted" aria-hidden="true" />
            Posting creates the vehicle units and stock movements and updates the purchase order in
            one step. It is not available yet; this draft does not change stock.
          </p>
          <Button type="button" variant="secondary" disabled aria-describedby="post-unavailable">
            <Check className="size-4" aria-hidden="true" />
            Post receipt
          </Button>
        </div>
        <p id="post-unavailable" className="sr-only">
          Posting receipts is not available yet.
        </p>
      </Card>
    </div>
  )
}
