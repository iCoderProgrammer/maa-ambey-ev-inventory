import { QueryClient } from '@tanstack/react-query'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { eventSchema as poEventSchema } from '@/features/purchasing/api/purchaseOrders'
import { eventLabels as poEventLabels } from '@/features/purchasing/lib/labels'
import { renderApp, testMe } from '@/test/render'
import type { GRNDetail, GRNLine, GRNUnit, PostResult } from './api/goodsReceipts'

// Placeholder data only ("GRN-000001", "CH-0001", "SR1"): nothing real. API
// answers are mocked; the transaction, locking, idempotency and rollback are
// proven by the Go/PostgreSQL tests.

const t = '2026-11-12T09:00:00Z'
const locs = [
  { id: 'l1', code: 'SR1', name: 'Showroom One' },
  { id: 'l2', code: 'SR2', name: 'Showroom Two' },
]
const me = (permissions: string[], locations = locs): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations },
})
const POSTER = [
  'goods_receipts.view',
  'goods_receipts.manage',
  'goods_receipts.post',
  'vehicle_units.view',
]

const unit = (
  no: number,
  chassis: string | null,
  condition: 'ACCEPTED' | 'REJECTED' = 'ACCEPTED',
): GRNUnit => ({
  id: `u${String(no)}`,
  unitNo: no,
  condition,
  chassisNumber: chassis,
  chassisNormalised: chassis,
  motorSerial: null,
  batterySerial: null,
  chargerSerial: null,
  note: null,
  warnings: chassis ? [] : ['CHASSIS_MISSING'],
})
const line: GRNLine = {
  id: 'gl1',
  purchaseOrderLineId: 'pl1',
  lineNo: 1,
  brandId: 'b1',
  brandName: 'Brand One',
  modelName: 'Model One',
  variantName: 'Standard',
  orderedQuantity: 5,
  postedQuantity: 1,
  otherDraftsQuantity: 0,
  availableQuantity: 4,
  receivedQuantity: 3,
  acceptedQuantity: 2,
  rejectedQuantity: 1,
  excessQuantity: 0,
  overReceipt: null,
  discrepancyNote: 'One scratched',
  units: [unit(1, 'CH-0001'), unit(2, 'CH-0002'), unit(3, 'CH-0003', 'REJECTED')],
  warnings: ['PARTIAL_DELIVERY'],
}
const draft: GRNDetail = {
  id: 'g1',
  number: 'GRN-000001',
  status: 'DRAFT',
  purchaseOrderId: 'po1',
  purchaseOrderNumber: 'PO-000001',
  purchaseOrderStatus: 'APPROVED',
  supplierId: 's1',
  supplierName: 'Example Supplier',
  receivedDate: '2026-11-10',
  supplierReference: 'DC-1',
  notes: null,
  createdBy: testMe.user.id,
  createdByName: 'Test User',
  cancelledBy: null,
  cancelledAt: null,
  cancelReason: null,
  postedBy: null,
  postedAt: null,
  locationId: null,
  locationCode: null,
  locationName: null,
  lineCount: 1,
  receivedQuantity: 3,
  acceptedQuantity: 2,
  rejectedQuantity: 1,
  unitCount: 3,
  version: 4,
  createdAt: t,
  updatedAt: t,
  purchaseOrderReceivable: true,
  lines: [line],
  documents: [],
  events: [
    {
      id: 'e1',
      action: 'CREATED',
      fromStatus: null,
      toStatus: 'DRAFT',
      actorId: testMe.user.id,
      actorName: 'Test User',
      note: null,
      createdAt: t,
    },
  ],
  stockPosted: false,
  actions: {
    edit: true,
    cancel: true,
    addDocument: false,
    authoriseOverReceipt: false,
    post: true,
  },
}
const postedDetail = (loc = locs[0]!): GRNDetail => ({
  ...draft,
  status: 'POSTED',
  version: draft.version + 1,
  postedBy: testMe.user.id,
  postedAt: t,
  locationId: loc.id,
  locationCode: loc.code,
  locationName: loc.name,
  purchaseOrderStatus: 'PARTIALLY_RECEIVED',
  purchaseOrderReceivable: true,
  stockPosted: true,
  events: [
    ...draft.events,
    {
      id: 'e2',
      action: 'POSTED',
      fromStatus: 'DRAFT',
      toStatus: 'POSTED',
      actorId: testMe.user.id,
      actorName: 'Test User',
      note: null,
      createdAt: t,
    },
  ],
  actions: {
    edit: false,
    cancel: false,
    addDocument: false,
    authoriseOverReceipt: false,
    post: false,
  },
})
const result = (loc = locs[0]!): PostResult => ({
  receipt: postedDetail(loc),
  vehicleUnits: [
    {
      vehicleUnitId: 'v1',
      unitNumber: 'MAE-VEH-000001',
      goodsReceiptUnitId: 'u1',
      chassisNumber: 'CH-0001',
      status: 'RECEIVED_PENDING_PDI',
    },
    {
      vehicleUnitId: 'v2',
      unitNumber: 'MAE-VEH-000002',
      goodsReceiptUnitId: 'u2',
      chassisNumber: 'CH-0002',
      status: 'RECEIVED_PENDING_PDI',
    },
  ],
})
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json(
    { error: { code, message: 'Refused by the server.', requestId: 'req-7', details } },
    { status },
  )

