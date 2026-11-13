/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { act, cleanup, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/timeline'
import type { TimelineMovement } from './api/timeline'
import { quantityChange, statusChange, utcOffsetLabel } from './lib/timeline'
import { dayStart, useUrlFilters } from './lib/urlFilters'

// E-02 stock movement timeline against a mocked API. Placeholder data only
// ("MAE-VEH-000001", "PRT-1", "GRN-000001"): nothing real.

const t1 = '2026-11-10T09:00:00Z'
const t2 = '2026-11-11T10:30:15Z'
const loc = { id: 'l1', code: 'SR1', name: 'Showroom One' }

const receipt: TimelineMovement = {
  id: 'mv1',
  itemType: 'VEHICLE_UNIT',
  movementType: 'RECEIPT',
  createdAt: t1,
  vehicleUnit: { id: 'u1', unitNumber: 'MAE-VEH-000001', chassisNumber: 'CH-0001' },
  stockItem: null,
  quantityBefore: '0.000',
  quantityAfter: '1.000',
  uom: null,
  fromStatus: null,
  toStatus: 'RECEIVED_PENDING_PDI',
  fromLocationId: null,
  fromLocationCode: null,
  fromLocationName: null,
  toLocationId: 'l1',
  toLocationCode: 'SR1',
  toLocationName: 'Showroom One',
  sourceType: 'GOODS_RECEIPT',
  sourceId: 'g1',
  sourceReference: 'GRN-000001',
  sourceHidden: false,
  supplierId: null,
  supplierName: null,
  reason: null,
  note: null,
  actorId: 'user-2',
  actorName: 'Example Poster',
}
const pdiPass: TimelineMovement = {
  ...receipt,
  id: 'mv2',
  movementType: 'PDI_PASS',
  createdAt: t2,
  quantityBefore: '1.000',
  fromStatus: 'RECEIVED_PENDING_PDI',
  toStatus: 'AVAILABLE',
  fromLocationId: 'l1',
  fromLocationCode: 'SR1',
  fromLocationName: 'Showroom One',
  sourceType: 'PDI_INSPECTION',
  sourceId: 'i1',
  sourceReference: 'PDI-000001',
  actorId: 'user-1',
  actorName: 'Test User',
}
const partsIssue: TimelineMovement = {
  ...receipt,
  id: 'mv3',
  itemType: 'STOCK_ITEM',
  movementType: 'PARTS_ISSUE',
  createdAt: t2,
  vehicleUnit: null,
  stockItem: { id: 'p1', sku: 'PRT-1', name: 'Brake pad' },
  quantityBefore: '10.000',
  quantityAfter: '8.500',
  uom: 'PCS',
  toStatus: null,
  fromLocationId: 'l1',
  fromLocationCode: 'SR1',
  fromLocationName: 'Showroom One',
  sourceType: 'STAFF_ACTION',
  sourceId: null,
  sourceReference: null,
  reason: 'Fitted in a repair',
}
const hiddenGRN: TimelineMovement = {
  ...receipt,
  id: 'mv4',
  sourceId: null,
  sourceReference: null,
  sourceHidden: true,
}

const page = <T,>(
  items: T[],
  over: Partial<{ page: number; limit: number; total: number }> = {},
) => ({
  items,
  page: 1,
  limit: 25,
  total: items.length,
  ...over,
})
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-9', details } }, { status })
const me = (permissions: string[], all = false): Me => ({
  ...testMe,
  permissions,
  locationScope: { all, locations: [loc] },
})
const VEHICLES = ['vehicle_units.view']
const BOTH = ['vehicle_units.view', 'stock.parts.view']
const FULL = [
  ...BOTH,
  'goods_receipts.view',
  'stock.adjustment.view',
  'stock_items.view',
  'suppliers.view',
]

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined
interface State {
  list?: TimelineMovement[]
  total?: number
  detail?: TimelineMovement
}

function apiMock(state: State = {}, handler: Handler = () => undefined) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/stock-movements') {
      const items = state.list ?? [pdiPass, partsIssue, receipt]
      const pg = Number(url.searchParams.get('page') ?? '1')
      return json(page(items, { page: pg, total: state.total ?? items.length }))
    }
    if (p.startsWith('/api/v1/stock-movements/')) {
      const id = p.split('/').at(-1)
      const found =
        state.detail ?? [pdiPass, partsIssue, receipt, hiddenGRN].find((m) => m.id === id)
      return found ? json(found) : Promise.resolve(err(404, 'NOT_FOUND'))
    }
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
const listCalls = (spy: Spy) =>
  spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://x'))
    .filter((u) => u.pathname === '/api/v1/stock-movements')
    .map((u) => Object.fromEntries(u.searchParams))
