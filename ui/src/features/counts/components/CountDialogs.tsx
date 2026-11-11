import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useRef, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { useStockItemList } from '@/features/parts/hooks/useStockItems'
import { useVehicleUnitList } from '@/features/vehicles/hooks/useVehicleUnits'
import {
  formatQuantity,
  newCountAction,
  OBSERVED_PATTERN,
  sameAction,
  type CountAction,
  type CountLine,
  type LineBody,
  type StockCount,
} from '../api/stockCounts'
import {
  useCountAction,
  useCountPerms,
  useRecordLine,
  useSubmitCount,
} from '../hooks/useStockCounts'
import { describeCountError, type CountError } from '../lib/countErrors'
import { expectedText, lineLabel, observedText, varianceText } from '../lib/labels'

type Draft = Parameters<typeof newCountAction>[0]

/**
 * The action to send for this attempt: the same request as the last attempt in
 * this dialog reuses its action (and Idempotency-Key), so a retry after an
 * unknown outcome cannot act twice; anything different is a new action.
 */
function useActionFor() {
  const last = useRef<CountAction | null>(null)
  return (draft: Draft): CountAction => {
    if (last.current && sameAction(last.current, draft)) return last.current
    last.current = newCountAction(draft)
    return last.current
  }
}

function FailureBox({ failure, count }: { failure: CountError; count?: StockCount }) {
  const stale = failure.staleLineIds
    .map((id) => count?.lines.find((l) => l.id === id))
    .filter((l): l is CountLine => Boolean(l))
  return (
    <div
      role="alert"
      className={`rounded-control border p-3 text-sm text-ink ${
        failure.kind === 'uncertain'
          ? 'border-warning/30 bg-warning-subtle'
          : 'border-danger/30 bg-danger-subtle'
      }`}
    >
      <p>{failure.message}</p>
      {failure.kind === 'stale' && (
        <>
          {stale.length > 0 && (
            <ul className="ml-5 mt-1 list-disc">
              {stale.map((l) => (
                <li key={l.id}>{lineLabel(l)}</li>
              ))}
            </ul>
          )}
          <p className="mt-1">
            {count?.status === 'DRAFT'
              ? 'Count those items again and record them, then submit.'
              : 'This count cannot be approved as it is. Reject it so the items can be counted again.'}
          </p>
        </>
      )}
      {failure.kind === 'invalid-input' &&
        failure.fields
          .filter((f) => f.field === 'version' || f.field === 'Idempotency-Key')
          .map((f) => (
            <p key={f.field} className="mt-1">
              {f.field}: {f.message}
            </p>
          ))}
      {failure.requestId && (
        <p className="mt-1 font-mono text-xs text-muted">Reference: {failure.requestId}</p>
      )}
    </div>
  )
}

function Shell({
  title,
  description,
  formId,
  pending,
  submitLabel,
  canSubmit = true,
  failure,
  count,
  onClose,
  children,
  danger,
}: {
  title: string
  description: string
  formId: string
  pending: boolean
  submitLabel: string
  canSubmit?: boolean
  failure: CountError | null
  count?: StockCount
  onClose: () => void
  children: ReactNode
  danger?: boolean
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
            disabled={!canSubmit}
            variant={danger ? 'danger' : 'primary'}
          >
            {failure?.retrySame ? 'Send again unchanged' : submitLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {children}
        {failure && <FailureBox failure={failure} count={count} />}
      </div>
    </Dialog>
  )
}

// ── Create ───────────────────────────────────────────────────────────────────

const createSchema = z.object({
  locationId: z.string().min(1, 'Choose a location'),
  note: z.string().trim().max(500, 'At most 500 characters'),
})
type CreateValues = z.input<typeof createSchema>

export function CreateCountDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void
  onCreated: (c: StockCount) => void
}) {
  const formId = useId()
  const perms = useCountPerms()
  const send = useCountAction()
  const actionFor = useActionFor()
  const [failure, setFailure] = useState<CountError | null>(null)
  const form = useForm<CreateValues, unknown, CreateValues>({
    resolver: zodResolver(createSchema),
    defaultValues: {
      locationId: perms.locations.length === 1 ? (perms.locations[0]?.id ?? '') : '',
      note: '',
    },
  })
  const errors = form.formState.errors
  const onSubmit = form.handleSubmit(async (v) => {
    if (send.isPending) return
    setFailure(null)
    const note = v.note.trim()
    const action = actionFor({
      kind: 'create',
      body: { locationId: v.locationId, ...(note ? { note } : {}) },
    })
    try {
      onCreated(await send.mutateAsync(action))
    } catch (e) {
      const d = describeCountError(e, true)
      for (const f of d.fields) {
        if (f.field === 'locationId' || f.field === 'note') form.setError(f.field, f)
      }
      setFailure(d)
    }
  })
  return (
    <Shell
      title="Start a stock count"
      description="Creates a draft count at one location. Nothing changes stock until another person approves the submitted count."
      formId={formId}
      pending={send.isPending}
      submitLabel="Start count"
      canSubmit={perms.locations.length > 0}
      failure={failure}
      onClose={onClose}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <Field
          label="Location"
          required
          hint={
            perms.locations.length === 0
              ? 'No location is assigned to you, so no count can be started. Ask an administrator to assign one.'
              : 'Your assigned locations.'
          }
          error={errors.locationId?.message}
        >
          {(p) => (
            <Select
              {...p}
              options={perms.locations.map((l) => ({
                value: l.id,
                label: `${l.code} — ${l.name}`,
              }))}
              placeholder="Choose a location"
              {...form.register('locationId')}
            />
          )}
        </Field>
        <Field label="Note" hint="Optional, e.g. which shelves." error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </Shell>
  )
}

