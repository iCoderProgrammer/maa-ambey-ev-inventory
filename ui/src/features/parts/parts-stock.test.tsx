/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/partsStock'
import type { PartsBalance, PartsMovement, PartsResult } from './api/partsStock'
import type { StockItemDetail } from './api/stockItems'
import { describePartsError } from './lib/stockErrors'
import { ApiError } from '@/lib/api/client'

// Placeholder data only ("PRT-1", "PCS", "Example Supplier"): nothing real. API
// answers are mocked; the database guarantees (no negative stock, one movement
// per post, idempotency under concurrency) are the Go/PostgreSQL tests.

const t = '2026-10-27T09:00:00Z'
const item: StockItemDetail = {
  id: 'i1',
  sku: 'PRT-1',
  skuNormalised: 'PRT-1',
  name: 'Brake pad set',
  description: null,
  categoryId: 'c1',
  categoryName: 'Brake parts',
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
  suppliers: [],
  suppliersHidden: false,
  compatibility: [],
}
const balance = (quantity: string, over: Partial<PartsBalance> = {}): PartsBalance => ({
  stockItemId: 'i1',
  sku: 'PRT-1',
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom',
  uom: 'PCS',
  quantity,
  version: 2,
  updatedAt: t,
  ...over,
})
const movement = (over: Partial<PartsMovement> = {}): PartsMovement => ({
  id: 'm1',
  stockItemId: 'i1',
  sku: 'PRT-1',
  movementType: 'PARTS_RECEIPT',
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom',
  uom: 'PCS',
  quantity: '5.000',
  quantityBefore: '0.000',
  quantityAfter: '5.000',
  sourceType: 'SUPPLIER_DELIVERY',
  supplierId: 's1',
  supplierName: 'Example Supplier One',
  sourceReference: 'DC-1',
  reason: null,
  note: null,
  actorId: 'u1',
  actorName: 'Asha',
  createdAt: t,
  ...over,
})
const issued = movement({
  id: 'm2',
  movementType: 'PARTS_ISSUE',
  quantity: '2.000',
  quantityBefore: '5.000',
  quantityAfter: '3.000',
  sourceType: 'STAFF_ACTION',
  supplierId: null,
  supplierName: null,
  sourceReference: 'JOB-1',
  reason: 'Fitted in a repair',
})
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json(
    { error: { code, message: 'Refused by the server.', requestId: 'req-4', details } },
    { status },
  )
const loc = { id: 'l1', code: 'SR1', name: 'Showroom' }
const me = (permissions: string[], locations = [loc]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations },
})
const supplier = (id: string, legalName: string, status: string) => ({
  id,
  legalName,
  tradeName: null,
  gstin: null,
  addressLine1: null,
  addressLine2: null,
  city: null,
  state: null,
  postalCode: null,
  status,
  verificationReference: null,
  verificationDocumentId: null,
  verifiedBy: null,
  verifiedAt: null,
  notes: null,
  createdBy: 'u-x',
  brandCount: 0,
  version: 1,
  createdAt: t,
  updatedAt: t,
})

type Handler = (url: URL, init: RequestInit | undefined) => Response | Error | undefined

function apiMock(
  handler: Handler = () => undefined,
  opts: { item?: StockItemDetail; balances?: PartsBalance[] } = {},
) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom instanceof Error) return Promise.reject(custom)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/stock-items/i1') return json(opts.item ?? item)
    if (p === '/api/v1/stock-items/i1/balances') return json(opts.balances ?? [balance('5.000')])
    if (p === '/api/v1/stock-items/i1/stock-movements') return json(page([issued, movement()]))
    if (p === '/api/v1/suppliers')
      return json(
        page([
          supplier('s1', 'Example Supplier One', 'PENDING_VERIFICATION'),
          supplier('s3', 'Example Supplier Old', 'ARCHIVED'),
        ]),
      )
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
const posts = (spy: Spy, op: string) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      init?.method === 'POST' &&
      new URL(u as string, 'http://x').pathname === `/api/v1/stock-items/i1/${op}`,
  ) as [string, RequestInit][]
const gets = (spy: Spy, suffix: string) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      (init?.method ?? 'GET') === 'GET' &&
      new URL(u as string, 'http://x').pathname === `/api/v1/stock-items/i1${suffix}`,
  ).length
