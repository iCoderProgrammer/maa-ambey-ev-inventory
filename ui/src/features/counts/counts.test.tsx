/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { ApiError } from '@/lib/api/client'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/stockCounts'
import type { CountLine, StockCount } from './api/stockCounts'
import { describeCountError } from './lib/countErrors'

// Placeholder data only ("SC-000001", "PRT-1", "PCS"): nothing real. API answers
// are mocked; posting, separation of duties, staleness and idempotency under
// concurrency are proven by the Go/PostgreSQL tests.

const t = '2026-10-27T09:00:00Z'
const ME = testMe.user.id
const loc = { id: 'l1', code: 'SR1', name: 'Showroom' }
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const VIEW = ['stock.adjustment.view']
const COUNTER = [...VIEW, 'stock.adjustment.create', 'stock_items.view', 'vehicle_units.view']
const REVIEWER = [...VIEW, 'stock.adjustment.approve']

const partLine = (over: Partial<CountLine> = {}): CountLine => ({
  id: 'ln1',
  itemType: 'STOCK_ITEM',
  stockItemId: 'i1',
  sku: 'PRT-1',
  itemName: 'Brake pad set',
  uom: 'PCS',
  vehicleUnitId: null,
  unitNumber: null,
  expectedStatus: null,
  found: null,
  expectedQuantity: '10.000',
  observedQuantity: '8.000',
  variance: '-2.000',
  reason: 'Two damaged',
  stockChanged: false,
  outcome: 'PENDING',
  postedMovementId: null,
  recordedBy: 'u-counter',
  recordedByName: 'Counter',
  recordedAt: t,
  ...over,
})
const vehicleLine = (over: Partial<CountLine> = {}): CountLine => ({
  ...partLine(),
  id: 'ln2',
  itemType: 'VEHICLE_UNIT',
  stockItemId: null,
  sku: null,
  itemName: null,
  uom: null,
  vehicleUnitId: 'v1',
  unitNumber: 'MAE-VEH-000001',
  expectedStatus: 'AVAILABLE',
  found: false,
  expectedQuantity: '1.000',
  observedQuantity: '0.000',
  variance: '-1.000',
  reason: 'Not on the floor',
  ...over,
})
const count = (over: Partial<StockCount> = {}): StockCount => ({
  id: 'c1',
  countNumber: 'SC-000001',
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom',
  status: 'SUBMITTED',
  approvalPolicy: 'APPROVER_REQUIRED',
  note: null,
  createdBy: 'u-counter',
  createdByName: 'Counter',
  createdAt: t,
  submittedBy: 'u-counter',
  submittedByName: 'Counter',
  submittedAt: t,
  decidedBy: null,
  decidedByName: null,
  decidedAt: null,
  decisionNote: null,
  lineCount: 1,
  varianceLines: 1,
  version: 3,
  updatedAt: t,
  lines: [partLine()],
  ...over,
})
const approved = (c: StockCount): StockCount => ({
  ...c,
  status: 'APPROVED',
  decidedBy: ME,
  decidedByName: 'Test User',
  decidedAt: t,
  version: c.version + 1,
  lines: c.lines.map((l) =>
    l.itemType === 'STOCK_ITEM' && l.variance !== '0.000'
      ? { ...l, outcome: 'POSTED', postedMovementId: 'm-' + l.id }
      : { ...l, outcome: l.variance === '0.000' ? 'NO_CHANGE' : 'DISCREPANCY_RECORDED' },
  ),
})
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json(
    { error: { code, message: 'Refused by the server.', requestId: 'req-5', details } },
    { status },
  )
const sku = {
  id: 'i1',
  sku: 'PRT-1',
  skuNormalised: 'PRT-1',
  name: 'Brake pad set',
  description: null,
  categoryId: 'cat',
  categoryName: 'Brakes',
  categoryStatus: 'ACTIVE',
  uom: 'PCS',
  trackingMode: 'QUANTITY',
  reorderLevel: null,
  status: 'ACTIVE',
  archivedBy: null,
  archivedAt: null,
  archiveReason: null,
  createdBy: 'u-x',
  version: 1,
  createdAt: t,
  updatedAt: t,
}

type Handler = (url: URL, init: RequestInit | undefined) => Response | Error | undefined