// ── Record a line ────────────────────────────────────────────────────────────

const lineSchema = z
  .object({
    kind: z.enum(['part', 'vehicle']),
    stockItemId: z.string(),
    observedQuantity: z.string().trim(),
    vehicleUnitId: z.string(),
    found: z.string(),
    reason: z.string().trim().max(300, 'At most 300 characters'),
  })
  .superRefine((v, ctx) => {
    if (v.kind === 'part') {
      if (!v.stockItemId)
        ctx.addIssue({ code: 'custom', path: ['stockItemId'], message: 'Choose the part' })
      if (!OBSERVED_PATTERN.test(v.observedQuantity))
        ctx.addIssue({
          code: 'custom',
          path: ['observedQuantity'],
          message: 'Enter the quantity counted: 0 or more, up to 3 decimals (e.g. 8 or 2.5)',
        })
    } else {
      if (!v.vehicleUnitId)
        ctx.addIssue({
          code: 'custom',
          path: ['vehicleUnitId'],
          message: 'Choose the vehicle unit',
        })
      if (v.found !== 'yes' && v.found !== 'no')
        ctx.addIssue({ code: 'custom', path: ['found'], message: 'Say whether it was found' })
    }
  })
type LineValues = z.input<typeof lineSchema>

const lineFieldNames = [
  'stockItemId',
  'observedQuantity',
  'vehicleUnitId',
  'found',
  'reason',
] as const