const keyOf = (c: [string, RequestInit]) => new Headers(c[1].headers).get('Idempotency-Key')

const VIEW = ['stock_items.view', 'stock.parts.view']
const ALL = [...VIEW, 'stock.parts.receive', 'stock.parts.issue', 'suppliers.view']
const result = (m: PartsMovement, b: PartsBalance): PartsResult => ({ movement: m, balance: b })

afterEach(() => {
  vi.restoreAllMocks()
})

describe('parts stock API client', () => {
  it('reads balances and history with the supported filters', async () => {
    const spy = apiMock()
    const b = await api.getPartsBalances('i1')
    expect(b[0]?.quantity).toBe('5.000')
    await api.listPartsMovements('i1', {
      locationId: 'l1',
      type: 'PARTS_ISSUE',
      page: 2,
      limit: 10,
    })
    const url = new URL(spy.mock.calls[1]?.[0] as string, 'http://x')
    expect(url.pathname).toBe('/api/v1/stock-items/i1/stock-movements')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      locationId: 'l1',
      type: 'PARTS_ISSUE',
      page: '2',
      limit: '10',
    })
  })

  it('posts receipts and issues with the frozen key and body, and checks the answer', async () => {
    const r = api.newPartsAction('receipt', 'i1', {
      locationId: 'l1',
      quantity: '5',
      uom: 'PCS',
      supplierId: 's1',
      sourceReference: 'DC-1',
    })
    expect(Object.isFrozen(r) && Object.isFrozen(r.body)).toBe(true)
    const spy = apiMock(() => Response.json({ data: result(movement(), balance('5.000')) }))
    await api.sendPartsAction(r)
    const c = posts(spy, 'receipts')[0] as [string, RequestInit]
    expect(keyOf(c)).toBe(r.idempotencyKey)
    expect(bodyOf(c[1])).toEqual({
      locationId: 'l1',
      quantity: '5',
      uom: 'PCS',
      supplierId: 's1',
      sourceReference: 'DC-1',
    })
    // An answer that is not this request's movement is refused.
    const i = api.newPartsAction('issue', 'i1', {
      locationId: 'l1',
      quantity: '1',
      uom: 'PCS',
      reason: 'x',
    })
    await expect(api.sendPartsAction(i)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    vi.restoreAllMocks()
    // A balance of another location, or a receipt that takes away, is refused too.
    apiMock(() =>
      Response.json({ data: result(movement(), balance('5.000', { locationId: 'l2' })) }),
    )
    await expect(api.sendPartsAction(r)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    vi.restoreAllMocks()
    apiMock(() =>
      Response.json({ data: result(movement({ quantityAfter: '0.000' }), balance('0.000')) }),
    )
    await expect(api.sendPartsAction(r)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })

  it('checks quantities as text, without floating point', () => {
    for (const ok of ['5', '0.001', '2.5', '999999999.999'])
      expect(api.isValidQuantity(ok)).toBe(true)
    for (const bad of ['0', '0.000', '-1', '1.2345', '1e3', '', ' 1', '1,5', '1234567890'])
      expect(api.isValidQuantity(bad)).toBe(false)
    expect(api.thousandths('2.5')).toBe(2500n)
    expect(api.thousandths('0.1') + api.thousandths('0.2')).toBe(api.thousandths('0.3'))
    expect(api.formatQuantity('15.000')).toBe('15')
    expect(api.sameBody({ a: '1', b: undefined }, { a: '1' })).toBe(true)
    expect(api.sameBody({ a: '1' }, { a: '2' })).toBe(false)
  })

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
    expect(Object.keys(balance('1.000')).sort()).toEqual(keys('PartsBalance'))
    expect(Object.keys(movement()).sort()).toEqual(keys('PartsMovement'))
    expect(keys('PartsResult')).toEqual(['balance', 'movement'])
    expect(keys('PartsReceiptInput')).toEqual([
      'locationId',
      'note',
      'quantity',
      'sourceReference',
      'supplierId',
      'uom',
    ])
    expect(keys('PartsIssueInput')).toEqual([
      'locationId',
      'note',
      'quantity',
      'reason',
      'sourceReference',
      'uom',
    ])
    for (const op of ['receipts', 'issues']) {
      expect(spec.paths[`/stock-items/{id}/${op}`]?.post?.parameters).toContainEqual(
        expect.objectContaining({ name: 'Idempotency-Key', in: 'header', required: true }),
      )
    }
  })
})

