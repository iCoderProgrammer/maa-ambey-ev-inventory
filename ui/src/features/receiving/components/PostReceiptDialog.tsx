import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { useId, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { Select } from '@/components/ui/Select'
import { useMe } from '@/features/access/hooks/useAccess'
import {
  newPostAction,
  samePost,
  type GRNDetail,
  type PostAction,
  type PostResult,
} from '../api/goodsReceipts'
import { useGoodsReceipt, usePostGoodsReceipt } from '../hooks/useGoodsReceipts'
import { describePostError, type PostError } from '../lib/postErrors'

const schema = z.object({
  locationId: z.string().min(1, 'Choose where the goods were received.'),
})
type Values = z.infer<typeof schema>

/**
 * Review and confirm a posting (D-07). Shows what the saved draft will do —
 * the server re-checks every quantity, unit and identifier and its answer wins —
 * and asks where the goods were received (the caller's assigned locations; no
 * default is assumed, OQ-24). One action and Idempotency-Key per confirmation.
 */
export function PostReceiptDialog({
  grn,
  onClose,
  onPosted,
}: {
  grn: GRNDetail
  onClose: () => void
  onPosted: (r: PostResult) => void
}) {
  const formId = useId()
  const me = useMe()
  const locations = me.data?.locationScope.locations ?? []
  const post = usePostGoodsReceipt()
  // Re-read after an unknown outcome or a conflict (the page's query; refreshed on error).
  const latest = useGoodsReceipt(grn.id)
  const current = latest.data ?? grn
  const [action, setAction] = useState<PostAction | null>(null)
  const [error, setError] = useState<PostError | null>(null)
  const form = useForm<Values, unknown, Values>({
    resolver: zodResolver(schema),
    defaultValues: { locationId: locations.length === 1 ? (locations[0]?.id ?? '') : '' },
  })
  const locationId = useWatch({ control: form.control, name: 'locationId' })
  const location = locations.find((l) => l.id === locationId)

  const lines = current.lines.filter((l) => l.id !== null)
  const lineNos = current.lines.map((l) => l.lineNo)
  const accepted = lines.reduce((n, l) => n + l.acceptedQuantity, 0)
  const rejected = lines.reduce((n, l) => n + l.rejectedQuantity, 0)
  const missingChassis = lines.reduce(
    (n, l) =>
      n + l.units.filter((u) => u.condition === 'ACCEPTED' && u.chassisNumber === null).length,
    0,
  )
  const stillDraft = current.status === 'DRAFT'
  // The version the user reviewed when the dialog opened (the page's query refreshes in place).
  const [openedVersion] = useState(grn.version)
  const changed = current.version !== openedVersion

  const submit = form.handleSubmit((v) => {
    if (post.isPending || !stillDraft) return
    const body = { version: current.version, locationId: v.locationId }
    // The same unchanged request (after an unknown outcome) keeps its key.
    const a =
      action && samePost(action, current.id, body) ? action : newPostAction(current.id, body)
    setAction(a)
    setError(null)
    post.mutateAsync(a).then(onPosted, (e: unknown) => {
      const d = describePostError(e, lineNos)
      setError(d)
      if (!d.uncertain) setAction(null)
      void latest.refetch()
    })
  })

  const retrySame = error?.uncertain === true && action !== null && stillDraft
  return (
    <Dialog
      open
      onClose={() => {
        if (!post.isPending) onClose()
      }}
      title={`Post ${current.number}?`}
      description="Posting finalises the receipt: it can no longer be edited or cancelled, and the accepted units become vehicle records in stock."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={post.isPending}>
            {stillDraft ? 'Cancel' : 'Close'}
          </Button>
          {!stillDraft && action !== null && current.status === 'POSTED' && (
            <Button
              disabled={post.isPending}
              onClick={() => {
                // Same key: the server returns the posting it already made (no second posting).
                post.mutateAsync(action).then(onPosted, (e: unknown) => {
                  setError(describePostError(e, lineNos))
                })
              }}
            >
              {post.isPending ? 'Loading…' : 'Show the posting result'}
            </Button>
          )}
          {stillDraft && (
            <Button type="submit" form={formId} disabled={post.isPending || locations.length === 0}>
              {post.isPending
                ? 'Posting…'
                : retrySame
                  ? 'Send the same request again'
                  : 'Post receipt'}
            </Button>
          )}
        </>
      }
    >
      <form
        id={formId}
        noValidate
        onSubmit={(e) => void submit(e)}
        className="flex flex-col gap-4 text-sm"
      >
        {!stillDraft && (
          <p
            role="status"
            className="rounded-control border border-info/30 bg-info-subtle px-3 py-2"
          >
            This receipt is now {current.status.toLowerCase()}
            {current.status === 'POSTED' ? ' — check its result on the page.' : '.'}
          </p>
        )}
        {stillDraft && changed && (
          <p
            role="status"
            className="rounded-control border border-warning/30 bg-warning-subtle px-3 py-2"
          >
            The receipt changed since you opened this dialog. Review the figures below before
            posting.
          </p>
        )}
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
          <dt className="text-muted">Purchase order</dt>
          <dd>
            {current.purchaseOrderNumber} · {current.supplierName}
          </dd>
          <dt className="text-muted">Received</dt>
          <dd>
            {current.receivedQuantity} units: {accepted} accepted, {rejected} rejected
          </dd>
        </dl>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <caption className="sr-only">Lines of {current.number} as saved</caption>
            <thead className="text-muted">
              <tr>
                <th scope="col" className="py-1 pr-2 font-medium">
                  PO line
                </th>
                <th scope="col" className="py-1 pr-2 font-medium">
                  Ordered
                </th>
                <th scope="col" className="py-1 pr-2 font-medium">
                  Posted earlier
                </th>
                <th scope="col" className="py-1 pr-2 font-medium">
                  This receipt
                </th>
                <th scope="col" className="py-1 font-medium">
                  Left after posting
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {lines.map((l) => (
                <tr key={l.purchaseOrderLineId}>
                  <td className="py-1.5 pr-2">
                    {l.lineNo}. {l.brandName} · {l.modelName} · {l.variantName}
                    {l.units.some((u) => u.condition === 'ACCEPTED' && u.chassisNumber) && (
                      <span className="block font-mono text-muted">
                        {l.units
                          .filter((u) => u.condition === 'ACCEPTED' && u.chassisNumber)
                          .map((u) => u.chassisNumber)
                          .join(', ')}
                      </span>
                    )}
                  </td>
                  <td className="py-1.5 pr-2">{l.orderedQuantity}</td>
                  <td className="py-1.5 pr-2">{l.postedQuantity}</td>
                  <td className="py-1.5 pr-2">
                    {l.receivedQuantity} ({l.acceptedQuantity} acc., {l.rejectedQuantity} rej.)
                  </td>
                  <td className="py-1.5">
                    {Math.max(l.orderedQuantity - l.postedQuantity - l.receivedQuantity, 0)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-1 text-xs text-muted">
            Figures from the saved draft. The server checks them again against the purchase order
            when you post.
          </p>
        </div>
        {locations.length === 0 ? (
          <p role="alert" className="text-danger">
            No location is assigned to you, so you cannot choose where the goods were received. Ask
            an administrator to assign one.
          </p>
        ) : (
          <Field
            label="Received at"
            required
            error={form.formState.errors.locationId?.message}
            hint="Where the goods physically arrived. The vehicles are registered there."
          >
            {(p) => (
              <Select
                {...p}
                placeholder="Choose a location"
                options={locations.map((l) => ({ value: l.id, label: `${l.code} — ${l.name}` }))}
                {...form.register('locationId')}
              />
            )}
          </Field>
        )}
        <div className="rounded-control border border-border bg-surface-muted/40 px-3 py-2">
          <p className="font-medium">What posting does</p>
          <ul className="mt-1 list-disc pl-5 text-muted">
            <li>
              {accepted} accepted unit{accepted === 1 ? '' : 's'} become vehicle record
              {accepted === 1 ? '' : 's'}
              {location ? ` at ${location.code}` : ''}, <strong>awaiting PDI — not saleable</strong>
              .
            </li>
            <li>
              {rejected} rejected unit{rejected === 1 ? '' : 's'} create no vehicle.
            </li>
            <li>The purchase order&apos;s received quantities and status are updated.</li>
            <li>The receipt becomes final and can no longer change.</li>
          </ul>
          {missingChassis > 0 && (
            <p className="mt-2 flex gap-1.5 text-warning">
              <AlertTriangle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
              {missingChassis} accepted unit{missingChassis === 1 ? ' has' : 's have'} no
              chassis/frame number;
              {missingChassis === 1 ? ' it' : ' they'} cannot pass PDI until one is recorded.
            </p>
          )}
        </div>
        {error && (
          <div
            role="alert"
            className="rounded-control border border-danger/30 bg-danger-subtle px-3 py-2"
          >
            <p>{error.message}</p>
            {error.blockers.length > 0 && (
              <ul className="mt-1 list-disc pl-5">
                {error.blockers.map((b, i) => (
                  <li key={`${b.key}-${String(i)}`}>
                    <span className="font-medium">{b.key}</span> — {b.message}
                  </li>
                ))}
              </ul>
            )}
            {latest.isFetching && (
              <p className="mt-1 flex items-center gap-1 text-xs text-muted">
                <RefreshCw className="size-3 animate-spin" aria-hidden="true" /> Checking the
                receipt again…
              </p>
            )}
            {error.requestId && (
              <p className="mt-1 font-mono text-xs text-muted">Reference: {error.requestId}</p>
            )}
          </div>
        )}
      </form>
    </Dialog>
  )
}
