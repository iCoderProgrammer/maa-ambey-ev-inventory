import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import type { Brand } from '@/features/catalogue/api/contracts'
import type { Supplier } from '@/features/suppliers/api/suppliers'
import { renderApp, testMe } from '@/test/render'
import type { PODetail, POLine, PurchaseOrder } from './api/purchaseOrders'

// Placeholder data only ("Example Supplier", "Brand One", 1000.00): nothing real.

const t = '2026-10-23T09:00:00Z'
const stamps = { version: 3, createdAt: t, updatedAt: t }
const po: PurchaseOrder = {
  id: 'po1',
  number: 'PO-000001',
  supplierId: 's1',
  supplierName: 'Example Supplier Private Limited',
  supplierStatus: 'VERIFIED',
  status: 'DRAFT',
  expectedDate: '2099-01-31',
  notes: null,
  currency: 'INR',
  createdBy: testMe.user.id,
  createdByName: 'Test User',
  submittedBy: null,
  submittedAt: null,
  approvalBasis: null,
  approvedBy: null,
  approvedAt: null,
  approvalNote: null,
  thresholdApplied: null,
  cancelledBy: null,
  cancelledAt: null,
  cancelReason: null,
  lineCount: 1,
  totalQuantity: 2,
  receivedQuantity: 0,
  remainingQuantity: 2,
  totalAmount: '1900.00',
  brandNames: ['Brand One'],
  ...stamps,
}
const line: POLine = {
  id: 'l1',
  lineNo: 1,
  brandId: 'b1',
  brandName: 'Brand One',
  brandStatus: 'ACTIVE',
  modelId: 'm1',
  modelName: 'Model One',
  modelStatus: 'ACTIVE',
  variantId: 'v1',
  variantName: 'Standard',
  variantStatus: 'ACTIVE',
  quantity: 2,
  unitCost: '950.00',
  lineTotal: '1900.00',
  receivedQuantity: 0,
  remainingQuantity: 2,
  referencePriceListId: null,
  referencePriceItemId: null,
  referenceUnitCost: null,
  referenceCapturedAt: null,
  referenceHidden: true,
  costVariance: null,
  note: null,
  warnings: [],
  createdAt: t,
  updatedAt: t,
}
const actions = { edit: true, submit: true, approve: false, return: false, cancel: true }
const draft: PODetail = {
  ...po,
  lines: [line],
  events: [
    {
      id: 'e1',
      action: 'CREATED',
      fromStatus: null,
      toStatus: 'DRAFT',
      actorId: testMe.user.id,
      actorName: 'Test User',
      note: null,
      createdAt: t,
    },
  ],
  blockers: [],
  approvalRequired: true,
  approvalReason: 'No approval threshold is configured, so every purchase order needs an approver.',
  actions,
}
const submitted: PODetail = {
  ...draft,
  status: 'SUBMITTED',
  createdBy: 'u-other',
  submittedBy: 'u-other',
  submittedAt: t,
  version: 4,
  approvalRequired: null,
  approvalReason: null,
  actions: { edit: false, submit: false, approve: true, return: true, cancel: false },
}
const settings = {
  approvalThreshold: null,
  firstTimeSupplierNeedsApproval: true,
  approvalReference: null,
  updatedBy: null,
  version: 1,
  updatedAt: null,
}
const supplier: Supplier = {
  id: 's1',
  legalName: 'Example Supplier Private Limited',
  tradeName: null,
  gstin: null,
  addressLine1: null,
  addressLine2: null,
  city: null,
  state: null,
  postalCode: null,
  status: 'VERIFIED',
  verificationReference: 'X',
  verificationDocumentId: 'a1',
  verifiedBy: 'u-x',
  verifiedAt: t,
  notes: null,
  createdBy: 'u-x',
  brandCount: 1,
  ...stamps,
}
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
const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({ ...testMe, permissions })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-7', details } }, { status })

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function api(handler: Handler = () => undefined, detail: PODetail = draft) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/purchase-orders') return json(page([po]))
    if (p === '/api/v1/purchase-orders/po1') return json(detail)
    if (p === '/api/v1/purchase-order-settings') return json(settings)
    if (p === '/api/v1/suppliers') return json(page([supplier]))
    if (p === '/api/v1/brands') return json(page([brand]))
    if (p === '/api/v1/models')
      return json(
        page([
          {
            id: 'm1',
            brandId: 'b1',
            name: 'Model One',
            internalCode: 'M1',
            manufacturerCode: null,
            vehicleType: null,
            lifecycleStatus: 'ACTIVE',
            ...provenance,
            ...stamps,
          },
        ]),
      )
    if (p === '/api/v1/variants')
      return json(
        page([
          {
            id: 'v1',
            brandId: 'b1',
            modelId: 'm1',
            name: 'Standard',
            variantCode: 'STD',
            manufacturerCode: null,
            lifecycleStatus: 'ACTIVE',
            ...provenance,
            ...stamps,
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

function bodyOf(init: RequestInit | undefined): unknown {
  return typeof init?.body === 'string' ? JSON.parse(init.body) : undefined
}

const VIEW = 'purchase_orders.view'
const BUYER = [
  VIEW,
  'purchase_orders.manage',
  'purchase_orders.cancel',
  'suppliers.view',
  'catalogue.view',
]

afterEach(() => {
  vi.restoreAllMocks()
})

describe('purchase order list', () => {
  it('is denied without purchase_orders.view and never calls the API', async () => {
    const spy = api()
    renderApp('/procurement/purchase-orders', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/purchase-order'))).toBe(false)
  })

  it('shows server totals, status and the approval policy; viewers get no create action', async () => {
    const user = userEvent.setup()
    const spy = api()
    renderApp('/procurement/purchase-orders', { me: me([VIEW]) })
    const table = await screen.findByRole('table', { name: 'Purchase orders' })
    expect(within(table).getByText('PO-000001')).toBeInTheDocument()
    expect(within(table).getByText('Draft')).toBeInTheDocument()
    expect(within(table).getByText('₹1,900.00')).toBeInTheDocument()
    expect(within(table).getByText(/0 \/ 2 received · 2 remaining/)).toBeInTheDocument()
    expect(await screen.findByText(/No approval threshold is configured/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'New purchase order' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Approval settings' })).not.toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Status'), 'SUBMITTED')
    await waitFor(() => {
      expect(
        spy.mock.calls.some(
          ([u]) => new URL(u as string, 'http://x').searchParams.get('status') === 'SUBMITTED',
        ),
      ).toBe(true)
    })
  })

  it('creates a draft with one idempotency key across retries and opens it after the server confirms', async () => {
    const user = userEvent.setup()
    const keys: string[] = []
    let n = 0
    api((u, init) => {
      if (u.pathname === '/api/v1/purchase-orders' && init?.method === 'POST') {
        keys.push(new Headers(init.headers).get('Idempotency-Key') ?? '')
        n++
        return n === 1
          ? err(500, 'INTERNAL_ERROR')
          : Response.json({ data: draft }, { status: 201 })
      }
      return undefined
    })
    const { router } = renderApp('/procurement/purchase-orders', { me: me(BUYER) })
    await user.click(await screen.findByRole('button', { name: 'New purchase order' }))
    const dialog = await screen.findByRole('dialog', { name: 'New purchase order' })
    await user.click(within(dialog).getByRole('button', { name: 'Create draft' }))
    expect(
      await within(dialog).findByText('Choose a supplier', { selector: 'span' }),
    ).toBeInTheDocument()
    await within(dialog).findByRole('option', { name: 'Example Supplier Private Limited' })
    await user.selectOptions(within(dialog).getByLabelText(/Supplier/), 's1')
    await user.click(within(dialog).getByRole('button', { name: 'Create draft' }))
    expect(await within(dialog).findByRole('alert')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/procurement/purchase-orders')
    await user.click(within(dialog).getByRole('button', { name: 'Create draft' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/procurement/purchase-orders/po1')
    })
    expect(keys).toHaveLength(2)
    expect(keys[0]).toMatch(/^[0-9a-f-]{36}$/)
    expect(keys[1]).toBe(keys[0])
  })
})

describe('purchase order detail', () => {
  it('shows lines, hidden snapshot, history and server blockers; submit refusals list them', async () => {
    const user = userEvent.setup()
    api(
      (u, init) =>
        u.pathname === '/api/v1/purchase-orders/po1/submit' && init?.method === 'POST'
          ? err(409, 'SUBMISSION_BLOCKED', [
              { field: 'supplier_brand_link', message: 'No current relationship with Brand One.' },
            ])
          : undefined,
      {
        ...draft,
        blockers: [
          {
            key: 'brand_active',
            message: 'Brand Brand One is not activated for purchase (Catalogue).',
            lineId: 'l1',
          },
        ],
      },
    )
    renderApp('/procurement/purchase-orders/po1', { me: me(BUYER) })
    expect(await screen.findByRole('heading', { level: 1, name: 'PO-000001' })).toBeInTheDocument()
    expect(screen.getByText(/Cannot be submitted yet/)).toBeInTheDocument()
    expect(screen.getByText('Brand activated for purchase')).toBeInTheDocument()
    expect(screen.getAllByText('Recorded (not visible to you)').length).toBeGreaterThan(0)
    expect(screen.getByText('Created')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Submit PO-000001?' })
    expect(within(dialog).getByText(/every purchase order needs an approver/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Submit purchase order' }))
    const alert = await within(dialog).findByRole('alert')
    expect(alert).toHaveTextContent(/cannot be submitted yet/)
    expect(alert).toHaveTextContent('Current verified supplier relationship with the brand')
    expect(alert).toHaveTextContent('req-7')
  })

  it('shows warnings and the visible snapshot with its difference', async () => {
    api(undefined, {
      ...draft,
      lines: [
        {
          ...line,
          referenceHidden: false,
          referencePriceListId: 'pl1',
          referencePriceItemId: 'pi1',
          referenceUnitCost: '900.00',
          referenceCapturedAt: t,
          costVariance: '50.00',
          warnings: ['PRICE_VARIANCE'],
        },
      ],
    })
    renderApp('/procurement/purchase-orders/po1', {
      me: me([...BUYER, 'catalogue.cost_price.view']),
    })
    expect(
      (await screen.findAllByText('Agreed cost differs from the catalogue cost')).length,
    ).toBeGreaterThan(0)
    expect(screen.getAllByText(/difference ₹50.00/).length).toBeGreaterThan(0)
  })

  it('adds a line through brand → model → variant and shows a duplicate refusal', async () => {
    const user = userEvent.setup()
    let sent: unknown
    api((u, init) => {
      if (u.pathname === '/api/v1/purchase-orders/po1/lines' && init?.method === 'POST') {
        sent = bodyOf(init)
        return err(409, 'DUPLICATE', [{ field: 'variantId', message: 'is already used' }])
      }
      return undefined
    })
    renderApp('/procurement/purchase-orders/po1', { me: me(BUYER) })
    await user.click(await screen.findByRole('button', { name: 'Add line' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add line' })
    await within(dialog).findByRole('option', { name: 'Brand One' })
    await user.selectOptions(within(dialog).getByLabelText(/Brand/), 'b1')
    await within(dialog).findByRole('option', { name: 'Model One' })
    await user.selectOptions(within(dialog).getByLabelText(/Model/), 'm1')
    await within(dialog).findByRole('option', { name: 'Standard' })
    await user.selectOptions(within(dialog).getByLabelText(/Variant/), 'v1')
    await user.clear(within(dialog).getByLabelText(/Quantity/))
    await user.type(within(dialog).getByLabelText(/Quantity/), '3')
    await user.type(within(dialog).getByLabelText(/Agreed unit cost/), '12.345')
    await user.click(within(dialog).getByRole('button', { name: 'Add line' }))
    expect(await within(dialog).findByText(/up to 2 decimals/)).toBeInTheDocument()
    expect(sent).toBeUndefined()
    await user.clear(within(dialog).getByLabelText(/Agreed unit cost/))
    await user.type(within(dialog).getByLabelText(/Agreed unit cost/), '1000.50')
    await user.click(within(dialog).getByRole('button', { name: 'Add line' }))
    expect(await within(dialog).findByText(/already on another line/)).toBeInTheDocument()
    expect(sent).toEqual({
      version: 3,
      brandId: 'b1',
      modelId: 'm1',
      variantId: 'v1',
      quantity: 3,
      unitCost: '1000.50',
      note: null,
    })
  })

  it('offers approval only as the server allows and shows success only after it confirms', async () => {
    const user = userEvent.setup()
    let approved = false
    api((u, init) => {
      if (u.pathname === '/api/v1/purchase-orders/po1/approve' && init?.method === 'POST') {
        approved = true
        return Response.json({
          data: {
            ...submitted,
            status: 'APPROVED',
            approvalBasis: 'APPROVER',
            approvedBy: testMe.user.id,
            approvedAt: t,
            version: 5,
            actions: { edit: false, submit: false, approve: false, return: false, cancel: true },
          },
        })
      }
      return undefined
    }, submitted)
    renderApp('/procurement/purchase-orders/po1', {
      me: me([VIEW, 'purchase_orders.approve']),
    })
    expect(await screen.findByText('Awaiting approval')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog', { name: 'Approve PO-000001?' })
    await user.click(within(dialog).getByRole('button', { name: 'Approve purchase order' }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(approved).toBe(true)
    expect(screen.getAllByText('Approved').length).toBeGreaterThan(0)
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument()
  })

  it('explains self-approval and stale versions without pretending success', async () => {
    const user = userEvent.setup()
    let n = 0
    api((u, init) => {
      if (u.pathname === '/api/v1/purchase-orders/po1/approve' && init?.method === 'POST') {
        n++
        return n === 1 ? err(403, 'SELF_APPROVAL_NOT_ALLOWED') : err(409, 'VERSION_CONFLICT')
      }
      return undefined
    }, submitted)
    renderApp('/procurement/purchase-orders/po1', { me: me([VIEW, 'purchase_orders.approve']) })
    await user.click(await screen.findByRole('button', { name: 'Approve' }))
    const dialog = await screen.findByRole('dialog', { name: 'Approve PO-000001?' })
    await user.click(within(dialog).getByRole('button', { name: 'Approve purchase order' }))
    expect(
      await within(dialog).findByText(/another authorised person must approve/),
    ).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Approve purchase order' }))
    expect(
      await within(dialog).findByRole('button', { name: 'Reload latest version' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Awaiting approval')).toBeInTheDocument()
  })

  it('keeps the heading and breadcrumbs when a detail link is denied without purchase_orders.view', async () => {
    const spy = api()
    renderApp('/procurement/purchase-orders/po1', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Purchase order' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Purchase Orders' })).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/purchase-order'))).toBe(false)
  })

  it('shows not-found for an unknown PO', async () => {
    api((u) => (u.pathname === '/api/v1/purchase-orders/nope' ? err(404, 'NOT_FOUND') : undefined))
    renderApp('/procurement/purchase-orders/nope', { me: me([VIEW]) })
    expect(await screen.findByText('This purchase order does not exist')).toBeInTheDocument()
  })
})
