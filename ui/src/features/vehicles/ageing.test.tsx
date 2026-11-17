/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/vehicleUnits'
import type { AgeingSummary, StockAgeing, VehicleUnit } from './api/vehicleUnits'

// E-07 stock ageing in the vehicle screens against a mocked API. Test-only
// values ("MAE-VEH-…", the LED-008 recommended bands): nothing real.

const t = '2026-11-10T09:00:00Z'
const loc = { id: 'l1', code: 'SR1', name: 'Showroom One' }
const bands = [
  { key: '0-30', label: '0–30 days', minDays: 0, maxDays: 30 },
  { key: '31-60', label: '31–60 days', minDays: 31, maxDays: 60 },
  { key: '61-90', label: '61–90 days', minDays: 61, maxDays: 90 },
  { key: 'over-90', label: 'Over 90 days', minDays: 91, maxDays: null },
]

const aged = (days: number, band: string, label: string): StockAgeing => ({
  status: 'AGED',
  originDate: '2026-10-17',
  originSource: 'GRN',
  ageDays: days,
  bandKey: band,
  bandLabel: label,
  missingReason: null,
  missingExplanation: null,
  asOf: '2026-11-16',
})
const missing: StockAgeing = {
  status: 'MISSING_ORIGIN',
  originDate: '2026-11-20',
  originSource: null,
  ageDays: null,
  bandKey: null,
  bandLabel: null,
  missingReason: 'FUTURE_ORIGIN_DATE',
  missingExplanation:
    'Its origin date is after today’s business date, so no age can be calculated.',
  asOf: '2026-11-16',
}

const base: VehicleUnit = {
  id: 'u1',
  unitNumber: 'MAE-VEH-000001',
  status: 'AVAILABLE',
  brandId: 'b1',
  brandName: 'Brand One',
  modelId: 'm1',
  modelName: 'Model One',
  variantId: 'v1',
  variantName: 'Standard',
  chassisNumber: 'CH-0001',
  chassisNormalised: 'CH-0001',
  motorSerial: null,
  batterySerial: null,
  chargerSerial: null,
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom One',
  sourceType: 'GOODS_RECEIPT',
  supplierId: null,
  supplierName: null,
  receivedDate: '2026-10-17',
  ageDays: 30,
  ageing: aged(30, '0-30', '0–30 days'),
  unitCost: null,
  costCurrency: null,
  costHidden: true,
  version: 1,
  createdAt: t,
  updatedAt: t,
}
const old: VehicleUnit = {
  ...base,
  id: 'u2',
  unitNumber: 'MAE-VEH-000002',
  chassisNumber: 'CH-0002',
  chassisNormalised: 'CH-0002',
  ageDays: 31,
  ageing: aged(31, '31-60', '31–60 days'),
}
const noOrigin: VehicleUnit = {
  ...base,
  id: 'u3',
  unitNumber: 'MAE-VEH-000003',
  chassisNumber: 'CH-0003',
  chassisNormalised: 'CH-0003',
  receivedDate: '2026-11-20',
  ageDays: null,
  ageing: missing,
}

const summary = (over: Partial<AgeingSummary> = {}): AgeingSummary => ({
  asOf: '2026-11-16',
  timeZone: 'Asia/Kolkata',
  configured: true,
  bandSet: { version: 1, status: 'PROPOSED', source: 'LED-008' },
  total: 3,
  withOrigin: 2,
  missingOrigin: 1,
  bands: bands.map((b) => ({ ...b, count: b.key === '0-30' || b.key === '31-60' ? 1 : 0 })),
  unbanded: 0,
  missingByReason: { FUTURE_ORIGIN_DATE: 1 },
  ...over,
})
const configuration = (configured = true) => ({
  configured,
  bandSet: configured
    ? {
        version: 1,
        status: 'PROPOSED',
        source: 'LED-008',
        createdById: null,
        createdByName: null,
        createdAt: t,
        approvedById: null,
        approvedByName: null,
        approvedAt: null,
        approvalReference: null,
        bands,
      }
    : null,
  timeZone: 'Asia/Kolkata',
  asOf: '2026-11-16',
  boundaryRule: 'minDays <= age <= maxDays',
})
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-7', details } }, { status })
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const VIEW = ['vehicle_units.view']

