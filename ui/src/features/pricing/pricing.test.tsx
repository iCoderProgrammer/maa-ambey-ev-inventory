import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import type { Brand, Variant, VehicleModel } from '@/features/catalogue/api/contracts'
import { renderApp, testMe } from '@/test/render'
import type { PriceItem, PriceList } from './api/pricing'
import { formatINR, localInputToRFC3339, pricePeriod, toLocalInput } from './lib/format'

// Placeholder data only ("Brand A", ₹1,000): no real price is invented.

const stamps = { version: 2, createdAt: '2026-10-13T09:00:00Z', updatedAt: '2026-10-13T09:00:00Z' }
const prov = {
  confidence: 'SUPPLIER_CONFIRMED' as const,
  sourceUrl: 'https://example.com/s',
  sourceDocumentId: null,
  sourceDate: '2026-10-01',
  verifiedBy: 'u-x',
  verifiedAt: '2026-10-13T09:00:00Z',
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
  lifecycleStatus: 'ACTIVE',
  ...prov,
  ...stamps,
}
const variant: Variant = {
  id: 'v1',
  brandId: 'b1',
  modelId: 'm1',
  name: 'Variant 1',
  variantCode: 'STD',
  manufacturerCode: null,
  lifecycleStatus: 'ACTIVE',
  ...prov,
  ...stamps,
}
const list: PriceList = {
  id: 'l1',
  brandId: 'b1',
  name: 'Showroom',
  currency: 'INR',
  lifecycleStatus: 'ACTIVE',
  createdBy: 'u-x',
  ...stamps,
}
const DAY = 86_400_000
function price(id: string, extra: Partial<PriceItem>): PriceItem {
  return {
    id,
    priceListId: 'l1',
    brandId: 'b1',
    modelId: 'm1',
    variantId: 'v1',
    priceType: 'SELLING',
    amount: '1000.00',
    currency: 'INR',
    effectiveFrom: new Date(Date.now() - DAY).toISOString(),
    effectiveTo: null,
    status: 'APPROVED',
    taxConfigurationId: null,
    sourceNote: null,
    createdBy: 'u-drafter',
    approvedBy: 'u-approver',
    approvedAt: '2026-10-13T09:00:00Z',
    approvalReference: 'REF-1',
    supersedesId: null,
    supersededById: null,
    ...stamps,
    ...extra,
  }
}
const current = price('p-cur', {})
const future = price('p-fut', {
  amount: '1200.00',
  effectiveFrom: new Date(Date.now() + 10 * DAY).toISOString(),
})
const draft = price('p-draft', {
  status: 'DRAFT',
  amount: '1300.00',
  approvedBy: null,
  approvedAt: null,
  approvalReference: null,
  createdBy: 'user-1',
  effectiveFrom: new Date(Date.now() + 20 * DAY).toISOString(),
})
const othersDraft = { ...draft, id: 'p-draft2', createdBy: 'u-drafter' }

const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({
  ...testMe,
  permissions: ['catalogue.view', ...permissions],
})
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-9', details } }, { status })

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function api(handler: Handler, items: PriceItem[] = [current, future, draft]) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    if (p === '/api/v1/variants/v1') return Promise.resolve(Response.json({ data: variant }))
    if (p === '/api/v1/models/m1') return Promise.resolve(Response.json({ data: model }))
    if (p === '/api/v1/brands/b1') return Promise.resolve(Response.json({ data: brand }))
    if (p === '/api/v1/colours') return Promise.resolve(Response.json({ data: page([]) }))
    if (p === '/api/v1/price-lists') return Promise.resolve(Response.json({ data: page([list]) }))
    if (p === '/api/v1/price-items') {
      const t = url.searchParams.get('priceType')
      return Promise.resolve(
        Response.json({ data: page(items.filter((i) => !t || i.priceType === t)) }),
      )
    }
    if (p.endsWith('/health')) {
      return Promise.resolve(
        Response.json({ data: { status: 'ok', service: 'api', version: 'v', time: 'x' } }),
      )
    }
    if (p.endsWith('/ready')) {
      return Promise.resolve(
        Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
      )
    }
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}

