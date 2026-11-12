import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, CheckCircle2, Paperclip, X, XCircle } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useFieldArray, useForm, useWatch, type UseFormReturn } from 'react-hook-form'
import { z } from 'zod'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { controlClass } from '@/components/ui/controlClass'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import type { Attachment } from '@/features/attachments/api/attachments'
import { AttachmentUploader } from '@/features/attachments/components/AttachmentUploader'
import { useAttachmentLimits } from '@/features/attachments/hooks/useAttachments'
import { cn } from '@/lib/utils/cn'
import {
  newInspectionAction,
  sameInspection,
  type ChecklistItem,
  type Disposition,
  type InspectionAction,
  type InspectionBody,
  type PdiOverview,
  type RecordResult,
} from '../api/pdi'
import { useRecordInspection } from '../hooks/usePdi'
import { confirmVerb, dispositionEffect, dispositionLabels, dispositionTone } from '../lib/labels'
import { describePdiError, fieldLabel, type PdiError } from '../lib/pdiErrors'

const MAX_NOTES = 2000
const MAX_NOTE = 500
const MAX_EVIDENCE = 20

const textareaClass = cn(controlClass, 'h-auto min-h-20 py-2 leading-relaxed')

/**
 * The form mirrors the server contract (API_CONVENTIONS D-06) so mistakes are
 * shown early; the server checks everything again and its verdict wins.
 */
function formSchema(allowed: Disposition[]) {
  return z
    .object({
      disposition: z.string(),
      items: z.array(
        z.object({
          checklistItemId: z.string(),
          outcome: z.enum(['', 'PASS', 'FAIL']),
          note: z.string().max(MAX_NOTE, `At most ${String(MAX_NOTE)} characters.`),
        }),
      ),
      notes: z.string().max(MAX_NOTES, `At most ${String(MAX_NOTES)} characters.`),
      attachmentIds: z
        .array(z.string())
        .max(MAX_EVIDENCE, `At most ${String(MAX_EVIDENCE)} files.`),
    })
    .superRefine((v, ctx) => {
      v.items.forEach((it, i) => {
        if (!it.outcome)
          ctx.addIssue({
            code: 'custom',
            path: ['items', i, 'outcome'],
            message: 'Choose Pass or Fail.',
          })
        if (it.outcome === 'FAIL' && !it.note.trim())
          ctx.addIssue({
            code: 'custom',
            path: ['items', i, 'note'],
            message: 'Describe what failed.',
          })
      })
      const d = v.disposition as Disposition
      if (!allowed.includes(d)) {
        ctx.addIssue({ code: 'custom', path: ['disposition'], message: 'Choose a disposition.' })
        return
      }
      if (d === 'PASS' && v.items.some((it) => it.outcome === 'FAIL'))
        ctx.addIssue({
          code: 'custom',
          path: ['disposition'],
          message: 'A vehicle with a failed item cannot pass. Choose Hold, or correct the outcome.',
        })
      if (d !== 'PASS') {
        if (!v.notes.trim())
          ctx.addIssue({
            code: 'custom',
            path: ['notes'],
            message: 'Describe the issue: notes are required for a hold or rejection.',
          })
        if (v.attachmentIds.length === 0)
          ctx.addIssue({
            code: 'custom',
            path: ['attachmentIds'],
            message: 'Add at least one photo or document as evidence for a hold or rejection.',
          })
      }
    })
}
type FormValues = z.infer<ReturnType<typeof formSchema>>

function toBody(v: FormValues, version: number): InspectionBody {
  return {
    version,
    disposition: v.disposition as Disposition,
    items: v.items.map((it) => ({
      checklistItemId: it.checklistItemId,
      outcome: it.outcome as 'PASS' | 'FAIL',
      ...(it.note.trim() ? { note: it.note.trim() } : {}),
    })),
    ...(v.notes.trim() ? { notes: v.notes.trim() } : {}),
    attachmentIds: v.attachmentIds,
  }
}