const lastList = (spy: Spy) => listCalls(spy).at(-1)
const writes = (spy: Spy) =>
  spy.mock.calls
    .filter(([, init]) => init?.method && init.method !== 'GET')
    .map(([u]) => (u instanceof Request ? u.url : (u as string)))

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('stock movement timeline API client (E-02)', () => {
  it('matches the server contract field for field (docs/openapi)', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<string, Record<string, { parameters?: { name: string; enum?: string[] }[] }>>
    }
    const props = (name: string) => spec.definitions[name]?.properties ?? {}
    const keys = (name: string) => Object.keys(props(name)).sort()
    expect(Object.keys(receipt).sort()).toEqual(keys('TimelineMovement'))
    expect(Object.keys(receipt.vehicleUnit ?? {}).sort()).toEqual(keys('TimelineVehicleUnit'))
    expect(Object.keys(partsIssue.stockItem ?? {}).sort()).toEqual(keys('TimelineStockItem'))
    expect(api.timelineMovementTypeSchema.options).toEqual(
      props('TimelineMovement').movementType?.enum,
    )
    expect(api.timelineSourceTypeSchema.options).toEqual(props('TimelineMovement').sourceType?.enum)
    expect(api.timelineItemTypeSchema.options).toEqual(props('TimelineMovement').itemType?.enum)
    // Every query parameter the client may send is one the server documents, and the reverse.
    const params: (keyof api.TimelineParams)[] = [
      'itemType',
      'vehicleUnitId',
      'stockItemId',
      'q',
      'movementType',
      'sourceType',
      'sourceReference',
      'actorId',
      'locationId',
      'from',
      'to',
      'order',
      'page',
      'limit',
    ]
    const documented = spec.paths['/stock-movements']?.get?.parameters?.map((p) => p.name) ?? []
    expect([...documented].sort()).toEqual([...params].sort())
    // Read-only: only GET exists on both routes.
    expect(Object.keys(spec.paths['/stock-movements'] ?? {})).toEqual(['get'])
    expect(Object.keys(spec.paths['/stock-movements/{id}'] ?? {})).toEqual(['get'])
  })

  it('sends only the filters that are set and reads a page', async () => {
    const spy = apiMock()
    const p = await api.listStockMovements({
      movementType: 'PDI_PASS',
      q: '',
      actorId: undefined,
      from: '2026-11-01T00:00:00.000Z',
      page: 2,
      limit: 25,
    })
    expect(p.items.map((m) => m.id)).toEqual(['mv2', 'mv3', 'mv1'])
    expect(lastList(spy)).toEqual({
      movementType: 'PDI_PASS',
      from: '2026-11-01T00:00:00.000Z',
      page: '2',
      limit: '25',
    })
    expect(writes(spy)).toEqual([])
  })

  it('refuses a hidden source that carries its reference, and mismatched item references', async () => {
    for (const bad of [
      { ...receipt, sourceHidden: true },
      { ...receipt, stockItem: { id: 'p1', sku: 'X', name: 'X' } },
      { ...partsIssue, stockItem: null },
      { ...receipt, movementType: 'SALE' },
    ]) {
      apiMock({ list: [bad as TimelineMovement] })
      await expect(api.listStockMovements()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
      vi.restoreAllMocks()
    }
  })

  it('turns calendar days into the instants the API expects (to is exclusive)', () => {
    const from = dayStart('2026-11-12')
    const to = dayStart('2026-11-12', 1)
    expect(new Date(from ?? '').getTime()).toBe(new Date(2026, 10, 12).getTime())
    expect(new Date(to ?? '').getTime()).toBe(new Date(2026, 10, 13).getTime())
    expect(dayStart('')).toBeUndefined()
    expect(dayStart('12/11/2026')).toBe('12/11/2026') // sent as typed: the server reports it
    expect(utcOffsetLabel()).toMatch(/^UTC[+-]\d{2}:\d{2}$/)
  })

  it('describes changes from the ledger values only', () => {
    expect(statusChange(receipt)).toEqual({ from: 'Not in stock', to: 'Awaiting PDI' })
    expect(statusChange(partsIssue)).toBeNull()
    expect(quantityChange(partsIssue)).toBe('10 → 8.5 PCS')
    expect(quantityChange(receipt)).toBeNull()
  })
})

