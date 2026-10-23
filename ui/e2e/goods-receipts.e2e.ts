import {
  expect,
  test,
  type APIRequestContext,
  type Browser,
  type Page,
  type TestInfo,
} from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Goods receipt drafts (D-03) against the real API. Placeholder data and
 * identifiers; throwaway database only. PROPOSED grants: E2E_CATALOGUE also
 * needs goods_receipts.view and .manage (not .over_receipt); E2E_PRICE_APPROVER
 * needs goods_receipts.view, .manage and .over_receipt; E2E_CATALOGUE_VIEWER
 * needs goods_receipts.view only.
 *
 * Goods are received against an approved purchase order, which needs the
 * eligible E2E_PO_* fixture described in purchase-orders.e2e.ts: the test
 * creates and submits one through the API and E2E_PRICE_APPROVER approves it.
 */

const pdf = Buffer.from('%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n')

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function call<T>(
  page: Page,
  method: 'GET' | 'POST',
  path: string,
  data?: object,
): Promise<T> {
  const res =
    method === 'GET'
      ? await page.request.get(`/api/v1${path}`)
      : await page.request.post(`/api/v1${path}`, {
          headers: { 'X-CSRF-Token': await csrf(page.request) },
          data,
        })
  expect(res.ok(), `${method} ${path}: ${await res.text()}`).toBe(true)
  return ((await res.json()) as { data: T }).data
}

interface Item {
  id: string
}
type ListOf<T> = { items: T[] }

