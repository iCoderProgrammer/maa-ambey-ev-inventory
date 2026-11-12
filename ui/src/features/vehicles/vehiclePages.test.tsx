/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import type { Brand } from '@/features/catalogue/api/contracts'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/vehicleUnits'
import type {
  VehicleUnit,
  VehicleUnitDetail,
  VehicleUnitDocument,
  VehicleUnitMovement,
} from './api/vehicleUnits'
import { isSaleable, notSaleableReason } from './lib/presentation'

// D-08 vehicle list and detail screens against a mocked API. Placeholder data
// only ("Brand One", "CH-0001", 95000.00): nothing real.

const t = '2026-11-10T09:00:00Z'
const stamps = { version: 3, createdAt: t, updatedAt: t }
const loc = { id: 'l1', code: 'SR1', name: 'Showroom One' }

const unit: VehicleUnit = {
  id: 'u1',
  unitNumber: 'MAE-VEH-000001',
  status: 'RECEIVED_PENDING_PDI',
  brandId: 'b1',
  brandName: 'Brand One',
  modelId: 'm1',
  modelName: 'Model One',
  variantId: 'v1',
  variantName: 'Standard',
  chassisNumber: 'CH-0001',
  chassisNormalised: 'CH-0001',
  motorSerial: 'MOT-0001',
  batterySerial: 'BAT-0001',
  chargerSerial: null,
  locationId: 'l1',
  locationCode: 'SR1',
  locationName: 'Showroom One',
  sourceType: 'GOODS_RECEIPT',
  supplierId: 's1',
  supplierName: 'Example Supplier',
  receivedDate: '2026-11-10',
  ageDays: 2,
  unitCost: null,
  costCurrency: null,
  costHidden: true,
  ...stamps,
}
const available: VehicleUnit = {
  ...unit,
  id: 'u2',
  unitNumber: 'MAE-VEH-000002',
  status: 'AVAILABLE',
  chassisNumber: 'CH-0002',
  chassisNormalised: 'CH-0002',
  motorSerial: null,
  batterySerial: null,
}
const challan: VehicleUnitDocument = {
  attachmentId: 'a1',
  sourceType: 'GOODS_RECEIPT',
  sourceId: 'g1',
  sourceReference: 'GRN-000001',
  documentType: 'DELIVERY_CHALLAN',
  fileName: 'challan.pdf',
  contentType: 'application/pdf',
  sizeBytes: 2048,
  createdAt: t,
}
const detail = (over: Partial<VehicleUnitDetail> = {}): VehicleUnitDetail => ({
  ...unit,
  source: {
    type: 'GOODS_RECEIPT',
    supplierId: 's1',
    supplierName: 'Example Supplier',
    purchaseOrderId: 'po1',
    purchaseOrderNumber: 'PO-000001',
    purchaseOrderLineId: 'pl1',
    goodsReceiptId: 'g1',
    goodsReceiptNumber: 'GRN-000001',
    goodsReceiptLineId: 'gl1',
    goodsReceiptUnitId: 'gu1',
    supplierReference: 'DC-1',
    receivedDate: '2026-11-10',
  },
  events: [
    {
      id: 'e1',
      action: 'REGISTERED',
      toStatus: 'RECEIVED_PENDING_PDI',
      sourceType: 'GOODS_RECEIPT',
      sourceId: 'g1',
      sourceReference: 'GRN-000001',
      actorId: 'user-2',
      actorName: 'Example Poster',
      note: null,
      createdAt: t,
    },
  ],
  documents: [challan],
  documentsHidden: false,
  ...over,
})
const movement = (over: Partial<VehicleUnitMovement> = {}): VehicleUnitMovement => ({
  id: 'mv1',
  movementType: 'RECEIPT',
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
  actorId: 'user-2',
  actorName: 'Example Poster',
  note: null,
  createdAt: t,
  ...over,
})
const hold = movement({
  id: 'mv2',
  movementType: 'QUALITY_HOLD',
  fromStatus: 'RECEIVED_PENDING_PDI',
  toStatus: 'QUALITY_HOLD',
  fromLocationId: 'l1',
  fromLocationCode: 'SR1',
  fromLocationName: 'Showroom One',
  sourceType: 'PDI_INSPECTION',
  sourceId: 'i1',
  sourceReference: 'PDI-000001',
  actorId: 'user-1',
  note: 'Scratched panel',
})
const brand: Brand = {
  id: 'b1',
  name: 'Brand One',
  slug: 'brand-one',
  lifecycleStatus: 'ACTIVE',
  authorisationStatus: 'AUTHORISED',
  manufacturerLegalName: 'Example Manufacturer',
  authorisationReference: 'R',
  authorisationEvidenceId: null,
  verifiedBy: 'u-x',
  verifiedAt: t,
  operational: true,
  ...stamps,
}
const provenance = {
  confidence: 'SUPPLIER_CONFIRMED',
  sourceUrl: 'https://example.com',
  sourceDocumentId: null,
  sourceDate: '2026-10-01',
  verifiedBy: 'u-x',
  verifiedAt: t,
}
const model = {
  id: 'm1',
  brandId: 'b1',
  name: 'Model One',
  internalCode: 'M1',
  manufacturerCode: null,
  vehicleType: null,
  lifecycleStatus: 'ACTIVE',
  ...provenance,
  ...stamps,
}
const variant = {
  id: 'v1',
  brandId: 'b1',
  modelId: 'm1',
  name: 'Standard',
  variantCode: 'STD',
  manufacturerCode: null,
  lifecycleStatus: 'ACTIVE',
  ...provenance,
  ...stamps,
}
const supplier = {
  id: 's1',
  legalName: 'Example Supplier',
  tradeName: null,
  gstin: null,
  addressLine1: null,
  addressLine2: null,
  city: null,
  state: null,
  postalCode: null,
  status: 'VERIFIED',
  verificationReference: 'X',
  verificationDocumentId: 'a9',
  verifiedBy: 'u-x',
  verifiedAt: t,
  notes: null,
  createdBy: 'u-x',
  brandCount: 1,
  ...stamps,
}
const overview = {
  unit: {
    id: 'u1',
    unitNumber: unit.unitNumber,
    status: unit.status,
    brandId: 'b1',
    locationId: 'l1',
    version: 3,
    saleable: false,
  },
  allowedDispositions: [],
  canRecord: false,
  checklist: [],
  inspections: [],
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
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-7', details } }, { status })
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions,
  locationScope: { all: false, locations: [loc] },
})
const VIEW = ['vehicle_units.view']
const FULL = [
  ...VIEW,
  'catalogue.view',
  'suppliers.view',
  'purchase_orders.view',
  'goods_receipts.view',
  'attachments.view',
]

