/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import * as api from './api/stockItems'
import type { ItemCategory, StockItem, StockItemDetail } from './api/stockItems'
import { stockItemKeys } from './hooks/useStockItems'

// Placeholder data only ("Brake parts", "BRK-PAD-01", "Example Supplier"): nothing real.

const t = '2026-10-27T09:00:00Z'
const category: ItemCategory = {
  id: 'c1',
  name: 'Brake parts',
  description: null,
  status: 'ACTIVE',
  archivedBy: null,
  archivedAt: null,
  archiveReason: null,
  activeItemCount: 1,
  createdBy: 'u-x',
  version: 1,
  createdAt: t,
  updatedAt: t,
}
const item: StockItem = {
  id: 'i1',
  sku: 'BRK-PAD-01',
  skuNormalised: 'BRK-PAD-01',
  name: 'Brake pad set',
  description: null,
  categoryId: 'c1',
  categoryName: 'Brake parts',
  categoryStatus: 'ACTIVE',
  uom: 'SET',
  trackingMode: 'QUANTITY',
  reorderLevel: '2.500',
  status: 'ACTIVE',
  archivedBy: null,
  archivedAt: null,
  archiveReason: null,
  createdBy: 'u-x',
  version: 3,
  createdAt: t,
  updatedAt: t,
}
const detail: StockItemDetail = {
  ...item,
  suppliers: [
    {
      id: 'l1',
      supplierId: 's1',
      supplierName: 'Example Supplier One',
      supplierStatus: 'PENDING_VERIFICATION',
      supplierPartNumber: 'SP-9',
      createdBy: 'u-x',
      createdAt: t,
      endedBy: null,
      endedAt: null,
    },
    {
      id: 'l0',
      supplierId: 's2',
      supplierName: 'Example Supplier Two',
      supplierStatus: 'VERIFIED',
      supplierPartNumber: null,
      createdBy: 'u-x',
      createdAt: t,
      endedBy: 'u-x',
      endedAt: t,
    },
  ],
  suppliersHidden: false,
  compatibility: [
    {
      id: 'm1',
      brandId: 'b1',
      brandName: 'Brand One',
      brandStatus: 'PENDING_VERIFICATION',
      modelId: null,
      modelName: null,
    },
  ],
}
const hidden: StockItemDetail = { ...detail, suppliers: [], suppliersHidden: true }
const archived: StockItemDetail = {
  ...detail,
  status: 'ARCHIVED',
  archivedBy: 'u-x',
  archivedAt: t,
  archiveReason: 'Discontinued',
}