/** A fake API: GETs answer from `state.count`; handler overrides first. */
function apiMock(
  state: { count: StockCount; list?: StockCount[] },
  handler: Handler = () => undefined,
) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom instanceof Error) return Promise.reject(custom)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/stock-counts') {
      const items = state.list ?? [state.count]
      const s = url.searchParams.get('status')
      return json(page(items.filter((c) => !s || c.status === s).map((c) => ({ ...c, lines: [] }))))
    }
    if (p === `/api/v1/stock-counts/${state.count.id}`) return json(state.count)
    if (p === '/api/v1/stock-items') return json(page([sku]))
    if (p === '/api/v1/vehicle-units') return json(page([]))
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
const bodyOf = (init: RequestInit | undefined): unknown =>
  typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
const posts = (spy: Spy, suffix: string) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      init?.method === 'POST' &&
      new URL(u as string, 'http://x').pathname === `/api/v1/stock-counts${suffix}`,
  ) as [string, RequestInit][]
const detailGets = (spy: Spy) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      (init?.method ?? 'GET') === 'GET' &&
      new URL(u as string, 'http://x').pathname === '/api/v1/stock-counts/c1',
  ).length
const keyOf = (c: [string, RequestInit]) => new Headers(c[1].headers).get('Idempotency-Key')

afterEach(() => {
  vi.restoreAllMocks()
})

describe('stock count API client', () => {
  it('matches the server contract field for field (docs/openapi)', () => {
    const spec = JSON.parse(
      readFileSync(
        path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json'),
        'utf8',
      ),
    ) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<
        string,
        Record<string, { parameters?: { name: string; in: string; required?: boolean }[] }>
      >
    }
    const keys = (n: string) => Object.keys(spec.definitions[n]?.properties ?? {}).sort()
    expect(Object.keys(count()).sort()).toEqual(keys('StockCount'))
    expect(Object.keys(partLine()).sort()).toEqual(keys('StockCountLine'))
    expect(keys('StockCountCreateInput')).toEqual(['locationId', 'note'])
    expect(keys('StockCountLineInput')).toEqual([
      'found',
      'observedQuantity',
      'reason',
      'stockItemId',
      'uom',
      'vehicleUnitId',
      'version',
    ])
    expect(keys('StockCountVersionInput')).toEqual(['version'])
    expect(keys('StockCountApproveInput')).toEqual(['note', 'version'])
    expect(keys('StockCountRejectInput')).toEqual(['reason', 'version'])
    expect(spec.definitions.StockCount?.properties.status?.enum).toEqual(
      api.countStatusSchema.options,
    )
    expect(spec.definitions.StockCountLine?.properties.outcome?.enum).toEqual(
      api.lineOutcomeSchema.options,
    )
    const key: unknown = expect.objectContaining({
      name: 'Idempotency-Key',
      in: 'header',
      required: true,
    })
    expect(spec.paths['/stock-counts']?.post?.parameters).toContainEqual(key)
    for (const op of ['approve', 'reject'])
      expect(spec.paths[`/stock-counts/{id}/${op}`]?.post?.parameters).toContainEqual(key)
    for (const op of ['lines', 'submit'])
      expect(
        spec.paths[`/stock-counts/{id}/${op}`]?.post?.parameters?.some(
          (p) => p.name === 'Idempotency-Key',
        ),
      ).toBe(false)
    expect(Object.keys(spec.paths['/stock-counts/{id}'] ?? {})).toEqual(['get'])
  })

  it('sends list filters, freezes actions and refuses an answer in the wrong state', async () => {
    const spy = apiMock({ count: count() })
    await api.listStockCounts({ status: 'SUBMITTED', locationId: 'l1', page: 2, limit: 10 })
    const url = new URL(spy.mock.calls[0]?.[0] as string, 'http://x')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      status: 'SUBMITTED',
      locationId: 'l1',
      page: '2',
      limit: '10',
    })
    const a = api.newCountAction({ kind: 'approve', countId: 'c1', body: { version: 3 } })
    expect(Object.isFrozen(a) && Object.isFrozen(a.body)).toBe(true)
    expect(api.sameAction(a, { kind: 'approve', countId: 'c1', body: { version: 3 } })).toBe(true)
    expect(api.sameAction(a, { kind: 'approve', countId: 'c1', body: { version: 4 } })).toBe(false)
    // The server answered with the count still SUBMITTED: not an approval.
    vi.restoreAllMocks()
    apiMock({ count: count() }, (u) =>
      u.pathname.endsWith('/approve') ? Response.json({ data: count() }) : undefined,
    )
    await expect(api.sendCountAction(a)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    // A vehicle line can never claim a posted movement.
    expect(() =>
      api.countLineSchema.parse(vehicleLine({ outcome: 'POSTED', postedMovementId: 'm1' })),
    ).toThrow()
  })

  it('maps errors without calling unknown outcomes failures', () => {
    const e = (status: number, code: string, details: { field: string; message: string }[] = []) =>
      new ApiError(status, { code, message: 'x', details })
    const stale = describeCountError(
      e(409, 'STOCK_CHANGED_SINCE_COUNT', [{ field: 'lines.ln1', message: 'x' }]),
      true,
    )
    expect(stale).toMatchObject({ kind: 'stale', staleLineIds: ['ln1'], retrySame: false })
    expect(describeCountError(e(403, 'SELF_APPROVAL_NOT_ALLOWED'), true).kind).toBe('self')
    expect(describeCountError(e(503, 'SERVICE_UNAVAILABLE'), true)).toMatchObject({
      kind: 'uncertain',
      retrySame: true,
    })
    expect(describeCountError(e(0, 'NETWORK_ERROR'), false)).toMatchObject({
      kind: 'uncertain',
      retrySame: false,
    })
    expect(describeCountError(e(409, 'VERSION_CONFLICT'), false).kind).toBe('version')
  })
})