interface State {
  list?: VehicleUnit[]
  summary?: AgeingSummary | Response
  configured?: boolean
}
function apiMock(state: State = {}) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/vehicle-units') return json(page(state.list ?? [base, old, noOrigin]))
    if (p === '/api/v1/vehicle-units/ageing-summary') {
      const s = state.summary ?? summary()
      return s instanceof Response ? Promise.resolve(s.clone()) : json(s)
    }
    if (p === '/api/v1/stock-ageing/configuration')
      return json(configuration(state.configured ?? true))
    if (p === '/api/v1/vehicle-units/u3')
      return json({ ...noOrigin, source: src, events: [], documents: [], documentsHidden: true })
    if (p === '/api/v1/vehicle-units/u1')
      return json({ ...base, source: src, events: [], documents: [], documentsHidden: true })
    if (p.endsWith('/stock-movements')) return json({ ...page([]), limit: 20 })
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
const src = {
  type: 'GOODS_RECEIPT',
  supplierId: null,
  supplierName: null,
  purchaseOrderId: null,
  purchaseOrderNumber: null,
  purchaseOrderLineId: null,
  goodsReceiptId: null,
  goodsReceiptNumber: null,
  goodsReceiptLineId: null,
  goodsReceiptUnitId: null,
  supplierReference: null,
  receivedDate: '2026-10-17',
  openingStockBatchId: null,
  openingStockBatchReference: null,
  openingStockRowNumber: null,
}
type Spy = ReturnType<typeof apiMock>
const callsTo = (spy: Spy, pathname: string) =>
  spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://x'))
    .filter((u) => u.pathname === pathname)
    .map((u) => Object.fromEntries(u.searchParams))

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('stock ageing API contract (E-07)', () => {
  it('matches the server field for field (docs/openapi)', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<string, Record<string, { parameters?: { name: string }[] }>>
    }
    const keys = (name: string) => Object.keys(spec.definitions[name]?.properties ?? {}).sort()
    expect(Object.keys(missing).sort()).toEqual(keys('StockAgeing'))
    expect(Object.keys(summary()).sort()).toEqual(keys('VehicleAgeingSummary'))
    expect(Object.keys(summary().bands[0] ?? {}).sort()).toEqual(keys('VehicleAgeingBandCount'))
    expect(Object.keys(configuration()).sort()).toEqual(keys('StockAgeingConfiguration'))
    expect(api.missingOriginReasonSchema.options).toEqual(
      spec.definitions.StockAgeing?.properties.missingReason?.enum,
    )
    expect(Object.keys(base)).toContain('ageing')
    const listParams = spec.paths['/vehicle-units']?.get?.parameters?.map((p) => p.name) ?? []
    expect(listParams).toEqual(expect.arrayContaining(['ageingBand', 'originStatus']))
    // Read-only for stock: the summary and configuration routes have GET only.
    expect(Object.keys(spec.paths['/vehicle-units/ageing-summary'] ?? {})).toEqual(['get'])
    expect(Object.keys(spec.paths['/stock-ageing/configuration'] ?? {})).toEqual(['get'])
  })

  it('refuses an ageing that is not self-consistent (an age without an origin, a band without an age)', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json({ data: page([{ ...noOrigin, ageing: { ...missing, ageDays: 0 } }]) }),
    )
    await expect(api.listVehicleUnits()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json({ data: page([{ ...base, ageDays: 29 }]) }),
    )
    await expect(api.listVehicleUnits()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(Response.json({ data: summary({ total: 4 }) }))
    await expect(api.getAgeingSummary()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })
})

