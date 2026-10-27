import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import type { Brand, BrandReadiness, Variant, VehicleModel } from './api/contracts'

// Placeholder data only ("Brand A", "M1"): no real brand, model, supplier or price is invented.

const ALL = [
  'catalogue.view',
  'catalogue.manage',
  'catalogue.verify',
  'catalogue.activate',
  'catalogue.archive',
  'catalogue.brand.authorise',
  'catalogue.brand.activate',
]
const viewer: Me = { ...testMe, permissions: ['catalogue.view'] }
const admin: Me = { ...testMe, permissions: ALL }

const stamps = { version: 3, createdAt: '2026-10-13T09:00:00Z', updatedAt: '2026-10-13T09:00:00Z' }
const unverified = {
  confidence: 'UNVERIFIED' as const,
  sourceUrl: null,
  sourceDocumentId: null,
  sourceDate: null,
  verifiedBy: null,
  verifiedAt: null,
}
function brand(id: string, name: string, extra: Partial<Brand> = {}): Brand {
  return {
    id,
    name,
    slug: name.toLowerCase().replace(/\s+/g, '-'),
    lifecycleStatus: 'PENDING_VERIFICATION',
    authorisationStatus: 'UNVERIFIED',
    manufacturerLegalName: null,
    authorisationReference: null,
    authorisationEvidenceId: null,
    verifiedBy: null,
    verifiedAt: null,
    operational: false,
    ...stamps,
    ...extra,
  }
}
function model(
  id: string,
  brandId: string,
  name: string,
  extra: Partial<VehicleModel> = {},
): VehicleModel {
  return {
    id,
    brandId,
    name,
    internalCode: name.toUpperCase().replace(/\s+/g, ''),
    manufacturerCode: null,
    vehicleType: null,
    lifecycleStatus: 'PENDING_VERIFICATION',
    ...unverified,
    ...stamps,
    ...extra,
  }
}
const brandA = brand('b-a', 'Brand A')
const brandB = brand('b-b', 'Brand B', { lifecycleStatus: 'ARCHIVED' })
const modelA1 = model('m-a1', 'b-a', 'Model A1')
const modelB1 = model('m-b1', 'b-b', 'Model B1')
const page = <T,>(items: T[], total = items.length) => ({ items, page: 1, limit: 25, total })

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function err(status: number, code: string, details?: { field: string; message: string }[]) {
  return Response.json(
    { error: { code, message: 'Refused by server.', requestId: 'req-77', details } },
    { status },
  )
}

/** Routes /api/v1 calls to handlers; health/ready answer OK. Returns the fetch spy. */
function api(handler: Handler) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const raw = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const url = new URL(raw, 'http://localhost')
    const res = handler(url, init)
    if (res) return Promise.resolve(res)
    if (url.pathname.endsWith('/health')) {
      return Promise.resolve(
        Response.json({ data: { status: 'ok', service: 'api', version: 'v', time: 'x' } }),
      )
    }
    if (url.pathname.endsWith('/ready')) {
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

function calls(spy: ReturnType<typeof api>, path: string) {
  return spy.mock.calls
    .map(([input, init]) => ({ url: new URL(input as string, 'http://localhost'), init }))
    .filter((c) => c.url.pathname === `/api/v1${path}`)
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('catalogue access', () => {
  it('denies the page without catalogue.view and never calls the catalogue API', async () => {
    const spy = api(() => undefined)
    renderApp('/admin/catalogue')
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { level: 1, name: 'Catalogue & Brands' }),
    ).toBeInTheDocument()
    expect(calls(spy, '/brands')).toHaveLength(0)
  })

  it('shows the server refusal when the API denies a granted-looking user', async () => {
    api((u) => (u.pathname === '/api/v1/brands' ? err(403, 'FORBIDDEN') : undefined))
    renderApp('/admin/catalogue', { me: viewer })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
  })
})