describe('URL filters (E-02)', () => {
  function Probe() {
    const { values, set } = useUrlFilters(['a', 'b', 'c'] as const)
    return (
      <div>
        <output aria-label="values">{JSON.stringify(values)}</output>
        <button
          onClick={() => {
            // Two changes before React re-renders: neither may be lost.
            set({ a: '1' })
            set({ b: '2' })
          }}
        >
          both
        </button>
        <button
          onClick={() => {
            set({ c: '3' })
          }}
        >
          one
        </button>
      </div>
    )
  }

  it('keeps two filters changed in the same frame (regression for the shared helper race)', async () => {
    const router = createMemoryRouter([{ path: '/', element: <Probe /> }], {
      initialEntries: ['/?page=4&c=x'],
    })
    render(<RouterProvider router={router} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'both' }))
    expect(Object.fromEntries(new URLSearchParams(router.state.location.search))).toEqual({
      c: 'x',
      a: '1',
      b: '2',
    })
    await user.click(screen.getByRole('button', { name: 'one' }))
    expect(Object.fromEntries(new URLSearchParams(router.state.location.search))).toEqual({
      c: '3',
      a: '1',
      b: '2',
    })
    // A navigation from outside (back/forward, a link) becomes the new base.
    await act(() => router.navigate('/?b=9'))
    await user.click(screen.getByRole('button', { name: 'one' }))
    expect(Object.fromEntries(new URLSearchParams(router.state.location.search))).toEqual({
      b: '9',
      c: '3',
    })
  })
})