describe('vehicle list stock ageing (E-07)', () => {
  it('shows the server’s age, band and origin, and a missing origin date as such (never 0 days)', async () => {
    apiMock()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    const table = await screen.findByRole('table', { name: 'Vehicle units' })
    const row = (unit: string) =>
      within(within(table).getByRole('link', { name: unit }).closest('tr') as HTMLElement)
    expect(within(table).getByRole('columnheader', { name: 'Stock age' })).toBeInTheDocument()
    expect(row('MAE-VEH-000001').getByText('30 days')).toBeInTheDocument()
    expect(row('MAE-VEH-000001').getByText('0–30 days')).toBeInTheDocument()
    expect(row('MAE-VEH-000001').getByText(/from goods receipt of/)).toBeInTheDocument()
    expect(row('MAE-VEH-000002').getByText('31 days')).toBeInTheDocument()
    expect(row('MAE-VEH-000002').getByText('31–60 days')).toBeInTheDocument()
    const m = row('MAE-VEH-000003')
    expect(m.getByText('No reliable origin date')).toBeInTheDocument()
    expect(m.getByText('Origin date after today')).toHaveAttribute(
      'title',
      expect.stringContaining('after today'),
    )
    expect(m.queryByText(/^0 days$/)).toBeNull()
  })

  it('shows the ageing summary for the same filters and filters by band or missing origin', async () => {
    const spy = apiMock()
    const user = userEvent.setup()
    renderApp('/inventory/vehicles?brandId=b1', { me: me(VIEW) })
    const panel = await screen.findByRole('region', { name: 'Stock ageing' })
    expect(within(panel).getByText(/3 vehicles as of/)).toBeInTheDocument()
    expect(within(panel).getByText(/proposed and awaiting owner approval/)).toBeInTheDocument()
    const tiles = within(panel).getByRole('list', { name: 'Vehicles by ageing band' })
    expect(within(tiles).getByRole('button', { name: /0–30 days\s*1/ })).toBeInTheDocument()
    expect(within(tiles).getByRole('button', { name: /Over 90 days\s*0/ })).toBeInTheDocument()
    expect(
      within(tiles).getByRole('button', { name: /No reliable origin date\s*1/ }),
    ).toBeInTheDocument()
    expect(within(panel).getByText('Origin date after today: 1')).toBeInTheDocument()
    // The summary used the list's filters.
    expect(callsTo(spy, '/api/v1/vehicle-units/ageing-summary').at(-1)).toEqual({ brandId: 'b1' })

    // Choosing a band filters the list and the summary on the server.
    await user.click(within(tiles).getByRole('button', { name: /31–60 days/ }))
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/vehicle-units').at(-1)).toMatchObject({
        ageingBand: '31-60',
        brandId: 'b1',
      })
    })
    expect(callsTo(spy, '/api/v1/vehicle-units/ageing-summary').at(-1)).toMatchObject({
      ageingBand: '31-60',
    })
    expect(screen.getByLabelText('Ageing band')).toHaveValue('31-60')
    // Missing origin replaces the band filter (they cannot combine).
    await user.click(within(tiles).getByRole('button', { name: /No reliable origin date/ }))
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/vehicle-units').at(-1)).toMatchObject({
        originStatus: 'MISSING',
      })
    })
    expect(callsTo(spy, '/api/v1/vehicle-units').at(-1)).not.toHaveProperty('ageingBand')
    expect(screen.getByLabelText('Origin date')).toHaveValue('MISSING')
  })

  it('offers band filters from the server configuration only', async () => {
    const spy = apiMock()
    const user = userEvent.setup()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    const select = await screen.findByLabelText('Ageing band')
    await waitFor(() => {
      expect(
        within(select)
          .getAllByRole('option')
          .map((o) => (o as HTMLOptionElement).value),
      ).toEqual(['', '0-30', '31-60', '61-90', 'over-90'])
    })
    await user.selectOptions(select, 'over-90')
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/vehicle-units').at(-1)).toMatchObject({ ageingBand: 'over-90' })
    })
    await user.selectOptions(screen.getByLabelText('Origin date'), 'KNOWN')
    await waitFor(() => {
      expect(callsTo(spy, '/api/v1/vehicle-units').at(-1)).toMatchObject({
        originStatus: 'KNOWN',
        ageingBand: 'over-90',
      })
    })
  })

  it('works without a band configuration: no band filter, ages without bands', async () => {
    apiMock({
      configured: false,
      list: [{ ...base, ageing: { ...base.ageing, bandKey: null, bandLabel: null } }],
      summary: summary({
        configured: false,
        bandSet: null,
        bands: [],
        unbanded: 1,
        total: 1,
        withOrigin: 1,
        missingOrigin: 0,
        missingByReason: {},
      }),
    })
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    const select = await screen.findByLabelText('Ageing band')
    await waitFor(() => {
      expect(select).toBeDisabled()
    })
    expect(within(select).getByRole('option', { name: 'No bands configured' })).toBeInTheDocument()
    const panel = await screen.findByRole('region', { name: 'Stock ageing' })
    expect(within(panel).getByText(/No ageing bands are configured/)).toBeInTheDocument()
    expect(within(panel).getByText('Aged, no band')).toBeInTheDocument()
    expect((await screen.findAllByText('No band')).length).toBeGreaterThan(0)
  })

  it('shows a summary failure with a retry, without hiding the list', async () => {
    let fail = true
    const ok = apiMock()
    const real = ok.getMockImplementation()
    ok.mockImplementation((input, init) => {
      const p = new URL(input as string, 'http://localhost').pathname
      if (fail && p === '/api/v1/vehicle-units/ageing-summary')
        return Promise.resolve(err(500, 'INTERNAL_ERROR'))
      return (real as typeof fetch)(input, init)
    })
    const user = userEvent.setup()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Stock ageing could not be loaded')
    expect(await screen.findByRole('table', { name: 'Vehicle units' })).toBeInTheDocument()
    fail = false
    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(await screen.findByRole('region', { name: 'Stock ageing' })).toBeInTheDocument()
  })

  it('is denied without vehicle_units.view and reaches no ageing data', async () => {
    const spy = apiMock()
    renderApp('/inventory/vehicles', { me: me([]) })
    expect(await screen.findByText(/do not have permission/i)).toBeInTheDocument()
    expect(callsTo(spy, '/api/v1/vehicle-units/ageing-summary')).toEqual([])
  })
})

describe('vehicle detail stock ageing (E-07)', () => {
  it('shows age, band and origin', async () => {
    apiMock()
    renderApp('/inventory/vehicles/u1', { me: me(VIEW) })
    expect(await screen.findByText(/30 days \(as of/)).toBeInTheDocument()
    expect(screen.getByText(/\(Goods receipt\)/)).toBeInTheDocument()
  })

  it('explains a missing origin date instead of showing an age', async () => {
    apiMock()
    renderApp('/inventory/vehicles/u3', { me: me(VIEW) })
    expect(await screen.findByText('Why no age')).toBeInTheDocument()
    expect(screen.getByText('Origin date after today')).toBeInTheDocument()
    expect(screen.getByText(/after today’s business date/)).toBeInTheDocument()
    expect(screen.queryByText('Ageing band')).toBeNull()
  })
})