function bodyOf(init: RequestInit | undefined): unknown {
  return typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('price formatting', () => {
  it('formats rupees and classifies periods from server data', () => {
    expect(formatINR('125000.00')).toBe('₹1,25,000.00')
    const now = new Date('2026-10-14T10:00:00Z')
    const base = price('x', { effectiveFrom: '2026-10-01T00:00:00Z' })
    expect(pricePeriod(base, now)).toBe('CURRENT')
    expect(pricePeriod({ ...base, effectiveTo: '2026-10-10T00:00:00Z' }, now)).toBe('PAST')
    expect(pricePeriod({ ...base, effectiveFrom: '2026-11-01T00:00:00Z' }, now)).toBe('FUTURE')
    expect(pricePeriod({ ...base, status: 'DRAFT' }, now)).toBe('DRAFT')
    const local = toLocalInput('2026-10-14T10:00:00Z')
    expect(localInputToRFC3339(local)).toBe('2026-10-14T10:00:00.000Z')
  })
})

describe('variant price history', () => {
  it('stays hidden without a price permission and never asks for prices', async () => {
    const spy = api(() => undefined)
    renderApp('/admin/catalogue/variants/v1', { me: me([]) })
    await screen.findByRole('heading', { level: 1, name: 'Variant 1' })
    expect(screen.queryByText('Price history')).not.toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/price'))).toBe(false)
  })

  it('shows current, future and draft prices with the snapshot warning; selling only without cost permission', async () => {
    const spy = api(() => undefined)
    renderApp('/admin/catalogue/variants/v1', { me: me(['catalogue.price.view']) })
    const table = await screen.findByRole('table', { name: 'Price history' })
    expect(within(table).getByText('₹1,000.00')).toBeInTheDocument()
    expect(within(table).getByText('Current')).toBeInTheDocument()
    expect(within(table).getByText('Future')).toBeInTheDocument()
    expect(within(table).getByText('Draft — not approved')).toBeInTheDocument()
    expect(screen.getAllByText(/never changes existing quotations/).length).toBeGreaterThan(0)
    // View only: no actions, no type switch, only SELLING requested.
    expect(
      screen.queryByRole('button', { name: /Draft price|Approve|Change price/ }),
    ).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Price type')).not.toBeInTheDocument()
    const priceCalls = spy.mock.calls
      .map(([u]) => new URL(u as string, 'http://x'))
      .filter((u) => u.pathname === '/api/v1/price-items')
    expect(priceCalls.every((u) => u.searchParams.get('priceType') === 'SELLING')).toBe(true)
  })

  it('offers approval only of other people’s drafts and explains server refusals', async () => {
    const user = userEvent.setup()
    api(
      (u) =>
        u.pathname === '/api/v1/price-items/p-draft2/approve'
          ? err(409, 'PRICE_OVERLAP', [{ field: 'conflictingPriceId', message: 'p-fut' }])
          : undefined,
      [current, draft, othersDraft],
    )
    renderApp('/admin/catalogue/variants/v1', {
      me: me(['catalogue.price.view', 'catalogue.price.approve']),
    })
    const table = await screen.findByRole('table', { name: 'Price history' })
    // Own draft (createdBy user-1 = test user): no Approve button; someone else's: one.
    expect(within(table).getAllByRole('button', { name: 'Approve' })).toHaveLength(1)
    await user.click(within(table).getByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog', { name: 'Approve price' })
    expect(within(dialog).getByText(/cannot approve a price you drafted/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Approve price' }))
    expect(
      await within(dialog).findByText(/already covers part of this period/),
    ).toBeInTheDocument()
  })

  it('lists the server’s unmet approval requirements', async () => {
    const user = userEvent.setup()
    api(
      (u) =>
        u.pathname === '/api/v1/price-items/p-draft2/approve'
          ? err(409, 'APPROVAL_BLOCKED', [
              { field: 'variant_verification', message: 'Verify the variant first.' },
            ])
          : undefined,
      [othersDraft],
    )
    renderApp('/admin/catalogue/variants/v1', {
      me: me(['catalogue.price.view', 'catalogue.price.approve']),
    })
    const table = await screen.findByRole('table', { name: 'Price history' })
    await user.click(within(table).getByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog', { name: 'Approve price' })
    await user.click(within(dialog).getByRole('button', { name: 'Approve price' }))
    expect(await within(dialog).findByText('Variant is verified')).toBeInTheDocument()
  })

  it('validates a draft, sends a decimal string and RFC 3339 dates, and supersedes with a link', async () => {
    const user = userEvent.setup()
    let sent: unknown
    api((u, init) => {
      if (u.pathname === '/api/v1/price-items' && init?.method === 'POST') {
        sent = bodyOf(init)
        return Response.json({ data: { ...draft, id: 'p-new' } }, { status: 201 })
      }
      return undefined
    })
    renderApp('/admin/catalogue/variants/v1', {
      me: me(['catalogue.price.view', 'catalogue.price.manage']),
    })
    const table = await screen.findByRole('table', { name: 'Price history' })
    await user.click(within(table).getAllByRole('button', { name: /Change price/ })[0]!)
    const dialog = await screen.findByRole('dialog', { name: 'Draft a replacement price' })
    expect(within(dialog).getByText(/Replaces the approved price of/)).toBeInTheDocument()
    expect(within(dialog).getByText(/no approved tax configuration exists/i)).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/Amount/), '12.345')
    await user.click(within(dialog).getByRole('button', { name: 'Save as draft' }))
    expect(await within(dialog).findByText(/Enter an amount like/)).toBeInTheDocument()
    await user.clear(within(dialog).getByLabelText(/Amount/))
    await user.type(within(dialog).getByLabelText(/Amount/), '1250.5')
    await user.click(within(dialog).getByRole('button', { name: 'Save as draft' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(sent).toMatchObject({
      priceListId: 'l1',
      variantId: 'v1',
      priceType: 'SELLING',
      amount: '1250.5',
      supersedesId: 'p-cur',
      effectiveTo: null,
    })
    expect((sent as { effectiveFrom: string }).effectiveFrom).toMatch(/Z$/)
  })

  it('looks up the price valid at a time and says when there is none', async () => {
    const user = userEvent.setup()
    let n = 0
    api((u) => {
      if (u.pathname !== '/api/v1/prices/lookup') return undefined
      n++
      return n === 1 ? Response.json({ data: current }) : err(404, 'PRICE_NOT_FOUND')
    })
    renderApp('/admin/catalogue/variants/v1', { me: me(['catalogue.price.view']) })
    await screen.findByRole('table', { name: 'Price history' })
    await user.click(screen.getByRole('button', { name: /Look up/ }))
    expect(await screen.findByText(/approved price valid from/)).toBeInTheDocument()
    const input = screen.getByLabelText(/Price valid at/)
    await user.clear(input)
    await user.type(input, '2020-01-01T10:00')
    await user.click(screen.getByRole('button', { name: /Look up/ }))
    expect(await screen.findByText('No approved price is valid at that time.')).toBeInTheDocument()
  })

  it('shows cost prices only to cost viewers, behind a type switch', async () => {
    const user = userEvent.setup()
    const cost = price('p-cost', { priceType: 'COST', amount: '700.00' })
    api(() => undefined, [current, cost])
    renderApp('/admin/catalogue/variants/v1', {
      me: me(['catalogue.price.view', 'catalogue.cost_price.view']),
    })
    await screen.findByRole('table', { name: 'Price history' })
    expect(screen.queryByText('₹700.00')).not.toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Price type'), 'COST')
    expect((await screen.findAllByText('₹700.00')).length).toBeGreaterThan(0)
  })

  it('explains an empty history and a brand without price lists', async () => {
    api((u) =>
      u.pathname === '/api/v1/price-lists' ? Response.json({ data: page([]) }) : undefined,
    )
    renderApp('/admin/catalogue/variants/v1', { me: me(['catalogue.price.view']) })
    expect(await screen.findByText('No price list for this brand')).toBeInTheDocument()
  })
})

describe('brand price lists', () => {
  it('creates a price list after the server confirms', async () => {
    const user = userEvent.setup()
    let sent: unknown
    api((u, init) => {
      if (u.pathname === '/api/v1/brands/b1/activation-readiness') {
        return Response.json({ data: { brandId: 'b1', ready: false, checks: [] } })
      }
      if (u.pathname === '/api/v1/models') return Response.json({ data: page([]) })
      if (u.pathname === '/api/v1/price-lists' && init?.method === 'POST') {
        sent = bodyOf(init)
        return err(409, 'DUPLICATE', [{ field: 'name', message: 'is already used' }])
      }
      return undefined
    })
    renderApp('/admin/catalogue/brands/b1', {
      me: me(['catalogue.price.view', 'catalogue.price.manage']),
    })
    expect(await screen.findByRole('table', { name: 'Price lists' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Add price list' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add price list' })
    await user.type(within(dialog).getByLabelText(/Name/), 'Showroom')
    await user.click(within(dialog).getByRole('button', { name: 'Add price list' }))
    expect(await within(dialog).findByText('Already used — choose another')).toBeInTheDocument()
    expect(sent).toEqual({ brandId: 'b1', name: 'Showroom' })
  })
})