/** Records (or re-records) one counted item on a draft. The server reads the expected value. */
export function LineDialog({
  count,
  line,
  onClose,
}: {
  count: StockCount
  /** Re-record this line: the item is fixed. */
  line?: CountLine
  onClose: () => void
}) {
  const formId = useId()
  const perms = useCountPerms()
  const record = useRecordLine(count.id)
  const [failure, setFailure] = useState<CountError | null>(null)
  const [search, setSearch] = useState('')
  const canPart = line ? line.itemType === 'STOCK_ITEM' : perms.stockItems
  const canVehicle = line ? line.itemType === 'VEHICLE_UNIT' : perms.vehicleUnits
  const form = useForm<LineValues, unknown, LineValues>({
    resolver: zodResolver(lineSchema),
    defaultValues: {
      kind: line?.itemType === 'VEHICLE_UNIT' || (!line && !canPart) ? 'vehicle' : 'part',
      stockItemId: line?.stockItemId ?? '',
      observedQuantity:
        line && line.itemType === 'STOCK_ITEM' ? formatQuantity(line.observedQuantity) : '',
      vehicleUnitId: line?.vehicleUnitId ?? '',
      found: line?.found === true ? 'yes' : line?.found === false ? 'no' : '',
      reason: line?.reason ?? '',
    },
  })
  const kind = useWatch({ control: form.control, name: 'kind' })
  const itemId = useWatch({ control: form.control, name: 'stockItemId' })
  const items = useStockItemList(
    { q: search || undefined, status: 'ACTIVE', trackingMode: 'QUANTITY', limit: 50 },
    { enabled: !line && kind === 'part' && perms.stockItems },
  )
  const units = useVehicleUnitList(
    { q: search || undefined, locationId: count.locationId, limit: 100 },
    { enabled: !line && kind === 'vehicle' && perms.vehicleUnits },
  )
  const chosen = items.data?.items.find((i) => i.id === itemId)
  const uom = line?.uom ?? chosen?.uom
  const errors = form.formState.errors

  const onSubmit = form.handleSubmit(async (v) => {
    if (record.isPending) return
    setFailure(null)
    const reason = v.reason.trim()
    const extra = reason ? { reason } : {}
    let body: LineBody
    if (v.kind === 'part') {
      if (!uom) {
        form.setError('stockItemId', { message: 'Choose the part' })
        return
      }
      body = {
        version: count.version,
        stockItemId: v.stockItemId,
        observedQuantity: v.observedQuantity,
        uom,
        ...extra,
      }
    } else {
      body = {
        version: count.version,
        vehicleUnitId: v.vehicleUnitId,
        found: v.found === 'yes',
        ...extra,
      }
    }
    try {
      await record.mutateAsync(body)
      onClose()
    } catch (e) {
      const d = describeCountError(e, false)
      for (const f of d.fields) {
        if (f.field === 'reason') {
          form.setError('reason', {
            message: 'Give a reason: this count differs from the stock record',
          })
        } else if ((lineFieldNames as readonly string[]).includes(f.field)) {
          form.setError(f.field as (typeof lineFieldNames)[number], { message: f.message })
        } else if (f.field === 'uom') {
          form.setError('stockItemId', { message: `Unit: ${f.message}` })
        }
      }
      setFailure(d)
    }
  })

  const searchBox = !line && (
    <Field label={kind === 'part' ? 'Find a part' : 'Find a vehicle unit'} hint="Optional filter.">
      {(p) => (
        <Input
          {...p}
          type="search"
          autoComplete="off"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
          }}
        />
      )}
    </Field>
  )

  return (
    <Shell
      title={line ? 'Record this item again' : 'Record a counted item'}
      description={`${count.countNumber} at ${count.locationCode}. The server reads what the stock record expects; nothing changes stock until the count is approved.`}
      formId={formId}
      pending={record.isPending}
      submitLabel="Save count"
      canSubmit={canPart || canVehicle}
      failure={failure}
      count={count}
      onClose={onClose}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        {!line && canPart && canVehicle && (
          <fieldset className="flex flex-col gap-1.5">
            <legend className="text-sm font-medium text-ink">What was counted</legend>
            <div className="flex flex-wrap gap-4 text-sm">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  value="part"
                  {...form.register('kind', {
                    onChange: () => {
                      setSearch('')
                    },
                  })}
                />
                A part or accessory
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  value="vehicle"
                  {...form.register('kind', {
                    onChange: () => {
                      setSearch('')
                    },
                  })}
                />
                A vehicle unit
              </label>
            </div>
          </fieldset>
        )}
        {!canPart && !canVehicle && (
          <p role="note" className="rounded-control bg-neutral-subtle px-3 py-2 text-sm text-ink">
            Choosing items needs permission to view spare parts or vehicle units.
          </p>
        )}
        {kind === 'part' && canPart && (
          <>
            {line ? (
              <p className="text-sm">
                <span className="font-medium">{lineLabel(line)}</span>
              </p>
            ) : (
              <>
                {searchBox}
                <Field
                  label="Part"
                  required
                  hint={
                    items.isError
                      ? 'Parts could not be loaded.'
                      : 'Active, quantity-tracked SKUs. Serial-tracked SKUs cannot be counted yet.'
                  }
                  error={errors.stockItemId?.message}
                >
                  {(p) => (
                    <Select
                      {...p}
                      key={items.isSuccess ? 'ready' : 'loading'}
                      options={(items.data?.items ?? []).map((i) => ({
                        value: i.id,
                        label: `${i.sku} — ${i.name} (${i.uom})`,
                      }))}
                      placeholder={items.isPending ? 'Loading…' : 'Choose the part'}
                      {...form.register('stockItemId')}
                    />
                  )}
                </Field>
              </>
            )}
            <Field
              label={uom ? `Quantity counted (${uom})` : 'Quantity counted'}
              required
              hint="What is physically there, in the SKU’s unit. 0 if none. Up to 3 decimals."
              error={errors.observedQuantity?.message}
            >
              {(p) => (
                <Input
                  {...p}
                  inputMode="decimal"
                  autoComplete="off"
                  {...form.register('observedQuantity')}
                />
              )}
            </Field>
          </>
        )}
        {kind === 'vehicle' && canVehicle && (
          <>
            {line ? (
              <p className="text-sm font-medium">{lineLabel(line)}</p>
            ) : (
              <>
                {searchBox}
                <Field
                  label="Vehicle unit"
                  required
                  hint={
                    units.isError
                      ? 'Vehicle units could not be loaded.'
                      : `Units recorded at ${count.locationCode}.`
                  }
                  error={errors.vehicleUnitId?.message}
                >
                  {(p) => (
                    <Select
                      {...p}
                      key={units.isSuccess ? 'ready' : 'loading'}
                      options={(units.data?.items ?? []).map((u) => ({
                        value: u.id,
                        label: [u.unitNumber, u.chassisNumber].filter(Boolean).join(' · '),
                      }))}
                      placeholder={units.isPending ? 'Loading…' : 'Choose the unit'}
                      {...form.register('vehicleUnitId')}
                    />
                  )}
                </Field>
              </>
            )}
            <fieldset className="flex flex-col gap-1.5">
              <legend className="text-sm font-medium text-ink">
                Was it there?
                <span className="ml-0.5 text-danger" aria-hidden="true">
                  *
                </span>
              </legend>
              <div className="flex flex-wrap gap-4 text-sm">
                <label className="flex items-center gap-2">
                  <input type="radio" value="yes" {...form.register('found')} />
                  Found
                </label>
                <label className="flex items-center gap-2">
                  <input type="radio" value="no" {...form.register('found')} />
                  Not found
                </label>
              </div>
              {errors.found?.message && (
                <p className="text-xs text-danger">{errors.found.message}</p>
              )}
              <p className="text-xs text-muted">
                A missing unit is recorded as a finding only: approving the count does not change
                the vehicle.
              </p>
            </fieldset>
          </>
        )}
        <Field
          label="Reason"
          hint="Required when your count differs from the stock record (the server checks). At most 300 characters."
          error={errors.reason?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
      </form>
    </Shell>
  )
}