type Handler = (url: URL, init: RequestInit | undefined) => Response | Promise<Response> | undefined
interface State {
  list?: VehicleUnit[]
  detail?: VehicleUnitDetail
  movements?: VehicleUnitMovement[]
  me?: Me
}

function apiMock(state: State = {}, handler: Handler = () => undefined) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/me') return json(state.me ?? me(VIEW))
    if (p === '/api/v1/vehicle-units') return json(page(state.list ?? [unit, available]))
    if (p === '/api/v1/vehicle-units/u1') return json(state.detail ?? detail())
    if (p === '/api/v1/vehicle-units/u1/stock-movements')
      return json({ ...page(state.movements ?? [movement(), hold]), limit: 20 })
    if (p === '/api/v1/vehicle-units/u1/pdi') return json(overview)
    if (p === '/api/v1/brands') return json(page([brand]))
    if (p === '/api/v1/models') return json(page([model]))
    if (p === '/api/v1/variants') return json(page([variant]))
    if (p === '/api/v1/suppliers') return json(page([supplier]))
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}
type Spy = ReturnType<typeof apiMock>
/** The query parameters of every GET /vehicle-units request, in order. */
const listCalls = (spy: Spy) =>
  spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://x'))
    .filter((u) => u.pathname === '/api/v1/vehicle-units')
    .map((u) => Object.fromEntries(u.searchParams))