type Handler = (
  url: URL,
  init: RequestInit | undefined,
) => Response | Error | Promise<Response> | undefined

function api(state: { detail: GRNDetail }, handler: Handler = () => undefined) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom instanceof Error) return Promise.reject(custom)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/goods-receipts/g1') return json(state.detail)
    if (p === '/api/v1/vehicle-units') return json({ items: [], page: 1, limit: 100, total: 0 })
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof api>
const posts = (spy: Spy) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      init?.method === 'POST' &&
      new URL(u as string, 'http://x').pathname === '/api/v1/goods-receipts/g1/post',
  ) as [string, RequestInit][]
const headerOf = (c: [string, RequestInit], h: string) => new Headers(c[1].headers).get(h)
const bodyOf = (c: [string, RequestInit] | undefined): unknown =>
  typeof c?.[1].body === 'string' ? JSON.parse(c[1].body) : undefined
const detailGets = (spy: Spy) =>
  spy.mock.calls.filter(
    ([u, i]) =>
      (i?.method ?? 'GET') === 'GET' &&
      new URL(u as string, 'http://x').pathname === '/api/v1/goods-receipts/g1',
  ).length

afterEach(() => {
  vi.restoreAllMocks()
})

async function openDialog(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: 'Post receipt' }))
  return screen.findByRole('dialog', { name: 'Post GRN-000001?' })
}

