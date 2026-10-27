/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createQueryClient } from '@/app/queryClient'
import { Permission, type Me } from '@/features/access/api/access'
import { meQueryKey } from '@/features/access/hooks/useAccess'
import { ApiError } from '@/lib/api/client'
import { testMe } from '@/test/render'
import * as api from './api/vehicleUnits'
import type { VehicleUnit, VehicleUnitDetail } from './api/vehicleUnits'
import {
  useVehicleUnit,
  useVehicleUnitList,
  useVehicleUnitPerms,
  vehicleUnitKeys,
} from './hooks/useVehicleUnits'
import { describeVehicleUnitReadError, HIDDEN_COST_TEXT, unitCostText } from './lib/presentation'

// Placeholder data only ("Brand One", "CH-0001", 95000.00): nothing real.

const t = '2026-10-26T12:00:00.123456+05:30'
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
  chassisNumber: 'ch-0001',
  chassisNormalised: 'CH-0001',
  motorSerial: null,
  batterySerial: null,
  chargerSerial: null,
  locationId: null,
  locationCode: null,
  locationName: null,
  sourceType: 'GOODS_RECEIPT',
  supplierId: 's1',
  supplierName: 'Example Supplier',
  receivedDate: '2026-10-23',
  unitCost: '95000.00',
  costCurrency: 'INR',
  costHidden: false,
  version: 1,
  createdAt: t,
  updatedAt: t,
}
const hidden: VehicleUnit = { ...unit, unitCost: null, costCurrency: null, costHidden: true }
const detail: VehicleUnitDetail = {
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
    receivedDate: '2026-10-23',
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
}

const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-7', details } }, { status })

function mockFetch(...responses: Response[]) {
  const spy = vi.spyOn(globalThis, 'fetch')
  for (const r of responses) spy.mockResolvedValueOnce(r)
  return spy
}
const urlOf = (call: unknown[]) => new URL(call[0] as string, 'http://localhost')

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}
/** The production query client (its retry rules included), without background refetches. */
function prodClient() {
  const client = createQueryClient()
  client.setDefaultOptions({
    queries: { ...client.getDefaultOptions().queries, refetchOnWindowFocus: false },
  })
  return client
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('vehicle unit API client', () => {
  it('sends search, filters and paging as query parameters and skips empty ones', async () => {
    const spy = mockFetch(Response.json({ data: page([unit]) }))
    const p = await api.listVehicleUnits({
      q: 'ch 00',
      status: 'AVAILABLE',
      brandId: 'b1',
      variantId: 'v1',
      supplierId: 's1',
      purchaseOrderId: 'po1',
      goodsReceiptId: 'g1',
      locationId: 'l1',
      page: 2,
      limit: 50,
    })
    expect(p.items[0]?.unitNumber).toBe('MAE-VEH-000001')
    const url = urlOf(spy.mock.calls[0] ?? [])
    expect(url.pathname).toBe('/api/v1/vehicle-units')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      q: 'ch 00',
      status: 'AVAILABLE',
      brandId: 'b1',
      variantId: 'v1',
      supplierId: 's1',
      purchaseOrderId: 'po1',
      goodsReceiptId: 'g1',
      locationId: 'l1',
      page: '2',
      limit: '50',
    })
    mockFetch(Response.json({ data: page([]) }))
    await api.listVehicleUnits({ q: '', brandId: undefined })
    expect(urlOf(vi.mocked(fetch).mock.calls[1] ?? []).search).toBe('')
  })

  it('fetches one unit by its encoded ID with GET only', async () => {
    const spy = mockFetch(Response.json({ data: detail }))
    const d = await api.getVehicleUnit('u 1/x')
    expect(d.source.goodsReceiptNumber).toBe('GRN-000001')
    expect(d.events[0]?.sourceReference).toBe('GRN-000001')
    expect(urlOf(spy.mock.calls[0] ?? []).pathname).toBe('/api/v1/vehicle-units/u%201%2Fx')
    expect(spy.mock.calls[0]?.[1]?.method ?? 'GET').toBe('GET')
  })

  it('exposes no create, update or delete call for the read-only API', () => {
    const calls = Object.entries(api).filter(([, v]) => typeof v === 'function')
    expect(calls.map(([k]) => k).sort()).toEqual(['getVehicleUnit', 'listVehicleUnits'])
  })

  it('rejects a hidden cost that carries an amount, and unknown statuses', async () => {
    mockFetch(Response.json({ data: page([{ ...hidden, unitCost: '95000.00' }]) }))
    await expect(api.listVehicleUnits()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    mockFetch(Response.json({ data: { ...detail, status: 'PARKED' } }))
    await expect(api.getVehicleUnit('u1')).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })

  it('matches the server contract field for field (docs/openapi)', () => {
    // The committed OpenAPI document generated from the Go handlers (make swagger).
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
    }
    const keys = (name: string) => Object.keys(spec.definitions[name]?.properties ?? {}).sort()
    expect(Object.keys(unit).sort()).toEqual(keys('VehicleUnit'))
    expect(Object.keys(detail).sort()).toEqual(keys('VehicleUnitDetail'))
    expect(Object.keys(detail.source).sort()).toEqual(keys('VehicleUnitSource'))
    expect(Object.keys(detail.events[0] ?? {}).sort()).toEqual(keys('VehicleUnitEvent'))
    expect(api.unitStatusSchema.options).toEqual(
      spec.definitions.VehicleUnit?.properties.status?.enum,
    )
    expect(Object.keys(spec.definitions.VehicleUnitPage?.properties ?? {}).sort()).toEqual([
      'items',
      'limit',
      'page',
      'total',
    ])
  })
})