const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({ ...testMe, permissions })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-8', details } }, { status })
const brand = (id: string, name: string, lifecycleStatus = 'PENDING_VERIFICATION') => ({
  id,
  name,
  slug: id,
  lifecycleStatus,
  authorisationStatus: 'UNVERIFIED',
  manufacturerLegalName: null,
  authorisationReference: null,
  authorisationEvidenceId: null,
  verifiedBy: null,
  verifiedAt: null,
  operational: false,
  version: 1,
  createdAt: t,
  updatedAt: t,
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

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function apiMock(handler: Handler = () => undefined, d: StockItemDetail = detail) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/stock-items') return json(page([item]))
    if (p === '/api/v1/stock-items/i1') return json(d)
    if (p === '/api/v1/stock-item-categories') return json(page([category]))
    if (p === '/api/v1/suppliers')
      return json(
        page([
          supplier('s1', 'Example Supplier One', 'PENDING_VERIFICATION'),
          supplier('s2', 'Example Supplier Two', 'VERIFIED'),
          supplier('s3', 'Example Supplier Old', 'ARCHIVED'),
        ]),
      )
    if (p === '/api/v1/brands')
      return json(page([brand('b1', 'Brand One'), brand('b9', 'Old Brand', 'ARCHIVED')]))
    if (p === '/api/v1/models')
      return json(
        page([
          {
            id: 'mo1',
            brandId: url.searchParams.get('brandId') ?? 'b1',
            name: 'Model One',
            manufacturerCode: null,
            internalCode: 'M1',
            vehicleType: 'TWO_WHEELER_EV',
            lifecycleStatus: 'PENDING_VERIFICATION',
            confidence: 'UNVERIFIED',
            sourceUrl: null,
            sourceDate: null,
            sourceDocumentId: null,
            verifiedBy: null,
            verifiedAt: null,
            version: 1,
            createdAt: t,
            updatedAt: t,
          },
        ]),
      )
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}

const bodyOf = (init: RequestInit | undefined): unknown =>
  typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
const calls = (spy: ReturnType<typeof apiMock>, method: string, p: string) =>
  spy.mock.calls.filter(
    ([u, init]) =>
      (init?.method ?? 'GET') === method && new URL(u as string, 'http://x').pathname === p,
  )

const VIEW = ['stock_items.view']
const ALL = [
  'stock_items.view',
  'stock_items.manage',
  'stock_items.archive',
  'suppliers.view',
  'catalogue.view',
]

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SKU API client', () => {
  it('sends list filters and paging, skipping empty ones', async () => {
    const spy = apiMock()
    await api.listStockItems({
      q: 'brk',
      status: 'ACTIVE',
      categoryId: 'c1',
      trackingMode: 'SERIAL',
      supplierId: 's1',
      brandId: 'b1',
      page: 2,
      limit: 50,
    })
    const url = new URL(spy.mock.calls[0]?.[0] as string, 'http://x')
    expect(url.pathname).toBe('/api/v1/stock-items')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      q: 'brk',
      status: 'ACTIVE',
      categoryId: 'c1',
      trackingMode: 'SERIAL',
      supplierId: 's1',
      brandId: 'b1',
      page: '2',
      limit: '50',
    })
    await api.listCategories({ q: '', status: undefined })
    expect(new URL(spy.mock.calls[1]?.[0] as string, 'http://x').search).toBe('')
  })

  it('uses the documented methods and paths, and has no delete, stock or reactivation call', async () => {
    const spy = apiMock(() => Response.json({ data: detail }))
    await api.createStockItem({
      sku: 'BRK-PAD-01',
      name: 'x',
      categoryId: 'c1',
      uom: 'SET',
      trackingMode: 'QUANTITY',
    })
    await api.updateStockItem('i 1', { version: 3, name: 'y' })
    await api.archiveStockItem('i1', { version: 3, reason: 'r' })
    spy.mockImplementation(() => Promise.resolve(Response.json({ data: category })))
    await api.createCategory({ name: 'Brake parts' })
    await api.updateCategory('c1', { version: 1, name: 'B' })
    await api.archiveCategory('c1', { version: 1, reason: 'r' })
    expect(
      spy.mock.calls.map(
        ([u, init]) => `${init?.method ?? 'GET'} ${new URL(u as string, 'http://x').pathname}`,
      ),
    ).toEqual([
      'POST /api/v1/stock-items',
      'PATCH /api/v1/stock-items/i%201',
      'POST /api/v1/stock-items/i1/archive',
      'POST /api/v1/stock-item-categories',
      'PATCH /api/v1/stock-item-categories/c1',
      'POST /api/v1/stock-item-categories/c1/archive',
    ])
    const fns = Object.entries(api)
      .filter(([, v]) => typeof v === 'function')
      .map(([k]) => k)
    expect(
      fns.filter((k) => /delete|remove|reactivat|stock(In|Out)|receipt|issue|balance/i.test(k)),
    ).toEqual([])
  })

  it('rejects answers that break the contract (hidden suppliers listed, unknown mode)', async () => {
    apiMock(() => Response.json({ data: { ...detail, suppliersHidden: true } }))
    await expect(api.getStockItem('i1')).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    vi.restoreAllMocks()
    apiMock(() => Response.json({ data: { ...detail, trackingMode: 'BATCH' } }))
    await expect(api.getStockItem('i1')).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })

  it('matches the server contract field for field (docs/openapi)', () => {
    const spec = JSON.parse(
      readFileSync(
        path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json'),
        'utf8',
      ),
    ) as { definitions: Record<string, { properties: Record<string, { enum?: string[] }> }> }
    const keys = (n: string) => Object.keys(spec.definitions[n]?.properties ?? {}).sort()
    expect(Object.keys(item).sort()).toEqual(keys('StockItem'))
    expect(Object.keys(detail).sort()).toEqual(keys('StockItemDetail'))
    expect(Object.keys(category).sort()).toEqual(keys('StockItemCategory'))
    expect(Object.keys(detail.suppliers[0] ?? {}).sort()).toEqual(keys('StockItemSupplier'))
    expect(Object.keys(detail.compatibility[0] ?? {}).sort()).toEqual(
      keys('StockItemCompatibility'),
    )
    expect(keys('StockItemCreate')).toEqual([
      'categoryId',
      'compatibility',
      'description',
      'name',
      'reorderLevel',
      'sku',
      'suppliers',
      'trackingMode',
      'uom',
    ])
    expect(keys('StockItemUpdate')).not.toContain('sku')
    expect(keys('StockItemUpdate')).not.toContain('trackingMode')
    expect(api.trackingModeSchema.options).toEqual(
      spec.definitions.StockItem?.properties.trackingMode?.enum,
    )
  })

  it('normalises like the server: trim and upper-case only, separators kept', () => {
    expect(api.normaliseSku('  brk-pad-01 ')).toBe('BRK-PAD-01')
    expect(new Set(['BRK-01', 'BRK.01', 'BRK01'].map(api.normaliseSku)).size).toBe(3)
    for (const ok of ['BRK-01', 'a.b/c_d-1', 'X']) expect(api.SKU_PATTERN.test(ok)).toBe(true)
    for (const bad of ['BRK 01', '-BRK', 'BRK#1', 'A'.repeat(41), ''])
      expect(api.SKU_PATTERN.test(bad)).toBe(false)
    for (const ok of ['PCS', 'SET', 'M2', 'L']) expect(api.UOM_PATTERN.test(ok)).toBe(true)
    for (const bad of ['pcs', '2L', 'PER PIECE', 'ABCDEFGHIJK'])
      expect(api.UOM_PATTERN.test(bad)).toBe(false)
    expect(api.isWholeNumber('5.000')).toBe(true)
    expect(api.isWholeNumber('5.5')).toBe(false)
    expect(api.formatLevel('2.500')).toBe('2.5')
    expect(api.formatLevel('5.000')).toBe('5')
    expect(stockItemKeys.list({ page: 1 })).not.toEqual(stockItemKeys.list({ page: 2 }))
  })
})