describe('brand list', () => {
  it('lists brands from the API with separate lifecycle, authorisation and purchase/sales status', async () => {
    api((u) =>
      u.pathname === '/api/v1/brands' ? Response.json({ data: page([brandA, brandB]) }) : undefined,
    )
    renderApp('/admin/catalogue', { me: viewer })
    const table = await screen.findByRole('table', { name: 'Brands' })
    const rowA = within(table).getByRole('row', { name: /Brand A/ })
    expect(within(rowA).getByText('Pending verification')).toBeInTheDocument()
    expect(within(rowA).getByText('Authorisation unverified')).toBeInTheDocument()
    expect(within(rowA).getByText('Not enabled for purchase & sales')).toBeInTheDocument()
    expect(within(table).getByText('Archived')).toBeInTheDocument()
    expect(screen.getByText('1–2 of 2 brands')).toBeInTheDocument()
    // A viewer gets no create button.
    expect(screen.queryByRole('button', { name: 'Add brand' })).not.toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent(/not confirmed facts/)
  })

  it('shows "enabled" only when the server says so', async () => {
    const live = brand('b-l', 'Brand L', {
      lifecycleStatus: 'ACTIVE',
      authorisationStatus: 'AUTHORISED',
      operational: true,
    })
    api((u) =>
      u.pathname === '/api/v1/brands' ? Response.json({ data: page([live]) }) : undefined,
    )
    renderApp('/admin/catalogue', { me: viewer })
    expect((await screen.findAllByText('Enabled for purchase & sales')).length).toBeGreaterThan(0)
  })

  it('sends search and filters to the server and explains no results', async () => {
    const user = userEvent.setup()
    const spy = api((u) =>
      u.pathname === '/api/v1/brands'
        ? Response.json({ data: page(u.searchParams.get('q') ? [] : [brandA]) })
        : undefined,
    )
    renderApp('/admin/catalogue', { me: viewer })
    await screen.findByRole('table', { name: 'Brands' })
    await user.type(screen.getByLabelText('Search brands'), 'zz{Enter}')
    expect(await screen.findByText('No brands match these filters')).toBeInTheDocument()
    expect(calls(spy, '/brands').some((c) => c.url.searchParams.get('q') === 'zz')).toBe(true)
    await user.click(screen.getByRole('button', { name: 'Clear filters' }))
    expect(await screen.findByRole('table', { name: 'Brands' })).toBeInTheDocument()
  })

  it('shows the empty state with the permitted action, and errors with a reference', async () => {
    api((u) => (u.pathname === '/api/v1/brands' ? Response.json({ data: page([]) }) : undefined))
    const { unmount } = renderApp('/admin/catalogue', { me: admin })
    expect(await screen.findByText('No brands yet')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Add brand' }).length).toBeGreaterThan(0)
    unmount()
    vi.restoreAllMocks()

    api((u) => (u.pathname === '/api/v1/brands' ? err(500, 'INTERNAL_ERROR') : undefined))
    renderApp('/admin/catalogue', { me: viewer })
    expect(await screen.findByText('Could not load brands')).toBeInTheDocument()
    expect(screen.getByText('req-77')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('validates the form, shows a server duplicate on the field, and opens the new brand only after the server confirms', async () => {
    const user = userEvent.setup()
    let attempts = 0
    const spy = api((u, init) => {
      if (u.pathname === '/api/v1/brands' && init?.method === 'POST') {
        attempts++
        return attempts === 1
          ? err(409, 'DUPLICATE', [{ field: 'name', message: 'is already used' }])
          : Response.json({ data: brand('b-new', 'Brand New') }, { status: 201 })
      }
      if (u.pathname === '/api/v1/brands') return Response.json({ data: page([]) })
      if (u.pathname === '/api/v1/brands/b-new')
        return Response.json({ data: brand('b-new', 'Brand New') })
      if (u.pathname.startsWith('/api/v1/brands/b-new/')) {
        return Response.json({ data: { brandId: 'b-new', ready: false, checks: [] } })
      }
      if (u.pathname === '/api/v1/models') return Response.json({ data: page([]) })
      return undefined
    })
    renderApp('/admin/catalogue', { me: admin })
    await user.click((await screen.findAllByRole('button', { name: 'Add brand' }))[0]!)
    const dialog = await screen.findByRole('dialog', { name: 'Add brand' })

    await user.click(within(dialog).getByRole('button', { name: 'Add brand' }))
    expect(await within(dialog).findByText('Name is required')).toBeInTheDocument()
    expect(calls(spy, '/brands').filter((c) => c.init?.method === 'POST')).toHaveLength(0)

    await user.type(within(dialog).getByLabelText(/Brand name/), 'Brand New')
    await user.click(within(dialog).getByRole('button', { name: 'Add brand' }))
    expect(await within(dialog).findByText('Already used — choose another')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Add brand' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Brand New' })).toBeInTheDocument()
    const posts = calls(spy, '/brands').filter((c) => c.init?.method === 'POST')
    const body = bodyOf(posts.at(-1)?.init)
    expect(body).toEqual({ name: 'Brand New' })
  })
})

describe('brand → model → variant selection', () => {
  it('only offers the chosen brand’s models and never archived brands when adding a variant', async () => {
    const user = userEvent.setup()
    const spy = api((u, init) => {
      if (u.pathname === '/api/v1/brands') return Response.json({ data: page([brandA, brandB]) })
      if (u.pathname === '/api/v1/models') {
        const b = u.searchParams.get('brandId')
        return Response.json({
          data: page([modelA1, modelB1].filter((m) => !b || m.brandId === b)),
        })
      }
      if (u.pathname === '/api/v1/variants' && init?.method === 'POST') {
        return err(422, 'VALIDATION_FAILED', [
          { field: 'brandId', message: "does not match the model's brand" },
        ])
      }
      if (u.pathname === '/api/v1/variants') return Response.json({ data: page([]) })
      return undefined
    })
    renderApp('/admin/catalogue/variants', { me: admin })
    await user.click((await screen.findAllByRole('button', { name: 'Add variant' }))[0]!)
    const dialog = await screen.findByRole('dialog', { name: 'Add variant' })
    const brandSelect = within(dialog).getByLabelText(/^Brand/)
    const modelSelect = within(dialog).getByLabelText(/^Model/)
    expect(modelSelect).toBeDisabled()
    // Archived brands are not offered.
    await waitFor(() => {
      expect(within(brandSelect).getByRole('option', { name: 'Brand A' })).toBeInTheDocument()
    })
    expect(within(brandSelect).queryByRole('option', { name: 'Brand B' })).not.toBeInTheDocument()

    await user.selectOptions(brandSelect, 'b-a')
    await waitFor(() => {
      expect(modelSelect).toBeEnabled()
    })
    const options = within(modelSelect)
      .getAllByRole('option')
      .map((o) => o.textContent)
    expect(options).toEqual(['Choose a model', 'Model A1 (MODELA1)'])
    expect(calls(spy, '/models').some((c) => c.url.searchParams.get('brandId') === 'b-a')).toBe(
      true,
    )

    await user.selectOptions(modelSelect, 'm-a1')
    await user.type(within(dialog).getByLabelText(/Variant name/), 'Variant 1')
    await user.type(within(dialog).getByLabelText(/Variant code/), 'std')
    await user.click(within(dialog).getByRole('button', { name: 'Add variant' }))
    // The server's refusal is shown; the brand never comes from the client.
    expect(await within(dialog).findByText("does not match the model's brand")).toBeInTheDocument()
    const sent = bodyOf(
      calls(spy, '/variants').find((c) => c.init?.method === 'POST')?.init,
    ) as Record<string, unknown>
    expect(sent).toEqual({
      modelId: 'm-a1',
      name: 'Variant 1',
      variantCode: 'STD',
      manufacturerCode: null,
    })
  })

  it('narrows the variants list model filter to the chosen brand', async () => {
    const user = userEvent.setup()
    api((u) => {
      if (u.pathname === '/api/v1/brands') return Response.json({ data: page([brandA, brandB]) })
      if (u.pathname === '/api/v1/models') {
        const b = u.searchParams.get('brandId')
        return Response.json({
          data: page([modelA1, modelB1].filter((m) => !b || m.brandId === b)),
        })
      }
      if (u.pathname === '/api/v1/variants') return Response.json({ data: page([]) })
      return undefined
    })
    renderApp('/admin/catalogue/variants', { me: viewer })
    const modelFilter = await screen.findByLabelText('Model')
    expect(modelFilter).toBeDisabled()
    await user.selectOptions(await screen.findByLabelText('Brand'), 'b-b')
    await waitFor(() => {
      expect(
        within(modelFilter)
          .getAllByRole('option')
          .map((o) => o.textContent),
      ).toEqual(['All models of this brand', 'Model B1 (MODELB1)'])
    })
  })
})

const readiness: BrandReadiness = {
  brandId: 'b-a',
  ready: false,
  checks: [
    { key: 'lifecycle', met: true, pending: false, message: 'ok' },
    {
      key: 'authorisation',
      met: false,
      pending: false,
      message: 'Record AUTHORISED status for the brand.',
    },
    {
      key: 'supplier_link',
      met: false,
      pending: true,
      message: 'Supplier master not available yet.',
    },
    { key: 'approved_prices', met: false, pending: true, message: 'Prices not available yet.' },
  ],
}

describe('brand detail', () => {
  it('shows readiness from the server, supplier and price gaps, and keeps activation unconfirmable while not ready', async () => {
    const user = userEvent.setup()
    const spy = api((u) => {
      if (u.pathname === '/api/v1/brands/b-a') return Response.json({ data: brandA })
      if (u.pathname === '/api/v1/brands/b-a/activation-readiness')
        return Response.json({ data: readiness })
      if (u.pathname === '/api/v1/models') return Response.json({ data: page([modelA1]) })
      return undefined
    })
    renderApp('/admin/catalogue/brands/b-a', { me: admin })
    expect(await screen.findByRole('heading', { level: 1, name: 'Brand A' })).toBeInTheDocument()
    expect(
      await screen.findByText('1 of 4 requirements met · 2 depend on features not built yet'),
    ).toBeInTheDocument()
    expect(screen.getByText(/Supplier master not available yet/)).toBeInTheDocument()
    expect(
      screen.getByText(/No verified, current supplier relationship is recorded/),
    ).toBeInTheDocument()
    expect(screen.getByText(/Authorisation not verified/)).toBeInTheDocument()
    expect(screen.getByText(/No price is shown or implied/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Enable…' }))
    const dialog = await screen.findByRole('dialog', {
      name: 'Enable brand for purchase and sales',
    })
    expect(
      within(dialog).getByRole('button', { name: 'Enable for purchase & sales' }),
    ).toBeDisabled()
    expect(calls(spy, '/brands/b-a/activate')).toHaveLength(0)
  })

  it('hides actions the user lacks permission for', async () => {
    api((u) => {
      if (u.pathname === '/api/v1/brands/b-a') return Response.json({ data: brandA })
      if (u.pathname === '/api/v1/brands/b-a/activation-readiness')
        return Response.json({ data: readiness })
      if (u.pathname === '/api/v1/models') return Response.json({ data: page([]) })
      return undefined
    })
    renderApp('/admin/catalogue/brands/b-a', { me: viewer })
    await screen.findByRole('heading', { level: 1, name: 'Brand A' })
    for (const name of ['Edit', 'Archive', 'Record', 'Enable…', 'Add model']) {
      expect(screen.queryByRole('button', { name })).not.toBeInTheDocument()
    }
  })

  it('reports an unknown brand as not found', async () => {
    api(() => undefined)
    renderApp('/admin/catalogue/brands/nope', { me: viewer })
    expect(await screen.findByText('This brand does not exist')).toBeInTheDocument()
  })
})

describe('model lifecycle actions', () => {
  function modelApi(activate: () => Response, archive?: (body: unknown) => Response) {
    return api((u, init) => {
      if (u.pathname === '/api/v1/models/m-a1/activate') return activate()
      if (u.pathname === '/api/v1/models/m-a1/archive' && archive) return archive(bodyOf(init))
      if (u.pathname === '/api/v1/models/m-a1') return Response.json({ data: modelA1 })
      if (u.pathname === '/api/v1/brands/b-a') return Response.json({ data: brandA })
      if (u.pathname === '/api/v1/variants') return Response.json({ data: page([] as Variant[]) })
      return undefined
    })
  }

  it('confirms activation and lists the server’s unmet requirements when refused', async () => {
    const user = userEvent.setup()
    modelApi(() =>
      err(409, 'ACTIVATION_BLOCKED', [
        { field: 'verification', message: "Record the model's source and verification first." },
      ]),
    )
    renderApp('/admin/catalogue/models/m-a1', { me: admin })
    expect(
      await screen.findByText(/No supplier or client confirmation has been recorded/),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Activate' }))
    const dialog = await screen.findByRole('dialog', { name: 'Activate model' })
    await user.click(within(dialog).getByRole('button', { name: 'Activate model' }))
    expect(await within(dialog).findByText('Source and verification recorded')).toBeInTheDocument()
    expect(within(dialog).getByText(/Activation is not allowed yet/)).toBeInTheDocument()
    // Still pending: nothing shows the model as active.
    expect(screen.getAllByText('Pending verification').length).toBeGreaterThan(0)
  })

  it('offers a reload when someone else changed the record', async () => {
    const user = userEvent.setup()
    modelApi(() => err(409, 'VERSION_CONFLICT'))
    renderApp('/admin/catalogue/models/m-a1', { me: admin })
    await user.click(await screen.findByRole('button', { name: 'Activate' }))
    const dialog = await screen.findByRole('dialog', { name: 'Activate model' })
    await user.click(within(dialog).getByRole('button', { name: 'Activate model' }))
    expect(await within(dialog).findByText(/Someone else changed this record/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Reload latest version' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  it('archives with a confirmation and an optional reason', async () => {
    const user = userEvent.setup()
    let sent: unknown
    modelApi(
      () => err(500, 'INTERNAL_ERROR'),
      (body) => {
        sent = body
        return Response.json({ data: { ...modelA1, lifecycleStatus: 'ARCHIVED', version: 4 } })
      },
    )
    renderApp('/admin/catalogue/models/m-a1', { me: admin })
    await user.click(await screen.findByRole('button', { name: 'Archive' }))
    const dialog = await screen.findByRole('dialog', { name: 'Archive model' })
    expect(within(dialog).getByText(/Nothing is deleted/)).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/Reason/), 'Discontinued')
    await user.click(within(dialog).getByRole('button', { name: 'Archive model' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(sent).toEqual({ version: 3, reason: 'Discontinued' })
  })

  it('reports network loss without claiming success', async () => {
    const user = userEvent.setup()
    api((u) => {
      if (u.pathname === '/api/v1/models/m-a1/activate') throw new TypeError('Failed to fetch')
      if (u.pathname === '/api/v1/models/m-a1') return Response.json({ data: modelA1 })
      if (u.pathname === '/api/v1/brands/b-a') return Response.json({ data: brandA })
      if (u.pathname === '/api/v1/variants') return Response.json({ data: page([]) })
      return undefined
    })
    renderApp('/admin/catalogue/models/m-a1', { me: admin })
    await user.click(await screen.findByRole('button', { name: 'Activate' }))
    const dialog = await screen.findByRole('dialog', { name: 'Activate model' })
    await user.click(within(dialog).getByRole('button', { name: 'Activate model' }))
    expect(
      await within(dialog).findByText(/Cannot reach the server. Nothing was saved/),
    ).toBeInTheDocument()
  })
})