describe('stock count lists', () => {
  it('lists counts from the API, filters by status and shows empty and error states', async () => {
    const user = userEvent.setup()
    const spy = apiMock({ count: count() })
    renderApp('/inventory/counts', { me: me(VIEW) })
    const table = await screen.findByRole('table', { name: 'Stock counts' })
    expect(within(table).getByText('SC-000001')).toBeInTheDocument()
    expect(within(table).getByText('Awaiting approval')).toBeInTheDocument()
    expect(screen.getByText(/No approval threshold is configured/)).toBeInTheDocument()
    // Only approvers see how many lines differ.
    expect(within(table).queryByText(/differ/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'New count' })).not.toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Status'), 'APPROVED')
    await waitFor(() => {
      expect(
        spy.mock.calls.some(
          ([u]) => new URL(u as string, 'http://x').searchParams.get('status') === 'APPROVED',
        ),
      ).toBe(true)
    })
    expect(await screen.findByText('No counts match these filters')).toBeInTheDocument()
    vi.restoreAllMocks()
    apiMock({ count: count(), list: [] })
    renderApp('/inventory/counts', { me: me(VIEW) })
    expect(await screen.findByText('No stock counts yet')).toBeInTheDocument()
    vi.restoreAllMocks()
    apiMock({ count: count() }, (u) =>
      u.pathname === '/api/v1/stock-counts' ? err(500, 'INTERNAL_ERROR') : undefined,
    )
    renderApp('/inventory/counts', { me: me(VIEW) })
    expect(await screen.findByText('Could not load stock counts')).toBeInTheDocument()
  })

  it('denies the pages without stock.adjustment.view and never calls the API', async () => {
    const spy = apiMock({ count: count() })
    renderApp('/inventory/counts', { me: me(['stock_items.view', 'stock.adjustment.create']) })
    expect(await screen.findByText(/Access denied/)).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/stock-counts'))).toBe(false)
  })

  it('shows submitted counts in the approval queue, flagging ones the user took part in', async () => {
    const mine = count({ id: 'c2', countNumber: 'SC-000002', createdBy: ME, submittedBy: ME })
    const spy = apiMock({
      count: count(),
      list: [count(), mine, count({ id: 'c3', status: 'DRAFT' })],
    })
    renderApp('/inventory/counts/approvals', { me: me(REVIEWER) })
    const table = await screen.findByRole('table', { name: 'Counts awaiting approval' })
    expect(within(table).getByText('SC-000001')).toBeInTheDocument()
    expect(within(table).getByText('SC-000002')).toBeInTheDocument()
    expect(within(table).getAllByText('You took part — someone else decides')).toHaveLength(1)
    expect(
      spy.mock.calls.some(
        ([u]) => new URL(u as string, 'http://x').searchParams.get('status') === 'SUBMITTED',
      ),
    ).toBe(true)
  })

  it('creates a count with one key and opens it only after the server confirms', async () => {
    const user = userEvent.setup()
    const draft = count({
      status: 'DRAFT',
      submittedBy: null,
      submittedByName: null,
      submittedAt: null,
      lines: [],
      lineCount: 0,
      varianceLines: 0,
      version: 1,
      createdBy: ME,
    })
    const spy = apiMock({ count: draft, list: [] }, (u, init) =>
      u.pathname === '/api/v1/stock-counts' && init?.method === 'POST'
        ? Response.json({ data: draft }, { status: 201 })
        : undefined,
    )
    renderApp('/inventory/counts', { me: me(COUNTER) })
    await user.click(await screen.findByRole('button', { name: 'New count' }))
    const dialog = await screen.findByRole('dialog', { name: 'Start a stock count' })
    await user.type(within(dialog).getByLabelText('Note'), 'Shelf A')
    await user.click(within(dialog).getByRole('button', { name: 'Start count' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'SC-000001' })).toBeInTheDocument()
    const c = posts(spy, '')
    expect(c).toHaveLength(1)
    expect(bodyOf(c[0]?.[1])).toEqual({ locationId: 'l1', note: 'Shelf A' })
    expect(keyOf(c[0] as [string, RequestInit])).toMatch(/^[A-Za-z0-9-]{8,}$/)
  })
})

