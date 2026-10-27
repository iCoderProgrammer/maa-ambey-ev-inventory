/// <reference types="node" />
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createQueryClient } from '@/app/queryClient'
import { Permission, type Me } from '@/features/access/api/access'
import { meQueryKey } from '@/features/access/hooks/useAccess'
import type { VehicleUnitDetail } from '@/features/vehicles/api/vehicleUnits'
import { vehicleUnitKeys } from '@/features/vehicles/hooks/useVehicleUnits'
import { ApiError } from '@/lib/api/client'
import { testMe } from '@/test/render'
import * as api from './api/stockMovements'
import type { StockMoveResult } from './api/stockMovements'
import { useDisplayMove, useStockMovementPerms } from './hooks/useStockMovements'
import { describeDisplayMoveError } from './lib/errors'

// Placeholder data only ("MAE-VEH-000001"): nothing real. The API answers are
// mocked here; the server's idempotency, locking and transition guarantees are
// proven by the Go/PostgreSQL tests (internal/modules/stock).

const t = '2026-10-27T12:00:00.123456+05:30'
const result = (over: Partial<StockMoveResult['movement']> = {}, unit = {}): StockMoveResult => ({
  movement: {
    id: 'm1',
    itemType: 'VEHICLE_UNIT',
    vehicleUnitId: 'u1',
    movementType: 'DISPLAY_ON',
    quantityBefore: 1,
    quantityAfter: 1,
    fromStatus: 'AVAILABLE',
    toStatus: 'DISPLAY',
    fromLocationId: 'l1',
    toLocationId: 'l1',
    sourceType: 'STAFF_ACTION',
    sourceId: null,
    sourceReference: null,
    actorId: 'user-1',
    note: null,
    createdAt: t,
    ...over,
  },
  unit: {
    id: 'u1',
    unitNumber: 'MAE-VEH-000001',
    status: 'DISPLAY',
    locationId: 'l1',
    version: 3,
    ...unit,
  },
})
const off = () =>
  result(
    { movementType: 'DISPLAY_OFF', fromStatus: 'DISPLAY', toStatus: 'AVAILABLE' },
    { status: 'AVAILABLE' },
  )

const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json(
    { error: { code, message: 'Refused by the server.', requestId: 'req-9', details } },
    { status },
  )

function mockFetch(...responses: (Response | Error)[]) {
  const spy = vi.spyOn(globalThis, 'fetch')
  for (const r of responses) {
    if (r instanceof Error) spy.mockRejectedValueOnce(r)
    else spy.mockResolvedValueOnce(r)
  }
  return spy
}
type FetchCall = [string, RequestInit]
const callOf = (spy: ReturnType<typeof mockFetch>, i: number) => spy.mock.calls[i] as FetchCall
const urlOf = (c: FetchCall) => new URL(c[0], 'http://localhost')
const headerOf = (c: FetchCall, name: string) => new Headers(c[1].headers).get(name)
const bodyOf = (c: FetchCall) => JSON.parse(c[1].body as string) as unknown

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}
/** The production query client (no mutation retries), without background refetches. */
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

