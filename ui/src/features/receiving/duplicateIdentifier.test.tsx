import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Me } from '@/features/access/api/access'
import { renderApp, testMe } from '@/test/render'
import type { GRNDetail, GRNLine } from './api/goodsReceipts'

// D-05 frontend: how the receipt editor shows a 409 DUPLICATE_IDENTIFIER
// (API_CONVENTIONS "Duplicate vehicle identifiers"). No route returns it yet —
// receipt posting (D-07) will — so these tests answer the draft save with the
// verified contract. They test the screen only; the PostgreSQL race itself is
// covered by the Go tests (TestConcurrentRegistrationsOfOneChassis…).
// Placeholder data only ("Brand One", "CH-0001"): nothing real.

const t = '2026-10-23T09:00:00Z'
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
  otherDraftsQuantity: 0,
  availableQuantity: 5,
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
const draft: GRNDetail = {
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
  purchaseOrderReceivable: true,
  lines: [line(), line({ purchaseOrderLineId: 'pl2', lineNo: 2, variantName: 'Pro' })],
  documents: [],
  events: [],
  stockPosted: false,
  actions: {
    edit: true,
    cancel: true,
    addDocument: true,
    authoriseOverReceipt: false,
    post: false,
  },
}

const me = (permissions: string[]): Me => ({ ...testMe, permissions })
const RECEIVER = ['goods_receipts.view', 'goods_receipts.manage', 'vehicle_units.view']
const conflict = (field: string) =>
  Response.json(
    {
      error: {
        code: 'DUPLICATE_IDENTIFIER',
        message:
          'This identifier is already registered to another vehicle. Correct it and try again. Nothing was saved.',
        requestId: 'req-dup',
        details: [{ field, message: 'is already registered' }],
      },
    },
    { status: 409 },
  )

type Put = (body: Record<string, unknown>) => Response | Promise<Response>

/** The receipt API; every PUT goes to `put`. Records every call made. */
function api(put: Put) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = new URL(input as string, 'http://localhost')
    const p = url.pathname
    const json = (data: unknown) => Promise.resolve(Response.json({ data }))
    if (p === '/api/v1/goods-receipts/g1' && init?.method === 'PUT') {
      const body =
        typeof init.body === 'string' ? (JSON.parse(init.body) as Record<string, unknown>) : {}
      return Promise.resolve(put(body))
    }
    if (p === '/api/v1/goods-receipts/g1') return json(draft)
    if (p === '/api/v1/attachments/limits') return json({ maxBytes: 1048576, allowedTypes: [] })
    if (p === '/api/v1/attachments') return json([])
    if (p.endsWith('/health'))
      return json({ status: 'ok', service: 'api', version: 'v', time: 'x' })
    if (p.endsWith('/ready'))
      return json({ status: 'ready', checks: { database: 'up', migrations: 'up' } })
    return Promise.resolve(
      Response.json({ error: { code: 'NOT_FOUND', message: 'No.' } }, { status: 404 }),
    )
  })
}

const puts = (spy: ReturnType<typeof api>) =>
  spy.mock.calls.filter(([, init]) => init?.method === 'PUT').length
const chassis = (l: number, u: number) =>
  screen.getByLabelText(`Chassis/frame number (line ${String(l)}, unit ${String(u)})`)

/** Two units on line 1 and one on line 2, with other entries to keep. */
async function fill(user: ReturnType<typeof userEvent.setup>) {
  await user.type(await screen.findByLabelText('Received (line 1)'), '2')
  await user.type(screen.getByLabelText('Discrepancy note (line 1)'), 'One scratched')
  await user.type(screen.getByLabelText('Received (line 2)'), '1')
  await user.click(screen.getByRole('button', { name: /Unit identifiers/ }))
  const [add1, add2] = screen.getAllByRole('button', { name: 'Add unit' })
  if (!add1 || !add2) throw new Error('no Add unit buttons')
  await user.click(add1)
  await user.type(chassis(1, 1), 'CH-0001')
  await user.click(add1)
  await user.type(chassis(1, 2), ' ch-0002 ')
  await user.type(screen.getByLabelText('Motor serial (line 1, unit 2)'), 'M-77')
  await user.click(add2)
  await user.type(chassis(2, 1), 'CH-0003')
}

