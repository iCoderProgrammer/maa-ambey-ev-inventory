import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createQueryClient } from '@/app/queryClient'
import { ApiError } from '@/lib/api/client'
import {
  activateItem,
  activationBlockers,
  createBrand,
  isVersionConflict,
  listBrands,
  listVariants,
  queryString,
  recordBrandAuthorisation,
  updateItem,
  verifyItem,
} from './api/catalogue'
import type { Brand, VehicleModel } from './api/contracts'
import {
  catalogueKeys,
  useActivateItem,
  useBrands,
  useCreateBrand,
  useModels,
} from './hooks/useCatalogue'

// Placeholder data only: no real brand, model or source is invented.
const stamps = {
  version: 1,
  createdAt: '2026-10-13T09:00:00Z',
  updatedAt: '2026-10-13T09:00:00Z',
}
const brand: Brand = {
  id: 'b1',
  name: 'Brand A',
  slug: 'brand-a',
  lifecycleStatus: 'PENDING_VERIFICATION',
  authorisationStatus: 'UNVERIFIED',
  manufacturerLegalName: null,
  authorisationReference: null,
  authorisationEvidenceId: null,
  verifiedBy: null,
  verifiedAt: null,
  operational: false,
  ...stamps,
}
const model: VehicleModel = {
  id: 'm1',
  brandId: 'b1',
  name: 'Model 1',
  internalCode: 'M1',
  manufacturerCode: null,
  vehicleType: null,
  lifecycleStatus: 'PENDING_VERIFICATION',
  confidence: 'SUPPLIER_CONFIRMED',
  sourceUrl: 'https://example.com/source',
  sourceDocumentId: null,
  sourceDate: '2026-10-01',
  verifiedBy: 'u1',
  verifiedAt: '2026-10-13T09:05:00.123456Z',
  ...stamps,
}

function page<T>(items: T[], total = items.length) {
  return { items, page: 1, limit: 25, total }
}

function errorResponse(
  status: number,
  code: string,
  details?: { field: string; message: string }[],
) {
  return Response.json(
    { error: { code, message: 'refused', requestId: 'req-1', details } },
    { status },
  )
}

function mockFetch(...responses: Response[]) {
  const spy = vi.spyOn(globalThis, 'fetch')
  for (const r of responses) spy.mockResolvedValueOnce(r)
  return spy
}