describe('recording and submitting', () => {
  const draft = () =>
    count({
      status: 'DRAFT',
      submittedBy: null,
      submittedByName: null,
      submittedAt: null,
      version: 2,
      createdBy: ME,
      lines: [partLine({ recordedBy: ME })],
    })

  it('records a part with the supported schema and maps the server reason error to the field', async () => {
    const user = userEvent.setup()
    const state = { count: draft() }
    let n = 0
    const spy = apiMock(state, (u, init) => {
      if (u.pathname === '/api/v1/stock-counts/c1/lines' && init?.method === 'POST') {
        n++
        if (n === 1)
          return err(422, 'VALIDATION_FAILED', [{ field: 'reason', message: 'is required' }])
        state.count = { ...state.count, version: 3 }
        return Response.json({ data: state.count })
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(COUNTER) })
    await user.click(await screen.findByRole('button', { name: 'Record item' }))
    const dialog = await screen.findByRole('dialog', { name: 'Record a counted item' })
    await user.click(within(dialog).getByRole('button', { name: 'Save count' }))
    expect(
      await within(dialog).findByText('Choose the part', { selector: 'span' }),
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(within(dialog).getByRole('option', { name: /PRT-1/ })).toBeInTheDocument()
    })
    await user.selectOptions(within(dialog).getByLabelText(/^Part/), 'i1')
    await user.type(within(dialog).getByLabelText(/Quantity counted/), '7')
    await user.click(within(dialog).getByRole('button', { name: 'Save count' }))
    expect(
      await within(dialog).findByText('Give a reason: this count differs from the stock record'),
    ).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/Quantity counted/)).toHaveValue('7')
    await user.type(within(dialog).getByLabelText('Reason'), 'Three missing')
    await user.click(within(dialog).getByRole('button', { name: 'Save count' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    const c = posts(spy, '/c1/lines')
    expect(bodyOf(c[1]?.[1])).toEqual({
      version: 2,
      stockItemId: 'i1',
      observedQuantity: '7',
      uom: 'PCS',
      reason: 'Three missing',
    })
    expect(keyOf(c[1] as [string, RequestInit])).toBeNull()
  })

  it('hides expected quantities from a counter and offers no cancel, delete or line removal', async () => {
    apiMock({ count: draft() })
    renderApp('/inventory/counts/c1', { me: me(COUNTER) })
    const table = await screen.findByRole('table', { name: 'Counted items' })
    expect(within(table).getByText('8 PCS')).toBeInTheDocument()
    expect(within(table).queryByText('10 PCS')).not.toBeInTheDocument()
    expect(within(table).queryByText(/Difference|Stock record/)).not.toBeInTheDocument()
    expect(screen.getByText(/shown to approvers only/)).toBeInTheDocument()
    expect(
      screen.getAllByRole('button', { name: 'Record PRT-1 — Brake pad set again' }),
    ).not.toHaveLength(0)
    expect(
      screen.queryByRole('button', { name: /Cancel count|Delete|Remove/ }),
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Approve|Reject/ })).not.toBeInTheDocument()
  })

  it('names the stale lines when submit is refused and refetches instead of submitting', async () => {
    const user = userEvent.setup()
    const state = { count: draft() }
    const spy = apiMock(state, (u, init) => {
      if (u.pathname === '/api/v1/stock-counts/c1/submit' && init?.method === 'POST') {
        state.count = { ...state.count, lines: [partLine({ recordedBy: ME, stockChanged: true })] }
        return err(409, 'STOCK_CHANGED_SINCE_COUNT', [
          { field: 'lines.ln1', message: 'stock changed after this line was counted' },
        ])
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(COUNTER) })
    await user.click(await screen.findByRole('button', { name: 'Submit for approval' }))
    const dialog = await screen.findByRole('dialog')
    const before = detailGets(spy)
    await user.click(within(dialog).getByRole('button', { name: 'Submit for approval' }))
    const alert = await within(dialog).findByRole('alert')
    expect(alert).toHaveTextContent('Stock moved after some lines were counted')
    expect(alert).toHaveTextContent('PRT-1 — Brake pad set')
    expect(alert).toHaveTextContent('Count those items again')
    await waitFor(() => {
      expect(detailGets(spy)).toBeGreaterThan(before)
    })
    expect(posts(spy, '/c1/submit')).toHaveLength(1)
    expect(await screen.findAllByText('Stock moved since counted')).not.toHaveLength(0)
    expect(screen.getAllByText('Draft').length).toBeGreaterThan(0)
  })
})