const lastList = (spy: Spy) => listCalls(spy).at(-1)

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('vehicle unit API client (D-08)', () => {
  it('sends the D-08 filters and reads a unit’s movements', async () => {
    const spy = apiMock()
    await api.listVehicleUnits({
      modelId: 'm1',
      receivedFrom: '2026-11-01',
      receivedTo: '2026-11-30',
      minAgeDays: 0,
      maxAgeDays: 30,
      sort: '-receivedDate',
    })
    expect(lastList(spy)).toEqual({
      modelId: 'm1',
      receivedFrom: '2026-11-01',
      receivedTo: '2026-11-30',
      minAgeDays: '0',
      maxAgeDays: '30',
      sort: '-receivedDate',
    })
    const m = await api.listVehicleUnitMovements('u1', { order: 'desc', page: 2, limit: 20 })
    expect(m.items.map((x) => x.movementType)).toEqual(['RECEIPT', 'QUALITY_HOLD'])
    const url = new URL(spy.mock.calls.at(-1)?.[0] as string, 'http://x')
    expect(url.pathname).toBe('/api/v1/vehicle-units/u1/stock-movements')
    expect(Object.fromEntries(url.searchParams)).toEqual({ order: 'desc', page: '2', limit: '20' })
  })

  it('refuses hidden documents that list files, and unknown movement types', async () => {
    apiMock({ detail: detail({ documentsHidden: true }) })
    await expect(api.getVehicleUnit('u1')).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    cleanup()
    vi.restoreAllMocks()
    apiMock({ movements: [movement({ movementType: 'SALE' as 'RECEIPT' })] })
    await expect(api.listVehicleUnitMovements('u1')).rejects.toMatchObject({
      code: 'INVALID_RESPONSE',
    })
  })

  it('matches the server contract for documents, movements and list parameters (docs/openapi)', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<string, Record<string, { parameters?: { name: string; enum?: string[] }[] }>>
    }
    const keys = (n: string) => Object.keys(spec.definitions[n]?.properties ?? {}).sort()
    expect(Object.keys(challan).sort()).toEqual(keys('VehicleUnitDocument'))
    expect(Object.keys(movement()).sort()).toEqual(keys('VehicleUnitMovement'))
    expect(Object.keys(detail()).sort()).toEqual(keys('VehicleUnitDetail'))
    expect(spec.definitions.VehicleUnitMovement?.properties.movementType?.enum).toEqual(
      api.unitMovementTypeSchema.options,
    )
    const params = spec.paths['/vehicle-units']?.get?.parameters ?? []
    for (const n of ['modelId', 'receivedFrom', 'receivedTo', 'minAgeDays', 'maxAgeDays', 'sort'])
      expect(params.map((p) => p.name)).toContain(n)
    expect(params.find((p) => p.name === 'sort')?.enum).toEqual([...api.vehicleUnitSorts])
    expect(
      spec.paths['/vehicle-units/{id}/stock-movements']?.get?.parameters?.map((p) => p.name),
    ).toEqual(['id', 'order', 'page', 'limit'])
  })
})

describe('vehicle status presentation', () => {
  it('treats AVAILABLE as the only saleable status and explains every other one', () => {
    for (const s of api.unitStatusSchema.options) {
      expect(isSaleable(s)).toBe(s === 'AVAILABLE')
      if (s !== 'AVAILABLE') expect(notSaleableReason[s]).toMatch(/not/i)
    }
  })
})

