import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowDownToLine, ArrowUpFromLine, PackageOpen } from 'lucide-react'
import { useId, useRef, useState, type ReactNode } from 'react'
import { useForm, useWatch, type FieldPath } from 'react-hook-form'
import { z } from 'zod'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import { Select } from '@/components/ui/Select'
import { EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import type { Column } from '@/components/ui/Table'
import { FilterSelect, Toolbar } from '@/features/catalogue/components/ListControls'
import { ResponsiveTable } from '@/features/catalogue/components/ResponsiveTable'
import { formatDateTime } from '@/features/catalogue/lib/presentation'
import { useSupplierList } from '@/features/suppliers/hooks/useSuppliers'
import {
  formatQuantity,
  isValidQuantity,
  newPartsAction,
  sameBody,
  thousandths,
  type IssueBody,
  type PartsAction,
  type PartsMovement,
  type PartsMovementType,
  type PartsResult,
  type ReceiptBody,
} from '../api/partsStock'
import type { StockItemDetail } from '../api/stockItems'
import {
  usePartsBalances,
  usePartsHistory,
  usePartsPost,
  usePartsStockPerms,
} from '../hooks/usePartsStock'
import { describePartsError, type PartsError } from '../lib/stockErrors'

const typeLabels: Record<PartsMovementType, string> = {
  PARTS_RECEIPT: 'Receipt',
  PARTS_ISSUE: 'Issue',
}

/**
 * Parts stock of one SKU (E-04): balances at the caller's locations, receipt and
 * issue forms, and the receipt/issue history. Everything shown comes from the
 * server; nothing here sets or computes a balance.
 */
export function PartsStockPanel({ item }: { item: StockItemDetail }) {
  const perms = usePartsStockPerms()
  const [open, setOpen] = useState<'receipt' | 'issue' | null>(null)
  const [done, setDone] = useState<PartsResult | null>(null)
  if (!perms.view && !perms.receive && !perms.issue && !perms.receiveWithoutSuppliers) return null
  const archived = item.status === 'ARCHIVED'
  const serial = item.trackingMode === 'SERIAL'
  const canAct = !archived && !serial
  const close = () => {
    setOpen(null)
  }
  return (
    <>
      <Card
        title="Stock"
        description="Quantities kept by the server from recorded receipts and issues. They cannot be edited."
        actions={
          canAct && (
            <>
              {perms.receive && (
                <Button
                  size="sm"
                  onClick={() => {
                    setDone(null)
                    setOpen('receipt')
                  }}
                >
                  <ArrowDownToLine className="size-3.5" aria-hidden="true" />
                  Receive stock
                </Button>
              )}
              {perms.issue && (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setDone(null)
                    setOpen('issue')
                  }}
                >
                  <ArrowUpFromLine className="size-3.5" aria-hidden="true" />
                  Issue stock
                </Button>
              )}
            </>
          )
        }
      >
        {serial && (
          <p
            role="note"
            className="mb-3 rounded-control bg-neutral-subtle px-3 py-2 text-sm text-ink"
          >
            Receipts and issues of <strong>serial-tracked</strong> items are not supported yet:
            there is no serial register. Nothing can be recorded here for this SKU.
          </p>
        )}
        {canAct && perms.receiveWithoutSuppliers && (
          <p className="mb-3 text-xs text-muted">
            Receiving stock also needs permission to view suppliers.
          </p>
        )}
        {done && <DoneNote result={done} />}
        {perms.view ? (
          <Balances itemId={item.id} uom={item.uom} />
        ) : (
          <p className="text-sm text-muted">
            Seeing balances needs permission to view parts stock.
          </p>
        )}
      </Card>
      {perms.view && <History itemId={item.id} />}
      {open === 'receipt' && (
        <ReceiptDialog
          item={item}
          onClose={close}
          onDone={(r) => {
            setDone(r)
            close()
          }}
        />
      )}
      {open === 'issue' && (
        <IssueDialog
          item={item}
          showBalance={perms.view}
          onClose={close}
          onDone={(r) => {
            setDone(r)
            close()
          }}
        />
      )}
    </>
  )
}

