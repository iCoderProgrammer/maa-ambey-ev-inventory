import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import type { GoodsReceipt, GRNDetail, GRNLine, ReceivablePO } from './api/goodsReceipts'

// Placeholder data only ("Example Supplier", "Brand One", "CH-0001"): nothing real.

const t = '2026-10-23T09:00:00Z'
const receipt: GoodsReceipt = {
  id: 'g1',
  number: 'GRN-000001',
  status: 'DRAFT',
  purchaseOrderId: 'po1',
  purchaseOrderNumber: 'PO-000001',
  purchaseOrderStatus: 'APPROVED',
  supplierId: 's1',
  supplierName: 'Example Supplier Private Limited',
  receivedDate: '2026-10-20',
  supplierReference: 'DC-1',
  notes: null,
  createdBy: testMe.user.id,
  createdByName: 'Test User',
  cancelledBy: null,
  cancelledAt: null,
  cancelReason: null,
  postedBy: null,
  postedAt: null,
  lineCount: 0,
  receivedQuantity: 0,
  acceptedQuantity: 0,
  rejectedQuantity: 0,
  unitCount: 0,
  version: 2,
  createdAt: t,
  updatedAt: t,
}
const line = (over: Partial<GRNLine> = {}): GRNLine => ({
  id: null,
  purchaseOrderLineId: 'pl1',
  lineNo: 1,
  brandId: 'b1',
  brandName: 'Brand One',
  modelName: 'Model One',
  variantName: 'Standard',
  orderedQuantity: 5,
  postedQuantity: 0,
  otherDraftsQuantity: 2,
  availableQuantity: 3,
  receivedQuantity: 0,
  acceptedQuantity: 0,
  rejectedQuantity: 0,
  excessQuantity: 0,
  overReceipt: null,
  discrepancyNote: null,
  units: [],
  warnings: [],
  ...over,
})
const actions = {
  edit: true,
  cancel: true,
  addDocument: true,
  authoriseOverReceipt: false,
  post: false,
}
const draft: GRNDetail = {
  ...receipt,
  purchaseOrderReceivable: true,
  lines: [
    line(),
    line({
      purchaseOrderLineId: 'pl2',
      lineNo: 2,
      variantName: 'Pro',
      otherDraftsQuantity: 0,
      availableQuantity: 2,
      orderedQuantity: 2,
    }),
  ],
  documents: [],
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
  stockPosted: false,
  actions,
}
const receivable: ReceivablePO = {
  id: 'po1',
  number: 'PO-000001',
  status: 'APPROVED',
  supplierId: 's1',
  supplierName: 'Example Supplier Private Limited',
  expectedDate: '2026-10-25',
  orderedQuantity: 7,
  postedQuantity: 0,
  draftQuantity: 2,
  availableQuantity: 5,
  openDraftCount: 1,
  brandNames: ['Brand One'],
}

const page = <T,>(items: T[]) => ({ items, page: 1, limit: 25, total: items.length })
const me = (permissions: string[]): Me => ({ ...testMe, permissions })
const err = (status: number, code: string, details?: { field: string; message: string }[]) =>
  Response.json({ error: { code, message: 'Refused.', requestId: 'req-9', details } }, { status })

type Handler = (url: URL, init: RequestInit | undefined) => Response | undefined

function api(handler: Handler = () => undefined, detail: GRNDetail = draft) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const custom = handler(url, init)
    if (custom) return Promise.resolve(custom)
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/goods-receipts') return json(page([receipt]))
    if (p === '/api/v1/goods-receipts/g1') return json(detail)
    if (p === '/api/v1/receivable-purchase-orders') return json(page([receivable]))
    if (p === '/api/v1/attachments/limits')
      return json({
        maxBytes: 1048576,
        allowedTypes: [
          { contentType: 'application/pdf', extensions: ['.pdf'], previewable: false },
        ],
      })
    if (p === '/api/v1/attachments') return json([])
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(err(404, 'NOT_FOUND'))
  })
}

function bodyOf(init: RequestInit | undefined): Record<string, unknown> {
  return typeof init?.body === 'string' ? (JSON.parse(init.body) as Record<string, unknown>) : {}
}

const VIEW = 'goods_receipts.view'
const RECEIVER = [VIEW, 'goods_receipts.manage', 'attachments.upload']

afterEach(() => {
  vi.restoreAllMocks()
})