describe('vehicle list page', () => {
  it('lists real units with identifiers, status, location, supplier and age; no cost without permission', async () => {
    apiMock()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    const table = await screen.findByRole('table', { name: 'Vehicle units' })
    const row = within(table).getByRole('link', { name: 'MAE-VEH-000001' }).closest('tr')
    expect(row).not.toBeNull()
    const r = within(row as HTMLElement)
    expect(r.getByText('Brand One · Model One · Standard')).toBeInTheDocument()
    expect(r.getByText('CH-0001')).toBeInTheDocument()
    expect(r.getByText('Motor MOT-0001 · Battery BAT-0001')).toBeInTheDocument()
    expect(r.getByText('Awaiting PDI')).toBeInTheDocument()
    expect(r.getByText('Not saleable')).toBeInTheDocument()
    expect(r.getByText('SR1 — Showroom One')).toBeInTheDocument()
    expect(r.getByText('Example Supplier')).toBeInTheDocument()
    expect(r.getByText('2 days in stock')).toBeInTheDocument()
    // The available unit carries no "not saleable" marker.
    const other = within(table).getByRole('link', { name: 'MAE-VEH-000002' }).closest('tr')
    expect(within(other as HTMLElement).queryByText('Not saleable')).toBeNull()
    expect(within(table).queryByRole('columnheader', { name: 'Cost' })).toBeNull()
    // Phones get the same rows as a labelled list (no sideways scrolling).
    expect(screen.getByRole('list', { name: 'Vehicle units' }).children).toHaveLength(2)
    // No reservation, sale or status-change action is offered.
    expect(screen.queryByRole('button', { name: /reserve|sell|transfer/i })).toBeNull()
    expect(screen.getByRole('link', { name: 'Vehicle Units' })).toBeInTheDocument()
  })

  it('shows the cost column only with the cost permission, never an amount the server hid', async () => {
    apiMock({
      list: [unit, { ...available, unitCost: '95000.00', costCurrency: 'INR', costHidden: false }],
    })
    renderApp('/inventory/vehicles', { me: me([...VIEW, 'vehicle_units.cost.view']) })
    const table = await screen.findByRole('table', { name: 'Vehicle units' })
    expect(within(table).getByRole('columnheader', { name: 'Cost' })).toBeInTheDocument()
    expect(within(table).getByText('₹95,000.00')).toBeInTheDocument()
    expect(within(table).getByText('Recorded (not visible to you)')).toBeInTheDocument()
  })

  it('searches on the server after a pause and returns to page 1', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    renderApp('/inventory/vehicles?page=3', { me: me(VIEW) })
    await screen.findByRole('table', { name: 'Vehicle units' })
    expect(lastList(spy)).toMatchObject({ page: '3' })
    await user.type(
      screen.getByLabelText('Search unit number, chassis, motor or battery'),
      'bat-0001',
    )
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ q: 'bat-0001', page: '1', limit: '25' })
    })
    // One request for the finished term, not one per keystroke.
    expect(listCalls(spy).filter((c) => c.q !== undefined)).toHaveLength(1)
  })

  it('keeps a filter chosen while a search is still pending (regression)', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    await screen.findByRole('table', { name: 'Vehicle units' })
    await user.type(screen.getByLabelText('Search unit number, chassis, motor or battery'), 'ch-')
    // Choose a status before the search pause ends: the search must not drop it.
    await user.selectOptions(screen.getByLabelText('Status'), 'QUALITY_HOLD')
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ q: 'ch-', status: 'QUALITY_HOLD', page: '1', limit: '25' })
    })
  })

  it('combines filters from real option lists and clears them all', async () => {
    const user = userEvent.setup()
    const spy = apiMock({}, () => undefined)
    const { router } = renderApp('/inventory/vehicles', { me: me(FULL) })
    await screen.findByRole('table', { name: 'Vehicle units' })
    // Model and variant wait for their parent.
    expect(screen.getByLabelText('Model')).toBeDisabled()
    await user.selectOptions(screen.getByLabelText('Status'), 'RECEIVED_PENDING_PDI')
    await user.selectOptions(await screen.findByLabelText('Brand'), 'b1')
    await waitFor(() => {
      expect(screen.getByLabelText('Model')).toBeEnabled()
    })
    await user.selectOptions(
      screen.getByLabelText('Model'),
      await screen.findByRole('option', { name: 'Model One' }),
    )
    await user.selectOptions(
      screen.getByLabelText('Variant'),
      await screen.findByRole('option', { name: 'Standard' }),
    )
    await user.selectOptions(screen.getByLabelText('Location'), 'l1')
    await user.selectOptions(screen.getByLabelText('Supplier'), 's1')
    await user.selectOptions(screen.getByLabelText('Saleable'), 'false')
    await user.type(screen.getByLabelText('Received from'), '2026-11-01')
    await user.type(screen.getByLabelText('Min. days in stock'), '1')
    await user.type(screen.getByLabelText('Max. days in stock'), '30{Enter}')
    await user.selectOptions(screen.getByLabelText('Sort by'), '-receivedDate')
    await waitFor(() => {
      expect(lastList(spy)).toEqual({
        status: 'RECEIVED_PENDING_PDI',
        saleable: 'false',
        brandId: 'b1',
        modelId: 'm1',
        variantId: 'v1',
        locationId: 'l1',
        supplierId: 's1',
        receivedFrom: '2026-11-01',
        minAgeDays: '1',
        maxAgeDays: '30',
        sort: '-receivedDate',
        page: '1',
        limit: '25',
      })
    })
    // The filters live in the URL (refresh and shared links keep them).
    expect(router.state.location.search).toContain('brandId=b1')
    // Changing the brand resets model and variant.
    await user.selectOptions(screen.getByLabelText('Brand'), '')
    await waitFor(() => {
      expect(lastList(spy)?.modelId).toBeUndefined()
    })
    await user.click(screen.getByRole('button', { name: 'Clear filters' }))
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ page: '1', limit: '25' })
    })
    expect(router.state.location.search).toBe('')
  })

  it('disables option lists the caller may not read instead of guessing values', async () => {
    const spy = apiMock()
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    await screen.findByRole('table', { name: 'Vehicle units' })
    expect(screen.getByLabelText('Brand')).toBeDisabled()
    expect(screen.getByLabelText('Supplier')).toBeDisabled()
    const paths = spy.mock.calls.map(([u]) => new URL(u as string, 'http://x').pathname)
    expect(paths).not.toContain('/api/v1/brands')
    expect(paths).not.toContain('/api/v1/suppliers')
  })

  it('pages on the server', async () => {
    const user = userEvent.setup()
    const spy = apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units'
        ? Response.json({
            data: page([unit], { page: Number(url.searchParams.get('page') ?? '1'), total: 30 }),
          })
        : undefined,
    )
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    await screen.findByText('1–25 of 30 vehicles')
    await user.click(screen.getByRole('button', { name: /next/i }))
    await waitFor(() => {
      expect(lastList(spy)).toEqual({ page: '2', limit: '25' })
    })
  })

  it('shows a skeleton while loading, then the empty inventory or no-results state', async () => {
    let release: (r: Response) => void = () => undefined
    apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units'
        ? new Promise<Response>((r) => {
            release = r
          })
        : undefined,
    )
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    expect(await screen.findByRole('status', { name: 'Loading vehicle units' })).toBeInTheDocument()
    release(Response.json({ data: page([]) }))
    expect(await screen.findByText('No vehicle units yet')).toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()
    apiMock({ list: [] })
    renderApp('/inventory/vehicles?q=nothing', { me: me(VIEW) })
    expect(await screen.findByText('No vehicles match these filters')).toBeInTheDocument()
  })

  it('reports invalid filters with the server’s field messages', async () => {
    apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units'
        ? err(422, 'VALIDATION_FAILED', [
            { field: 'maxAgeDays', message: 'must not be less than minAgeDays' },
          ])
        : undefined,
    )
    renderApp('/inventory/vehicles?minAgeDays=10&maxAgeDays=2', { me: me(VIEW) })
    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText('Max. days in stock')).toBeInTheDocument()
    expect(within(alert).getByText(/must not be less than minAgeDays/)).toBeInTheDocument()
  })

  it('offers a retry after a server error and recovers', async () => {
    const user = userEvent.setup()
    let fail = true
    apiMock({}, (url) => {
      if (url.pathname !== '/api/v1/vehicle-units' || !fail) return undefined
      return err(500, 'INTERNAL_ERROR')
    })
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    expect(await screen.findByText('Could not load vehicle units')).toBeInTheDocument()
    fail = false
    await user.click(screen.getByRole('button', { name: 'Try again' }))
    expect(await screen.findByRole('table', { name: 'Vehicle units' })).toBeInTheDocument()
  })

  it('denies access without the permission (no request) and on a 403 from the server', async () => {
    const spy = apiMock()
    renderApp('/inventory/vehicles', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(listCalls(spy)).toHaveLength(0)
    cleanup()
    vi.restoreAllMocks()
    apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units' ? err(403, 'FORBIDDEN') : undefined,
    )
    renderApp('/inventory/vehicles', { me: me(VIEW) })
    expect(
      await screen.findByText(/You do not have permission to view vehicle units/),
    ).toBeInTheDocument()
  })

  it('sends the user to sign in when the session has ended (401)', async () => {
    apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units' ? err(401, 'SESSION_EXPIRED') : undefined,
    )
    const { router } = renderApp('/inventory/vehicles', { me: me(VIEW) })
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/login')
    })
  })
})

