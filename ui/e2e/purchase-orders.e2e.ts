import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Purchase orders (D-02) against the real API. Placeholder data; throwaway
 * database only. PROPOSED grants: E2E_CATALOGUE also needs purchase_orders.view,
 * .manage, .approve and .cancel; E2E_PRICE_APPROVER needs purchase_orders.view and
 * .approve; E2E_CATALOGUE_VIEWER needs purchase_orders.view.
 *
 * No brand can be activated through the API yet (warranty terms and identifier
 * rules are pending checks), so the approval flow needs an eligible fixture
 * prepared directly in the throwaway database and named in E2E_PO_SUPPLIER,
 * E2E_PO_BRAND (with one active model and variant) and E2E_PO_COST_LIST (an
 * active cost price list with an approved COST price for that variant).
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function post(page: Page, path: string, data: object): Promise<{ id: string }> {
  const res = await page.request.post(`/api/v1${path}`, {
    headers: { 'X-CSRF-Token': await csrf(page.request) },
    data,
  })
  expect(res.ok(), await res.text()).toBe(true)
  return ((await res.json()) as { data: { id: string } }).data
}

async function openPurchaseOrders(page: Page, isMobile: boolean) {
  if (isMobile) await page.goto('/procurement/purchase-orders')
  else
    await page
      .getByRole('navigation', { name: 'Main' })
      .getByRole('link', { name: /Purchase Orders/ })
      .click()
  await expect(page.getByRole('heading', { level: 1, name: 'Purchase Orders' })).toBeVisible()
}

async function newDraft(page: Page, supplier: string, expected: string) {
  await page.getByRole('button', { name: 'New purchase order' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'New purchase order' })
  await dialog.getByLabel(/Supplier/).selectOption({ label: supplier })
  await dialog.getByLabel(/Expected delivery date/).fill(expected)
  await dialog.getByRole('button', { name: 'Create draft' }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('heading', { level: 1, name: /^PO-\d{6}$/ })).toBeVisible()
}

async function addLine(page: Page, brand: string, qty: string, cost: string, costList?: string) {
  await page.getByRole('button', { name: 'Add line' }).click()
  const dialog = page.getByRole('dialog', { name: 'Add line' })
  await dialog.getByLabel(/Brand/).selectOption({ label: brand })
  await dialog.getByLabel(/Model/).selectOption({ index: 1 })
  await dialog.getByLabel(/Variant/).selectOption({ index: 1 })
  await dialog.getByLabel(/Quantity/).fill(qty)
  await dialog.getByLabel(/Agreed unit cost/).fill(cost)
  if (costList)
    await dialog.getByLabel(/Catalogue cost reference/).selectOption({ label: costList })
  await dialog.getByRole('button', { name: 'Add line' }).click()
  await expect(dialog).toHaveCount(0)
}

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