// ── Submit ───────────────────────────────────────────────────────────────────

export function SubmitDialog({ count, onClose }: { count: StockCount; onClose: () => void }) {
  const formId = useId()
  const submit = useSubmitCount(count.id)
  const [failure, setFailure] = useState<CountError | null>(null)
  const run = async () => {
    if (submit.isPending) return
    setFailure(null)
    try {
      await submit.mutateAsync(count.version)
      onClose()
    } catch (err) {
      setFailure(describeCountError(err, false))
    }
  }
  return (
    <Shell
      title={`Submit ${count.countNumber} for approval?`}
      description="The lines become fixed. Someone with approval permission who did not create, count or submit it must then approve or reject it. Submitting changes no stock."
      formId={formId}
      pending={submit.isPending}
      submitLabel="Submit for approval"
      failure={failure}
      count={count}
      onClose={onClose}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          void run()
        }}
      >
        <p className="text-sm">
          {count.lineCount} counted {count.lineCount === 1 ? 'item' : 'items'} at{' '}
          {count.locationCode}. No approval threshold is configured, so every count needs an
          approver.
        </p>
      </form>
    </Shell>
  )
}

// ── Approve / reject ─────────────────────────────────────────────────────────

const rejectSchema = z.object({
  text: z
    .string()
    .trim()
    .min(1, 'Give the reason for rejecting')
    .max(500, 'At most 500 characters'),
})
const approveSchema = z.object({ text: z.string().trim().max(500, 'At most 500 characters') })
type DecisionValues = z.input<typeof approveSchema>

/**
 * Approve (posts the parts adjustments) or reject a submitted count. The
 * version is taken when the dialog opens, so an unchanged retry after an
 * unknown outcome is the same request with the same Idempotency-Key.
 */