describe('vehicle unit query keys and hooks', () => {
  it('give every filter and page combination its own cache entry', () => {
    const a = JSON.stringify(vehicleUnitKeys.list({ q: 'ch', page: 1 }))
    expect(JSON.stringify(vehicleUnitKeys.list({ q: 'ch', page: 2 }))).not.toBe(a)
    expect(JSON.stringify(vehicleUnitKeys.list({ q: 'ch', page: 1, brandId: 'b1' }))).not.toBe(a)
    expect(
      JSON.stringify(vehicleUnitKeys.list({ q: 'ch', page: 1, status: 'AVAILABLE' })),
    ).not.toBe(a)
    // Parameters the server never sees do not split the cache.
    expect(vehicleUnitKeys.list({ q: '', brandId: undefined, page: 1 })).toEqual(
      vehicleUnitKeys.list({ page: 1 }),
    )
    expect(vehicleUnitKeys.detail('u1')).not.toEqual(vehicleUnitKeys.detail('u2'))
  })

  it('load a list with its filters, and a new page with a new request', async () => {
    const spy = mockFetch(
      Response.json({ data: page([unit]) }),
      Response.json({ data: { ...page([]), page: 2 } }),
    )
    const client = prodClient()
    const { result, rerender } = renderHook(
      (p: { page: number }) => useVehicleUnitList({ brandId: 'b1', page: p.page }),
      {
        wrapper: wrapper(client),
        initialProps: { page: 1 },
      },
    )
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(result.current.data?.items).toHaveLength(1)
    rerender({ page: 2 })
    await waitFor(() => {
      expect(result.current.data?.page).toBe(2)
    })
    expect(spy.mock.calls.map((c) => urlOf(c).search)).toEqual([
      '?brandId=b1&page=1',
      '?brandId=b1&page=2',
    ])
  })

  it('load one unit by ID and stay idle without an ID or when disabled', async () => {
    const spy = mockFetch(Response.json({ data: detail }))
    const client = prodClient()
    renderHook(() => useVehicleUnit(undefined), { wrapper: wrapper(client) })
    renderHook(() => useVehicleUnit('u1', { enabled: false }), { wrapper: wrapper(client) })
    renderHook(() => useVehicleUnitList({}, { enabled: false }), { wrapper: wrapper(client) })
    expect(spy).not.toHaveBeenCalled()
    const { result } = renderHook(() => useVehicleUnit('u1'), { wrapper: wrapper(client) })
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(urlOf(spy.mock.calls[0] ?? []).pathname).toBe('/api/v1/vehicle-units/u1')
    expect(client.getQueryData(vehicleUnitKeys.detail('u1'))).toEqual(detail)
  })

  it.each([
    [403, 'FORBIDDEN'],
    [404, 'NOT_FOUND'],
  ])('ask once on %i and never retry or widen the request', async (status, code) => {
    const spy = mockFetch(err(status, code), Response.json({ data: detail }))
    const { result } = renderHook(() => useVehicleUnit('u1'), { wrapper: wrapper(prodClient()) })
    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
    expect(spy).toHaveBeenCalledTimes(1)
    expect(result.current.error).toMatchObject({ status, code })
  })

  it('treat an out-of-scope location filter as invalid without retrying it unfiltered', async () => {
    const spy = mockFetch(
      err(422, 'VALIDATION_FAILED', [
        { field: 'locationId', message: 'is not one of your locations' },
      ]),
      Response.json({ data: page([unit]) }),
    )
    const { result } = renderHook(() => useVehicleUnitList({ locationId: 'l2' }), {
      wrapper: wrapper(prodClient()),
    })
    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
    expect(spy).toHaveBeenCalledTimes(1)
    expect(urlOf(spy.mock.calls[0] ?? []).searchParams.get('locationId')).toBe('l2')
    expect(result.current.data).toBeUndefined()
    const e = describeVehicleUnitReadError(result.current.error)
    expect(e.state).toBe('invalid-filter')
    expect(e.fields).toEqual([{ field: 'locationId', message: 'is not one of your locations' }])
  })

  it('read the two vehicle unit permissions from the server contract only', () => {
    const client = prodClient()
    const seed = (permissions: string[]) => {
      client.setQueryData<Me>(meQueryKey, { ...testMe, permissions })
    }
    const spy = vi.spyOn(globalThis, 'fetch')
    seed([])
    const { result, rerender } = renderHook(() => useVehicleUnitPerms(), {
      wrapper: wrapper(client),
    })
    expect(result.current).toMatchObject({ view: false, viewCost: false })
    seed([Permission.VehicleUnitView, Permission.CostPriceView])
    rerender()
    expect(result.current).toMatchObject({ view: true, viewCost: false })
    seed([Permission.VehicleUnitView, Permission.VehicleUnitCostView])
    rerender()
    expect(result.current).toMatchObject({ view: true, viewCost: true })
    expect(spy).not.toHaveBeenCalled()
  })
})