test.describe('purchase orders (D-02)', () => {
  test('without purchase_orders.view there is no link and the page is denied', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page, '/', 'active')
    if (!isMobile) {
      await expect(
        page
          .getByRole('navigation', { name: 'Main' })
          .getByRole('link', { name: /Purchase Orders/ }),
      ).toHaveCount(0)
    }
    await page.goto('/procurement/purchase-orders')
    await expect(page.getByText('Access denied')).toBeVisible()
    expect((await page.request.get('/api/v1/purchase-orders')).status()).toBe(403)
    expect((await page.request.get('/api/v1/purchase-order-settings')).status()).toBe(403)
    expect(problems).toEqual([])
  })

  test('draft with lines and server totals; submission refused for an unverified supplier and inactive brand', async ({
    page,
    isMobile,
  }, testInfo) => {
    test.skip(!hasAccount('catalogue'), 'needs E2E_CATALOGUE with purchase order grants')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const run = `${testInfo.project.name} ${Date.now().toString(36)}`
    await signIn(page, '/', 'catalogue')
    const supplier = `E2E PO Draft Supplier ${run}`
    const brand = `E2E PO Draft Brand ${run}`
    await post(page, '/suppliers', { legalName: supplier })
    const b = await post(page, '/brands', { name: brand })
    const m = await post(page, '/models', { brandId: b.id, name: 'E2E Model', internalCode: 'M1' })
    await post(page, '/variants', { modelId: m.id, name: 'E2E Variant', variantCode: 'STD' })

    await openPurchaseOrders(page, isMobile)
    await expect(
      page
        .getByText(/approval threshold|No approval threshold/)
        .filter({ visible: true })
        .first(),
    ).toBeVisible()

    // Client validation, then the server creates a numbered draft.
    await page.getByRole('button', { name: 'New purchase order' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'New purchase order' })
    await dialog.getByRole('button', { name: 'Create draft' }).click()
    await expect(dialog.getByText('Choose a supplier', { exact: true }).last()).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()
    await newDraft(page, `${supplier} (pending verification)`, future(30))
    const poUrl = page.url()
    const number = (await page.getByRole('heading', { level: 1 }).textContent()) ?? ''
    await expect(
      page.getByText('Draft', { exact: true }).filter({ visible: true }).first(),
    ).toBeVisible()

    // Line: validation, server-side totals, edit.
    await page.getByRole('button', { name: 'Add line' }).click()
    dialog = page.getByRole('dialog', { name: 'Add line' })
    await dialog
      .getByLabel(/Brand/)
      .selectOption({ label: `${brand} (not activated for purchase)` })
    await dialog.getByLabel(/Model/).selectOption({ index: 1 })
    await dialog.getByLabel(/Variant/).selectOption({ index: 1 })
    await dialog.getByLabel(/Quantity/).fill('0')
    await dialog.getByLabel(/Agreed unit cost/).fill('12.345')
    await dialog.getByRole('button', { name: 'Add line' }).click()
    await expect(dialog.getByText('A whole number from 1 to 10000')).toBeVisible()
    await expect(dialog.getByText(/up to 2 decimals/)).toBeVisible()
    await dialog.getByLabel(/Quantity/).fill('2')
    await dialog.getByLabel(/Agreed unit cost/).fill('1000.5')
    await dialog.getByRole('button', { name: 'Add line' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('₹2,001.00').filter({ visible: true }).first()).toBeVisible()

    // The same variant twice is refused by the server.
    await page.getByRole('button', { name: 'Add line' }).click()
    dialog = page.getByRole('dialog', { name: 'Add line' })
    await dialog
      .getByLabel(/Brand/)
      .selectOption({ label: `${brand} (not activated for purchase)` })
    await dialog.getByLabel(/Model/).selectOption({ index: 1 })
    await dialog.getByLabel(/Variant/).selectOption({ index: 1 })
    await dialog.getByLabel(/Agreed unit cost/).fill('1')
    await dialog.getByRole('button', { name: 'Add line' }).click()
    await expect(dialog.getByText(/already on another line/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    await page.getByRole('button', { name: 'Edit line 1' }).click()
    dialog = page.getByRole('dialog', { name: 'Edit line 1' })
    await dialog.getByLabel(/Quantity/).fill('3')
    await dialog.getByRole('button', { name: 'Save line' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('₹3,001.50').filter({ visible: true }).first()).toBeVisible()

    // Prerequisites are shown, and submission is refused by the server.
    await expect(page.getByText(/Cannot be submitted yet/)).toBeVisible()
    await page.getByRole('button', { name: 'Submit', exact: true }).click()
    dialog = page.getByRole('dialog', { name: `Submit ${number}?` })
    await dialog.getByRole('button', { name: 'Submit purchase order' }).click()
    const alert = dialog.getByRole('alert')
    await expect(alert).toContainText('cannot be submitted yet')
    await expect(alert).toContainText('Verified supplier')
    await expect(alert).toContainText('Brand activated for purchase')
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Persisted: reload shows the same draft and totals.
    await page.reload()
    await expect(page.getByRole('heading', { level: 1, name: number })).toBeVisible()
    await expect(page.getByText('₹3,001.50').filter({ visible: true }).first()).toBeVisible()
    await page.goto('/procurement/purchase-orders')
    await page.getByRole('searchbox').fill(number)
    await expect(page.getByRole('link', { name: number })).toBeVisible()

    // Cancel keeps it as history.
    await page.goto(poUrl)
    await page.getByRole('button', { name: 'Cancel PO' }).click()
    dialog = page.getByRole('dialog', { name: `Cancel ${number}?` })
    await dialog.getByRole('button', { name: 'Cancel purchase order' }).click()
    await expect(dialog.getByText('A reason is required')).toBeVisible()
    await dialog.getByLabel(/Reason/).fill('E2E cancel')
    await dialog.getByRole('button', { name: 'Cancel purchase order' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText(/Cancelled .*E2E cancel/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add line' })).toHaveCount(0)

    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })

  test('submit, no self-approval, approval by another user, cost snapshot and read-only viewer', async ({
    page,
    browser,
    isMobile,
  }, testInfo) => {
    const supplier = process.env.E2E_PO_SUPPLIER
    const brand = process.env.E2E_PO_BRAND
    const costList = process.env.E2E_PO_COST_LIST
    test.skip(
      !supplier ||
        !brand ||
        !costList ||
        !hasAccount('catalogue') ||
        !hasAccount('priceApprover') ||
        !hasAccount('catalogueViewer'),
      'needs the eligible E2E_PO_* fixture and three accounts (throwaway database only)',
    )
    test.setTimeout(150_000)
    const problems = watchConsole(page)
    await signIn(page, '/', 'catalogue')
    await openPurchaseOrders(page, isMobile)
    await newDraft(page, supplier ?? '', future(14))
    const poUrl = page.url()
    const poId = poUrl.split('/').pop() ?? ''
    const number = (await page.getByRole('heading', { level: 1 }).textContent()) ?? ''
    await addLine(page, brand ?? '', '2', '950', costList)
    await expect(page.getByText('₹1,900.00').filter({ visible: true }).first()).toBeVisible()
    await expect(
      page
        .getByText('Agreed cost differs from the catalogue cost')
        .filter({ visible: true })
        .first(),
    ).toBeVisible()
    await expect(page.getByText(/Cannot be submitted yet/)).toHaveCount(0)

    // Submit: no threshold is configured, so it waits for an approver.
    await page.getByRole('button', { name: 'Submit', exact: true }).click()
    const submit = page.getByRole('dialog', { name: `Submit ${number}?` })
    await expect(submit.getByText(/every purchase order needs an approver/)).toBeVisible()
    await submit.getByRole('button', { name: 'Submit purchase order' }).click()
    await expect(submit).toHaveCount(0)
    await expect(
      page.getByText('Awaiting approval').filter({ visible: true }).first(),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add line' })).toHaveCount(0)
    // The creator is never offered approval, and the server refuses it anyway.
    await expect(page.getByRole('button', { name: 'Approve', exact: true })).toHaveCount(0)
    await expect(page.getByText(/another authorised person must approve/)).toBeVisible()
    const detail = (await (await page.request.get(`/api/v1/purchase-orders/${poId}`)).json()) as {
      data: { version: number }
    }
    const self = await page.request.post(`/api/v1/purchase-orders/${poId}/approve`, {
      headers: { 'X-CSRF-Token': await csrf(page.request) },
      data: { version: detail.data.version },
    })
    expect(self.status()).toBe(403)
    expect(((await self.json()) as { error: { code: string } }).error.code).toBe(
      'SELF_APPROVAL_NOT_ALLOWED',
    )

    const ctx = async () =>
      browser.newContext({
        baseURL: testInfo.project.use.baseURL,
        viewport: testInfo.project.use.viewport,
        isMobile: testInfo.project.use.isMobile,
        hasTouch: testInfo.project.use.hasTouch,
      })
    // Another user approves through the screen.
    const approverCtx = await ctx()
    const approver = await approverCtx.newPage()
    const approverProblems = watchConsole(approver)
    await signIn(approver, '/', 'priceApprover')
    await approver.goto(poUrl)
    await approver.getByRole('button', { name: 'Approve', exact: true }).click()
    const approve = approver.getByRole('dialog', { name: `Approve ${number}?` })
    await approve.getByLabel(/Approval note/).fill('E2E approved')
    await approve.getByRole('button', { name: 'Approve purchase order' }).click()
    await expect(approve).toHaveCount(0)
    await expect(
      approver.getByText('Approved', { exact: true }).filter({ visible: true }).first(),
    ).toBeVisible()
    await expect(approver.getByText('E2E approved').filter({ visible: true }).first()).toBeVisible()
    // The approver may not see catalogue cost prices: the snapshot is hidden, the agreed cost is not.
    await expect(
      approver.getByText('Recorded (not visible to you)').filter({ visible: true }).first(),
    ).toBeVisible()
    await expect(approver.getByText('₹950.00').filter({ visible: true }).first()).toBeVisible()
    expect(approverProblems).toEqual([])
    await approverCtx.close()

    // A read-only viewer sees it with no actions.
    const viewerCtx = await ctx()
    const viewer = await viewerCtx.newPage()
    await signIn(viewer, '/', 'catalogueViewer')
    await viewer.goto(poUrl)
    await expect(viewer.getByRole('heading', { level: 1, name: number })).toBeVisible()
    for (const label of ['Submit', 'Approve', 'Edit', 'Cancel PO', 'Return to draft']) {
      await expect(viewer.getByRole('button', { name: label, exact: true })).toHaveCount(0)
    }
    await viewerCtx.close()

    // The creator sees the approval and the frozen snapshot after a reload.
    await page.reload()
    await expect(
      page.getByText('Approved', { exact: true }).filter({ visible: true }).first(),
    ).toBeVisible()
    await expect(
      page
        .getByText(/difference ₹50.00/)
        .filter({ visible: true })
        .first(),
    ).toBeVisible()
    await expect(page.getByText('Created', { exact: true })).toBeVisible()
    await expect(
      page.getByText('Submitted', { exact: true }).filter({ visible: true }).first(),
    ).toBeVisible()
    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })
})