export interface InspectionFormProps {
  overview: PdiOverview
  canUpload: boolean
  onRecorded: (r: RecordResult) => void
}

/** Checklist, defect notes, private evidence, disposition and confirmation. */
export function InspectionForm({ overview, canUpload, onRecorded }: InspectionFormProps) {
  const allowed = overview.allowedDispositions
  const checklist = overview.checklist
  const schema = useMemo(() => formSchema(allowed), [allowed])
  const form = useForm<FormValues, unknown, FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      disposition: allowed.length === 1 ? allowed[0] : '',
      items: checklist.map((c) => ({ checklistItemId: c.id, outcome: '', note: '' })),
      notes: '',
      attachmentIds: [],
    },
  })
  const [evidence, setEvidence] = useState<Attachment[]>([])
  const [review, setReview] = useState<InspectionBody | null>(null)

  const setEvidenceIds = (list: Attachment[]) => {
    setEvidence(list)
    form.setValue(
      'attachmentIds',
      list.map((a) => a.id),
      { shouldValidate: form.formState.isSubmitted },
    )
  }

  return (
    <form
      noValidate
      aria-label="PDI inspection"
      onSubmit={(e) =>
        void form.handleSubmit((v) => {
          setReview(toBody(v, overview.unit.version))
        })(e)
      }
      className="flex flex-col gap-5"
    >
      <ChecklistSection form={form} checklist={checklist} allowed={allowed} />
      <NotesSection form={form} />
      <EvidenceSection
        form={form}
        evidence={evidence}
        canUpload={canUpload}
        locationId={overview.unit.locationId}
        onAdd={(a) => {
          if (a.locationId !== overview.unit.locationId) return
          setEvidenceIds([...evidence, a])
        }}
        onRemove={(id) => {
          setEvidenceIds(evidence.filter((a) => a.id !== id))
        }}
      />
      <DispositionSection form={form} allowed={allowed} checklistEmpty={checklist.length === 0} />
      <div className="flex flex-wrap items-center justify-end gap-3 border-t border-border pt-4">
        <p className="mr-auto text-xs text-muted">
          You will review the inspection before it is recorded.
        </p>
        <Button type="submit">Review inspection</Button>
      </div>
      {review && (
        <ConfirmDialog
          unitId={overview.unit.id}
          unitNumber={overview.unit.unitNumber}
          body={review}
          checklist={checklist}
          evidence={evidence}
          onClose={() => {
            setReview(null)
          }}
          onServerFields={(fields) => {
            for (const f of fields) {
              const m = /^items\[(\d+)\]\.(note|outcome)$/.exec(f.field)
              if (m)
                form.setError(`items.${Number(m[1])}.${m[2] as 'note' | 'outcome'}`, {
                  message: f.message,
                })
              else if (f.field === 'notes' || f.field === 'disposition')
                form.setError(f.field, { message: f.message })
              else if (f.field.startsWith('attachmentIds'))
                form.setError('attachmentIds', { message: f.message })
            }
          }}
          onRecorded={(r) => {
            setReview(null)
            onRecorded(r)
          }}
        />
      )}
    </form>
  )
}

// ── Sections ────────────────────────────────────────────────────────────────