describe('goods receipt list', () => {
  it('is denied without goods_receipts.view and never calls the API', async () => {
    const spy = api()
    renderApp('/procurement/goods-receipts', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/goods-receipts'))).toBe(false)
  })

  it('keeps the heading and breadcrumbs when a detail link is denied without goods_receipts.view', async () => {
    const spy = api()
    renderApp('/procurement/goods-receipts/g1', { me: me([]) })
    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Goods receipt' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Goods Receipts' })).toBeInTheDocument()
    expect(spy.mock.calls.some(([u]) => (u as string).includes('/goods-receipts'))).toBe(false)
  })

  it('lists receipts with server totals and says drafts change no stock; viewers cannot create', async () => {
    api()
    renderApp('/procurement/goods-receipts', { me: me([VIEW]) })
    const table = await screen.findByRole('table', { name: 'Goods receipts' })
    expect(within(table).getByText('GRN-000001')).toBeInTheDocument()
    expect(within(table).getByText('PO-000001')).toBeInTheDocument()
    expect(within(table).getByText('Draft')).toBeInTheDocument()
    expect(screen.getByText(/does not change stock/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'New goods receipt' })).not.toBeInTheDocument()
  })

  it('shows an empty state and a load error with retry', async () => {
    let fail = true
    api((u) => {
      if (u.pathname !== '/api/v1/goods-receipts') return undefined
      return fail ? err(500, 'INTERNAL_ERROR') : Response.json({ data: page([]) })
    })
    const user = userEvent.setup()
    renderApp('/procurement/goods-receipts', { me: me(RECEIVER) })
    expect(await screen.findByText('Could not load goods receipts')).toBeInTheDocument()
    fail = false
    await user.click(screen.getByRole('button', { name: /try again/i }))
    expect(await screen.findByText('No goods receipts yet')).toBeInTheDocument()
  })

  it('starts a draft against an approved PO with one idempotency key and opens it after the server confirms', async () => {
    const user = userEvent.setup()
    const keys: string[] = []
    let body: Record<string, unknown> = {}
    let n = 0
    api((u, init) => {
      if (u.pathname === '/api/v1/goods-receipts' && init?.method === 'POST') {
        keys.push(new Headers(init.headers).get('Idempotency-Key') ?? '')
        body = bodyOf(init)
        n++
        return n === 1
          ? err(500, 'INTERNAL_ERROR')
          : Response.json({ data: draft }, { status: 201 })
      }
      return undefined
    })
    const { router } = renderApp('/procurement/goods-receipts', { me: me(RECEIVER) })
    await user.click(
      (await screen.findAllByRole('button', { name: 'New goods receipt' })).at(0) as HTMLElement,
    )
    const dialog = await screen.findByRole('dialog', { name: 'New goods receipt' })
    expect(await within(dialog).findByText(/5 of 7 units still to receive/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Start draft' }))
    expect(await within(dialog).findByText(/Choose the purchase order/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('radio'))
    await user.type(within(dialog).getByLabelText('Supplier delivery reference'), 'DC-1')
    await user.click(within(dialog).getByRole('button', { name: 'Start draft' }))
    expect(await within(dialog).findByRole('alert')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/procurement/goods-receipts')
    await user.click(within(dialog).getByRole('button', { name: 'Start draft' }))
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/procurement/goods-receipts/g1')
    })
    expect(keys).toHaveLength(2)
    expect(keys[1]).toBe(keys[0])
    expect(body).toMatchObject({ purchaseOrderId: 'po1', supplierReference: 'DC-1' })
  })
})