describe('posting a goods receipt (D-07)', () => {
  it('offers Post only for a postable draft and the right permission', async () => {
    api({ detail: { ...draft, actions: { ...draft.actions, post: false } } })
    const view = renderApp('/procurement/goods-receipts/g1', { me: me(POSTER) })
    expect(await screen.findByRole('heading', { level: 1, name: 'GRN-000001' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Post receipt' })).not.toBeInTheDocument()
    view.unmount()
    vi.restoreAllMocks()
    api({
      detail: {
        ...postedDetail(),
        actions: { ...draft.actions, edit: false, cancel: false, post: false },
      },
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER) })
    expect(await screen.findByText(/SR1 — Showroom One/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Post receipt' })).not.toBeInTheDocument()
  })

  it('shows the saved draft, requires a location and sends nothing on cancel', async () => {
    const user = userEvent.setup()
    const spy = api({ detail: draft })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER) })
    const dialog = await openDialog(user)
    const row = within(dialog).getByRole('row', { name: /Brand One/ })
    expect(
      within(row)
        .getAllByRole('cell')
        .map((c) => c.textContent),
    ).toEqual([
      '1. Brand One · Model One · StandardCH-0001, CH-0002',
      '5',
      '1',
      '3 (2 acc., 1 rej.)',
      '1',
    ])
    expect(within(dialog).getByText(/awaiting PDI — not saleable/)).toBeInTheDocument()
    expect(within(dialog).getByText(/1 rejected unit create no vehicle/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(
      await within(dialog).findByText('Choose where the goods were received.'),
    ).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(posts(spy)).toHaveLength(0)
  })

  it('posts once with one key and CSRF, then shows the registered units awaiting PDI', async () => {
    const user = userEvent.setup()
    const state = { detail: draft }
    let release: (() => void) | undefined
    const invalidate = vi.spyOn(QueryClient.prototype, 'invalidateQueries')
    const spy = api(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post')
        return new Promise<Response>((resolve) => {
          release = () => {
            state.detail = postedDetail(locs[1])
            resolve(Response.json({ data: result(locs[1]) }))
          }
        })
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER) })
    const dialog = await openDialog(user)
    await user.selectOptions(within(dialog).getByLabelText(/Received at/), 'l2')
    expect(within(dialog).getByText(/at SR2/)).toBeInTheDocument()
    await user.dblClick(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(within(dialog).getByRole('button', { name: 'Posting…' })).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.queryByText(/GRN-000001 posted\./)).not.toBeInTheDocument()
    const calls = posts(spy)
    expect(calls).toHaveLength(1)
    const [call] = calls as [[string, RequestInit]]
    expect(bodyOf(call)).toEqual({ version: 4, locationId: 'l2' })
    expect(headerOf(call, 'Idempotency-Key')).toMatch(/^[0-9a-f-]{36}$/)
    expect(headerOf(call, 'X-CSRF-Token')).toBe('test-csrf-token')
    release?.()
    expect(await screen.findByText(/GRN-000001 posted\./)).toBeInTheDocument()
    const units = screen.getByRole('list', { name: 'Vehicle units registered by this receipt' })
    expect(within(units).getAllByText('Awaiting PDI')).toHaveLength(2)
    expect(within(units).queryByText('Available')).not.toBeInTheDocument()
    expect(within(units).getByRole('link', { name: 'MAE-VEH-000001' })).toHaveAttribute(
      'href',
      '/procurement/pdi/v1',
    )
    expect(screen.queryByRole('button', { name: 'Post receipt' })).not.toBeInTheDocument()
    const keys = invalidate.mock.calls.map(([f]) => JSON.stringify(f?.queryKey))
    for (const k of [
      '["purchase-orders"]',
      '["vehicle-units"]',
      '["pdi"]',
      '["goods-receipts","list"]',
    ])
      expect(keys).toContain(k)
  })

  it('explains POSTING_BLOCKED and key reuse, and uses a new key after a refusal', async () => {
    const user = userEvent.setup()
    let answer = err(409, 'POSTING_BLOCKED', [
      { field: 'lines[0].units', message: 'capture every received unit' },
    ])
    const spy = api({ detail: draft }, (url, init) =>
      init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post'
        ? answer
        : undefined,
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, [locs[0]!]) })
    const dialog = await openDialog(user)
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(
      await within(dialog).findByText(/cannot be posted yet\. Nothing was changed/),
    ).toBeInTheDocument()
    expect(within(dialog).getByText('Line 1: units')).toBeInTheDocument()
    answer = err(409, 'IDEMPOTENCY_KEY_REUSED')
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(
      await within(dialog).findByText(/could not be matched to the earlier attempt/),
    ).toBeInTheDocument()
    const [first, second] = posts(spy) as [[string, RequestInit], [string, RequestInit]]
    expect(headerOf(first, 'Idempotency-Key')).not.toBe(headerOf(second, 'Idempotency-Key'))
  })

  it('reloads on a version conflict and asks for a fresh confirmation', async () => {
    const user = userEvent.setup()
    const state = { detail: draft }
    const spy = api(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post') {
        state.detail = { ...draft, version: 5 }
        return err(409, 'VERSION_CONFLICT')
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, [locs[0]!]) })
    const dialog = await openDialog(user)
    const before = detailGets(spy)
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(await within(dialog).findByText(/changed by someone else/)).toBeInTheDocument()
    await waitFor(() => {
      expect(detailGets(spy)).toBeGreaterThan(before)
    })
    expect(
      await within(dialog).findByText(/changed since you opened this dialog/),
    ).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    await waitFor(() => {
      expect(posts(spy)).toHaveLength(2)
    })
    expect(bodyOf(posts(spy)[1])).toEqual({ version: 5, locationId: 'l1' })
  })

  it('handles 403 without claiming anything', async () => {
    const user = userEvent.setup()
    api({ detail: draft }, (url, init) =>
      init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post'
        ? err(403, 'FORBIDDEN')
        : undefined,
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, [locs[0]!]) })
    const dialog = await openDialog(user)
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(await within(dialog).findByText(/do not have permission to post/)).toBeInTheDocument()
    expect(screen.queryByText(/GRN-000001 posted\./)).not.toBeInTheDocument()
  })

  it('after an unknown outcome re-reads the receipt and retries with the same key', async () => {
    const user = userEvent.setup()
    let fail = true
    const state = { detail: draft }
    const spy = api(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post') {
        if (fail) {
          fail = false
          return new TypeError('Failed to fetch')
        }
        return Response.json({ data: result() })
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, [locs[0]!]) })
    const dialog = await openDialog(user)
    const before = detailGets(spy)
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(await within(dialog).findByText(/may or may not have been posted/)).toBeInTheDocument()
    await waitFor(() => {
      expect(detailGets(spy)).toBeGreaterThan(before)
    })
    await user.click(within(dialog).getByRole('button', { name: 'Send the same request again' }))
    expect(await screen.findByText(/GRN-000001 posted\./)).toBeInTheDocument()
    const [a, b] = posts(spy) as [[string, RequestInit], [string, RequestInit]]
    expect(headerOf(a, 'Idempotency-Key')).toBe(headerOf(b, 'Idempotency-Key'))
    expect(a[1].body).toBe(b[1].body)
  })

  it('when the re-read shows the receipt posted, fetches the result with the same key', async () => {
    const user = userEvent.setup()
    let fail = true
    const state = { detail: draft }
    const spy = api(state, (url, init) => {
      if (init?.method === 'POST' && url.pathname === '/api/v1/goods-receipts/g1/post') {
        if (fail) {
          fail = false
          state.detail = postedDetail() // it was posted; the answer was lost
          return new TypeError('Failed to fetch')
        }
        return Response.json({ data: result() }, { headers: { 'Idempotent-Replayed': 'true' } })
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, [locs[0]!]) })
    const dialog = await openDialog(user)
    await user.click(within(dialog).getByRole('button', { name: 'Post receipt' }))
    expect(await within(dialog).findByText(/This receipt is now posted/)).toBeInTheDocument()
    expect(
      within(dialog).queryByRole('button', { name: /Post receipt|Send the same/ }),
    ).not.toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Show the posting result' }))
    expect(await screen.findByText(/GRN-000001 posted\./)).toBeInTheDocument()
    const [a, b] = posts(spy) as [[string, RequestInit], [string, RequestInit]]
    expect(headerOf(a, 'Idempotency-Key')).toBe(headerOf(b, 'Idempotency-Key'))
  })

  it('blocks posting while the editor has unsaved changes and without a location', async () => {
    const user = userEvent.setup()
    api({ detail: draft })
    renderApp('/procurement/goods-receipts/g1', { me: me(POSTER, []) })
    const dialog = await openDialog(user)
    expect(within(dialog).getByText(/No location is assigned to you/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Post receipt' })).toBeDisabled()
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await user.type(screen.getByLabelText(/Notes on the delivery/), 'x')
    expect(
      await screen.findByText(/You have unsaved changes\. Save the draft before posting/),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Post receipt' })).toBeDisabled()
  })
})

describe('purchase order history (D-07)', () => {
  it('knows GOODS_RECEIVED and keeps the earlier actions', () => {
    for (const action of [
      'CREATED',
      'SUBMITTED',
      'APPROVED',
      'APPROVAL_NOT_REQUIRED',
      'RETURNED',
      'CANCELLED',
      'GOODS_RECEIVED',
    ]) {
      const e = poEventSchema.parse({
        id: 'e',
        action,
        fromStatus: 'APPROVED',
        toStatus: 'PARTIALLY_RECEIVED',
        actorId: 'u',
        actorName: 'U',
        note: 'GRN-000001',
        createdAt: t,
      })
      expect(poEventLabels[e.action]).toBeTruthy()
    }
    expect(poEventLabels.GOODS_RECEIVED).toBe('Goods received')
  })
})