function ChecklistSection({
  form,
  checklist,
  allowed,
}: {
  form: UseFormReturn<FormValues, unknown, FormValues>
  checklist: ChecklistItem[]
  allowed: Disposition[]
}) {
  const { fields } = useFieldArray({ control: form.control, name: 'items' })
  const values = useWatch({ control: form.control, name: 'items' })
  const failed = values.filter((v) => v.outcome === 'FAIL').length
  const answered = values.filter((v) => v.outcome).length
  if (checklist.length === 0)
    return (
      <Card title="Checklist" description="No checklist item applies to this vehicle.">
        <p role="note" className="flex gap-2 text-sm text-ink">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
          <span>
            The PDI checklist is not configured, so this vehicle cannot pass PDI.
            {allowed.includes('HOLD') && ' It can still be put on hold with notes and evidence.'}
            {allowed.includes('REJECT') && ' It can still be rejected with notes and evidence.'}
          </span>
        </p>
      </Card>
    )
  return (
    <Card
      title="Checklist"
      description="Record an outcome for every item. A failed item needs a note."
      badge={
        <span className="text-xs text-muted" aria-live="polite">
          {answered} of {checklist.length} answered{failed > 0 ? ` · ${String(failed)} failed` : ''}
        </span>
      }
      contentClassName="p-0"
    >
      <ol className="divide-y divide-border">
        {fields.map((f, i) => {
          const item = checklist[i]
          if (!item) return null
          const err = form.formState.errors.items?.[i]
          const outcome = values[i]?.outcome
          return (
            <li key={f.id} className="flex flex-col gap-3 px-4 py-3 sm:px-5">
              <fieldset className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                <legend className="sr-only">{item.label}</legend>
                <div className="min-w-0">
                  <p className="text-sm font-medium text-ink" aria-hidden="true">
                    {item.label}
                  </p>
                  <p className="font-mono text-xs text-muted">
                    {item.code}
                    {item.brandName ? ` · ${item.brandName} only` : ''}
                  </p>
                  {item.description && (
                    <p className="mt-1 text-xs text-muted">{item.description}</p>
                  )}
                </div>
                <div
                  className="flex shrink-0 gap-2"
                  role="radiogroup"
                  aria-label={`${item.label} outcome`}
                >
                  {(['PASS', 'FAIL'] as const).map((o) => (
                    <label
                      key={o}
                      className={cn(
                        'inline-flex min-h-9 cursor-pointer items-center gap-2 rounded-control border px-3 text-sm font-medium',
                        'has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary/30',
                        outcome === o
                          ? o === 'PASS'
                            ? 'border-success/40 bg-success-subtle text-success'
                            : 'border-danger/40 bg-danger-subtle text-danger'
                          : 'border-border bg-surface text-ink hover:bg-surface-muted',
                      )}
                    >
                      <input
                        type="radio"
                        value={o}
                        className="sr-only"
                        {...form.register(`items.${i}.outcome`)}
                      />
                      {o === 'PASS' ? (
                        <CheckCircle2 className="size-4" aria-hidden="true" />
                      ) : (
                        <XCircle className="size-4" aria-hidden="true" />
                      )}
                      {o === 'PASS' ? 'Pass' : 'Fail'}
                    </label>
                  ))}
                </div>
              </fieldset>
              {err?.outcome?.message && (
                <p role="alert" className="text-xs font-medium text-danger">
                  {item.label}: {err.outcome.message}
                </p>
              )}
              {(outcome === 'FAIL' || err?.note) && (
                <Field
                  label={`Defect note — ${item.label}`}
                  required={outcome === 'FAIL'}
                  error={err?.note?.message}
                  hint={`What is wrong? Up to ${String(MAX_NOTE)} characters.`}
                >
                  {(p) => (
                    <textarea
                      {...p}
                      rows={2}
                      className={textareaClass}
                      {...form.register(`items.${i}.note`)}
                    />
                  )}
                </Field>
              )}
            </li>
          )
        })}
      </ol>
    </Card>
  )
}

function NotesSection({ form }: { form: UseFormReturn<FormValues, unknown, FormValues> }) {
  const disposition = useWatch({ control: form.control, name: 'disposition' })
  const required = disposition === 'HOLD' || disposition === 'REJECT'
  return (
    <Card
      title="Inspection notes"
      description="Required for a hold or rejection: describe the issue."
    >
      <Field
        label="Defect notes"
        required={required}
        error={form.formState.errors.notes?.message}
        hint={`Up to ${String(MAX_NOTES)} characters. Visible in the vehicle's inspection history.`}
      >
        {(p) => <textarea {...p} rows={3} className={textareaClass} {...form.register('notes')} />}
      </Field>
    </Card>
  )
}