describe('goods receipt draft editor', () => {
  it('saves a partial receipt with units, sends every PO line and keeps Post receipt separate and unavailable', async () => {
    const user = userEvent.setup()
    let sent: Record<string, unknown> = {}
    api((u, init) => {
      if (u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT') {
        sent = bodyOf(init)
        return Response.json({ data: { ...draft, version: 3 } })
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    expect(await screen.findByRole('heading', { level: 1, name: 'GRN-000001' })).toBeInTheDocument()
    expect(screen.getAllByText(/does not change stock/).length).toBeGreaterThan(0)
    // Server availability is shown per line.
    expect(screen.getAllByText('On other drafts')).toHaveLength(2)
    await user.type(screen.getByLabelText('Received (line 1)'), '2')
    await user.type(screen.getByLabelText('Rejected (line 1)'), '1')
    await user.type(screen.getByLabelText('Discrepancy note (line 1)'), 'One scratched')
    await user.click(screen.getByRole('button', { name: /Unit identifiers/ }))
    expect(screen.getByText(/0 of 2 units captured/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Add unit' }))
    await waitFor(() => {
      expect(screen.getByLabelText('Chassis/frame number (line 1, unit 1)')).toHaveFocus()
    })
    await user.keyboard(' ch-0001 ')
    await user.click(screen.getByRole('button', { name: /Review and save/ }))
    expect(screen.getByRole('button', { name: 'Post receipt' })).toBeDisabled()
    expect(
      screen.getByText(/It is not available yet; this draft does not change stock/),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    await waitFor(() => {
      expect(sent.version).toBe(2)
    })
    const lines = sent.lines as Record<string, unknown>[]
    expect(lines).toHaveLength(2)
    expect(lines[0]).toMatchObject({
      purchaseOrderLineId: 'pl1',
      receivedQuantity: 2,
      acceptedQuantity: 1,
      rejectedQuantity: 1,
      discrepancyNote: 'One scratched',
      overReceiptReason: null,
      units: [{ condition: 'ACCEPTED', chassisNumber: 'ch-0001', motorSerial: null }],
    })
    expect(lines[1]).toMatchObject({ purchaseOrderLineId: 'pl2', receivedQuantity: 0, units: [] })
    expect(await screen.findByText(/Draft saved .* Stock is unchanged/)).toBeInTheDocument()
  })

  it('warns about over-receipt and keeps entries when the server refuses it without an authorised override', async () => {
    const user = userEvent.setup()
    api((u, init) =>
      u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT'
        ? err(409, 'OVER_RECEIPT_REQUIRES_OVERRIDE', [
            { field: 'lines[0].receivedQuantity', message: 'only 3 left on PO line 1' },
          ])
        : undefined,
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.type(await screen.findByLabelText('Received (line 1)'), '4')
    expect(screen.getByText(/Over-receipt:/)).toBeInTheDocument()
    expect(screen.getByText(/Saving will be refused until someone allowed/)).toBeInTheDocument()
    expect(screen.queryByLabelText(/Over-receipt reason/)).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    const alert = await screen.findByText(/More is received than the purchase order has left/)
    expect(alert.closest('[role="alert"]')).toHaveTextContent('Line 1: received')
    expect(screen.getByText('only 3 left on PO line 1', { selector: 'span' })).toBeInTheDocument()
    expect(screen.getByLabelText('Received (line 1)')).toHaveValue('4')
  })

  it('lets an authorised user give the over-receipt reason', async () => {
    const user = userEvent.setup()
    let sent: Record<string, unknown> = {}
    api(
      (u, init) => {
        if (u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT') {
          sent = bodyOf(init)
          return Response.json({ data: { ...draft, version: 3 } })
        }
        return undefined
      },
      { ...draft, actions: { ...actions, authoriseOverReceipt: true } },
    )
    renderApp('/procurement/goods-receipts/g1', {
      me: me([...RECEIVER, 'goods_receipts.over_receipt']),
    })
    await user.type(await screen.findByLabelText('Received (line 1)'), '4')
    await user.type(screen.getByLabelText(/Over-receipt reason/), 'Supplier sent one extra')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    await waitFor(() => {
      expect((sent.lines as Record<string, unknown>[] | undefined)?.[0]).toMatchObject({
        receivedQuantity: 4,
        overReceiptReason: 'Supplier sent one extra',
      })
    })
  })

  it('shows duplicate chassis numbers at once and places server duplicates on the unit', async () => {
    const user = userEvent.setup()
    let puts = 0
    api((u, init) => {
      if (u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT') {
        puts++
        return err(409, 'DUPLICATE', [
          {
            field: 'lines[0].units[0].chassisNumber',
            message: 'is already on the draft goods receipt GRN-000007',
          },
        ])
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.type(await screen.findByLabelText('Received (line 1)'), '2')
    await user.click(screen.getByRole('button', { name: /Unit identifiers/ }))
    await user.click(screen.getByRole('button', { name: 'Add unit' }))
    await user.type(screen.getByLabelText('Chassis/frame number (line 1, unit 1)'), 'CH-1')
    await user.click(screen.getByRole('button', { name: 'Add unit' }))
    await user.type(screen.getByLabelText('Chassis/frame number (line 1, unit 2)'), ' ch-1')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(
      await screen.findByText('Same chassis/frame number as line 1, unit 1'),
    ).toBeInTheDocument()
    expect(puts).toBe(0)
    await user.clear(screen.getByLabelText('Chassis/frame number (line 1, unit 2)'))
    await user.type(screen.getByLabelText('Chassis/frame number (line 1, unit 2)'), 'CH-2')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(
      await screen.findByText('is already on the draft goods receipt GRN-000007', {
        selector: 'span',
      }),
    ).toBeInTheDocument()
    expect(screen.getByText(/already on another goods receipt/)).toBeInTheDocument()
    expect(screen.getByLabelText('Chassis/frame number (line 1, unit 1)')).toHaveValue('CH-1')
  })

  it('offers a reload after a version conflict and explains network failures', async () => {
    const user = userEvent.setup()
    let mode: 'conflict' | 'network' = 'conflict'
    api((u, init) => {
      if (u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT') {
        if (mode === 'network') throw new TypeError('Failed to fetch')
        return err(409, 'VERSION_CONFLICT')
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.type(await screen.findByLabelText('Received (line 2)'), '1')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(await screen.findByRole('button', { name: 'Reload latest version' })).toBeInTheDocument()
    mode = 'network'
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(
      await screen.findByText(/Cannot reach the server. Nothing was saved/),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Received (line 2)')).toHaveValue('1')
  })

  it('refuses a forbidden save with the server message', async () => {
    const user = userEvent.setup()
    api((u, init) =>
      u.pathname === '/api/v1/goods-receipts/g1' && init?.method === 'PUT'
        ? err(403, 'FORBIDDEN')
        : undefined,
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.type(await screen.findByLabelText('Received (line 2)'), '1')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(await screen.findByText(/You do not have permission to do this/)).toBeInTheDocument()
  })
})

describe('goods receipt read-only and documents', () => {
  const saved: GRNDetail = {
    ...draft,
    lineCount: 1,
    receivedQuantity: 1,
    acceptedQuantity: 1,
    unitCount: 1,
    lines: [
      line({
        id: 'gl1',
        receivedQuantity: 1,
        acceptedQuantity: 1,
        warnings: ['PARTIAL_DELIVERY'],
        units: [
          {
            id: 'u1',
            unitNo: 1,
            condition: 'ACCEPTED',
            chassisNumber: 'CH-0001',
            chassisNormalised: 'CH-0001',
            motorSerial: 'M-1',
            batterySerial: null,
            chargerSerial: null,
            note: null,
            warnings: [],
          },
        ],
      }),
      draft.lines[1] as GRNLine,
    ],
    documents: [
      {
        id: 'd1',
        attachmentId: 'a1',
        documentType: 'DELIVERY_CHALLAN',
        note: null,
        fileName: 'challan.pdf',
        contentType: 'application/pdf',
        addedBy: testMe.user.id,
        addedByName: 'Test User',
        createdAt: t,
      },
    ],
    actions: {
      edit: false,
      cancel: false,
      addDocument: false,
      authoriseOverReceipt: false,
      post: false,
    },
  }

  it('shows viewers the lines, units and documents without editing; downloads need attachments.view', async () => {
    api(undefined, saved)
    renderApp('/procurement/goods-receipts/g1', { me: me([VIEW]) })
    expect(await screen.findByText(/Chassis CH-0001/)).toBeInTheDocument()
    expect(screen.getAllByText('Partial delivery').length).toBeGreaterThan(0)
    expect(screen.getAllByText('challan.pdf').length).toBeGreaterThan(0)
    expect(screen.queryByRole('button', { name: 'Save draft' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancel draft' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Download/ })).not.toBeInTheDocument()
    expect(screen.getByText(/Opening documents needs permission/)).toBeInTheDocument()
  })

  it('reports a refused download', async () => {
    const user = userEvent.setup()
    api(
      (u) => (u.pathname.startsWith('/api/v1/attachments/a1') ? err(404, 'NOT_FOUND') : undefined),
      saved,
    )
    renderApp('/procurement/goods-receipts/g1', { me: me([VIEW, 'attachments.view']) })
    await user.click(
      (await screen.findAllByRole('button', { name: 'Download challan.pdf' })).at(0) as HTMLElement,
    )
    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('cancels a draft only with a reason', async () => {
    const user = userEvent.setup()
    let sent: Record<string, unknown> = {}
    api((u, init) => {
      if (u.pathname === '/api/v1/goods-receipts/g1/cancel') {
        sent = bodyOf(init)
        return Response.json({
          data: { ...draft, status: 'CANCELLED', version: 3, actions: saved.actions },
        })
      }
      return undefined
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.click(await screen.findByRole('button', { name: 'Cancel draft' }))
    const dialog = await screen.findByRole('dialog', { name: 'Cancel GRN-000001?' })
    await user.click(within(dialog).getByRole('button', { name: 'Cancel draft' }))
    expect(await within(dialog).findByText('A reason is required')).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText(/Reason/), 'Wrong PO')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel draft' }))
    await waitFor(() => {
      expect(sent).toEqual({ version: 2, reason: 'Wrong PO' })
    })
  })
})
