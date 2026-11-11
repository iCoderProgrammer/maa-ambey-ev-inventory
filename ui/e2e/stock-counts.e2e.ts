import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Stock counts and adjustment approval (E-05) against the real API and
 * PostgreSQL: start a count, record a part (reason required for a difference),
 * submit, approve as another person (one adjustment posted), stock changed
 * since the count (refused, then rejected), reload, read-only and no-permission
 * users. Placeholder data; throwaway database only. PROPOSED grants:
 * E2E_CATALOGUE also needs stock.adjustment.view/create (with the E-04 parts
 * grants); E2E_PRICE_APPROVER needs stock.adjustment.view/approve;
 * E2E_CATALOGUE_VIEWER stock.adjustment.view; all three the same location.
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function post<T = { id: string }>(
  page: Page,
  path: string,
  data: object,
  key?: string,
): Promise<T> {
  const res = await page.request.post(`/api/v1${path}`, {
    headers: {
      'X-CSRF-Token': await csrf(page.request),
      ...(key ? { 'Idempotency-Key': key } : {}),
    },
    data,
  })
  expect(res.ok(), await res.text()).toBe(true)
  return ((await res.json()) as { data: T }).data
}

async function balance(page: Page, itemId: string): Promise<string> {
  const res = await page.request.get(`/api/v1/stock-items/${itemId}/balances`)
  return ((await res.json()) as { data: { quantity: string }[] }).data[0]?.quantity ?? 'none'
}

const visible = (page: Page, text: string | RegExp) =>
  page.getByText(text).filter({ visible: true }).first()