describe('approving and rejecting', () => {
  it('shows the comparison to a reviewer and confirms the server result after approval', async () => {
    const user = userEvent.setup()
    const state = {
      count: count({ lines: [partLine(), vehicleLine()], lineCount: 2, varianceLines: 2 }),
    }
    const spy = apiMock(state, (u, init) => {
      if (u.pathname === '/api/v1/stock-counts/c1/approve' && init?.method === 'POST') {
        state.count = approved(state.count)
        return Response.json({ data: state.count })
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    const table = await screen.findByRole('table', { name: 'Counted items' })
    expect(within(table).getByText('10 PCS')).toBeInTheDocument()
    expect(within(table).getByText('−2 PCS')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog', { name: 'Approve SC-000001?' })
    expect(within(dialog).getByText(/1 parts adjustment will be posted/)).toBeInTheDocument()
    expect(
      within(dialog).getByText(/1 vehicle finding is recorded only: no vehicle is changed/),
    ).toBeInTheDocument()
    // Double click: one request.
    await user.dblClick(within(dialog).getByRole('button', { name: 'Approve and post' }))
    expect(
      await screen.findByText(/SC-000001 approved\. 1 adjustment was posted/),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/1 vehicle finding recorded \(no vehicle changed\)/),
    ).toBeInTheDocument()
    expect(posts(spy, '/c1/approve')).toHaveLength(1)
    expect(bodyOf(posts(spy, '/c1/approve')[0]?.[1])).toEqual({ version: 3 })
    // 16: the vehicle line is a finding, the parts line a posted adjustment.
    expect(await screen.findAllByText('Finding recorded — vehicle not changed')).not.toHaveLength(0)
    expect(screen.getAllByText('Adjustment posted')).not.toHaveLength(0)
    expect(screen.queryByRole('button', { name: /^Approve$|^Reject$/ })).not.toBeInTheDocument()
  })

  it('requires a reason to reject and shows the rejection from the server', async () => {
    const user = userEvent.setup()
    const state = { count: count() }
    const spy = apiMock(state, (u, init) => {
      if (u.pathname === '/api/v1/stock-counts/c1/reject' && init?.method === 'POST') {
        state.count = {
          ...state.count,
          status: 'REJECTED',
          decidedBy: ME,
          decidedAt: t,
          decisionNote: 'Recount',
          version: 4,
          lines: state.count.lines.map((l) => ({ ...l, outcome: 'NOT_POSTED' as const })),
        }
        return Response.json({ data: state.count })
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    await user.click(await screen.findByRole('button', { name: 'Reject' }))
    const dialog = await screen.findByRole('dialog', { name: 'Reject SC-000001?' })
    await user.click(within(dialog).getByRole('button', { name: 'Reject count' }))
    expect(await within(dialog).findByText('Give the reason for rejecting')).toBeInTheDocument()
    expect(posts(spy, '/c1/reject')).toHaveLength(0)
    await user.type(within(dialog).getByLabelText(/Reason/), 'Recount')
    await user.click(within(dialog).getByRole('button', { name: 'Reject count' }))
    expect(await screen.findByText(/SC-000001 rejected\. Nothing was posted/)).toBeInTheDocument()
    expect(screen.getAllByText('Not posted (rejected)')).not.toHaveLength(0)
    expect(bodyOf(posts(spy, '/c1/reject')[0]?.[1])).toEqual({ version: 3, reason: 'Recount' })
  })

  it('does not offer a decision to a participant, and handles the server 403 for unknown ones', async () => {
    const user = userEvent.setup()
    apiMock({ count: count({ lines: [partLine({ recordedBy: ME })] }) })
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    expect(await screen.findByText(/You took part in this count/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Approve$|^Reject$/ })).not.toBeInTheDocument()
    vi.restoreAllMocks()
    // An earlier recorder is known only to the server: 403 is authoritative.
    const spy = apiMock({ count: count() }, (u, init) =>
      u.pathname.endsWith('/approve') && init?.method === 'POST'
        ? err(403, 'SELF_APPROVAL_NOT_ALLOWED')
        : undefined,
    )
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    await user.click(await screen.findByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog')
    const before = detailGets(spy)
    await user.click(within(dialog).getByRole('button', { name: 'Approve and post' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'You took part in this count',
    )
    await waitFor(() => {
      expect(detailGets(spy)).toBeGreaterThan(before)
    })
    expect(screen.queryByText(/approved\./)).not.toBeInTheDocument()
  })

  it('does not mark the count approved when stock changed, and names the lines', async () => {
    const user = userEvent.setup()
    const state = { count: count() }
    apiMock(state, (u, init) => {
      if (u.pathname.endsWith('/approve') && init?.method === 'POST') {
        state.count = { ...state.count, lines: [partLine({ stockChanged: true })] }
        return err(409, 'STOCK_CHANGED_SINCE_COUNT', [{ field: 'lines.ln1', message: 'x' }])
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    await user.click(await screen.findByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Approve and post' }))
    const alert = await within(dialog).findByRole('alert')
    expect(alert).toHaveTextContent('PRT-1 — Brake pad set')
    expect(alert).toHaveTextContent('Reject it so the items can be counted again')
    expect(await screen.findAllByText('Stock moved since counted')).not.toHaveLength(0)
    expect(screen.queryByText(/approved\./)).not.toBeInTheDocument()
    expect(screen.getAllByText('Awaiting approval').length).toBeGreaterThan(0)
  })

  it('retries an unknown outcome with the same key and body, and a change is a new key', async () => {
    const user = userEvent.setup()
    const state = { count: count() }
    let n = 0
    const spy = apiMock(state, (u, init) => {
      if (u.pathname.endsWith('/approve') && init?.method === 'POST') {
        n++
        if (n < 3) return new TypeError('Failed to fetch')
        state.count = approved(state.count)
        return Response.json({ data: state.count })
      }
      return undefined
    })
    renderApp('/inventory/counts/c1', { me: me(REVIEWER) })
    await user.click(await screen.findByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Approve and post' }))
    expect(await within(dialog).findByText(/may or may not have been done/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Send again unchanged' }))
    await within(dialog).findByText(/may or may not have been done/)
    // Changing the note is a different request: new key.
    await user.type(within(dialog).getByLabelText(/Note/), 'ok')
    await user.click(within(dialog).getByRole('button', { name: 'Send again unchanged' }))
    expect(await screen.findByText(/SC-000001 approved/)).toBeInTheDocument()
    const c = posts(spy, '/c1/approve')
    expect(c).toHaveLength(3)
    expect(keyOf(c[0] as [string, RequestInit])).toBe(keyOf(c[1] as [string, RequestInit]))
    expect(bodyOf(c[0]?.[1])).toEqual(bodyOf(c[1]?.[1]))
    expect(keyOf(c[2] as [string, RequestInit])).not.toBe(keyOf(c[0] as [string, RequestInit]))
    expect(bodyOf(c[2]?.[1])).toEqual({ version: 3, note: 'ok' })
  })

  it('reloads the count from the server on a fresh visit (no local state)', async () => {
    const spy = apiMock({ count: approved(count()) })
    renderApp('/inventory/counts/c1', { me: me(VIEW) })
    expect(await screen.findAllByText('Adjustment posted')).not.toHaveLength(0)
    expect(detailGets(spy)).toBe(1)
    expect(screen.queryByText('10 PCS')).not.toBeInTheDocument()
  })
})