describe('stock movement API client', () => {
  it('puts a unit on display: POST, encoded path, version body and the action key', async () => {
    const action = api.newDisplayAction('PutOn', 'u 1/x', 2, '  Front window  ')
    const spy = mockFetch(
      Response.json({ data: result({ vehicleUnitId: 'u 1/x' }, { id: 'u 1/x' }) }),
    )
    const r = await api.sendDisplayAction(action)
    expect(r.unit.status).toBe('DISPLAY')
    const c = callOf(spy, 0)
    expect(c[1].method).toBe('POST')
    expect(urlOf(c).pathname).toBe('/api/v1/vehicle-units/u%201%2Fx/put-on-display')
    expect(bodyOf(c)).toEqual({ version: 2, note: 'Front window' })
    expect(headerOf(c, 'Idempotency-Key')).toBe(action.idempotencyKey)
    expect(headerOf(c, 'Content-Type')).toBe('application/json')
  })

  it('takes a unit off display; the body has no note when none was given', async () => {
    const spy = mockFetch(Response.json({ data: off() }))
    const action = api.newDisplayAction('TakeOff', 'u1', 3)
    const r = await api.sendDisplayAction(action)
    expect(r.movement.movementType).toBe('DISPLAY_OFF')
    const c = callOf(spy, 0)
    expect(c[1].method).toBe('POST')
    expect(urlOf(c).pathname).toBe('/api/v1/vehicle-units/u1/take-off-display')
    expect(bodyOf(c)).toEqual({ version: 3 })
    expect(headerOf(c, 'Idempotency-Key')).toBe(action.idempotencyKey)
  })

  it('gives every new action its own valid key and freezes it', () => {
    const a = api.newDisplayAction('PutOn', 'u1', 2)
    const b = api.newDisplayAction('PutOn', 'u1', 2)
    expect(a.idempotencyKey).toMatch(/^[A-Za-z0-9_-]{8,100}$/)
    expect(b.idempotencyKey).not.toBe(a.idempotencyKey)
    expect(Object.isFrozen(a)).toBe(true)
    expect(() => {
      ;(a as { version: number }).version = 9
    }).toThrow()
  })

  it('refuses to send a malformed action instead of inventing a key or version', async () => {
    const spy = vi.spyOn(globalThis, 'fetch')
    const a = api.newDisplayAction('PutOn', 'u1', 2)
    for (const bad of [
      { ...a, idempotencyKey: '' },
      { ...a, idempotencyKey: 'has spaces in it' },
      { ...a, version: 0 },
      { ...a, unitId: '' },
    ]) {
      await expect(api.sendDisplayAction(bad)).rejects.toThrow()
    }
    expect(spy).not.toHaveBeenCalled()
  })

  it('validates the answer: shape, statuses, and that it is the movement asked for', async () => {
    const a = api.newDisplayAction('PutOn', 'u1', 2)
    for (const bad of [
      result({ toStatus: 'PARKED' as never }),
      result({ quantityAfter: 2 }),
      result({}, { id: 'u2' }), // movement of another unit than the one returned
      result({ vehicleUnitId: 'u2' }, { id: 'u2' }), // another unit than the one asked for
      off(), // the opposite move
    ]) {
      mockFetch(Response.json({ data: bad }))
      await expect(api.sendDisplayAction(a)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
    }
  })

  it('matches the server contract field for field (docs/openapi)', () => {
    const specPath = path.resolve(import.meta.dirname, '../../../../docs/openapi/swagger.json')
    const spec = JSON.parse(readFileSync(specPath, 'utf8')) as {
      definitions: Record<string, { properties: Record<string, { enum?: string[] }> }>
      paths: Record<
        string,
        Record<string, { parameters: { name: string; in: string; required?: boolean }[] }>
      >
    }
    const keys = (name: string) => Object.keys(spec.definitions[name]?.properties ?? {}).sort()
    const r = result()
    expect(Object.keys(r).sort()).toEqual(keys('StockMoveResult'))
    expect(Object.keys(r.movement).sort()).toEqual(keys('StockMovement'))
    expect(Object.keys(r.unit).sort()).toEqual(keys('StockUnitState'))
    expect(keys('StockDisplayInput')).toEqual(['note', 'version'])
    expect(api.movementTypeSchema.options).toEqual(
      spec.definitions.StockMovement?.properties.movementType?.enum,
    )
    for (const move of Object.values(api.DisplayMove)) {
      const op = spec.paths[`/vehicle-units/{id}/${move.path}`]?.post
      expect(op?.parameters).toContainEqual(
        expect.objectContaining({ name: 'Idempotency-Key', in: 'header', required: true }),
      )
    }
    // The only movement write routes are the two display moves; nothing edits or lists movements.
    const movementPaths = Object.keys(spec.paths).filter(
      (p) => p.includes('movement') || p.includes('display') || p.endsWith('/status'),
    )
    expect(movementPaths.sort()).toEqual([
      '/vehicle-units/{id}/put-on-display',
      '/vehicle-units/{id}/take-off-display',
    ])
  })

  it('exposes only the display action calls: no generic, list or delete call', () => {
    const calls = Object.entries(api).filter(([, v]) => typeof v === 'function')
    expect(calls.map(([k]) => k).sort()).toEqual(['newDisplayAction', 'sendDisplayAction'])
  })
})

describe('display move permission', () => {
  it('is its own key, granted only when the server lists it', async () => {
    expect(Permission.VehicleUnitDisplayManage).toBe('vehicle_units.display.manage')
    const run = async (permissions: string[]) => {
      const client = prodClient()
      const me: Me = { ...testMe, permissions }
      client.setQueryData(meQueryKey, me)
      const { result: r } = renderHook(() => useStockMovementPerms(), {
        wrapper: wrapper(client),
      })
      await waitFor(() => {
        expect(r.current.me.data).toBeDefined()
      })
      return r.current.manageDisplay
    }
    expect(await run([Permission.VehicleUnitDisplayManage])).toBe(true)
    // Viewing units, their cost, receiving, approving or every location do not imply it.
    expect(
      await run([
        Permission.VehicleUnitView,
        Permission.VehicleUnitCostView,
        Permission.GoodsReceiptManage,
        Permission.PurchaseOrderApprove,
        'locations.all',
      ]),
    ).toBe(false)
    expect(await run([])).toBe(false)
  })
})

describe('display move hook', () => {
  const seed = (client: QueryClient) => {
    const unit = { id: 'u1', status: 'AVAILABLE', version: 2 } as unknown as VehicleUnitDetail
    client.setQueryData(vehicleUnitKeys.detail('u1'), unit)
    client.setQueryData(vehicleUnitKeys.list({ page: 1 }), {
      items: [unit],
      page: 1,
      limit: 25,
      total: 1,
    })
    client.setQueryData(vehicleUnitKeys.list({ status: 'AVAILABLE', page: 2 }), {
      items: [],
      page: 2,
      limit: 25,
      total: 1,
    })
    client.setQueryData(['goods-receipts', 'list', {}], { items: [] })
  }
  const stale = (client: QueryClient, key: readonly unknown[]) =>
    client.getQueryState(key)?.isInvalidated ?? false

  it('refetches every vehicle unit list and detail after the server confirms, and nothing else', async () => {
    mockFetch(Response.json({ data: result() }))
    const client = prodClient()
    seed(client)
    const { result: h } = renderHook(() => useDisplayMove(), { wrapper: wrapper(client) })
    const action = api.newDisplayAction('PutOn', 'u1', 2)
    await act(() => h.current.mutateAsync(action))
    expect(stale(client, vehicleUnitKeys.detail('u1'))).toBe(true)
    expect(stale(client, vehicleUnitKeys.list({ page: 1 }))).toBe(true)
    expect(stale(client, vehicleUnitKeys.list({ status: 'AVAILABLE', page: 2 }))).toBe(true)
    expect(stale(client, ['goods-receipts', 'list', {}])).toBe(false)
    // The cache is never written with a status the client made up.
    expect(client.getQueryData<{ status: string }>(vehicleUnitKeys.detail('u1'))?.status).toBe(
      'AVAILABLE',
    )
  })

  it('leaves the cached status alone when the server refuses, and does not retry', async () => {
    for (const [status, code] of [
      [401, 'UNAUTHENTICATED'],
      [403, 'FORBIDDEN'],
      [404, 'NOT_FOUND'],
      [422, 'VALIDATION_FAILED'],
      [409, 'IDEMPOTENCY_KEY_REUSED'],
    ] as const) {
      const spy = mockFetch(err(status, code))
      const client = prodClient()
      client.setQueryData(meQueryKey, testMe)
      seed(client)
      const { result: h } = renderHook(() => useDisplayMove(), { wrapper: wrapper(client) })
      await act(async () => {
        await h.current.mutateAsync(api.newDisplayAction('PutOn', 'u1', 2)).catch(() => undefined)
      })
      await waitFor(() => {
        expect(h.current.isError).toBe(true)
      })
      expect(h.current.error).toMatchObject({ status, code })
      expect(spy).toHaveBeenCalledTimes(1)
      expect(client.getQueryData<{ status: string }>(vehicleUnitKeys.detail('u1'))?.status).toBe(
        'AVAILABLE',
      )
      expect(stale(client, vehicleUnitKeys.detail('u1'))).toBe(false)
      vi.restoreAllMocks()
    }
  })

  it('reloads the unit after a stale version or a wrong state, without changing it locally', async () => {
    for (const code of ['VERSION_CONFLICT', 'INVALID_STATE']) {
      mockFetch(err(409, code))
      const client = prodClient()
      seed(client)
      const { result: h } = renderHook(() => useDisplayMove(), { wrapper: wrapper(client) })
      await act(async () => {
        await h.current.mutateAsync(api.newDisplayAction('PutOn', 'u1', 2)).catch(() => undefined)
      })
      expect(stale(client, vehicleUnitKeys.detail('u1'))).toBe(true)
      expect(client.getQueryData<{ status: string }>(vehicleUnitKeys.detail('u1'))?.status).toBe(
        'AVAILABLE',
      )
      vi.restoreAllMocks()
    }
  })

  it('retries the same action with the same key and body after an unknown outcome', async () => {
    const spy = mockFetch(new TypeError('Failed to fetch'), Response.json({ data: result() }))
    const client = prodClient()
    const { result: h } = renderHook(() => useDisplayMove(), { wrapper: wrapper(client) })
    const action = api.newDisplayAction('PutOn', 'u1', 2, 'window')
    await act(async () => {
      await h.current.mutateAsync(action).catch(() => undefined)
    })
    // One attempt only: the hook does not retry on its own, and does not claim the move failed.
    expect(spy).toHaveBeenCalledTimes(1)
    await waitFor(() => {
      expect(h.current.isError).toBe(true)
    })
    expect(describeDisplayMoveError(h.current.error)).toMatchObject({
      kind: 'uncertain',
      retrySameAction: true,
    })
    await act(() => h.current.mutateAsync(action))
    expect(spy).toHaveBeenCalledTimes(2)
    const [first, second] = [callOf(spy, 0), callOf(spy, 1)]
    expect(headerOf(second, 'Idempotency-Key')).toBe(headerOf(first, 'Idempotency-Key'))
    expect(bodyOf(second)).toEqual(bodyOf(first))
    expect(urlOf(second).pathname).toBe(urlOf(first).pathname)
    // A new decision is a new action: a new key.
    mockFetch(Response.json({ data: off() }))
    await act(() => h.current.mutateAsync(api.newDisplayAction('TakeOff', 'u1', 3)))
    expect(headerOf(callOf(vi.mocked(fetch), 2), 'Idempotency-Key')).not.toBe(action.idempotencyKey)
  })
})

describe('display move errors', () => {
  const describeHttp = async (status: number, code: string, details?: []) => {
    mockFetch(err(status, code, details))
    const e = await api
      .sendDisplayAction(api.newDisplayAction('PutOn', 'u1', 2))
      .catch((x: unknown) => x)
    expect(e).toBeInstanceOf(ApiError)
    return describeDisplayMoveError(e)
  }

  it('tells a stale version, a wrong state and a reused key apart', async () => {
    expect(await describeHttp(409, 'VERSION_CONFLICT')).toMatchObject({
      kind: 'stale-version',
      reload: true,
      retrySameAction: false,
    })
    expect(await describeHttp(409, 'INVALID_STATE')).toMatchObject({
      kind: 'invalid-state',
      reload: true,
      retrySameAction: false,
      message: 'Refused by the server.',
    })
    expect(await describeHttp(409, 'IDEMPOTENCY_KEY_REUSED')).toMatchObject({
      kind: 'key-reused',
      reload: true,
      retrySameAction: false,
      requestId: 'req-9',
    })
  })

  it('maps 401, 403, 404 and 422 safely and never suggests a retry', async () => {
    expect(await describeHttp(401, 'UNAUTHENTICATED')).toMatchObject({
      kind: 'signed-out',
      retrySameAction: false,
    })
    expect(await describeHttp(403, 'FORBIDDEN')).toMatchObject({
      kind: 'forbidden',
      retrySameAction: false,
    })
    const nf = await describeHttp(404, 'NOT_FOUND')
    expect(nf).toMatchObject({ kind: 'not-found', retrySameAction: false })
    expect(nf.message).toBe('This vehicle unit was not found, or it is not available to you.')
    mockFetch(err(422, 'VALIDATION_FAILED', [{ field: 'version', message: 'is required' }]))
    const e = await api
      .sendDisplayAction(api.newDisplayAction('PutOn', 'u1', 2))
      .catch((x: unknown) => x)
    expect(describeDisplayMoveError(e)).toMatchObject({
      kind: 'invalid-input',
      fields: [{ field: 'version', message: 'is required' }],
      retrySameAction: false,
    })
  })

  it('treats network failures, timeouts and server errors as unknown outcomes', async () => {
    expect(await describeHttp(500, 'INTERNAL_ERROR')).toMatchObject({
      kind: 'uncertain',
      retrySameAction: true,
    })
    expect(await describeHttp(503, 'REQUEST_TIMEOUT')).toMatchObject({ kind: 'uncertain' })
    // A 2xx answer the client cannot read: the move may have been made; reload, do not call it failed.
    mockFetch(Response.json({ data: { unexpected: true } }))
    const e = await api
      .sendDisplayAction(api.newDisplayAction('PutOn', 'u1', 2))
      .catch((x: unknown) => x)
    expect(describeDisplayMoveError(e)).toMatchObject({
      kind: 'uncertain',
      reload: true,
      retrySameAction: false,
    })
    expect(describeDisplayMoveError(new Error('x'))).toMatchObject({
      kind: 'error',
      retrySameAction: false,
    })
  })
})
