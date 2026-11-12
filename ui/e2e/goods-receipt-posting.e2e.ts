import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Goods receipt posting (D-07) against the real API and PostgreSQL. Placeholder
 * data; throwaway database only. Needs the eligible E2E_PO_* fixture
 * (purchase-orders.e2e.ts). PROPOSED grants: E2E_CATALOGUE needs the D-02/D-03
 * grants plus goods_receipts.post and vehicle_units.view and one location in
 * scope; E2E_PRICE_APPROVER (approves the PO) has goods_receipts.view/manage but
 * NOT goods_receipts.post.
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function call<T>(
  page: Page,
  method: 'GET' | 'POST' | 'PUT',
  path: string,
  data?: object,
  key?: string,
): Promise<T> {
  const headers: Record<string, string> = {}
  if (method !== 'GET') headers['X-CSRF-Token'] = await csrf(page.request)
  if (key) headers['Idempotency-Key'] = key
  const res = await page.request.fetch(`/api/v1${path}`, { method, headers, data })
  expect(res.ok(), `${method} ${path}: ${await res.text()}`).toBe(true)
  return ((await res.json()) as { data: T }).data
}

type ListOf<T> = { items: T[] }
interface Detail {
  id: string
  number: string
  version: number
  status: string
  lines: { purchaseOrderLineId: string }[]
}

function future(days: number): string {
  const d = new Date(Date.now() + days * 86_400_000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function approvedPO(page: Page, approver: Page, quantity: number) {
  const s = await call<ListOf<{ id: string }>>(
    page,
    'GET',
    `/suppliers?q=${encodeURIComponent(process.env.E2E_PO_SUPPLIER ?? '')}`,
  )
  const b = await call<ListOf<{ id: string }>>(
    page,
    'GET',
    `/brands?q=${encodeURIComponent(process.env.E2E_PO_BRAND ?? '')}`,
  )
  const brandId = b.items[0]?.id ?? ''
  const modelId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/models?brandId=${brandId}`)).items[0]?.id ??
    ''
  const variantId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/variants?modelId=${modelId}`)).items[0]
      ?.id ?? ''
  const po = await call<{ id: string; number: string; version: number }>(
    page,
    'POST',
    '/purchase-orders',
    {
      supplierId: s.items[0]?.id,
      expectedDate: future(7),
      lines: [{ brandId, modelId, variantId, quantity, unitCost: '950.00' }],
    },
  )
  const submitted = await call<{ version: number }>(
    page,
    'POST',
    `/purchase-orders/${po.id}/submit`,
    { version: po.version },
  )
  await call(approver, 'POST', `/purchase-orders/${po.id}/approve`, { version: submitted.version })
  return po
}

const unit = (condition: string, chassis: string) => ({ condition, chassisNumber: chassis })

function save(page: Page, d: Detail, units: { condition: string; chassisNumber: string }[]) {
  return call<Detail>(page, 'PUT', `/goods-receipts/${d.id}`, {
    version: d.version,
    receivedDate: future(0),
    supplierReference: 'DC-POST',
    lines: [
      {
        purchaseOrderLineId: d.lines[0]?.purchaseOrderLineId,
        receivedQuantity: 3,
        acceptedQuantity: 2,
        rejectedQuantity: 1,
        discrepancyNote: 'One scratched',
        units,
      },
    ],
  })
}