function DoneNote({ result }: { result: PartsResult }) {
  const m = result.movement
  const sign = m.movementType === 'PARTS_RECEIPT' ? '+' : '−'
  return (
    <p role="status" className="mb-3 rounded-control bg-success-subtle px-3 py-2 text-sm text-ink">
      {typeLabels[m.movementType]} recorded: {sign}
      {formatQuantity(m.quantity)} {m.uom} at {m.locationCode}. Balance there is now{' '}
      <strong>
        {formatQuantity(result.balance.quantity)} {result.balance.uom}
      </strong>
      .
    </p>
  )
}

function Balances({ itemId, uom }: { itemId: string; uom: string }) {
  const q = usePartsBalances(itemId)
  if (q.isPending) return <LoadingState label="Loading balances…" />
  if (q.isError && !q.data)
    return (
      <ErrorState
        error={q.error}
        title="Could not load the balances"
        onRetry={() => void q.refetch()}
      />
    )
  if (q.data.length === 0)
    return (
      <EmptyState title="No stock recorded" icon={<PackageOpen className="size-6 text-muted/80" />}>
        No receipt has been recorded for this SKU at your locations, so its balance there is 0 {uom}
        .
      </EmptyState>
    )
  const cols: Column<(typeof q.data)[number]>[] = [
    { key: 'loc', header: 'Location', render: (b) => `${b.locationCode} — ${b.locationName}` },
    {
      key: 'qty',
      header: 'Balance',
      render: (b) => (
        <span className="font-medium">
          {formatQuantity(b.quantity)} {b.uom}
        </span>
      ),
    },
    { key: 'at', header: 'Last change', render: (b) => formatDateTime(b.updatedAt) },
  ]
  return (
    <ResponsiveTable caption="Balances" columns={cols} rows={q.data} rowKey={(b) => b.locationId} />
  )
}