describe('stock on the SKU page', () => {
  it('is not shown without any parts permission and never calls the stock API', async () => {
    const spy = apiMock()
    renderApp('/inventory/parts/i1', { me: me(['stock_items.view']) })
    await screen.findByRole('heading', { level: 1, name: 'PRT-1' })
    expect(screen.queryByText('Stock')).not.toBeInTheDocument()
    expect(gets(spy, '/balances') + gets(spy, '/stock-movements')).toBe(0)
  })

  it('shows balances and history read-only for a viewer, without supplier names', async () => {
    const hiddenIssue = { ...issued }
    const hiddenReceipt = movement({ supplierId: null, supplierName: null })
    apiMock((u) =>
      u.pathname === '/api/v1/stock-items/i1/stock-movements'
        ? Response.json({ data: page([hiddenIssue, hiddenReceipt]) })
        : undefined,
    )
    renderApp('/inventory/parts/i1', { me: me(VIEW) })
    const bal = await screen.findByRole('table', { name: 'Balances' })
    expect(within(bal).getByText('5 PCS')).toBeInTheDocument()
    const hist = await screen.findByRole('table', { name: 'Receipts and issues' })
    expect(within(hist).getByText('−2 PCS')).toBeInTheDocument()
    expect(within(hist).getByText('Reason: Fitted in a repair')).toBeInTheDocument()
    expect(screen.queryByText(/Example Supplier/)).not.toBeInTheDocument()
    expect(screen.getByText(/Suppliers are not shown without permission/)).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Receive stock|Issue stock/ }),
    ).not.toBeInTheDocument()
  })

  it('says when there is no stock yet, distinct from an error, and sends history filters', async () => {
    const user = userEvent.setup()
    const spy = apiMock(
      (u) =>
        u.pathname === '/api/v1/stock-items/i1/stock-movements'
          ? Response.json({ data: page([]) })
          : undefined,
      { balances: [] },
    )
    renderApp('/inventory/parts/i1', { me: me(VIEW) })
    expect(await screen.findByText('No stock recorded')).toBeInTheDocument()
    expect(await screen.findByText('No receipts or issues yet')).toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Type'), 'PARTS_ISSUE')
    await waitFor(() => {
      expect(
        spy.mock.calls.some(
          ([u]) => new URL(u as string, 'http://x').searchParams.get('type') === 'PARTS_ISSUE',
        ),
      ).toBe(true)
    })
    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument()
  })

  it('shows an error state when balances cannot be loaded', async () => {
    apiMock((u) =>
      u.pathname === '/api/v1/stock-items/i1/balances' ? err(500, 'INTERNAL_ERROR') : undefined,
    )
    renderApp('/inventory/parts/i1', { me: me(VIEW) })
    expect(await screen.findByText('Could not load the balances')).toBeInTheDocument()
  })

  it('offers no receipt or issue for a serial-tracked or archived SKU', async () => {
    apiMock(undefined, { item: { ...item, trackingMode: 'SERIAL' } })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    expect(await screen.findByText(/serial-tracked/)).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Receive stock|Issue stock/ }),
    ).not.toBeInTheDocument()
    vi.restoreAllMocks()
    apiMock(undefined, {
      item: { ...item, status: 'ARCHIVED', archivedBy: 'u', archivedAt: t, archiveReason: 'x' },
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    expect(await screen.findAllByText(/This SKU is archived/)).not.toHaveLength(0)
    await screen.findByRole('table', { name: 'Balances' })
    expect(
      screen.queryByRole('button', { name: /Receive stock|Issue stock/ }),
    ).not.toBeInTheDocument()
  })

  it('offers receiving only with suppliers.view as well, and issuing only with its own key', async () => {
    apiMock()
    renderApp('/inventory/parts/i1', { me: me([...VIEW, 'stock.parts.receive']) })
    await screen.findByRole('table', { name: 'Balances' })
    expect(screen.queryByRole('button', { name: 'Receive stock' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Issue stock' })).not.toBeInTheDocument()
    expect(screen.getByText(/also needs permission to view suppliers/)).toBeInTheDocument()
  })
})

describe('receiving stock', () => {
  it('validates, sends the SKU unit and one key, and refreshes balance and history after success', async () => {
    const user = userEvent.setup()
    let sent: unknown
    const spy = apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-items/i1/receipts') {
        sent = bodyOf(init)
        return Response.json({
          data: result(
            movement({ quantityBefore: '5.000', quantityAfter: '10.000' }),
            balance('10.000'),
          ),
        })
      }
      return undefined
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Receive stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Receive stock' })
    expect(within(dialog).getByLabelText(/Unit \(the SKU's unit\)/)).toHaveTextContent('PCS')
    await user.click(within(dialog).getByRole('button', { name: 'Record receipt' }))
    expect(await within(dialog).findByText('Supplier is required')).toBeInTheDocument()
    expect(within(dialog).getByText('Enter the challan or invoice number')).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/Quantity/), '1.2345')
    await user.click(within(dialog).getByRole('button', { name: 'Record receipt' }))
    expect(await within(dialog).findByText(/up to 3 decimals \(e.g. 5/)).toBeInTheDocument()
    expect(posts(spy, 'receipts')).toHaveLength(0)
    // Archived suppliers are not offered; pending ones say so.
    const sup = within(dialog).getByLabelText(/Supplier/)
    await waitFor(() => {
      expect(
        within(sup).getByRole('option', { name: /Example Supplier One \(pending verification\)/ }),
      ).toBeInTheDocument()
    })
    expect(within(sup).queryByRole('option', { name: /Old/ })).not.toBeInTheDocument()
    await user.selectOptions(sup, 's1')
    await user.clear(within(dialog).getByLabelText(/Quantity/))
    await user.type(within(dialog).getByLabelText(/Quantity/), '5')
    await user.type(within(dialog).getByLabelText(/Challan or invoice/), ' DC-1 ')
    const before = gets(spy, '/balances') + gets(spy, '/stock-movements')
    await user.click(within(dialog).getByRole('button', { name: 'Record receipt' }))
    expect(await screen.findByText(/Receipt recorded: \+5 PCS at SR1/)).toBeInTheDocument()
    expect(sent).toEqual({
      locationId: 'l1',
      quantity: '5',
      uom: 'PCS',
      supplierId: 's1',
      sourceReference: 'DC-1',
    })
    expect(screen.queryByRole('dialog', { name: 'Receive stock' })).not.toBeInTheDocument()
    await waitFor(() => {
      expect(gets(spy, '/balances') + gets(spy, '/stock-movements')).toBeGreaterThan(before + 1)
    })
  })

  it('keeps the same key and body for a retry after an unknown outcome; a change makes a new key', async () => {
    const user = userEvent.setup()
    let n = 0
    const spy = apiMock((u) => {
      if (u.pathname !== '/api/v1/stock-items/i1/receipts') return undefined
      n++
      if (n === 1) return new TypeError('Failed to fetch')
      if (n === 2) return err(503, 'SERVICE_UNAVAILABLE')
      return Response.json({ data: result(movement(), balance('5.000')) })
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Receive stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Receive stock' })
    const sup = within(dialog).getByLabelText(/Supplier/)
    await waitFor(() => {
      expect(within(sup).getAllByRole('option').length).toBeGreaterThan(1)
    })
    await user.selectOptions(sup, 's1')
    await user.type(within(dialog).getByLabelText(/Quantity/), '5')
    await user.type(within(dialog).getByLabelText(/Challan or invoice/), 'DC-1')
    await user.click(within(dialog).getByRole('button', { name: 'Record receipt' }))
    expect(await within(dialog).findByText(/may or may not have been recorded/)).toBeInTheDocument()
    // The form is kept; the button now says it sends the same request again.
    expect(within(dialog).getByLabelText(/Quantity/)).toHaveValue('5')
    await user.click(within(dialog).getByRole('button', { name: 'Send again unchanged' }))
    await within(dialog).findByText(/may or may not have been recorded/)
    // Changing a value is a new decision: a new key.
    await user.clear(within(dialog).getByLabelText(/Quantity/))
    await user.type(within(dialog).getByLabelText(/Quantity/), '6')
    await user.click(within(dialog).getByRole('button', { name: 'Send again unchanged' }))
    await screen.findByText(/Receipt recorded/)
    const [a, b, c] = posts(spy, 'receipts') as [
      [string, RequestInit],
      [string, RequestInit],
      [string, RequestInit],
    ]
    expect(keyOf(b)).toBe(keyOf(a))
    expect(bodyOf(b[1])).toEqual(bodyOf(a[1]))
    expect(keyOf(c)).not.toBe(keyOf(a))
    expect(bodyOf(c[1])).toMatchObject({ quantity: '6' })
  })

  it('cannot be submitted twice while the first submission is in flight', async () => {
    const user = userEvent.setup()
    let release: (r: Response) => void = () => undefined
    const spy = vi.spyOn(globalThis, 'fetch')
    apiMock()
    spy.mockImplementation((input) => {
      const u = new URL(input as string, 'http://localhost')
      if (u.pathname === '/api/v1/stock-items/i1/receipts')
        return new Promise<Response>((res) => {
          release = res
        })
      const p = u.pathname
      const json = (data: unknown) => Promise.resolve(Response.json({ data }))
      if (p === '/api/v1/stock-items/i1') return json(item)
      if (p === '/api/v1/stock-items/i1/balances') return json([balance('5.000')])
      if (p === '/api/v1/stock-items/i1/stock-movements') return json(page([]))
      if (p === '/api/v1/suppliers')
        return json(page([supplier('s1', 'Example Supplier One', 'VERIFIED')]))
      return Promise.resolve(err(404, 'NOT_FOUND'))
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Receive stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Receive stock' })
    const sup = within(dialog).getByLabelText(/Supplier/)
    await waitFor(() => {
      expect(within(sup).getAllByRole('option').length).toBeGreaterThan(1)
    })
    await user.selectOptions(sup, 's1')
    await user.type(within(dialog).getByLabelText(/Quantity/), '5')
    await user.type(within(dialog).getByLabelText(/Challan or invoice/), 'DC-1')
    const button = within(dialog).getByRole('button', { name: 'Record receipt' })
    await user.click(button)
    await user.click(button)
    await user.click(button)
    expect(posts(spy, 'receipts')).toHaveLength(1)
    release(Response.json({ data: result(movement(), balance('5.000')) }))
    await screen.findByText(/Receipt recorded/)
  })

  it('says when no location is assigned and cannot be sent', async () => {
    const user = userEvent.setup()
    apiMock()
    renderApp('/inventory/parts/i1', { me: me(ALL, []) })
    await user.click(await screen.findByRole('button', { name: 'Receive stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Receive stock' })
    expect(within(dialog).getByText(/No location is assigned to you/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Record receipt' })).toBeDisabled()
  })
})

describe('issuing stock', () => {
  it('shows the available balance as information, sends the issue and refreshes', async () => {
    const user = userEvent.setup()
    let sent: unknown
    apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-items/i1/issues') {
        sent = bodyOf(init)
        return Response.json({ data: result(issued, balance('3.000')) })
      }
      return undefined
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Issue stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Issue stock' })
    expect(await within(dialog).findByText('5 PCS')).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Record issue' }))
    expect(await within(dialog).findByText('Give the reason for this issue')).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/Quantity/), '6')
    expect(within(dialog).getByText(/the server will refuse it/)).toBeInTheDocument()
    await user.clear(within(dialog).getByLabelText(/Quantity/))
    await user.type(within(dialog).getByLabelText(/Quantity/), '2')
    await user.type(within(dialog).getByLabelText(/^Reason/), 'Fitted in a repair')
    await user.type(within(dialog).getByLabelText(/^Reference/), 'JOB-1')
    await user.click(within(dialog).getByRole('button', { name: 'Record issue' }))
    expect(await screen.findByText(/Issue recorded: −2 PCS at SR1/)).toBeInTheDocument()
    expect(sent).toEqual({
      locationId: 'l1',
      quantity: '2',
      uom: 'PCS',
      reason: 'Fitted in a repair',
      sourceReference: 'JOB-1',
    })
  })

  it('keeps the form and refreshes the balance on INSUFFICIENT_STOCK, without changing it locally', async () => {
    const user = userEvent.setup()
    let refused = false
    const spy = apiMock((u) => {
      if (u.pathname === '/api/v1/stock-items/i1/issues') {
        refused = true
        return err(409, 'INSUFFICIENT_STOCK', [
          { field: 'quantity', message: 'is more than the available balance' },
        ])
      }
      // After the refusal the server says only 1 is left (someone else issued meanwhile).
      if (u.pathname === '/api/v1/stock-items/i1/balances' && refused)
        return Response.json({ data: [balance('1.000')] })
      return undefined
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Issue stock' }))
    const dialog = await screen.findByRole('dialog', { name: 'Issue stock' })
    await within(dialog).findByText('5 PCS')
    await user.type(within(dialog).getByLabelText(/Quantity/), '4')
    await user.type(within(dialog).getByLabelText(/^Reason/), 'Repair')
    await user.click(within(dialog).getByRole('button', { name: 'Record issue' }))
    expect(await within(dialog).findByText(/not enough stock at this location/)).toBeInTheDocument()
    expect(within(dialog).getByText('More than the available balance')).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/Quantity/)).toHaveValue('4')
    expect(within(dialog).getByLabelText(/^Reason/)).toHaveValue('Repair')
    // The balance shown is the server's new answer, not a local calculation.
    expect(await within(dialog).findByText('1 PCS')).toBeInTheDocument()
    expect(posts(spy, 'issues')).toHaveLength(1)
    expect(screen.queryByText(/Issue recorded/)).not.toBeInTheDocument()
  })

  it('shows key reuse, wrong unit and permission answers safely', async () => {
    const user = userEvent.setup()
    for (const [status, code, text, details] of [
      [409, 'IDEMPOTENCY_KEY_REUSED', /could not be matched to the earlier attempt/, undefined],
      [
        422,
        'VALIDATION_FAILED',
        /Unit: must be the SKU's unit of measure, SET/,
        [{ field: 'uom', message: "must be the SKU's unit of measure, SET" }],
      ],
      [403, 'FORBIDDEN', /do not have permission/, undefined],
      [409, 'TRACKING_NOT_SUPPORTED', /no serial register/, undefined],
    ] as const) {
      apiMock((u) =>
        u.pathname === '/api/v1/stock-items/i1/issues'
          ? err(status, code, details ? [...details] : undefined)
          : undefined,
      )
      const { unmount } = renderApp('/inventory/parts/i1', { me: me(ALL) })
      await user.click(await screen.findByRole('button', { name: 'Issue stock' }))
      const dialog = await screen.findByRole('dialog', { name: 'Issue stock' })
      await user.type(within(dialog).getByLabelText(/Quantity/), '1')
      await user.type(within(dialog).getByLabelText(/^Reason/), 'Repair')
      await user.click(within(dialog).getByRole('button', { name: 'Record issue' }))
      expect(await within(dialog).findByText(text)).toBeInTheDocument()
      expect(within(dialog).getByLabelText(/Quantity/)).toHaveValue('1')
      unmount()
      vi.restoreAllMocks()
    }
  })
})

describe('parts error mapping', () => {
  it('never calls an unknown outcome a failure', () => {
    const net = new ApiError(0, { code: 'NETWORK_ERROR', message: 'x' })
    expect(describePartsError(net)).toMatchObject({ kind: 'uncertain', retrySame: true })
    expect(
      describePartsError(new ApiError(504, { code: 'GATEWAY_TIMEOUT', message: 'x' })),
    ).toMatchObject({
      kind: 'uncertain',
      retrySame: true,
    })
    expect(
      describePartsError(new ApiError(200, { code: 'INVALID_RESPONSE', message: 'x' })),
    ).toMatchObject({
      kind: 'uncertain',
      retrySame: false,
    })
    expect(
      describePartsError(new ApiError(401, { code: 'UNAUTHENTICATED', message: 'x' })),
    ).toMatchObject({
      kind: 'signed-out',
    })
    expect(
      describePartsError(new ApiError(404, { code: 'NOT_FOUND', message: 'x' })),
    ).toMatchObject({
      kind: 'not-found',
      retrySame: false,
    })
  })
})