describe('vehicle unit presentation', () => {
  it('uses the server permission keys', () => {
    expect(Permission.VehicleUnitView).toBe('vehicle_units.view')
    expect(Permission.VehicleUnitCostView).toBe('vehicle_units.cost.view')
  })

  it('never shows a hidden cost as an amount, and shows a cost only when sent and permitted', () => {
    expect(unitCostText(hidden, true)).toBe(HIDDEN_COST_TEXT)
    expect(unitCostText(hidden, false)).toBe(HIDDEN_COST_TEXT)
    expect(unitCostText(hidden, true)).not.toMatch(/₹|0/)
    expect(unitCostText(unit, true)).toBe('₹95,000.00')
    // Sent but the UI has no permission recorded (e.g. permissions changed meanwhile): not shown.
    expect(unitCostText(unit, false)).toBeNull()
    // Not hidden but not sent: nothing invented.
    expect(unitCostText({ unitCost: null, costHidden: false }, true)).toBeNull()
  })

  it.each([
    [401, 'UNAUTHENTICATED', 'signed-out', false],
    [403, 'FORBIDDEN', 'forbidden', false],
    [404, 'NOT_FOUND', 'not-found', false],
    [422, 'VALIDATION_FAILED', 'invalid-filter', false],
    [500, 'INTERNAL_ERROR', 'error', true],
  ])('maps %i %s to the %s state', (status, code, state, retry) => {
    const e = describeVehicleUnitReadError(
      new ApiError(status, { code, message: 'Refused.', requestId: 'req-7' }),
    )
    expect(e).toMatchObject({ state, retry, requestId: 'req-7' })
  })

  it('does not reveal whether a concealed unit exists', () => {
    const e = describeVehicleUnitReadError(new ApiError(404, { code: 'NOT_FOUND', message: 'x' }))
    expect(e.message).toBe('This vehicle unit was not found, or it is not available to you.')
  })

  it('maps network failures and unknown errors to retryable states', () => {
    expect(
      describeVehicleUnitReadError(new ApiError(0, { code: 'NETWORK_ERROR', message: 'x' })),
    ).toMatchObject({ state: 'offline', retry: true })
    expect(describeVehicleUnitReadError(new Error('boom'))).toMatchObject({
      state: 'error',
      retry: true,
    })
  })
})