function expectKept() {
  expect(chassis(1, 1)).toHaveValue('CH-0001')
  expect(chassis(1, 2)).toHaveValue(' ch-0002 ')
  expect(screen.getByLabelText('Motor serial (line 1, unit 2)')).toHaveValue('M-77')
  expect(chassis(2, 1)).toHaveValue('CH-0003')
}

function expectOnlyUnit12Marked() {
  const field = chassis(1, 2)
  expect(field).toHaveAttribute('aria-invalid', 'true')
  expect(field).toHaveAccessibleDescription(/Already registered to another vehicle/)
  for (const other of [
    chassis(1, 1),
    chassis(2, 1),
    screen.getByLabelText('Motor serial (line 1, unit 2)'),
  ]) {
    expect(other).not.toHaveAttribute('aria-invalid')
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('duplicate vehicle identifiers in the receipt editor (D-05)', () => {
  it('compares chassis numbers trimmed and upper-cased only: separators keep numbers apart', async () => {
    const user = userEvent.setup()
    const sent: Record<string, unknown>[] = []
    const spy = api((b) => {
      sent.push(b)
      return Response.json({ data: { ...draft, version: 3 } })
    })
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await user.type(await screen.findByLabelText('Received (line 1)'), '3')
    await user.click(screen.getByRole('button', { name: /Unit identifiers/ }))
    const add = screen.getByRole('button', { name: 'Add unit' })
    for (const [i, v] of ['CH 0001', 'CH/0001', 'CH0001'].entries()) {
      await user.click(add)
      await user.type(chassis(1, i + 1), v)
    }
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    await waitFor(() => {
      expect(puts(spy)).toBe(1)
    })
    const units = (sent[0]?.lines as { units: { chassisNumber: string }[] }[])[0]?.units
    // Sent as captured (trimmed); the server keeps the printed form and compares the normalised one.
    expect(units?.map((u) => u.chassisNumber)).toEqual(['CH 0001', 'CH/0001', 'CH0001'])
    expect(screen.queryByText(/Same chassis\/frame number/)).not.toBeInTheDocument()
  })

  it('catches a repeat across lines in another case and with spaces before sending, and clears it once corrected', async () => {
    const user = userEvent.setup()
    const spy = api(() => Response.json({ data: { ...draft, version: 3 } }))
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await fill(user)
    await user.clear(chassis(2, 1))
    await user.type(chassis(2, 1), '  ch-0001')
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(
      await screen.findByText('Same chassis/frame number as line 1, unit 1'),
    ).toBeInTheDocument()
    expect(chassis(2, 1)).toHaveAttribute('aria-invalid', 'true')
    expect(puts(spy)).toBe(0)
    await user.type(chassis(2, 1), '9')
    await waitFor(() => {
      expect(screen.queryByText(/Same chassis\/frame number/)).not.toBeInTheDocument()
    })
  })

  it('immediate 409: marks the conflicting unit, keeps every entry, never retries and saves after correction', async () => {
    const user = userEvent.setup()
    let refuse = true
    const spy = api(() =>
      refuse
        ? conflict('lines[0].units[1].chassisNumber')
        : Response.json({ data: { ...draft, version: 3 } }),
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await fill(user)
    await user.click(screen.getByRole('button', { name: 'Save draft' }))

    const alert = (await screen.findByText(/Correct the highlighted number/)).closest(
      '[role="alert"]',
    ) as HTMLElement
    expect(alert).toHaveTextContent('Line 1, unit 2: chassis/frame number')
    expect(alert).toHaveTextContent('Reference: req-dup')
    // A duplicate is not a stale version: no reload is offered, nothing is claimed as saved.
    expect(within(alert).queryByRole('button', { name: /Reload/ })).not.toBeInTheDocument()
    expect(within(alert).queryByRole('link')).not.toBeInTheDocument()
    expect(screen.queryByText(/Draft saved/)).not.toBeInTheDocument()
    expectOnlyUnit12Marked()
    expectKept()
    expect(screen.getByRole('button', { name: /Unit identifiers/ })).toHaveAttribute(
      'aria-current',
      'step',
    )
    await new Promise((r) => setTimeout(r, 200))
    expect(puts(spy)).toBe(1)

    // Correcting the number clears its mark; the user saves again explicitly.
    await user.clear(chassis(1, 2))
    await user.type(chassis(1, 2), 'CH-0004')
    await waitFor(() => {
      expect(chassis(1, 2)).not.toHaveAttribute('aria-invalid')
    })
    refuse = false
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(await screen.findByText(/Draft saved .* Stock is unchanged/)).toBeInTheDocument()
    expect(puts(spy)).toBe(2)
  })

  it('delayed 409 after a locally valid form (another registration won the race): same field, entries kept, no retry', async () => {
    const user = userEvent.setup()
    let release: (() => void) | undefined
    const spy = api(
      () =>
        new Promise<Response>((resolve) => {
          release = () => {
            resolve(conflict('lines[0].units[1].chassisNumber'))
          }
        }),
    )
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await fill(user)
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    // The local check found nothing; the screen waits for the server's answer.
    await waitFor(() => {
      expect(puts(spy)).toBe(1)
    })
    expect(screen.queryByText(/Same chassis\/frame number/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Save draft/ })).toBeDisabled()
    // A second click while waiting sends nothing more.
    await user.click(screen.getByRole('button', { name: /Save draft/ }))
    expect(puts(spy)).toBe(1)
    await new Promise((r) => setTimeout(r, 300))
    release?.()

    expect(await screen.findByText(/Correct the highlighted number/)).toBeInTheDocument()
    expectOnlyUnit12Marked()
    expectKept()
    expect(screen.queryByText(/Draft saved/)).not.toBeInTheDocument()
    await new Promise((r) => setTimeout(r, 200))
    expect(puts(spy)).toBe(1)
  })

  it('an unknown or out-of-range path marks no field and shows the general message', async () => {
    const user = userEvent.setup()
    api(() => conflict('lines[0].units[7].chassisNumber'))
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await fill(user)
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    expect(
      await screen.findByText(/Check the chassis\/frame numbers and try again/),
    ).toBeInTheDocument()
    for (const f of [chassis(1, 1), chassis(1, 2), chassis(2, 1)]) {
      expect(f).not.toHaveAttribute('aria-invalid')
    }
    expectKept()
  })

  it('does not look up, link to or reveal the existing vehicle, and calls no vehicle API', async () => {
    const user = userEvent.setup()
    const spy = api(() => conflict('lines[0].units[1].chassisNumber'))
    renderApp('/procurement/goods-receipts/g1', { me: me(RECEIVER) })
    await fill(user)
    await user.click(screen.getByRole('button', { name: 'Save draft' }))
    await screen.findByText(/Correct the highlighted number/)
    const paths = spy.mock.calls.map(([u]) => new URL(u as string, 'http://localhost').pathname)
    expect(paths.some((p) => p.includes('vehicle-units'))).toBe(false)
    expect(paths.some((p) => /duplicate|check|exists/i.test(p))).toBe(false)
    // Only the sidebar's module link exists; nothing points at a particular unit.
    const unitLinks = screen
      .queryAllByRole('link')
      .filter((l) => /\/inventory\/vehicles\/./.test(l.getAttribute('href') ?? ''))
    expect(unitLinks).toHaveLength(0)
  })
})