describe('SKU list', () => {
  it('is denied without stock_items.view and never calls the API', async () => {
    const spy = apiMock()
    renderApp('/inventory/parts', { me: me(['suppliers.view', 'catalogue.view']) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/stock-item'))).toBe(false)
  })

  it('lists SKUs read-only for a viewer; supplier and brand filters need their permissions', async () => {
    const user = userEvent.setup()
    const spy = apiMock()
    renderApp('/inventory/parts', { me: me(VIEW) })
    const table = await screen.findByRole('table', { name: 'SKUs' })
    expect(within(table).getByText('BRK-PAD-01')).toBeInTheDocument()
    expect(within(table).getByText('2.5')).toBeInTheDocument()
    expect(within(table).getByText('Quantity')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent(/records no stock/)
    expect(screen.queryByRole('button', { name: 'Add SKU' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Supplier')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Compatible brand')).not.toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => /\/suppliers|\/brands/.test(u as string))).toBe(false)
    await user.selectOptions(screen.getByLabelText('Tracking'), 'SERIAL')
    await waitFor(() => {
      expect(
        spy.mock.calls.some(
          ([u]) => new URL(u as string, 'http://x').searchParams.get('trackingMode') === 'SERIAL',
        ),
      ).toBe(true)
    })
  })

  it('says when filters match nothing, and when there are no SKUs yet', async () => {
    apiMock((u) =>
      u.pathname === '/api/v1/stock-items' ? Response.json({ data: page([]) }) : undefined,
    )
    renderApp('/inventory/parts?status=ARCHIVED', { me: me(VIEW) })
    expect(await screen.findByText('No SKUs match these filters')).toBeInTheDocument()
  })
})

// Long typed flows: allow for a busy machine (each takes about 1–2 s alone).
describe('SKU editor', { timeout: 15_000 }, () => {
  it('validates before sending, then creates with trimmed code, upper-cased unit and links', async () => {
    const user = userEvent.setup()
    let sent: unknown
    const spy = apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-items' && init?.method === 'POST') {
        sent = bodyOf(init)
        return Response.json({ data: detail }, { status: 201 })
      }
      return undefined
    })
    const { router } = renderApp('/inventory/parts', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Add SKU' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add SKU' })
    await user.click(within(dialog).getByRole('button', { name: 'Add SKU' }))
    for (const f of [
      'SKU is required',
      'Name is required',
      'Category is required',
      'Choose how the item is tracked',
    ])
      expect(await within(dialog).findByText(f)).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/SKU code/), 'BRK PAD')
    await user.type(within(dialog).getByLabelText(/^Name/), 'Brake pad set')
    await user.selectOptions(within(dialog).getByLabelText(/Category/), 'c1')
    await user.type(within(dialog).getByLabelText(/Unit of measure/), 'per piece')
    await user.selectOptions(within(dialog).getByLabelText(/Tracking mode/), 'SERIAL')
    await user.type(within(dialog).getByLabelText(/Reorder level/), '2.5')
    await user.click(within(dialog).getByRole('button', { name: 'Add SKU' }))
    expect(
      await within(dialog).findByText("1–40 letters, digits, '.', '_', '/' or '-' — no spaces"),
    ).toBeInTheDocument()
    expect(within(dialog).getByText(/1–10 letters or digits/)).toBeInTheDocument()
    expect(within(dialog).getByText(/whole number/)).toBeInTheDocument()
    expect(calls(spy, 'POST', '/api/v1/stock-items')).toHaveLength(0)
    await user.clear(within(dialog).getByLabelText(/SKU code/))
    await user.type(within(dialog).getByLabelText(/SKU code/), '  brk.pad-01 ')
    await user.clear(within(dialog).getByLabelText(/Unit of measure/))
    await user.type(within(dialog).getByLabelText(/Unit of measure/), 'pcs')
    await user.clear(within(dialog).getByLabelText(/Reorder level/))
    await user.type(within(dialog).getByLabelText(/Reorder level/), '4')
    // Suppliers: archived ones are not offered; pending ones say so.
    await user.click(within(dialog).getByRole('button', { name: 'Add supplier' }))
    const sel = await within(dialog).findByLabelText('Supplier 1')
    await waitFor(() => {
      expect(
        within(sel).getByRole('option', { name: /Example Supplier One \(pending verification\)/ }),
      ).toBeInTheDocument()
    })
    expect(within(sel).queryByRole('option', { name: /Old/ })).not.toBeInTheDocument()
    await user.selectOptions(sel, 's1')
    // Compatibility: models come from the catalogue for the chosen brand only.
    await user.click(within(dialog).getByRole('button', { name: 'Add compatibility' }))
    const b = await within(dialog).findByLabelText('Brand 1')
    await waitFor(() => {
      expect(within(b).getByRole('option', { name: 'Brand One' })).toBeInTheDocument()
    })
    expect(within(b).queryByRole('option', { name: 'Old Brand' })).not.toBeInTheDocument()
    await user.selectOptions(b, 'b1')
    const m = within(dialog).getByLabelText('Model 1')
    await waitFor(() => {
      expect(within(m).getByRole('option', { name: 'Model One' })).toBeInTheDocument()
    })
    expect(
      spy.mock.calls.some(
        ([u]) => new URL(u as string, 'http://x').searchParams.get('brandId') === 'b1',
      ),
    ).toBe(true)
    await user.selectOptions(m, 'mo1')
    await user.click(within(dialog).getByRole('button', { name: 'Add SKU' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/inventory/parts/i1')
    })
    expect(sent).toEqual({
      sku: 'brk.pad-01',
      name: 'Brake pad set',
      description: null,
      categoryId: 'c1',
      uom: 'PCS',
      trackingMode: 'SERIAL',
      reorderLevel: '4',
      suppliers: [{ supplierId: 's1', supplierPartNumber: null }],
      compatibility: [{ brandId: 'b1', modelId: 'mo1' }],
    })
  })

  it('puts a server duplicate on the SKU field and keeps every entry', async () => {
    const user = userEvent.setup()
    apiMock((u, init) =>
      u.pathname === '/api/v1/stock-items' && init?.method === 'POST'
        ? err(409, 'DUPLICATE', [{ field: 'sku', message: 'is already used' }])
        : undefined,
    )
    renderApp('/inventory/parts', { me: me(VIEW.concat('stock_items.manage')) })
    await user.click(await screen.findByRole('button', { name: 'Add SKU' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add SKU' })
    await user.type(within(dialog).getByLabelText(/SKU code/), 'brk-pad-01')
    await user.type(within(dialog).getByLabelText(/^Name/), 'Brake pad set')
    await user.selectOptions(within(dialog).getByLabelText(/Category/), 'c1')
    await user.type(within(dialog).getByLabelText(/Unit of measure/), 'SET')
    await user.selectOptions(within(dialog).getByLabelText(/Tracking mode/), 'QUANTITY')
    await user.click(within(dialog).getByRole('button', { name: 'Add SKU' }))
    expect(await within(dialog).findByText('Already used — choose another')).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/SKU code/)).toHaveValue('brk-pad-01')
    expect(within(dialog).getByLabelText(/^Name/)).toHaveValue('Brake pad set')
    // Without suppliers.view / catalogue.view there is no supplier or brand picker, and no call.
    expect(within(dialog).getByText(/needs permission to view suppliers/)).toBeInTheDocument()
    expect(within(dialog).getByText(/needs permission to view the catalogue/)).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Add supplier' })).not.toBeInTheDocument()
  })

  it('maps nested server errors to the right row', async () => {
    const user = userEvent.setup()
    apiMock((u, init) =>
      u.pathname === '/api/v1/stock-items/i1' && init?.method === 'PATCH'
        ? err(422, 'VALIDATION_FAILED', [
            { field: 'compatibility[0].modelId', message: 'belongs to another brand' },
          ])
        : undefined,
    )
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit SKU' })
    const m = await within(dialog).findByLabelText('Model 1')
    await waitFor(() => {
      expect(within(m).getByRole('option', { name: 'Model One' })).toBeInTheDocument()
    })
    await user.selectOptions(m, 'mo1')
    await user.click(within(dialog).getByRole('button', { name: 'Save SKU' }))
    expect(await within(dialog).findByText('belongs to another brand')).toBeInTheDocument()
  })

  it('edits with the code and tracking mode read-only and sends only what changed', async () => {
    const user = userEvent.setup()
    let sent: unknown
    const spy = apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-items/i1' && init?.method === 'PATCH') {
        sent = bodyOf(init)
        return Response.json({ data: { ...detail, name: 'Renamed', version: 4 } })
      }
      return undefined
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit SKU' })
    expect(within(dialog).getByLabelText('SKU code (cannot be changed)')).toHaveTextContent(
      'BRK-PAD-01',
    )
    expect(within(dialog).getByLabelText('Tracking mode (cannot be changed)')).toHaveTextContent(
      'Quantity',
    )
    expect(within(dialog).queryByRole('textbox', { name: /SKU code/ })).not.toBeInTheDocument()
    // Nothing changed: nothing is sent.
    await user.click(within(dialog).getByRole('button', { name: 'Save SKU' }))
    expect(await within(dialog).findByText(/Nothing has changed/)).toBeInTheDocument()
    expect(calls(spy, 'PATCH', '/api/v1/stock-items/i1')).toHaveLength(0)
    await user.clear(within(dialog).getByLabelText(/^Name/))
    await user.type(within(dialog).getByLabelText(/^Name/), 'Renamed')
    await user.clear(within(dialog).getByLabelText(/Reorder level/))
    await user.type(within(dialog).getByLabelText(/Reorder level/), '1.125')
    await user.click(within(dialog).getByRole('button', { name: 'Save SKU' }))
    await waitFor(() => {
      expect(sent).toEqual({ version: 3, name: 'Renamed', reorderLevel: '1.125' })
    })
    // The detail is fetched again from the server after it confirmed.
    await waitFor(() => {
      expect(calls(spy, 'GET', '/api/v1/stock-items/i1').length).toBeGreaterThan(1)
    })
  })

  it('shows the stored supplier, brand and model once their options have loaded', async () => {
    const user = userEvent.setup()
    const withModel: StockItemDetail = {
      ...detail,
      compatibility: [
        {
          id: 'm1',
          brandId: 'b1',
          brandName: 'Brand One',
          brandStatus: 'PENDING_VERIFICATION',
          modelId: 'mo1',
          modelName: 'Model One',
        },
      ],
    }
    apiMock(undefined, withModel)
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit SKU' })
    // The options arrive after the form is filled; the selects must still show the stored values.
    await waitFor(() => {
      expect(within(dialog).getByLabelText('Supplier 1', { exact: true })).toHaveValue('s1')
    })
    await waitFor(() => {
      expect(within(dialog).getByLabelText('Brand 1', { exact: true })).toHaveValue('b1')
    })
    await waitFor(() => {
      expect(within(dialog).getByLabelText('Model 1', { exact: true })).toHaveValue('mo1')
    })
    expect(within(dialog).getByLabelText(/Category/)).toHaveValue('c1')
  })

  it('shows a stale version with a reload offer', async () => {
    const user = userEvent.setup()
    apiMock((u, init) =>
      u.pathname === '/api/v1/stock-items/i1' && init?.method === 'PATCH'
        ? err(409, 'VERSION_CONFLICT')
        : undefined,
    )
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit SKU' })
    await user.type(within(dialog).getByLabelText(/^Name/), ' X')
    await user.click(within(dialog).getByRole('button', { name: 'Save SKU' }))
    expect(
      await within(dialog).findByRole('button', { name: /Reload latest version/ }),
    ).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/^Name/)).toHaveValue('Brake pad set X')
  })
})