describe('stock movement timeline screens (E-02)', () => {
  it('lists the ledger with readable events, statuses, quantities, actors and sources', async () => {
    const spy = apiMock()
    renderApp('/inventory/movements', { me: me(FULL) })
    expect(screen.getByRole('status', { name: 'Loading stock movements' })).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: 'Stock movements' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows).toHaveLength(3)
    // Newest first, exactly as the server ordered them.
    expect(
      within(rows[0] as HTMLElement).getByRole('link', { name: 'Passed PDI' }),
    ).toBeInTheDocument()
    expect(rows[0]).toHaveTextContent('Awaiting PDI')
    expect(rows[0]).toHaveTextContent('Available')
    expect(rows[0]).toHaveTextContent('You')
    expect(
      within(rows[0] as HTMLElement).getByRole('link', { name: 'PDI-000001' }),
    ).toHaveAttribute('href', '/procurement/pdi/u1')
    expect(rows[1]).toHaveTextContent('Parts issued')
    expect(rows[1]).toHaveTextContent('10 → 8.5 PCS')
    expect(within(rows[1] as HTMLElement).getByRole('link', { name: 'PRT-1' })).toHaveAttribute(
      'href',
      '/inventory/parts/p1',
    )
    expect(rows[2]).toHaveTextContent('Not in stock')
    expect(rows[2]).toHaveTextContent('Example Poster')
    expect(
      within(rows[2] as HTMLElement).getByRole('link', { name: 'GRN-000001' }),
    ).toHaveAttribute('href', '/procurement/goods-receipts/g1')
    expect(
      within(rows[2] as HTMLElement).getByRole('link', { name: 'MAE-VEH-000001' }),
    ).toHaveAttribute('href', '/inventory/vehicles/u1')
    // Phone layout: the same rows as a labelled card list.
    expect(
      within(screen.getByRole('list', { name: 'Stock movements' })).getAllByRole('listitem'),
    ).toHaveLength(3)
    // Times are in the user's zone, with the offset stated.
    expect(screen.getByRole('note')).toHaveTextContent(utcOffsetLabel())
    // Read-only: no control changes a movement, and nothing but GETs were sent.
    expect(
      screen.queryByRole('button', { name: /edit|delete|reverse|correct|undo/i }),
    ).not.toBeInTheDocument()
    expect(writes(spy)).toEqual([])
    expect(lastList(spy)).toEqual({ page: '1', limit: '25' })
  })

  it('links only what the user may open and never shows a hidden source', async () => {
    apiMock({ list: [hiddenGRN, receipt] })
    renderApp('/inventory/movements', { me: me(VEHICLES) })
    const table = await screen.findByRole('table', { name: 'Stock movements' })
    const [hidden, visible] = within(table).getAllByRole('row').slice(1)
    if (!hidden || !visible) throw new Error('expected two rows')
    expect(hidden).toHaveTextContent('Not visible to you')
    expect(within(hidden).queryByText(/GRN-/)).not.toBeInTheDocument()
    // A reference the server returned, but no goods_receipts.view: text, not a link.
    expect(within(visible).getByText('GRN-000001').closest('a')).toBeNull()
    // No item-type filter without parts access.
    expect(screen.queryByLabelText('Item type')).not.toBeInTheDocument()
  })

  it('sends each filter, combines them, keeps them in the URL and clears them', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    const { router } = renderApp('/inventory/movements', { me: me(FULL) })
    await screen.findByRole('table', { name: 'Stock movements' })
    await user.selectOptions(screen.getByLabelText('Item type'), 'VEHICLE_UNIT')
    await user.selectOptions(screen.getByLabelText('Movement'), 'PDI_PASS')
    await user.selectOptions(screen.getByLabelText('Source'), 'PDI_INSPECTION')
    await user.type(screen.getByLabelText('Source reference'), 'pdi-000001{Enter}')
    await user.selectOptions(screen.getByLabelText('Recorded by'), 'user-1')
    await user.selectOptions(screen.getByLabelText('Location'), 'l1')
    await user.type(screen.getByLabelText('From date'), '2026-11-01')
    await user.type(screen.getByLabelText('To date'), '2026-11-30')
    await user.selectOptions(screen.getByLabelText('Order'), 'asc')
    await user.type(screen.getByLabelText('Search unit number, chassis or SKU'), 'veh-0001')
    await waitFor(() => {
      expect(lastList(spy)).toEqual({
        q: 'veh-0001',
        itemType: 'VEHICLE_UNIT',
        movementType: 'PDI_PASS',
        sourceType: 'PDI_INSPECTION',
        sourceReference: 'pdi-000001',
        actorId: 'user-1',
        locationId: 'l1',
        from: new Date(2026, 10, 1).toISOString(),
        to: new Date(2026, 11, 1).toISOString(),
        order: 'asc',
        page: '1',
        limit: '25',
      })
    })
    expect(Object.fromEntries(new URLSearchParams(router.state.location.search))).toMatchObject({
      movementType: 'PDI_PASS',
      from: '2026-11-01',
      to: '2026-11-30',
      order: 'asc',
    })
    await user.click(screen.getByRole('button', { name: 'Clear filters' }))
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ page: '1', limit: '25' })
    })
    expect(router.state.location.search).toBe('')
  })

  it('keeps a filter chosen while a search is still pending', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    renderApp('/inventory/movements', { me: me(BOTH) })
    await screen.findByRole('table', { name: 'Stock movements' })
    await user.type(screen.getByLabelText('Search unit number, chassis or SKU'), 'prt')
    await user.selectOptions(screen.getByLabelText('Movement'), 'PARTS_ISSUE')
    await waitFor(() => {
      expect(lastList(spy)).toEqual({
        q: 'prt',
        movementType: 'PARTS_ISSUE',
        page: '1',
        limit: '25',
      })
    })
  })

  it('pages on the server and returns to page 1 when a filter changes', async () => {
    const user = userEvent.setup()
    const spy = apiMock({ total: 60 })
    renderApp('/inventory/movements', { me: me(BOTH) })
    await screen.findByRole('table', { name: 'Stock movements' })
    await user.click(screen.getByRole('button', { name: /next/i }))
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ page: '2', limit: '25' })
    })
    await user.selectOptions(screen.getByLabelText('Movement'), 'RECEIPT')
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ movementType: 'RECEIPT', page: '1', limit: '25' })
    })
  })

  it('opens pinned filters from the URL and removes them', async () => {
    const user = userEvent.setup()
    const spy = apiMock({ list: [receipt] })
    renderApp('/inventory/movements?vehicleUnitId=u1&actorId=user-2', { me: me(BOTH) })
    await screen.findByRole('table', { name: 'Stock movements' })
    expect(lastList(spy)).toEqual({
      vehicleUnitId: 'u1',
      actorId: 'user-2',
      page: '1',
      limit: '25',
    })
    // Another person's filter is named from the rows, not looked up elsewhere.
    expect(screen.getByLabelText('Recorded by')).toHaveDisplayValue('Example Poster')
    await user.click(screen.getByRole('button', { name: 'Remove filter: Vehicle MAE-VEH-000001' }))
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ actorId: 'user-2', page: '1', limit: '25' })
    })
  })

  it('shows empty, no-results, invalid-filter, error and retry states', async () => {
    const user = userEvent.setup()
    apiMock({ list: [] })
    renderApp('/inventory/movements', { me: me(BOTH) })
    expect(await screen.findByText('No stock movements yet')).toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()

    apiMock({ list: [] })
    renderApp('/inventory/movements?movementType=REJECT', { me: me(BOTH) })
    expect(await screen.findByText('No movements match these filters')).toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()

    apiMock({}, (u) =>
      u.pathname === '/api/v1/stock-movements'
        ? err(422, 'VALIDATION_FAILED', [
            { field: 'locationId', message: 'is not one of your locations' },
          ])
        : undefined,
    )
    renderApp('/inventory/movements?locationId=l9', { me: me(BOTH) })
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Location is not one of your locations')
    cleanup()
    vi.restoreAllMocks()

    let fail = true
    const spy = apiMock({}, (u) => {
      if (u.pathname !== '/api/v1/stock-movements' || !fail) return undefined
      throw new TypeError('Failed to fetch')
    })
    renderApp('/inventory/movements', { me: me(BOTH) })
    expect(await screen.findByText('Could not load stock movements')).toBeInTheDocument()
    fail = false
    await user.click(screen.getByRole('button', { name: /try again|retry/i }))
    expect(await screen.findByRole('table', { name: 'Stock movements' })).toBeInTheDocument()
    expect(listCalls(spy).length).toBeGreaterThanOrEqual(2)
  })

  it('denies access without either view permission and sends no request', async () => {
    const spy = apiMock()
    renderApp('/inventory/movements', { me: me(['goods_receipts.view', 'stock.adjustment.view']) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(listCalls(spy)).toEqual([])
    expect(screen.queryByRole('link', { name: /Stock Movements/ })).not.toBeInTheDocument()
  })

  it('shows access denied when the server refuses (403) and signs out on 401', async () => {
    apiMock({}, (u) =>
      u.pathname === '/api/v1/stock-movements' ? err(403, 'FORBIDDEN') : undefined,
    )
    renderApp('/inventory/movements', { me: me(BOTH) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()

    apiMock({}, (u) =>
      u.pathname === '/api/v1/stock-movements' ? err(401, 'UNAUTHENTICATED') : undefined,
    )
    const { router } = renderApp('/inventory/movements', { me: me(BOTH) })
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/login')
    })
  })

  it('shows one movement with its values, related links and no edit controls', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    const { router } = renderApp('/inventory/movements?movementType=PDI_PASS', { me: me(FULL) })
    await screen.findByRole('table', { name: 'Stock movements' })
    await user.click(screen.getAllByRole('link', { name: 'Passed PDI' })[0] as HTMLElement)
    expect(await screen.findByRole('heading', { level: 1, name: 'Passed PDI' })).toBeInTheDocument()
    const card =
      screen.getByRole('heading', { name: 'Movement' }).closest('section') ?? document.body
    expect(card).toHaveTextContent('Awaiting PDI')
    expect(card).toHaveTextContent('Available')
    expect(card).toHaveTextContent('SR1 — Showroom One')
    expect(card).toHaveTextContent('You')
    expect(screen.getByRole('link', { name: 'PDI-000001' })).toHaveAttribute(
      'href',
      '/procurement/pdi/u1',
    )
    expect(screen.getByRole('link', { name: 'All movements of MAE-VEH-000001' })).toHaveAttribute(
      'href',
      '/inventory/movements?vehicleUnitId=u1',
    )
    expect(screen.getByRole('link', { name: 'All movements recorded by you' })).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /edit|delete|reverse|correct|undo/i }),
    ).not.toBeInTheDocument()
    expect(writes(spy)).toEqual([])
    // Back keeps the list filters.
    await user.click(screen.getByRole('link', { name: 'Back to stock movements' }))
    expect(router.state.location.search).toBe('?movementType=PDI_PASS')
  })

  it('shows a hidden source on the detail without its reference, and 404 for an unseen movement', async () => {
    apiMock()
    renderApp('/inventory/movements/mv4', { me: me(VEHICLES) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Received into stock' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Not visible to you')).toBeInTheDocument()
    expect(screen.queryByText(/All movements from/)).not.toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()

    apiMock()
    renderApp('/inventory/movements/unknown', { me: me(VEHICLES) })
    expect(await screen.findByText('Movement not found')).toBeInTheDocument()
  })
})