test.describe('goods receipt posting (D-07)', () => {
  test('blocked until complete, stale version, single posting, units awaiting PDI, PO history, others denied', async ({
    page,
    browser,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') || !hasAccount('priceApprover') || !process.env.E2E_PO_SUPPLIER,
      'needs E2E_CATALOGUE_*, E2E_PRICE_APPROVER_* and the E2E_PO_* fixture (throwaway database only)',
    )
    test.setTimeout(180_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/', 'catalogue')
    const actx = await browser.newContext({ ...info.project.use })
    const approver = await actx.newPage()
    await signIn(approver, '/', 'priceApprover')
    const me = (
      await call<{ locationScope: { locations: { id: string; code: string; name: string }[] } }>(
        page,
        'GET',
        '/me',
      )
    ).locationScope.locations[0]
    expect(me, 'E2E_CATALOGUE needs a location in scope').toBeTruthy()
    const loc = me

    // An approved PO of 3 and a draft receipt: 3 received (2 accepted, 1 rejected), only 2 units captured.
    const po = await approvedPO(page, approver, 3)
    let d = await call<Detail>(
      page,
      'POST',
      '/goods-receipts',
      { purchaseOrderId: po.id, receivedDate: future(0), supplierReference: 'DC-POST' },
      `grn-${tag}`,
    )
    d = await save(page, d, [unit('ACCEPTED', `CH-${tag}-1`), unit('ACCEPTED', `CH-${tag}-2`)])
    await page.goto(`/procurement/goods-receipts/${d.id}`)
    await expect(page.getByRole('heading', { level: 1, name: d.number })).toBeVisible()

    // Scenario 3: incomplete capture is refused; nothing is registered.
    await page.getByRole('button', { name: 'Post receipt' }).click()
    let dialog = page.getByRole('dialog', { name: `Post ${d.number}?` })
    await expect(dialog.getByText(/awaiting PDI — not saleable/)).toBeVisible()
    await dialog.getByLabel(/Received at/).selectOption({ label: `${loc.code} — ${loc.name}` })
    await dialog.getByRole('button', { name: 'Post receipt' }).click()
    await expect(dialog.getByText(/cannot be posted yet\. Nothing was changed/)).toBeVisible()
    await expect(dialog.getByText('Line 1: units')).toBeVisible()
    expect(
      (await call<ListOf<unknown>>(page, 'GET', `/vehicle-units?goodsReceiptId=${d.id}`)).items,
    ).toHaveLength(0)
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Complete the capture (another tab saves it); scenario 6: the open page is now stale.
    d = await call<Detail>(page, 'GET', `/goods-receipts/${d.id}`)
    await page.reload()
    await page.getByRole('button', { name: 'Post receipt' }).click()
    dialog = page.getByRole('dialog', { name: `Post ${d.number}?` })
    await dialog.getByLabel(/Received at/).selectOption({ label: `${loc.code} — ${loc.name}` })
    d = await save(page, d, [
      unit('ACCEPTED', `CH-${tag}-1`),
      unit('ACCEPTED', `CH-${tag}-2`),
      unit('REJECTED', `CH-${tag}-R`),
    ])
    await dialog.getByRole('button', { name: 'Post receipt' }).click()
    await expect(dialog.getByText(/changed by someone else/)).toBeVisible()
    await expect(dialog.getByText(/changed since you opened this dialog/)).toBeVisible()

    // Scenarios 1 + 2: confirm with a double click: one request, one posting.
    const postRequests: string[] = []
    page.on('request', (r) => {
      if (r.method() === 'POST' && r.url().endsWith(`/goods-receipts/${d.id}/post`))
        postRequests.push(r.headers()['idempotency-key'] ?? '')
    })
    await dialog.getByRole('button', { name: 'Post receipt' }).dblclick()
    await expect(
      page.getByText(new RegExp(`${d.number} posted\\.`)).filter({ visible: true }),
    ).toBeVisible()
    expect(postRequests).toHaveLength(1)
    const units = page.getByRole('list', { name: 'Vehicle units registered by this receipt' })
    await expect(units.getByText('Awaiting PDI')).toHaveCount(2)
    await expect(units.getByText('Available')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Post receipt' })).toHaveCount(0)
    const vehicles = await call<
      ListOf<{ status: string; locationId: string; chassisNumber: string }>
    >(page, 'GET', `/vehicle-units?goodsReceiptId=${d.id}`)
    expect(vehicles.items.map((v) => v.status).sort()).toEqual([
      'RECEIVED_PENDING_PDI',
      'RECEIVED_PENDING_PDI',
    ])
    expect(vehicles.items.every((v) => v.locationId === loc.id)).toBe(true)
    expect(vehicles.items.some((v) => v.chassisNumber === `CH-${tag}-R`)).toBe(false)
    // Reload: the committed state comes back from the server.
    await page.reload()
    await expect(
      page.getByText(`${loc.code} — ${loc.name}`).filter({ visible: true }).first(),
    ).toBeVisible()
    await expect(
      page
        .getByRole('list', { name: 'Vehicle units registered by this receipt' })
        .getByText('Awaiting PDI'),
    ).toHaveCount(2)
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)

    // Scenario 7: the PO page shows the posting in its history (3 of 3 received → RECEIVED).
    await page.goto(`/procurement/purchase-orders/${po.id}`)
    await expect(page.getByText('Goods received').filter({ visible: true }).first()).toBeVisible()
    await expect(page.getByText(d.number).filter({ visible: true }).first()).toBeVisible()
    const poDetail = await call<{ status: string; lines: { receivedQuantity: number }[] }>(
      page,
      'GET',
      `/purchase-orders/${po.id}`,
    )
    expect(poDetail.status).toBe('RECEIVED')
    expect(poDetail.lines[0]?.receivedQuantity).toBe(3)

    // Scenario 5: a user without goods_receipts.post sees no Post action and the API refuses.
    const po2 = await approvedPO(page, approver, 1)
    let d2 = await call<Detail>(
      approver,
      'POST',
      '/goods-receipts',
      { purchaseOrderId: po2.id, receivedDate: future(0) },
      `grn2-${tag}`,
    )
    d2 = await call<Detail>(approver, 'PUT', `/goods-receipts/${d2.id}`, {
      version: d2.version,
      receivedDate: future(0),
      lines: [
        {
          purchaseOrderLineId: d2.lines[0]?.purchaseOrderLineId,
          receivedQuantity: 1,
          acceptedQuantity: 1,
          rejectedQuantity: 0,
          units: [unit('ACCEPTED', `CH-${tag}-X`)],
        },
      ],
    })
    await approver.goto(`/procurement/goods-receipts/${d2.id}`)
    await expect(approver.getByRole('heading', { level: 1, name: d2.number })).toBeVisible()
    await expect(approver.getByRole('button', { name: 'Post receipt' })).toHaveCount(0)
    const denied = await approver.request.post(`/api/v1/goods-receipts/${d2.id}/post`, {
      headers: { 'X-CSRF-Token': await csrf(approver.request), 'Idempotency-Key': `deny-${tag}` },
      data: { version: d2.version, locationId: loc.id },
    })
    expect(denied.status()).toBe(403)
    expect(
      (await call<ListOf<unknown>>(page, 'GET', `/vehicle-units?goodsReceiptId=${d2.id}`)).items,
    ).toHaveLength(0)
    await actx.close()
    expect(problems).toEqual([])
  })
})