function History({ itemId }: { itemId: string }) {
  const perms = usePartsStockPerms()
  const [filters, setFilters] = useState<{ locationId: string; type: string; page: number }>({
    locationId: '',
    type: '',
    page: 1,
  })
  const q = usePartsHistory(itemId, {
    locationId: filters.locationId || undefined,
    type: (filters.type || undefined) as PartsMovementType | undefined,
    page: filters.page,
    limit: 25,
  })
  const filtered = Boolean(filters.locationId || filters.type)
  const cols: Column<PartsMovement>[] = [
    { key: 'at', header: 'When', render: (m) => formatDateTime(m.createdAt) },
    {
      key: 'type',
      header: 'Type',
      render: (m) => (
        <Badge tone={m.movementType === 'PARTS_RECEIPT' ? 'success' : 'info'}>
          {typeLabels[m.movementType]}
        </Badge>
      ),
    },
    {
      key: 'qty',
      header: 'Quantity',
      render: (m) => (
        <span className="whitespace-nowrap">
          {m.movementType === 'PARTS_RECEIPT' ? '+' : '−'}
          {formatQuantity(m.quantity)} {m.uom}
          <span className="block text-xs text-muted">
            {formatQuantity(m.quantityBefore)} → {formatQuantity(m.quantityAfter)}
          </span>
        </span>
      ),
    },
    { key: 'loc', header: 'Location', render: (m) => m.locationCode },
    {
      key: 'source',
      header: 'Source',
      render: (m) => (
        <span className="text-xs">
          {m.supplierName && <span className="block font-medium">{m.supplierName}</span>}
          {m.sourceReference ?? (m.movementType === 'PARTS_ISSUE' ? '—' : '')}
          {m.reason && <span className="block text-muted">Reason: {m.reason}</span>}
          {m.note && <span className="block text-muted">Note: {m.note}</span>}
        </span>
      ),
    },
    { key: 'by', header: 'By', render: (m) => m.actorName ?? '—' },
  ]
  let body: ReactNode
  if (q.isPending) body = <LoadingState label="Loading history…" />
  else if (q.isError && !q.data)
    body = (
      <ErrorState
        error={q.error}
        title="Could not load the history"
        onRetry={() => void q.refetch()}
      />
    )
  else if (q.data.total === 0)
    body = filtered ? (
      <EmptyState title="Nothing matches these filters">Change or clear the filters.</EmptyState>
    ) : (
      <EmptyState title="No receipts or issues yet">
        Recorded receipts and issues of this SKU at your locations appear here.
      </EmptyState>
    )
  else
    body = (
      <div className={q.isPlaceholderData ? 'opacity-70' : undefined}>
        <ResponsiveTable
          caption="Receipts and issues"
          columns={cols}
          rows={q.data.items}
          rowKey={(m) => m.id}
        />
        <Pagination
          page={q.data.page}
          limit={q.data.limit}
          total={q.data.total}
          noun={['entry', 'entries']}
          onPage={(page) => {
            setFilters((f) => ({ ...f, page }))
          }}
        />
      </div>
    )
  return (
    <Card
      title="Receipts and issues"
      description={`Newest first. Recorded entries cannot be edited or deleted.${perms.supplierView ? '' : ' Suppliers are not shown without permission to view suppliers.'}`}
    >
      <Toolbar
        active={filtered}
        onClear={() => {
          setFilters({ locationId: '', type: '', page: 1 })
        }}
      >
        <FilterSelect
          label="Location"
          value={filters.locationId}
          options={perms.locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
          allLabel="All my locations"
          onChange={(v) => {
            setFilters((f) => ({ ...f, locationId: v, page: 1 }))
          }}
        />
        <FilterSelect
          label="Type"
          value={filters.type}
          options={Object.entries(typeLabels).map(([value, label]) => ({ value, label }))}
          allLabel="Receipts and issues"
          onChange={(v) => {
            setFilters((f) => ({ ...f, type: v, page: 1 }))
          }}
        />
      </Toolbar>
      {body}
    </Card>
  )
}

// ── Receipt and issue forms ─────────────────────────────────────────────────

const quantityRule = z
  .string()
  .trim()
  .refine((v) => isValidQuantity(v), {
    message: 'Enter a quantity above 0 with up to 3 decimals (e.g. 5 or 2.5)',
  })
const optional = (max: number) =>
  z
    .string()
    .trim()
    .max(max, `At most ${String(max)} characters`)

const receiptSchema = z.object({
  locationId: z.string().min(1, 'Choose a location'),
  quantity: quantityRule,
  supplierId: z.string().min(1, 'Supplier is required'),
  sourceReference: z
    .string()
    .trim()
    .min(1, 'Enter the challan or invoice number')
    .max(100, 'At most 100 characters'),
  note: optional(500),
})
type ReceiptValues = z.input<typeof receiptSchema>

const issueSchema = z.object({
  locationId: z.string().min(1, 'Choose a location'),
  quantity: quantityRule,
  reason: z
    .string()
    .trim()
    .min(1, 'Give the reason for this issue')
    .max(300, 'At most 300 characters'),
  sourceReference: optional(100),
  note: optional(500),
})
type IssueValues = z.input<typeof issueSchema>

/**
 * The action to send for these values: the same values as the last attempt in
 * this form reuse its action (and so its Idempotency-Key), so a retry after an
 * unknown outcome cannot record twice; any change is a new action.
 */
function useActionFor() {
  const last = useRef<PartsAction | null>(null)
  return (make: () => PartsAction, body: object): PartsAction => {
    if (last.current && sameBody(last.current.body, body)) return last.current
    last.current = make()
    return last.current
  }
}

function useSubmit<V extends Record<string, unknown>>(
  itemId: string,
  onDone: (r: PartsResult) => void,
  setError: (name: FieldPath<V>, e: { message: string }) => void,
  names: readonly string[],
) {
  const post = usePartsPost(itemId)
  const [failure, setFailure] = useState<PartsError | null>(null)
  const run = async (action: PartsAction) => {
    if (post.isPending) return
    setFailure(null)
    try {
      onDone(await post.mutateAsync(action))
    } catch (e) {
      const d = describePartsError(e)
      for (const f of d.fields) {
        if (names.includes(f.field)) setError(f.field as FieldPath<V>, { message: f.message })
      }
      setFailure(d)
    }
  }
  return { post, failure, run }
}

function FormShell({
  title,
  description,
  formId,
  pending,
  submitLabel,
  canSubmit,
  failure,
  onClose,
  children,
}: {
  title: string
  description: string
  formId: string
  pending: boolean
  submitLabel: string
  canSubmit: boolean
  failure: PartsError | null
  onClose: () => void
  children: ReactNode
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
          <Button type="submit" form={formId} loading={pending} disabled={!canSubmit}>
            {failure?.retrySame ? 'Send again unchanged' : submitLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {children}
        {failure && (
          <div
            role="alert"
            className={`rounded-control border p-3 text-sm text-ink ${
              failure.kind === 'uncertain'
                ? 'border-warning/30 bg-warning-subtle'
                : 'border-danger/30 bg-danger-subtle'
            }`}
          >
            <p>{failure.message}</p>
            {failure.kind === 'invalid-input' &&
              failure.fields
                .filter((f) => f.field === 'uom' || f.field === 'Idempotency-Key')
                .map((f) => (
                  <p key={f.field} className="mt-1">
                    {f.field === 'uom' ? 'Unit' : f.field}: {f.message}
                  </p>
                ))}
            {failure.requestId && (
              <p className="mt-1 font-mono text-xs text-muted">Reference: {failure.requestId}</p>
            )}
          </div>
        )}
      </div>
    </Dialog>
  )
}

function LocationField({
  error,
  registration,
}: {
  error?: string
  registration: ReturnType<ReturnType<typeof useForm>['register']>
}) {
  const perms = usePartsStockPerms()
  return (
    <Field
      label="Location"
      required
      hint={
        perms.locations.length === 0
          ? 'No location is assigned to you, so nothing can be recorded. Ask an administrator to assign one.'
          : 'Your assigned locations.'
      }
      error={error}
    >
      {(p) => (
        <Select
          {...p}
          options={perms.locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
          placeholder="Choose a location"
          {...registration}
        />
      )}
    </Field>
  )
}

function UnitDisplay({ uom }: { uom: string }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-ink">Unit</span>
      <span
        className="rounded-control border border-border bg-neutral-subtle px-3 py-2 font-mono text-sm"
        aria-label={`Unit (the SKU's unit): ${uom}`}
      >
        {uom}
      </span>
      <span className="text-xs text-muted">The SKU’s unit. No conversion.</span>
    </div>
  )
}

const receiptNames = ['locationId', 'quantity', 'supplierId', 'sourceReference', 'note'] as const

function ReceiptDialog({
  item,
  onClose,
  onDone,
}: {
  item: StockItemDetail
  onClose: () => void
  onDone: (r: PartsResult) => void
}) {
  const formId = useId()
  const perms = usePartsStockPerms()
  const suppliers = useSupplierList({ limit: 100 })
  const form = useForm<ReceiptValues, unknown, ReceiptValues>({
    resolver: zodResolver(receiptSchema),
    defaultValues: {
      locationId: perms.locations.length === 1 ? (perms.locations[0]?.id ?? '') : '',
      quantity: '',
      supplierId: '',
      sourceReference: '',
      note: '',
    },
  })
  const errors = form.formState.errors
  const actionFor = useActionFor()
  const { post, failure, run } = useSubmit<ReceiptValues>(
    item.id,
    onDone,
    form.setError,
    receiptNames,
  )
  const onSubmit = form.handleSubmit((v) => {
    const body: ReceiptBody = {
      locationId: v.locationId,
      quantity: v.quantity.trim(),
      uom: item.uom,
      supplierId: v.supplierId,
      sourceReference: v.sourceReference.trim(),
      ...(v.note.trim() ? { note: v.note.trim() } : {}),
    }
    return run(actionFor(() => newPartsAction('receipt', item.id, body), body))
  })
  const options = (suppliers.data?.items ?? [])
    .filter((s) => s.status !== 'ARCHIVED')
    .map((s) => ({
      value: s.id,
      label: s.legalName + (s.status === 'PENDING_VERIFICATION' ? ' (pending verification)' : ''),
    }))
  return (
    <FormShell
      title="Receive stock"
      description={`${item.sku} — adds to the balance at the chosen location once the server records it.`}
      formId={formId}
      pending={post.isPending}
      submitLabel="Record receipt"
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
        <LocationField
          error={errors.locationId?.message}
          registration={form.register('locationId')}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Quantity"
            required
            hint="Up to 3 decimals."
            error={errors.quantity?.message}
          >
            {(p) => (
              <Input {...p} inputMode="decimal" autoComplete="off" {...form.register('quantity')} />
            )}
          </Field>
          <UnitDisplay uom={item.uom} />
        </div>
        <Field
          label="Supplier"
          required
          hint={suppliers.isError ? 'Suppliers could not be loaded.' : undefined}
          error={errors.supplierId?.message}
        >
          {(p) => (
            <Select
              {...p}
              key={suppliers.isSuccess ? 'ready' : 'loading'}
              options={options}
              placeholder={suppliers.isPending ? 'Loading…' : 'Choose the supplier'}
              {...form.register('supplierId')}
            />
          )}
        </Field>
        <Field
          label="Challan or invoice number"
          required
          hint="As printed on the supplier’s delivery paperwork."
          error={errors.sourceReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('sourceReference')} />}
        </Field>
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormShell>
  )
}

const issueNames = ['locationId', 'quantity', 'reason', 'sourceReference', 'note'] as const

function IssueDialog({
  item,
  showBalance,
  onClose,
  onDone,
}: {
  item: StockItemDetail
  showBalance: boolean
  onClose: () => void
  onDone: (r: PartsResult) => void
}) {
  const formId = useId()
  const perms = usePartsStockPerms()
  const balances = usePartsBalances(item.id, { enabled: showBalance })
  const form = useForm<IssueValues, unknown, IssueValues>({
    resolver: zodResolver(issueSchema),
    defaultValues: {
      locationId: perms.locations.length === 1 ? (perms.locations[0]?.id ?? '') : '',
      quantity: '',
      reason: '',
      sourceReference: '',
      note: '',
    },
  })
  const errors = form.formState.errors
  const locationId = useWatch({ control: form.control, name: 'locationId' })
  const quantity = useWatch({ control: form.control, name: 'quantity' })
  const actionFor = useActionFor()
  const { post, failure, run } = useSubmit<IssueValues>(item.id, onDone, form.setError, issueNames)
  const onSubmit = form.handleSubmit((v) => {
    const body: IssueBody = {
      locationId: v.locationId,
      quantity: v.quantity.trim(),
      uom: item.uom,
      reason: v.reason.trim(),
      ...(v.sourceReference.trim() ? { sourceReference: v.sourceReference.trim() } : {}),
      ...(v.note.trim() ? { note: v.note.trim() } : {}),
    }
    return run(actionFor(() => newPartsAction('issue', item.id, body), body))
  })
  // Information only: the server checks the balance again when it records the issue.
  const available = locationId
    ? (balances.data?.find((b) => b.locationId === locationId)?.quantity ??
      (balances.isSuccess ? '0.000' : null))
    : null
  const over =
    available !== null &&
    isValidQuantity(quantity.trim()) &&
    thousandths(quantity.trim()) > thousandths(available)
  return (
    <FormShell
      title="Issue stock"
      description={`${item.sku} — takes from the balance at the chosen location once the server records it.`}
      formId={formId}
      pending={post.isPending}
      submitLabel="Record issue"
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
        <LocationField
          error={errors.locationId?.message}
          registration={form.register('locationId')}
        />
        {available !== null && (
          <p className="text-sm text-ink" role="status">
            Available there now:{' '}
            <strong>
              {formatQuantity(available)} {item.uom}
            </strong>
            <span className="block text-xs text-muted">
              Others may record issues meanwhile; the server checks again when you submit.
            </span>
          </p>
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Quantity"
            required
            hint={
              over
                ? 'More than the balance shown above: the server will refuse it.'
                : 'Up to 3 decimals.'
            }
            error={errors.quantity?.message}
          >
            {(p) => (
              <Input {...p} inputMode="decimal" autoComplete="off" {...form.register('quantity')} />
            )}
          </Field>
          <UnitDisplay uom={item.uom} />
        </div>
        <Field
          label="Reason"
          required
          hint="Why the stock leaves (e.g. used in a repair)."
          error={errors.reason?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('reason')} />}
        </Field>
        <Field
          label="Reference"
          hint="Optional, e.g. a job card number."
          error={errors.sourceReference?.message}
        >
          {(p) => <Input {...p} autoComplete="off" {...form.register('sourceReference')} />}
        </Field>
        <Field label="Note" error={errors.note?.message}>
          {(p) => <Input {...p} autoComplete="off" {...form.register('note')} />}
        </Field>
      </form>
    </FormShell>
  )
}
