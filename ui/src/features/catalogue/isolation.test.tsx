import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import type { Brand, Variant, VehicleModel } from './api/contracts'

/**
 * Brand isolation in the UI (C-06): the brand choice constrains model/variant
 * options and filters, stale cross-brand filters are dropped before reaching the
 * server, price data is requested only for the record's own brand, and
 * cost/authorisation details follow permissions. Brand names are arbitrary data.
 */

const stamps = { version: 1, createdAt: '2026-10-13T09:00:00Z', updatedAt: '2026-10-13T09:00:00Z' }
const prov = {
  confidence: 'SUPPLIER_CONFIRMED' as const,
  sourceUrl: 'https://example.com/s',
  sourceDocumentId: null,
  sourceDate: '2026-10-01',
  verifiedBy: 'u-x',
  verifiedAt: '2026-10-13T09:00:00Z',
}
const brand = (id: string, name: string): Brand => ({
  id,
  name,
  slug: id,
  lifecycleStatus: 'PENDING_VERIFICATION',
  authorisationStatus: 'UNVERIFIED',
  manufacturerLegalName: null,
  authorisationReference: null,
  authorisationEvidenceId: null,
  verifiedBy: null,
  verifiedAt: null,
  operational: false,
  ...stamps,
})
const model = (id: string, brandId: string, name: string): VehicleModel => ({
  id,
  brandId,
  name,
  internalCode: id.toUpperCase(),
  manufacturerCode: null,
  vehicleType: null,
  lifecycleStatus: 'ACTIVE',
  ...prov,
  ...stamps,
})
// An arbitrary brand name nobody coded for.
const qx = brand('b-qx', 'Quixotic Volt Works')
const zy = brand('b-zy', 'Zephyr Yards')
const qxModel = model('m-qx', 'b-qx', 'QX One')
const zyModel = model('m-zy', 'b-zy', 'ZY One')
const zyVariant: Variant = {
  id: 'v-zy',
  brandId: 'b-zy',
  modelId: 'm-zy',
  name: 'ZY Standard',
  variantCode: 'STD',
  manufacturerCode: null,
  lifecycleStatus: 'ACTIVE',
  ...prov,
  ...stamps,
}
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({ ...testMe, permissions })

function api() {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/brands') return json(page([qx, zy]))
    if (p === '/api/v1/brands/b-zy') return json(zy)
    if (p === '/api/v1/brands/b-qx') return json(qx)
    if (p === '/api/v1/brands/b-qx/activation-readiness') {
      return json({ brandId: 'b-qx', ready: false, checks: [] })
    }
    if (p === '/api/v1/models') {
      const b = url.searchParams.get('brandId')
      return json(page([qxModel, zyModel].filter((m) => !b || m.brandId === b)))
    }
    if (p === '/api/v1/models/m-qx') return json(qxModel)
    if (p === '/api/v1/models/m-zy') return json(zyModel)
    if (p === '/api/v1/variants/v-zy') return json(zyVariant)
    if (p === '/api/v1/variants' || p === '/api/v1/colours') return json(page([]))
    if (p === '/api/v1/price-lists' || p === '/api/v1/price-items') return json(page([]))
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(
      Response.json({ error: { code: 'NOT_FOUND', message: 'x' } }, { status: 404 }),
    )
  })
}

function requests(spy: ReturnType<typeof api>, path: string) {
  return spy.mock.calls
    .map(([u]) => new URL(u as string, 'http://localhost'))
    .filter((u) => u.pathname === `/api/v1${path}`)
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('brand isolation in the catalogue UI', () => {
  it('drops a model filter of another brand from a stale link before asking the server', async () => {
    const spy = api()
    const { router } = renderApp('/admin/catalogue/variants?brandId=b-zy&modelId=m-qx', {
      me: me(['catalogue.view']),
    })
    await waitFor(() => {
      expect(router.state.location.search).toBe('?brandId=b-zy')
    })
    await screen.findByText('No variants match these filters')
    const variantCalls = requests(spy, '/variants')
    expect(variantCalls.length).toBeGreaterThan(0)
    // The mismatched pair never reached the API.
    expect(
      variantCalls.some(
        (u) => u.searchParams.get('brandId') === 'b-zy' && u.searchParams.get('modelId') === 'm-qx',
      ),
    ).toBe(false)
  })

  it('keeps a matching brand and model filter', async () => {
    const spy = api()
    renderApp('/admin/catalogue/variants?brandId=b-zy&modelId=m-zy', { me: me(['catalogue.view']) })
    await waitFor(() => {
      expect(requests(spy, '/variants').some((u) => u.searchParams.get('modelId') === 'm-zy')).toBe(
        true,
      )
    })
  })

  it('clears the model filter when the brand changes', async () => {
    const user = userEvent.setup()
    api()
    const { router } = renderApp('/admin/catalogue/variants?brandId=b-zy&modelId=m-zy', {
      me: me(['catalogue.view']),
    })
    const brandFilter = await screen.findByLabelText('Brand')
    await within(brandFilter).findByRole('option', { name: 'Quixotic Volt Works' })
    await user.selectOptions(brandFilter, 'b-qx')
    await waitFor(() => {
      expect(router.state.location.search).toBe('?brandId=b-qx')
    })
    const modelFilter = screen.getByLabelText('Model')
    await waitFor(() => {
      expect(
        within(modelFilter)
          .getAllByRole('option')
          .map((o) => o.textContent),
      ).toEqual(['All models of this brand', 'QX One (M-QX)'])
    })
  })

  it('asks for price lists of the variant’s own brand only', async () => {
    const spy = api()
    renderApp('/admin/catalogue/variants/v-zy', {
      me: me(['catalogue.view', 'catalogue.price.view']),
    })
    await screen.findByText('No price list for this brand')
    const lists = requests(spy, '/price-lists')
    expect(lists.length).toBeGreaterThan(0)
    expect(lists.every((u) => u.searchParams.get('brandId') === 'b-zy')).toBe(true)
  })

  it('shows an arbitrary brand from data with its activation warning; authorisation editing needs its permission', async () => {
    api()
    renderApp('/admin/catalogue/brands/b-qx', { me: me(['catalogue.view']) })
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Quixotic Volt Works' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Authorisation not verified/)).toBeInTheDocument()
    // Details are readable; recording authorisation and enabling need their own permissions.
    expect(screen.getByRole('heading', { name: 'Authorisation' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Record' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Enable…' })).not.toBeInTheDocument()
    // No price permission: no price sections and no price requests.
    expect(screen.queryByText('Price lists')).not.toBeInTheDocument()
  })
})