function future(days: number): string {
  const d = new Date(Date.now() + days * 86_400_000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function noSidewaysScroll(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )
  expect(overflow).toBeLessThanOrEqual(1)
}

function context(browser: Browser, testInfo: TestInfo) {
  return browser.newContext({
    baseURL: testInfo.project.use.baseURL,
    viewport: testInfo.project.use.viewport,
    isMobile: testInfo.project.use.isMobile,
    hasTouch: testInfo.project.use.hasTouch,
  })
}

/** An APPROVED purchase order of `quantity` units of the fixture variant. */
async function approvedPO(page: Page, approver: Page, quantity: number) {
  const supplier = process.env.E2E_PO_SUPPLIER ?? ''
  const brand = process.env.E2E_PO_BRAND ?? ''
  const s = await call<ListOf<Item>>(page, 'GET', `/suppliers?q=${encodeURIComponent(supplier)}`)
  const b = await call<ListOf<Item>>(page, 'GET', `/brands?q=${encodeURIComponent(brand)}`)
  const brandId = b.items[0]?.id ?? ''
  const m = await call<ListOf<Item>>(page, 'GET', `/models?brandId=${brandId}`)
  const modelId = m.items[0]?.id ?? ''
  const v = await call<ListOf<Item>>(page, 'GET', `/variants?modelId=${modelId}`)
  const po = await call<{ id: string; number: string; version: number }>(
    page,
    'POST',
    '/purchase-orders',
    {
      supplierId: s.items[0]?.id,
      expectedDate: future(7),
      lines: [{ brandId, modelId, variantId: v.items[0]?.id, quantity, unitCost: '950.00' }],
    },
  )
  const submitted = await call<{ version: number; status: string }>(
    page,
    'POST',
    `/purchase-orders/${po.id}/submit`,
    { version: po.version },
  )
  const approved = await call<{ status: string }>(
    approver,
    'POST',
    `/purchase-orders/${po.id}/approve`,
    {
      version: submitted.version,
    },
  )
  expect(approved.status).toBe('APPROVED')
  return po
}

async function startDraft(page: Page, poNumber: string, reference: string) {
  await page.goto('/procurement/goods-receipts')
  await expect(page.getByRole('heading', { level: 1, name: 'Goods Receipts' })).toBeVisible()
  await page.getByRole('button', { name: 'New goods receipt' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'New goods receipt' })
  await dialog.getByLabel('Search PO number or supplier').fill(poNumber)
  const radio = dialog.getByRole('radio', { name: new RegExp(`^${poNumber} `) })
  await radio.check()
  await expect(radio).toBeChecked()
  await dialog.getByLabel('Supplier delivery reference').fill(reference)
  await dialog.getByRole('button', { name: 'Start draft' }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('heading', { level: 1, name: /^GRN-\d{6}$/ })).toBeVisible()
  return page.url()
}

const chassisField = (page: Page, unit: number) =>
  page.getByLabel(`Chassis/frame number (line 1, unit ${String(unit)})`)

test.describe('goods receipts (D-03)', () => {
  test('without goods_receipts.view there is no link and the page and API are denied', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page, '/', 'active')
    if (!isMobile) {
      await expect(
        page
          .getByRole('navigation', { name: 'Main' })
          .getByRole('link', { name: /Goods Receipts/ }),
      ).toHaveCount(0)
    }
    await page.goto('/procurement/goods-receipts')
    await expect(page.getByText('Access denied')).toBeVisible()
    expect((await page.request.get('/api/v1/goods-receipts')).status()).toBe(403)
    expect((await page.request.get('/api/v1/receivable-purchase-orders')).status()).toBe(403)
    const res = await page.request.post('/api/v1/goods-receipts', {
      headers: { 'X-CSRF-Token': await csrf(page.request) },
      data: {},
    })
    expect(res.status()).toBe(403)
    expect(problems).toEqual([])
  })

  test('partial draft with units and documents; over-receipt needs an authorised override; duplicates and stale saves are refused; drafts change no stock', async ({
    page,
    browser,
    isMobile,
  }, testInfo) => {
    test.skip(
      !process.env.E2E_PO_SUPPLIER ||
        !process.env.E2E_PO_BRAND ||
        !hasAccount('catalogue') ||
        !hasAccount('priceApprover') ||
        !hasAccount('catalogueViewer'),
      'needs the eligible E2E_PO_* fixture and three accounts (throwaway database only)',
    )
    test.setTimeout(180_000)
    const problems = watchConsole(page)
    const run = `${isMobile ? 'M' : 'D'}${Date.now().toString(36).toUpperCase()}`
    await signIn(page, '/', 'catalogue')
    const approverCtx = await context(browser, testInfo)
    const approver = await approverCtx.newPage()
    const approverProblems = watchConsole(approver)
    await signIn(approver, '/', 'priceApprover')
    const po = await approvedPO(page, approver, 3)

    // The PO page offers "Receive goods"; the receipt list starts empty for it.
    await page.goto(`/procurement/purchase-orders/${po.id}`)
    const receipts = page.locator('section', {
      has: page.getByRole('heading', { name: 'Goods receipts' }),
    })
    await expect(receipts.getByText('No goods receipt yet.')).toBeVisible()
    if (!isMobile) {
      await page
        .getByRole('navigation', { name: 'Main' })
        .getByRole('link', { name: /Goods Receipts/ })
        .click()
      await expect(page.getByRole('heading', { level: 1, name: 'Goods Receipts' })).toBeVisible()
    }
    const grnUrl = await startDraft(page, po.number, `DC-${run}`)
    const grnId = grnUrl.split('/').pop() ?? ''
    await expect(page.getByText(/does not change stock/).first()).toBeVisible()

    // Step 1: a partial delivery — 2 of 3 arrived, 1 rejected.
    await page.getByLabel('Received (line 1)').fill('2')
    await page.getByLabel('Rejected (line 1)').fill('1')
    await page.getByLabel('Discrepancy note (line 1)').fill('One unit with a cracked panel')
    // Step 2: unit identifiers, scanner style (each new unit's chassis field gets focus).
    await page.getByRole('button', { name: /Unit identifiers/ }).click()
    await page.getByRole('button', { name: 'Add unit' }).click()
    await expect(chassisField(page, 1)).toBeFocused()
    await page.keyboard.type(`ch-${run}-1`)
    await page.getByLabel('Motor serial (line 1, unit 1)').fill(`MOT-${run}-1`)
    await page.getByRole('button', { name: 'Add unit' }).click()
    await expect(chassisField(page, 2)).toBeFocused()
    // Duplicate on the same receipt: shown at once, nothing is sent.
    await page.keyboard.type(` CH-${run}-1 `)
    await page.getByRole('button', { name: 'Save draft' }).click()
    await expect(page.getByText('Same chassis/frame number as line 1, unit 1')).toBeVisible()
    await chassisField(page, 2).fill(`CH-${run}-2`)
    await page.getByLabel('Condition (line 1, unit 2)').selectOption('REJECTED')
    // Review: Post receipt is separate and not available.
    await page.getByRole('button', { name: /Review and save/ }).click()
    await expect(page.getByRole('button', { name: 'Post receipt' })).toBeDisabled()
    await page.getByRole('button', { name: 'Save draft' }).click()
    await expect(page.getByText(/Draft saved .* Stock is unchanged/)).toBeVisible()

    // Persistence: a reload shows what the server stored, with its warnings.
    await page.reload()
    await expect(page.getByLabel('Received (line 1)')).toHaveValue('2')
    await expect(page.getByText('Partial delivery').first()).toBeVisible()
    await page.getByRole('button', { name: /Unit identifiers/ }).click()
    await expect(chassisField(page, 1)).toHaveValue(`ch-${run}-1`)
    await expect(page.getByLabel('Condition (line 1, unit 2)')).toHaveValue('REJECTED')

    // No stock change: the PO's received quantity and status are untouched.
    const poAfter = await call<{ status: string; receivedQuantity: number }>(
      page,
      'GET',
      `/purchase-orders/${po.id}`,
    )
    expect(poAfter).toMatchObject({ status: 'APPROVED', receivedQuantity: 0 })

    // Step 3: a supplier document, uploaded privately and attached.
    await page.getByRole('button', { name: /Supplier documents/ }).click()
    await page.getByRole('button', { name: 'Attach document' }).click()
    const doc = page.getByRole('dialog', { name: 'Attach document' })
    await doc.getByLabel('Choose files to upload').setInputFiles({
      name: `challan-${run}.pdf`,
      mimeType: 'application/pdf',
      buffer: pdf,
    })
    await expect(doc.getByText('Uploaded and stored privately.')).toBeVisible()
    await doc.getByRole('button', { name: 'Attach document' }).click()
    await expect(doc).toHaveCount(0)
    await expect(
      page.getByText(`challan-${run}.pdf`).filter({ visible: true }).first(),
    ).toBeVisible()
    const download = page.waitForEvent('download')
    await page
      .getByRole('button', { name: `Download challan-${run}.pdf` })
      .filter({ visible: true })
      .first()
      .click()
    expect((await download).suggestedFilename()).toBe(`challan-${run}.pdf`)

    // Over-receipt without the permission: warned, then refused by the server; entries kept.
    await page.getByRole('button', { name: /Delivered lines/ }).click()
    await page.getByLabel('Received (line 1)').fill('4')
    await expect(page.getByText(/Saving will be refused until someone allowed/)).toBeVisible()
    await page.getByRole('button', { name: 'Save draft' }).click()
    await expect(page.getByText(/More is received than the purchase order has left/)).toBeVisible()
    await expect(page.getByLabel('Received (line 1)')).toHaveValue('4')

    // Someone allowed to authorise over-receipt does so with a reason.
    await approver.goto(grnUrl)
    await approver.getByLabel('Received (line 1)').fill('4')
    await approver.getByLabel('Rejected (line 1)').fill('1')
    await approver
      .getByLabel(/Over-receipt reason/)
      .fill('Supplier sent one extra unit; agreed by phone')
    await approver.getByRole('button', { name: 'Save draft' }).click()
    await expect(approver.getByText(/Draft saved/)).toBeVisible()
    await expect(approver.getByText(/Authorised by/).first()).toBeVisible()

    // The first screen is now stale: its save is refused and it offers the latest version.
    await page.getByLabel('Received (line 1)').fill('2')
    await page.getByRole('button', { name: 'Save draft' }).click()
    await expect(page.getByRole('button', { name: 'Reload latest version' }).first()).toBeVisible()
    await page.getByRole('button', { name: 'Reload latest version' }).first().click()
    await expect(page.getByLabel('Received (line 1)')).toHaveValue('4')
    await expect(page.getByText(/Authorised by .*agreed by phone/).first()).toBeVisible()

    // A second draft cannot take a chassis number already on the first.
    await startDraft(approver, po.number, `DC-${run}-B`)
    await approver.getByLabel('Received (line 1)').fill('1')
    await approver.getByLabel(/Over-receipt reason/).fill('Second delivery beyond the order')
    await approver.getByRole('button', { name: /Unit identifiers/ }).click()
    await approver.getByRole('button', { name: 'Add unit' }).click()
    await chassisField(approver, 1).fill(`CH-${run}-1`)
    await approver.getByRole('button', { name: 'Save draft' }).click()
    await expect(
      approver.getByText(/is already on the draft goods receipt GRN-\d{6}/).first(),
    ).toBeVisible()
    await expect(chassisField(approver, 1)).toHaveValue(`CH-${run}-1`)
    await approver.getByRole('button', { name: 'Cancel draft' }).click()
    const cancel = approver.getByRole('dialog', { name: /^Cancel GRN-/ })
    await cancel.getByRole('button', { name: 'Cancel draft' }).click()
    await expect(cancel.getByText('A reason is required')).toBeVisible()
    await cancel.getByLabel(/Reason/).fill('Entered by mistake')
    await cancel.getByRole('button', { name: 'Cancel draft' }).click()
    await expect(cancel).toHaveCount(0)
    await expect(approver.getByText(/Cancelled .*Entered by mistake/)).toBeVisible()
    expect(approverProblems).toEqual([])
    await approverCtx.close()

    // A read-only viewer sees the draft, its units and documents, but cannot change or open them.
    const viewerCtx = await context(browser, testInfo)
    const viewer = await viewerCtx.newPage()
    const viewerProblems = watchConsole(viewer)
    await signIn(viewer, '/', 'catalogueViewer')
    await viewer.goto(grnUrl)
    await expect(viewer.getByText(`Chassis ch-${run}-1`, { exact: false })).toBeVisible()
    await expect(viewer.getByRole('button', { name: 'Save draft' })).toHaveCount(0)
    await expect(viewer.getByText(/Opening documents needs permission/)).toBeVisible()
    const denied = await viewer.request.put(`/api/v1/goods-receipts/${grnId}`, {
      headers: { 'X-CSRF-Token': await csrf(viewer.request) },
      data: { version: 1, receivedDate: future(0), lines: [] },
    })
    expect(denied.status()).toBe(403)
    expect(viewerProblems).toEqual([])
    await viewerCtx.close()

    // Still no stock change after everything above.
    const poEnd = await call<{ status: string; receivedQuantity: number }>(
      page,
      'GET',
      `/purchase-orders/${po.id}`,
    )
    expect(poEnd).toMatchObject({ status: 'APPROVED', receivedQuantity: 0 })
    await noSidewaysScroll(page)
    if (process.env.E2E_EVIDENCE_DIR) {
      await page.getByRole('button', { name: /Delivered lines/ }).click()
      await page.screenshot({
        path: `${process.env.E2E_EVIDENCE_DIR}/${isMobile ? 'mobile' : 'desktop'}-grn-draft.png`,
        fullPage: true,
      })
    }
    expect(problems).toEqual([])
  })
})