export function DecisionDialog({
  count,
  decision,
  onClose,
  onDone,
}: {
  count: StockCount
  decision: 'approve' | 'reject'
  onClose: () => void
  onDone: (c: StockCount) => void
}) {
  const formId = useId()
  const send = useCountAction()
  const actionFor = useActionFor()
  // The version when the dialog opened: an unchanged retry is the same request.
  const [version] = useState(count.version)
  const [failure, setFailure] = useState<CountError | null>(null)
  const approve = decision === 'approve'
  const form = useForm<DecisionValues, unknown, DecisionValues>({
    resolver: zodResolver(approve ? approveSchema : rejectSchema),
    defaultValues: { text: '' },
  })
  const posts = count.lines.filter((l) => l.itemType === 'STOCK_ITEM' && l.variance !== '0.000')
  const findings = count.lines.filter(
    (l) => l.itemType === 'VEHICLE_UNIT' && l.variance !== '0.000',
  )
  const decided = count.status === 'APPROVED' || count.status === 'REJECTED'
  const onSubmit = form.handleSubmit(async (v) => {
    if (send.isPending) return
    setFailure(null)
    const text = v.text.trim()
    const action = actionFor(
      approve
        ? { kind: 'approve', countId: count.id, body: { version, ...(text ? { note: text } : {}) } }
        : { kind: 'reject', countId: count.id, body: { version, reason: text } },
    )
    try {
      onDone(await send.mutateAsync(action))
    } catch (e) {
      const d = describeCountError(e, true)
      for (const f of d.fields) {
        if (f.field === 'note' || f.field === 'reason') form.setError('text', f)
      }
      setFailure(d)
    }
  })
  return (
    <Shell
      title={approve ? `Approve ${count.countNumber}?` : `Reject ${count.countNumber}?`}
      description={
        approve
          ? 'The server checks every line against the stock record again, then posts the adjustments. If stock moved since the count, nothing is posted.'
          : 'Nothing is posted and stock is unchanged. The items can then be counted again in a new count.'
      }
      formId={formId}
      pending={send.isPending}
      submitLabel={approve ? 'Approve and post' : 'Reject count'}
      canSubmit={!(decided && failure?.kind !== 'uncertain')}
      failure={failure}
      count={count}
      onClose={onClose}
      danger={!approve}
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void onSubmit(e)}
        className="flex flex-col gap-4"
      >
        <DecisionSummary count={count} />
        {approve && (
          <p className="text-sm">
            {posts.length === 0
              ? 'No parts line differs from the stock record: approving posts no adjustment.'
              : `${String(posts.length)} parts ${posts.length === 1 ? 'adjustment' : 'adjustments'} will be posted to the stock ledger.`}
            {findings.length > 0 &&
              ` ${String(findings.length)} vehicle ${findings.length === 1 ? 'finding is' : 'findings are'} recorded only: no vehicle is changed.`}
          </p>
        )}
        <Field
          label={approve ? 'Note' : 'Reason'}
          required={!approve}
          hint={approve ? 'Optional.' : 'Shown on the count.'}
          error={form.formState.errors.text?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('text')} />}
        </Field>
        {failure?.kind === 'uncertain' && decided && (
          <p role="status" className="rounded-control bg-info-subtle px-3 py-2 text-sm">
            The server now shows this count as <strong>{count.status.toLowerCase()}</strong>
            {count.decidedByName ? ` by ${count.decidedByName}` : ''}.
          </p>
        )}
      </form>
    </Shell>
  )
}

/** Expected, counted and difference per line, as the server computed them (reviewers only). */
function DecisionSummary({ count }: { count: StockCount }) {
  return (
    <div className="max-h-64 overflow-auto rounded-control border border-border">
      <table className="w-full text-left text-sm">
        <caption className="sr-only">Counted items</caption>
        <thead className="bg-neutral-subtle text-xs text-muted">
          <tr>
            <th scope="col" className="px-3 py-2 font-medium">
              Item
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              Record
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              Counted
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              Difference
            </th>
          </tr>
        </thead>
        <tbody>
          {count.lines.map((l) => (
            <tr key={l.id} className="border-t border-border">
              <td className="px-3 py-2">{lineLabel(l)}</td>
              <td className="px-3 py-2">{expectedText(l)}</td>
              <td className="px-3 py-2">{observedText(l)}</td>
              <td className="px-3 py-2 font-medium">{varianceText(l)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