test.describe('stock counts', () => {
  test('count, submit, approve once, refuse stale stock, and stay read-only for others', async ({
    page,
    browser,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') || !hasAccount('priceApprover'),
      'Set E2E_CATALOGUE_* and E2E_PRICE_APPROVER_* (see DEVELOPMENT_SETUP)',
    )
    test.setTimeout(150_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/counts', 'catalogue')
    const meRes = await page.request.get('/api/v1/me')
    const locationId = (
      (await meRes.json()) as { data: { locationScope: { locations: { id: string }[] } } }
    ).data.locationScope.locations[0]?.id
    expect(locationId, 'E2E_CATALOGUE needs a location in scope').toBeTruthy()

    // Disposable stock: a SKU with 10 PCS received at the location.
    const categoryId = (await post(page, '/stock-item-categories', { name: `Count test ${tag}` }))
      .id
    const itemId = (
      await post(page, '/stock-items', {
        sku: `CNT-${tag}`,
        name: 'Count test part',
        categoryId,
        uom: 'PCS',
        trackingMode: 'QUANTITY',
      })
    ).id
    const supplierId = (await post(page, '/suppliers', { legalName: `Count Supplier ${tag}` })).id
    await post(
      page,
      `/stock-items/${itemId}/receipts`,
      { locationId, quantity: '10', uom: 'PCS', supplierId, sourceReference: 'DC-C1' },
      `rcpt-${tag}`,
    )

    // Start a count in the UI.
    await page.getByRole('button', { name: 'New count' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'Start a stock count' })
    await dialog.getByLabel('Note').fill(`Shelf ${tag}`)
    await dialog.getByRole('button', { name: 'Start count' }).click()
    await expect(page.getByRole('heading', { level: 1, name: /^SC-/ })).toBeVisible()
    const countUrl = page.url()
    const countNumber = (await page.getByRole('heading', { level: 1 }).textContent()) ?? ''

    // Record the part: a difference needs a reason (the server says so).
    await page.getByRole('button', { name: 'Record item' }).click()
    dialog = page.getByRole('dialog', { name: 'Record a counted item' })
    await dialog.getByLabel('Find a part').fill(`CNT-${tag}`)
    await expect(dialog.getByRole('option', { name: new RegExp(`CNT-${tag}`) })).toHaveCount(1)
    await dialog.getByLabel(/^Part/).selectOption({ label: `CNT-${tag} — Count test part (PCS)` })
    await dialog.getByLabel(/Quantity counted/).fill('8')
    await dialog.getByRole('button', { name: 'Save count' }).click()
    await expect(
      dialog.getByText('Give a reason: this count differs from the stock record'),
    ).toBeVisible()
    await expect(dialog.getByLabel(/Quantity counted/)).toHaveValue('8')
    await dialog.getByLabel('Reason').fill('Two damaged')
    await dialog.getByRole('button', { name: 'Save count' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, '8 PCS')).toBeVisible()
    // The counter does not see the stock record or the difference.
    await expect(page.getByText('10 PCS').filter({ visible: true })).toHaveCount(0)
    expect(await balance(page, itemId)).toBe('10.000')

    // Submit: still no stock change; the counter cannot decide.
    await page.getByRole('button', { name: 'Submit for approval' }).click()
    dialog = page.getByRole('dialog')
    await dialog.getByRole('button', { name: 'Submit for approval' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, 'Awaiting approval')).toBeVisible()
    await expect(page.getByRole('button', { name: /^Approve$|^Reject$/ })).toHaveCount(0)
    expect(await balance(page, itemId)).toBe('10.000')

    // Another person approves from the queue: one adjustment is posted.
    const ctx = await browser.newContext({ ...info.project.use })
    const approver = await ctx.newPage()
    const approverProblems = watchConsole(approver)
    await signIn(approver, '/inventory/counts/approvals', 'priceApprover')
    await approver.getByRole('link', { name: countNumber }).filter({ visible: true }).click()
    await expect(visible(approver, '10 PCS')).toBeVisible()
    await expect(visible(approver, '−2 PCS')).toBeVisible()
    await approver.getByRole('button', { name: 'Approve', exact: true }).click()
    dialog = approver.getByRole('dialog', { name: `Approve ${countNumber}?` })
    await expect(dialog.getByText(/1 parts adjustment will be posted/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Approve and post' }).dblclick()
    await expect(visible(approver, /approved\. 1 adjustment was posted/)).toBeVisible()
    await expect(visible(approver, 'Adjustment posted')).toBeVisible()
    expect(await balance(page, itemId)).toBe('8.000')

    // Stock moves after a count is submitted: approval is refused, nothing posted; reject it.
    const c2 = await post<{ id: string; version: number; countNumber: string }>(
      page,
      '/stock-counts',
      { locationId },
      `cnt2-${tag}`,
    )
    const r2 = await post<{ version: number }>(page, `/stock-counts/${c2.id}/lines`, {
      version: c2.version,
      stockItemId: itemId,
      observedQuantity: '5',
      uom: 'PCS',
      reason: 'Three missing',
    })
    await post(page, `/stock-counts/${c2.id}/submit`, { version: r2.version })
    await post(
      page,
      `/stock-items/${itemId}/issues`,
      { locationId, quantity: '1', uom: 'PCS', reason: 'Fitted in a repair' },
      `iss-${tag}`,
    )
    await approver.goto(`/inventory/counts/${c2.id}`)
    await approver.getByRole('button', { name: 'Approve', exact: true }).click()
    dialog = approver.getByRole('dialog')
    await dialog.getByRole('button', { name: 'Approve and post' }).click()
    await expect(dialog.getByText(/Stock moved after some lines were counted/)).toBeVisible()
    await expect(dialog.getByText('Reject it so the items can be counted again')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()
    await expect(visible(approver, 'Stock moved since counted')).toBeVisible()
    expect(await balance(page, itemId)).toBe('7.000')
    await approver.getByRole('button', { name: 'Reject', exact: true }).click()
    dialog = approver.getByRole('dialog')
    await dialog.getByLabel(/Reason/).fill('Stock moved; recount')
    await dialog.getByRole('button', { name: 'Reject count' }).click()
    await expect(visible(approver, /rejected\. Nothing was posted/)).toBeVisible()
    expect(await balance(page, itemId)).toBe('7.000')
    // Reload: everything comes back from the server.
    await approver.reload()
    await expect(visible(approver, 'Not posted (rejected)')).toBeVisible()
    const overflow = await approver.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    await ctx.close()

    // The counter's view after reload: approved, no decision buttons, no cancel/delete.
    await page.goto(countUrl)
    await expect(visible(page, 'Adjustment posted')).toBeVisible()
    await expect(
      page.getByRole('button', { name: /Approve|Reject|Delete|Remove|Cancel count/ }),
    ).toHaveCount(0)

    // A viewer reads only; a user without grants is denied.
    if (hasAccount('catalogueViewer')) {
      const vctx = await browser.newContext()
      const viewer = await vctx.newPage()
      await signIn(viewer, countUrl.replace(/^https?:\/\/[^/]+/, ''), 'catalogueViewer')
      await expect(visible(viewer, 'Adjustment posted')).toBeVisible()
      await expect(
        viewer.getByRole('button', { name: /Record item|Approve|New count/ }),
      ).toHaveCount(0)
      await vctx.close()
    }
    const nctx = await browser.newContext()
    const nobody = await nctx.newPage()
    await signIn(nobody, '/inventory/counts', 'active')
    await expect(visible(nobody, 'Access denied')).toBeVisible()
    await nctx.close()
    expect([...problems, ...approverProblems]).toEqual([])
  })
})