describe('vehicle detail page', () => {
  it('shows overview, identifiers, procurement links, documents, PDI and movement history', async () => {
    apiMock()
    renderApp('/inventory/vehicles/u1', { me: me(FULL) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'MAE-VEH-000001' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Awaiting PDI — not saleable until it passes inspection.'),
    ).toBeInTheDocument()
    expect(screen.getByText('u1')).toBeInTheDocument() // internal ID
    expect(screen.getByText('CH-0001')).toBeInTheDocument()
    expect(screen.getByText('MOT-0001')).toBeInTheDocument()
    expect(screen.getByText('BAT-0001')).toBeInTheDocument()
    expect(screen.getAllByText('Not recorded')).toHaveLength(1) // charger
    expect(screen.getByRole('link', { name: 'PO-000001' })).toHaveAttribute(
      'href',
      '/procurement/purchase-orders/po1',
    )
    const grnLinks = screen.getAllByRole('link', { name: 'GRN-000001' })
    expect(grnLinks[0]).toHaveAttribute('href', '/procurement/goods-receipts/g1')
    expect(screen.getByText(/by Example Poster/)).toBeInTheDocument()
    // Documents the server listed, with a private download.
    const docs = screen.getByRole('list', { name: 'Vehicle documents' })
    expect(within(docs).getByText('challan.pdf')).toBeInTheDocument()
    expect(within(docs).getByText(/Delivery challan · GRN-000001 · 2.0 KB/)).toBeInTheDocument()
    // PDI: the shared inspection history and the PDI screen link.
    expect(await screen.findByText('No inspection recorded yet')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open PDI' })).toHaveAttribute(
      'href',
      '/procurement/pdi/u1',
    )
    // Ledger history, oldest first, with sources linked.
    const history = await screen.findByRole('list', { name: 'Stock movements' })
    const entries = within(history).getAllByRole('listitem')
    expect(entries).toHaveLength(2)
    expect(within(entries[0] as HTMLElement).getByText('Received into stock')).toBeInTheDocument()
    expect(within(entries[1] as HTMLElement).getByText('Put on quality hold')).toBeInTheDocument()
    expect(within(entries[1] as HTMLElement).getByText('Scratched panel')).toBeInTheDocument()
    expect(within(entries[1] as HTMLElement).getByText(/You/)).toBeInTheDocument()
    expect(
      within(entries[1] as HTMLElement).getByRole('link', { name: 'PDI-000001' }),
    ).toHaveAttribute('href', '/procurement/pdi/u1')
    // Read-only: no edit, delete, reserve or sell actions anywhere.
    expect(screen.queryByRole('button', { name: /edit|delete|reserve|sell/i })).toBeNull()
  })

  it('downloads a document through the attachment API', async () => {
    const user = userEvent.setup()
    URL.createObjectURL = vi.fn(() => 'blob:x')
    URL.revokeObjectURL = vi.fn()
    const spy = apiMock({}, (url) => {
      if (url.pathname === '/api/v1/attachments/a1/link')
        return Response.json({
          data: {
            url: '/api/v1/attachments/a1/content?sig=x',
            expiresAt: t,
            disposition: 'attachment',
          },
        })
      if (url.pathname === '/api/v1/attachments/a1/content')
        return new Response(new Blob(['%PDF']), { headers: { 'Content-Type': 'application/pdf' } })
      return undefined
    })
    renderApp('/inventory/vehicles/u1', { me: me(FULL) })
    await user.click(await screen.findByRole('button', { name: 'Download challan.pdf' }))
    await waitFor(() => {
      expect(spy.mock.calls.some(([u]) => (u as string).includes('/attachments/a1/content'))).toBe(
        true,
      )
    })
  })

  it('hides documents and links the caller may not open', async () => {
    apiMock({ detail: detail({ documents: [], documentsHidden: true }) })
    renderApp('/inventory/vehicles/u1', { me: me(VIEW) })
    expect(
      await screen.findByText('You do not have permission to open documents, so none are listed.'),
    ).toBeInTheDocument()
    // Without purchase order / goods receipt view, references are text, not links.
    expect(screen.getByText('PO-000001').closest('a')).toBeNull()
    expect(screen.queryByRole('link', { name: 'GRN-000001' })).toBeNull()
  })

  it('shows the cost only when the server sends it', async () => {
    apiMock({ detail: detail({ unitCost: '95000.00', costCurrency: 'INR', costHidden: false }) })
    renderApp('/inventory/vehicles/u1', { me: me([...VIEW, 'vehicle_units.cost.view']) })
    expect(await screen.findByText('₹95,000.00')).toBeInTheDocument()
    cleanup()
    vi.restoreAllMocks()
    apiMock()
    renderApp('/inventory/vehicles/u1', { me: me(VIEW) })
    await screen.findByRole('heading', { level: 1, name: 'MAE-VEH-000001' })
    expect(screen.queryByText(/₹/)).toBeNull()
    expect(screen.getByText('Recorded (not visible to you)')).toBeInTheDocument()
  })

  it('pages the movement history on the server', async () => {
    const user = userEvent.setup()
    const spy = apiMock({}, (url) =>
      url.pathname === '/api/v1/vehicle-units/u1/stock-movements'
        ? Response.json({
            data: page([movement()], {
              page: Number(url.searchParams.get('page') ?? '1'),
              limit: 20,
              total: 25,
            }),
          })
        : undefined,
    )
    renderApp('/inventory/vehicles/u1', { me: me(VIEW) })
    await screen.findByText('1–20 of 25 movements')
    await user.click(screen.getByRole('button', { name: /next/i }))
    await waitFor(() => {
      const last = spy.mock.calls
        .map(([u]) => new URL(u as string, 'http://x'))
        .filter((u) => u.pathname.endsWith('/stock-movements'))
        .at(-1)
      expect(Object.fromEntries(last?.searchParams ?? [])).toEqual({
        order: 'asc',
        page: '2',
        limit: '20',
      })
    })
  })

  it('says a unit was not found without hinting that it exists elsewhere', async () => {
    apiMock()
    renderApp('/inventory/vehicles/nope', { me: me(VIEW) })
    expect(await screen.findByText('Vehicle unit not found')).toBeInTheDocument()
    expect(screen.getByText(/not available to you/)).toBeInTheDocument()
  })

  it('keeps the list filters when going back from a vehicle', async () => {
    const user = userEvent.setup()
    apiMock()
    const { router } = renderApp('/inventory/vehicles?status=RECEIVED_PENDING_PDI', {
      me: me(VIEW),
    })
    const table = await screen.findByRole('table', { name: 'Vehicle units' })
    await user.click(within(table).getByRole('link', { name: 'MAE-VEH-000001' }))
    await screen.findByRole('heading', { level: 1, name: 'MAE-VEH-000001' })
    await user.click(screen.getByRole('link', { name: 'Back to vehicle units' }))
    await waitFor(() => {
      expect(router.state.location.search).toBe('?status=RECEIVED_PENDING_PDI')
    })
  })
})