function EvidenceSection({
  form,
  evidence,
  canUpload,
  locationId,
  onAdd,
  onRemove,
}: {
  form: UseFormReturn<FormValues, unknown, FormValues>
  evidence: Attachment[]
  canUpload: boolean
  locationId: string | null
  onAdd: (a: Attachment) => void
  onRemove: (id: string) => void
}) {
  const limits = useAttachmentLimits(canUpload)
  const error = form.formState.errors.attachmentIds?.message
  return (
    <Card
      title="Evidence"
      description="Photos or documents of the inspection. Stored privately; attached to this inspection only."
    >
      <div className="flex flex-col gap-4">
        {!canUpload ? (
          <p role="note" className="text-sm text-muted">
            You do not have permission to upload files, so you cannot add evidence. A hold or
            rejection needs evidence.
          </p>
        ) : limits.isPending ? (
          <p className="text-sm text-muted">Loading upload limits…</p>
        ) : limits.isError ? (
          <p role="alert" className="text-sm text-danger">
            Could not load the upload limits.{' '}
            <Button variant="ghost" size="sm" onClick={() => void limits.refetch()}>
              Try again
            </Button>
          </p>
        ) : (
          <AttachmentUploader limits={limits.data} locationId={locationId} onUploaded={onAdd} />
        )}
        {evidence.length > 0 && (
          <ul aria-label="Evidence for this inspection" className="flex flex-col gap-2">
            {evidence.map((a) => (
              <li
                key={a.id}
                className="flex items-center gap-3 rounded-control border border-border bg-surface px-3 py-2 text-sm"
              >
                <Paperclip className="size-4 shrink-0 text-muted" aria-hidden="true" />
                <span className="min-w-0 flex-1 truncate" title={a.fileName}>
                  {a.fileName}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove ${a.fileName} from this inspection`}
                  onClick={() => {
                    onRemove(a.id)
                  }}
                >
                  <X className="size-4" aria-hidden="true" />
                </Button>
              </li>
            ))}
          </ul>
        )}
        {error && (
          <p role="alert" className="text-xs font-medium text-danger">
            {error}
          </p>
        )}
      </div>
    </Card>
  )
}

function DispositionSection({
  form,
  allowed,
  checklistEmpty,
}: {
  form: UseFormReturn<FormValues, unknown, FormValues>
  allowed: Disposition[]
  checklistEmpty: boolean
}) {
  const value = useWatch({ control: form.control, name: 'disposition' })
  const error = form.formState.errors.disposition?.message
  return (
    <Card title="Disposition" description="The decision this inspection records.">
      <fieldset className="flex flex-col gap-3">
        <legend className="sr-only">Disposition</legend>
        <div className="grid gap-3 sm:grid-cols-3">
          {allowed.map((d) => {
            const disabled = d === 'PASS' && checklistEmpty
            return (
              <label
                key={d}
                className={cn(
                  'flex cursor-pointer flex-col gap-2 rounded-card border p-3 text-sm',
                  'has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary/30',
                  disabled && 'cursor-not-allowed opacity-60',
                  value === d
                    ? 'border-primary bg-primary/5'
                    : 'border-border bg-surface hover:bg-surface-muted',
                )}
              >
                <span className="flex items-center gap-2">
                  <input
                    type="radio"
                    value={d}
                    disabled={disabled}
                    className="size-4 accent-primary"
                    {...form.register('disposition')}
                  />
                  <Badge tone={dispositionTone[d]}>{dispositionLabels[d]}</Badge>
                </span>
                <span className="text-xs text-muted">
                  {disabled ? 'Not possible: no checklist is configured.' : dispositionEffect[d]}
                </span>
              </label>
            )
          })}
        </div>
        {allowed.length === 1 && allowed[0] === 'REJECT' && (
          <p role="note" className="text-xs text-muted">
            A vehicle on quality hold can only be rejected here: releasing it to available stock or
            re-inspecting it is not approved yet.
          </p>
        )}
        {error && (
          <p role="alert" className="text-xs font-medium text-danger">
            {error}
          </p>
        )}
      </fieldset>
    </Card>
  )
}

// ── Confirmation ────────────────────────────────────────────────────────────

function ConfirmDialog({
  unitId,
  unitNumber,
  body,
  checklist,
  evidence,
  onClose,
  onServerFields,
  onRecorded,
}: {
  unitId: string
  unitNumber: string
  body: InspectionBody
  checklist: ChecklistItem[]
  evidence: Attachment[]
  onClose: () => void
  onServerFields: (fields: { field: string; message: string }[]) => void
  onRecorded: (r: RecordResult) => void
}) {
  const record = useRecordInspection()
  const [action, setAction] = useState<InspectionAction | null>(null)
  const [error, setError] = useState<PdiError | null>(null)
  const failed = body.items.filter((i) => i.outcome === 'FAIL')
  const labels = checklist.map((c) => c.label)

  const submit = () => {
    // The same unchanged request (e.g. after an unknown outcome) keeps its key.
    const a =
      action && sameInspection(action, unitId, body) ? action : newInspectionAction(unitId, body)
    setAction(a)
    setError(null)
    // mutateAsync, not mutate callbacks: the refetch after success re-keys (unmounts) this form,
    // and per-call mutate callbacks of an unmounted observer never run.
    record.mutateAsync(a).then(onRecorded, (e: unknown) => {
      const d = describePdiError(e, true)
      setError(d)
      if (d.fields.length > 0) onServerFields(d.fields)
      // A refused request used up nothing; a changed form must not reuse the key.
      if (!d.retrySame) setAction(null)
    })
  }

  const d = body.disposition
  return (
    <Dialog
      open
      onClose={() => {
        if (!record.isPending) onClose()
      }}
      title={`${dispositionLabels[d]} ${unitNumber}?`}
      description="Check the inspection. Once recorded it cannot be edited or deleted."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={record.isPending}>
            {error ? 'Back to the form' : 'Cancel'}
          </Button>
          <Button
            variant={d === 'PASS' ? 'primary' : 'danger'}
            onClick={submit}
            disabled={
              record.isPending || (error !== null && !error.retrySame && error.kind !== 'error')
            }
          >
            {record.isPending
              ? 'Recording…'
              : error?.retrySame
                ? 'Send again unchanged'
                : confirmVerb[d]}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4 text-sm">
        <div className="flex items-center gap-2">
          <Badge tone={dispositionTone[d]}>{dispositionLabels[d]}</Badge>
          <span>{dispositionEffect[d]}</span>
        </div>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2">
          <dt className="text-muted">Checklist</dt>
          <dd>
            {body.items.length === 0
              ? 'Not configured'
              : `${String(body.items.length)} items, ${String(failed.length)} failed`}
          </dd>
          <dt className="text-muted">Evidence</dt>
          <dd>{evidence.length === 0 ? 'None' : `${String(evidence.length)} file(s)`}</dd>
        </dl>
        {failed.length > 0 && (
          <div>
            <p className="font-medium">Failed items</p>
            <ul className="mt-1 list-disc pl-5 text-muted">
              {failed.map((f) => (
                <li key={f.checklistItemId}>
                  {checklist.find((c) => c.id === f.checklistItemId)?.label ?? 'Item'}: {f.note}
                </li>
              ))}
            </ul>
          </div>
        )}
        {body.notes && (
          <div>
            <p className="font-medium">Notes</p>
            <p className="whitespace-pre-wrap text-muted">{body.notes}</p>
          </div>
        )}
        {error && (
          <div
            role="alert"
            className="rounded-control border border-danger/30 bg-danger-subtle px-3 py-2 text-danger"
          >
            <p>{error.message}</p>
            {error.fields.length > 0 && (
              <ul className="mt-1 list-disc pl-5">
                {error.fields.map((f) => (
                  <li key={f.field + f.message}>
                    {fieldLabel(f.field, labels)}: {f.message}
                  </li>
                ))}
              </ul>
            )}
            {error.requestId && (
              <p className="mt-1 font-mono text-xs text-muted">Reference: {error.requestId}</p>
            )}
          </div>
        )}
      </div>
    </Dialog>
  )
}