describe('SKU detail', () => {
  it('shows suppliers (current and ended) and compatibility as staff-entered metadata', async () => {
    apiMock()
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    const sup = await screen.findByRole('table', { name: 'Suppliers of this SKU' })
    expect(within(sup).getByText('Example Supplier One')).toBeInTheDocument()
    expect(within(sup).getByText('Pending verification')).toBeInTheDocument()
    expect(within(sup).getByText(/^Ended/)).toBeInTheDocument()
    expect(
      within(screen.getByRole('table', { name: 'Compatibility' })).getByText('All models'),
    ).toBeInTheDocument()
    expect(screen.getByText(/not an approved manufacturer fitment/)).toBeInTheDocument()
  })

  it('never shows supplier names without suppliers.view', async () => {
    apiMock(undefined, hidden)
    renderApp('/inventory/parts/i1', { me: me(VIEW) })
    expect(await screen.findByText(/need permission to view suppliers/)).toBeInTheDocument()
    expect(screen.queryByText(/Example Supplier/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Archive' })).not.toBeInTheDocument()
  })

  it('archives only with a reason, after confirming', async () => {
    const user = userEvent.setup()
    let sent: unknown
    const spy = apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-items/i1/archive') {
        sent = bodyOf(init)
        return Response.json({ data: archived })
      }
      return undefined
    })
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    await user.click(await screen.findByRole('button', { name: 'Archive' }))
    const dialog = await screen.findByRole('dialog', { name: 'Archive SKU' })
    expect(within(dialog).getByText(/no reactivation/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Archive SKU' }))
    expect(await within(dialog).findByText('Give a reason')).toBeInTheDocument()
    expect(calls(spy, 'POST', '/api/v1/stock-items/i1/archive')).toHaveLength(0)
    await user.type(within(dialog).getByLabelText(/Reason/), ' Discontinued ')
    await user.click(within(dialog).getByRole('button', { name: 'Archive SKU' }))
    await waitFor(() => {
      expect(sent).toEqual({ version: 3, reason: 'Discontinued' })
    })
  })

  it('keeps an archived SKU readable, with no edit, archive or reactivation action', async () => {
    apiMock(undefined, archived)
    renderApp('/inventory/parts/i1', { me: me(ALL) })
    expect(await screen.findByText(/This SKU is archived/)).toBeInTheDocument()
    expect(screen.getByText('Discontinued')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Suppliers of this SKU' })).toBeInTheDocument()
    for (const name of ['Edit', 'Archive', /reactivat/i, /restore/i])
      expect(screen.queryByRole('button', { name })).not.toBeInTheDocument()
  })

  it('says when a SKU does not exist', async () => {
    apiMock()
    renderApp('/inventory/parts/nope', { me: me(VIEW) })
    expect(await screen.findByText('This SKU does not exist')).toBeInTheDocument()
  })
})

describe('categories', () => {
  it('creates, shows a duplicate on the name, and refuses archiving with active SKUs', async () => {
    const user = userEvent.setup()
    let posts = 0
    apiMock((u, init) => {
      if (u.pathname === '/api/v1/stock-item-categories' && init?.method === 'POST') {
        posts++
        return err(409, 'DUPLICATE', [{ field: 'name', message: 'is already used' }])
      }
      if (u.pathname === '/api/v1/stock-item-categories/c1/archive')
        return Response.json(
          {
            error: {
              code: 'INVALID_STATE',
              message: 'This category still has active SKUs. Move or archive them first.',
              requestId: 'r',
            },
          },
          { status: 409 },
        )
      return undefined
    })
    renderApp('/inventory/parts/categories', { me: me(ALL) })
    const table = await screen.findByRole('table', { name: 'Categories' })
    expect(within(table).getByText('Brake parts')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Add category' }))
    const add = await screen.findByRole('dialog', { name: 'Add category' })
    await user.click(within(add).getByRole('button', { name: 'Add category' }))
    expect(await within(add).findByText('Name is required')).toBeInTheDocument()
    await user.type(within(add).getByLabelText(/^Name/), 'brake PARTS')
    await user.click(within(add).getByRole('button', { name: 'Add category' }))
    expect(await within(add).findByText('Already used — choose another')).toBeInTheDocument()
    expect(posts).toBe(1)
    await user.click(within(add).getByRole('button', { name: 'Cancel' }))
    // The responsive table renders the row twice (table and phone card); use the first.
    await user.click(
      screen.getAllByRole('button', { name: 'Archive Brake parts' })[0] as HTMLElement,
    )
    const arch = await screen.findByRole('dialog', { name: 'Archive category' })
    await user.type(within(arch).getByLabelText(/Reason/), 'Unused')
    await user.click(within(arch).getByRole('button', { name: 'Archive category' }))
    expect(await within(arch).findByText(/still has active SKUs/)).toBeInTheDocument()
  })

  it('is read-only without manage and archive permissions', async () => {
    apiMock()
    renderApp('/inventory/parts/categories', { me: me(VIEW) })
    await screen.findByRole('table', { name: 'Categories' })
    expect(screen.queryByRole('button', { name: 'Add category' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Archive|Edit/ })).not.toBeInTheDocument()
  })
})
