import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Parts receipts and issues (E-04) against the real API and PostgreSQL: receive,
 * issue, balance and history refresh, insufficient stock, a lost response retried
 * with the same request (recorded once), double clicks, serial SKUs, read-only and
 * no-permission users. Placeholder data; throwaway database only. PROPOSED grants:
 * E2E_CATALOGUE also needs stock.parts.view/receive/issue, stock_items.view/manage
 * and suppliers.view/manage, plus a location in its scope (cmd/access scope-add);
 * E2E_CATALOGUE_VIEWER needs stock_items.view and stock.parts.view and the same location.
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function post(page: Page, path: string, data: object): Promise<string> {
  const res = await page.request.post(`/api/v1${path}`, {
    headers: { 'X-CSRF-Token': await csrf(page.request) },
    data,
  })
  expect(res.ok(), await res.text()).toBe(true)
  return ((await res.json()) as { data: { id: string } }).data.id
}

async function movementCount(page: Page, itemId: string): Promise<number> {
  const res = await page.request.get(`/api/v1/stock-items/${itemId}/stock-movements`)
  return ((await res.json()) as { data: { total: number } }).data.total
}

async function noSidewaysScroll(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )
  expect(overflow).toBeLessThanOrEqual(1)
}

const visible = (page: Page, text: string | RegExp) =>
  page.getByText(text).filter({ visible: true }).first()

test.describe('parts receipts and issues', () => {
  test('receive, issue, refuse, retry once, and stay read-only for others', async ({
    page,
    browser,
  }, info) => {
    test.skip(!hasAccount('catalogue'), 'Set E2E_CATALOGUE_* (see DEVELOPMENT_SETUP)')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/parts', 'catalogue')

    // Disposable master data through the API.
    const categoryId = await post(page, '/stock-item-categories', { name: `Stock test ${tag}` })
    const itemId = await post(page, '/stock-items', {
      sku: `STK-${tag}`,
      name: 'Test part',
      categoryId,
      uom: 'PCS',
      trackingMode: 'QUANTITY',
    })
    const serialId = await post(page, '/stock-items', {
      sku: `SER-${tag}`,
      name: 'Test serial part',
      categoryId,
      uom: 'PCS',
      trackingMode: 'SERIAL',
    })
    await post(page, '/suppliers', { legalName: `Test Stock Supplier ${tag}` })

    await page.goto(`/inventory/parts/${itemId}`)
    await expect(visible(page, 'No stock recorded')).toBeVisible()

    // Receive 10: the balance and history come back from the server.
    await page.getByRole('button', { name: 'Receive stock' }).click()
    let dialog = page.getByRole('dialog', { name: 'Receive stock' })
    await dialog
      .getByLabel(/Supplier/)
      .selectOption({ label: `Test Stock Supplier ${tag} (pending verification)` })
    await dialog.getByLabel(/Quantity/).fill('10.5555')
    await dialog.getByLabel(/Challan or invoice/).fill('DC-1')
    await dialog.getByRole('button', { name: 'Record receipt' }).click()
    await expect(dialog.getByText(/up to 3 decimals \(e.g. 5/)).toBeVisible()
    await dialog.getByLabel(/Quantity/).fill('10')
    // Double click: one receipt.
    await dialog.getByRole('button', { name: 'Record receipt' }).dblclick()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, /Receipt recorded: \+10 PCS/)).toBeVisible()
    await expect(visible(page, '10 PCS')).toBeVisible()
    expect(await movementCount(page, itemId)).toBe(1)

    // Issue 3 -> 7.
    await page.getByRole('button', { name: 'Issue stock' }).click()
    dialog = page.getByRole('dialog', { name: 'Issue stock' })
    await expect(dialog.getByText('10 PCS')).toBeVisible()
    await dialog.getByLabel(/Quantity/).fill('3')
    await dialog.getByLabel(/^Reason/).fill('Fitted in a repair')
    await dialog.getByRole('button', { name: 'Record issue' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, /Issue recorded: −3 PCS/)).toBeVisible()
    await expect(visible(page, '7 PCS')).toBeVisible()

    // Issue more than there is: refused by the server, the form is kept.
    await page.getByRole('button', { name: 'Issue stock' }).click()
    dialog = page.getByRole('dialog', { name: 'Issue stock' })
    await dialog.getByLabel(/Quantity/).fill('100')
    await dialog.getByLabel(/^Reason/).fill('Too many')
    await dialog.getByRole('button', { name: 'Record issue' }).click()
    await expect(dialog.getByText(/not enough stock at this location/)).toBeVisible()
    await expect(dialog.getByLabel(/Quantity/)).toHaveValue('100')
    await dialog.getByRole('button', { name: 'Cancel' }).click()
    expect(await movementCount(page, itemId)).toBe(2)

    // The response of a receipt is lost after the server recorded it: the retry
    // sends the same request (same key) and it is counted once.
    let lose = true
    await page.route(`**/api/v1/stock-items/${itemId}/receipts`, async (route) => {
      if (!lose) return route.continue()
      lose = false
      await route.fetch()
      await route.abort('failed')
    })
    await page.getByRole('button', { name: 'Receive stock' }).click()
    dialog = page.getByRole('dialog', { name: 'Receive stock' })
    await dialog
      .getByLabel(/Supplier/)
      .selectOption({ label: `Test Stock Supplier ${tag} (pending verification)` })
    await dialog.getByLabel(/Quantity/).fill('2')
    await dialog.getByLabel(/Challan or invoice/).fill('DC-2')
    await dialog.getByRole('button', { name: 'Record receipt' }).click()
    await expect(dialog.getByText(/may or may not have been recorded/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Send again unchanged' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, '9 PCS')).toBeVisible()
    expect(await movementCount(page, itemId)).toBe(3)
    await page.unroute(`**/api/v1/stock-items/${itemId}/receipts`)

    // History: newest first, filter by type, no edit or delete.
    // (A table on desktop, cards on phones: match what is visible.)
    await expect(visible(page, 'DC-2')).toBeVisible()
    await page.getByLabel('Type').selectOption('PARTS_ISSUE')
    await expect(page.getByText('DC-2').filter({ visible: true })).toHaveCount(0)
    await expect(visible(page, 'Reason: Fitted in a repair')).toBeVisible()
    await expect(page.getByRole('button', { name: /Delete|Edit entry/ })).toHaveCount(0)
    await page.reload()
    await expect(visible(page, '9 PCS')).toBeVisible()
    await noSidewaysScroll(page)

    // Serial-tracked: no forms.
    await page.goto(`/inventory/parts/${serialId}`)
    await expect(visible(page, /serial-tracked/)).toBeVisible()
    await expect(page.getByRole('button', { name: /Receive stock|Issue stock/ })).toHaveCount(0)

    // A viewer reads, cannot act; a user without grants sees no stock.
    if (hasAccount('catalogueViewer')) {
      const ctx = await browser.newContext()
      const viewer = await ctx.newPage()
      await signIn(viewer, `/inventory/parts/${itemId}`, 'catalogueViewer')
      await expect(visible(viewer, '9 PCS')).toBeVisible()
      await expect(viewer.getByRole('button', { name: /Receive stock|Issue stock/ })).toHaveCount(0)
      await ctx.close()
    }
    const ctx = await browser.newContext()
    const nobody = await ctx.newPage()
    await signIn(nobody, `/inventory/parts/${itemId}`, 'active')
    await expect(visible(nobody, 'Access denied')).toBeVisible()
    await ctx.close()
    expect(problems).toEqual([])
  })
})