function call(spy: ReturnType<typeof mockFetch>, i = 0) {
  const [url, init] = spy.mock.calls[i]!
  return {
    url: url as string,
    method: init?.method,
    body: typeof init?.body === 'string' ? (JSON.parse(init.body) as unknown) : undefined,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('catalogue API client', () => {
  it('lists with only the filters that are set', async () => {
    const spy = mockFetch(Response.json({ data: page([brand], 3) }))
    const result = await listBrands({ q: 'a b', status: 'PENDING_VERIFICATION', page: 1 })
    expect(call(spy).url).toBe('/api/v1/brands?q=a+b&status=PENDING_VERIFICATION&page=1')
    expect(result.total).toBe(3)
    expect(result.items[0]?.name).toBe('Brand A')
    expect(queryString({ brandId: undefined, q: '' })).toBe('')
  })

  it('sends mutations to the documented routes with the version', async () => {
    const spy = mockFetch(
      Response.json({ data: brand }, { status: 201 }),
      Response.json({ data: model }),
      Response.json({ data: model }),
      Response.json({ data: { ...model, lifecycleStatus: 'ACTIVE' } }),
      Response.json({ data: brand }),
    )
    await createBrand({ name: 'Brand A' })
    await updateItem('models', 'm1', { version: 1, manufacturerCode: null })
    await verifyItem('models', 'm1', {
      version: 1,
      confidence: 'SUPPLIER_CONFIRMED',
      sourceUrl: 'https://example.com/source',
      sourceDate: '2026-10-01',
    })
    const active = await activateItem('models', 'm1', 2)
    await recordBrandAuthorisation('b1', { version: 1, authorisationStatus: 'UNVERIFIED' })

    expect(call(spy, 0)).toMatchObject({
      url: '/api/v1/brands',
      method: 'POST',
      body: { name: 'Brand A' },
    })
    expect(call(spy, 1)).toMatchObject({
      url: '/api/v1/models/m1',
      method: 'PATCH',
      body: { version: 1, manufacturerCode: null },
    })
    expect(call(spy, 2)).toMatchObject({ url: '/api/v1/models/m1/verify', method: 'POST' })
    expect(call(spy, 3)).toMatchObject({
      url: '/api/v1/models/m1/activate',
      method: 'POST',
      body: { version: 2 },
    })
    expect(call(spy, 4)).toMatchObject({ url: '/api/v1/brands/b1/authorisation', method: 'PUT' })
    expect(active.lifecycleStatus).toBe('ACTIVE')
  })

  it('exposes the unmet checks of a refused activation', async () => {
    mockFetch(
      errorResponse(409, 'ACTIVATION_BLOCKED', [
        { field: 'verification', message: 'Record the model source first.' },
        { field: 'parent', message: 'Activate the brand first.' },
      ]),
    )
    const err = await activateItem('variants', 'v1', 1).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(activationBlockers(err)).toEqual([
      { key: 'verification', message: 'Record the model source first.' },
      { key: 'parent', message: 'Activate the brand first.' },
    ])
    expect(isVersionConflict(err)).toBe(false)
  })

  it('reports stale versions, permission and relationship errors from the server', async () => {
    mockFetch(
      errorResponse(409, 'VERSION_CONFLICT'),
      errorResponse(403, 'FORBIDDEN'),
      errorResponse(422, 'VALIDATION_FAILED', [
        { field: 'brandId', message: "does not match the model's brand" },
      ]),
    )
    expect(
      isVersionConflict(
        await updateItem('colours', 'c1', { version: 1, name: 'X' }).catch((e: unknown) => e),
      ),
    ).toBe(true)
    await expect(createBrand({ name: 'X' })).rejects.toMatchObject({
      status: 403,
      code: 'FORBIDDEN',
    })
    await expect(listVariants({ brandId: 'b2', modelId: 'm1' })).rejects.toMatchObject({
      code: 'VALIDATION_FAILED',
      details: [{ field: 'brandId', message: "does not match the model's brand" }],
    })
  })

  it('rejects a response that does not match the contract instead of trusting it', async () => {
    const withoutOperational: Partial<Brand> = { ...brand }
    delete withoutOperational.operational
    mockFetch(Response.json({ data: page([withoutOperational]) }))
    await expect(listBrands()).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })
})

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

function testClient() {
  const client = createQueryClient()
  client.setDefaultOptions({ queries: { retry: false, refetchOnWindowFocus: false } })
  return client
}

describe('catalogue hooks', () => {
  it('load a list and report an empty result as an empty page', async () => {
    mockFetch(Response.json({ data: page([]) }))
    const client = testClient()
    const { result } = renderHook(() => useModels({ brandId: 'b1' }), { wrapper: wrapper(client) })
    expect(result.current.isPending).toBe(true)
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(result.current.data?.items).toEqual([])
  })

  it('do not call the API when disabled (no catalogue.view)', () => {
    const spy = vi.spyOn(globalThis, 'fetch')
    renderHook(() => useBrands({}, { enabled: false }), { wrapper: wrapper(testClient()) })
    expect(spy).not.toHaveBeenCalled()
  })

  it('surface a permission error from the server', async () => {
    mockFetch(errorResponse(403, 'FORBIDDEN'), Response.json({ data: {} }))
    const { result } = renderHook(() => useBrands(), { wrapper: wrapper(testClient()) })
    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
    expect(result.current.error).toMatchObject({ status: 403 })
  })

  it('change the cache only after the server confirms', async () => {
    const client = testClient()
    client.setQueryData(catalogueKeys.item('models', 'm1'), model)
    mockFetch(
      errorResponse(409, 'ACTIVATION_BLOCKED', [
        { field: 'parent', message: 'The brand is archived.' },
      ]),
    )
    const { result } = renderHook(() => useActivateItem('models'), { wrapper: wrapper(client) })

    await act(async () => {
      await result.current.mutateAsync({ id: 'm1', version: 1 }).catch(() => undefined)
    })
    expect(client.getQueryData(catalogueKeys.item('models', 'm1'))).toEqual(model)
    await waitFor(() => {
      expect(activationBlockers(result.current.error)).toEqual([
        { key: 'parent', message: 'The brand is archived.' },
      ])
    })

    const active = { ...model, lifecycleStatus: 'ACTIVE' as const, version: 2 }
    mockFetch(Response.json({ data: active }))
    await act(async () => {
      await result.current.mutateAsync({ id: 'm1', version: 1 })
    })
    expect(client.getQueryData(catalogueKeys.item('models', 'm1'))).toEqual(active)
  })

  it('refresh lists after a confirmed create', async () => {
    const client = testClient()
    client.setQueryData(catalogueKeys.brands({}), page([]))
    mockFetch(Response.json({ data: brand }, { status: 201 }))
    const { result } = renderHook(() => useCreateBrand(), { wrapper: wrapper(client) })
    await act(async () => {
      await result.current.mutateAsync({ name: 'Brand A' })
    })
    expect(client.getQueryState(catalogueKeys.brands({}))?.isInvalidated).toBe(true)
    expect(client.getQueryData(catalogueKeys.brand('b1'))).toEqual(brand)
  })
})
