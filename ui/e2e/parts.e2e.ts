import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Spare part and accessory SKU master (E-03) against the real API: categories,
 * SKU create with validation and duplicate handling, suppliers and brand/model
 * compatibility, read-only code and tracking mode on edit, archive with reason,
 * category archive refused while it has active SKUs, read-only viewer with
 * suppliers hidden. Placeholder data; throwaway database only. PROPOSED grants:
 * E2E_CATALOGUE also needs stock_items.view/manage/archive (plus suppliers.view,
 * suppliers.manage and catalogue.*, which the supplier and catalogue tests use);
 * E2E_CATALOGUE_VIEWER needs stock_items.view only (no suppliers.view).
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

async function noSidewaysScroll(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )
  expect(overflow).toBeLessThanOrEqual(1)
}

test.describe('spare parts and accessories', () => {
  test('are denied without stock_items.view', async ({ page }) => {
    const problems = watchConsole(page)
    await signIn(page, '/inventory/parts')
    await expect(page.getByText('Access denied')).toBeVisible()
    expect(problems).toEqual([])
  })

  test('categories and SKUs: create, duplicates, links, edit, archive', async ({ page }, info) => {
    test.skip(!hasAccount('catalogue'), 'Set E2E_CATALOGUE_* (see DEVELOPMENT_SETUP)')
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/parts/categories', 'catalogue')

    // Category through the UI; a second one in another case is a duplicate.
    await page.getByRole('button', { name: 'Add category' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'Add category' })
    await dialog.getByLabel(/^Name/).fill(`Brake parts ${tag}`)
    await dialog.getByRole('button', { name: 'Add category' }).click()
    await expect(dialog).toHaveCount(0)
    await page.getByRole('button', { name: 'Add category' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add category' })
    await dialog.getByLabel(/^Name/).fill(`BRAKE PARTS ${tag.toLowerCase()}`)
    await dialog.getByRole('button', { name: 'Add category' }).click()
    await expect(dialog.getByText('Already used — choose another')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Disposable linked records through the API: a brand with a model, and a supplier.
    const brandId = await post(page, '/brands', { name: `Test Brand ${tag}` })
    await post(page, '/models', { brandId, name: `Test Model ${tag}`, internalCode: `TM${tag}` })
    await post(page, '/suppliers', { legalName: `Test Parts Supplier ${tag}` })

    // SKU: invalid values are caught before sending; then created with links.
    await page.goto('/inventory/parts')
    await page.getByRole('button', { name: 'Add SKU' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add SKU' })
    await dialog.getByLabel(/SKU code/).fill(`BRK ${tag}`)
    await dialog.getByLabel(/^Name/).fill('Brake pad set')
    await dialog.getByLabel(/Category/).selectOption({ label: `Brake parts ${tag}` })
    await dialog.getByLabel(/Unit of measure/).fill('per piece')
    await dialog.getByLabel(/Tracking mode/).selectOption('SERIAL')
    await dialog.getByLabel(/Reorder level/).fill('2.5')
    await dialog.getByRole('button', { name: 'Add SKU' }).click()
    await expect(dialog.getByText(/— no spaces/)).toBeVisible()
    await expect(dialog.getByText(/whole number/)).toBeVisible()
    await dialog.getByLabel(/SKU code/).fill(`  brk-${tag.toLowerCase()} `)
    await dialog.getByLabel(/Unit of measure/).fill('set')
    await dialog.getByLabel(/Reorder level/).fill('3')
    await dialog.getByRole('button', { name: 'Add supplier' }).click()
    await dialog
      .getByLabel('Supplier 1', { exact: true })
      .selectOption({ label: `Test Parts Supplier ${tag} (pending verification)` })
    await dialog.getByLabel('Supplier part number 1').fill('SP-1')
    await dialog.getByRole('button', { name: 'Add compatibility' }).click()
    await dialog.getByLabel('Brand 1', { exact: true }).selectOption({ label: `Test Brand ${tag}` })
    await dialog.getByLabel('Model 1', { exact: true }).selectOption({ label: `Test Model ${tag}` })
    await dialog.getByRole('button', { name: 'Add SKU' }).click()
    await expect(dialog).toHaveCount(0)
    const sku = `brk-${tag.toLowerCase()}`
    await expect(page.getByRole('heading', { level: 1, name: sku })).toBeVisible()
    const detailUrl = page.url()
    await expect(page.getByText('Serial — Each piece will be identified')).toBeVisible()
    await expect(
      page
        .getByRole('link', { name: `Test Parts Supplier ${tag}` })
        .filter({ visible: true })
        .first(),
    ).toBeVisible()
    await expect(
      page.getByText(`Test Model ${tag}`).filter({ visible: true }).first(),
    ).toBeVisible()

    // The same code in another case is a duplicate; entries stay.
    await page.goto('/inventory/parts')
    await page.getByRole('button', { name: 'Add SKU' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add SKU' })
    await dialog.getByLabel(/SKU code/).fill(sku.toUpperCase())
    await dialog.getByLabel(/^Name/).fill('Second')
    await dialog.getByLabel(/Category/).selectOption({ label: `Brake parts ${tag}` })
    await dialog.getByLabel(/Unit of measure/).fill('PCS')
    await dialog.getByLabel(/Tracking mode/).selectOption('QUANTITY')
    await dialog.getByRole('button', { name: 'Add SKU' }).click()
    await expect(dialog.getByText('Already used — choose another')).toBeVisible()
    await expect(dialog.getByLabel(/^Name/)).toHaveValue('Second')
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // The category cannot be archived while the SKU is active.
    await page.goto(`/inventory/parts/categories?q=${encodeURIComponent(tag)}`)
    await page
      .getByRole('button', { name: `Archive Brake parts ${tag}` })
      .first()
      .click()
    dialog = page.getByRole('dialog', { name: 'Archive category' })
    await dialog.getByLabel(/Reason/).fill('Test')
    await dialog.getByRole('button', { name: 'Archive category' }).click()
    await expect(dialog.getByText(/still has active SKUs/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Edit: code and tracking mode read-only; only the name changes; persists after reload.
    await page.goto(detailUrl)
    await page.getByRole('button', { name: 'Edit' }).click()
    dialog = page.getByRole('dialog', { name: 'Edit SKU' })
    await expect(dialog.getByLabel('SKU code (cannot be changed)')).toHaveText(sku)
    await expect(dialog.getByLabel('Tracking mode (cannot be changed)')).toHaveText('Serial')
    await dialog.getByLabel(/^Name/).fill('Brake pad set (front)')
    await dialog.getByRole('button', { name: 'Save SKU' }).click()
    await expect(dialog).toHaveCount(0)
    await page.reload()
    await expect(page.getByText('Brake pad set (front)').first()).toBeVisible()
    await noSidewaysScroll(page)

    // Archive with a reason: readable, no actions, code kept.
    await page.getByRole('button', { name: 'Archive' }).click()
    dialog = page.getByRole('dialog', { name: 'Archive SKU' })
    await dialog.getByRole('button', { name: 'Archive SKU' }).click()
    await expect(dialog.getByText('Give a reason')).toBeVisible()
    await dialog.getByLabel(/Reason/).fill('Discontinued')
    await dialog.getByRole('button', { name: 'Archive SKU' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText(/This SKU is archived/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit' })).toHaveCount(0)
    await expect(
      page
        .getByRole('link', { name: `Test Parts Supplier ${tag}` })
        .filter({ visible: true })
        .first(),
    ).toBeVisible()

    // Now the category can be archived; filters find the archived SKU.
    await page.goto(`/inventory/parts/categories?q=${encodeURIComponent(tag)}`)
    await page
      .getByRole('button', { name: `Archive Brake parts ${tag}` })
      .first()
      .click()
    dialog = page.getByRole('dialog', { name: 'Archive category' })
    await dialog.getByLabel(/Reason/).fill('Test done')
    await dialog.getByRole('button', { name: 'Archive category' }).click()
    await expect(dialog).toHaveCount(0)
    await page.goto(`/inventory/parts?q=${encodeURIComponent(sku)}&status=ARCHIVED`)
    await expect(page.getByRole('link', { name: sku }).first()).toBeVisible()
    await page.goto(`/inventory/parts?q=${encodeURIComponent(sku)}&status=ACTIVE`)
    await expect(page.getByText('No SKUs match these filters')).toBeVisible()
    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })

  test('a viewer reads SKUs without actions; suppliers only with suppliers.view', async ({
    page,
  }) => {
    test.skip(!hasAccount('catalogueViewer'), 'Set E2E_CATALOGUE_VIEWER_* (see DEVELOPMENT_SETUP)')
    const problems = watchConsole(page)
    await signIn(page, '/inventory/parts', 'catalogueViewer')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { permissions: string[] }
    }
    const seesSuppliers = me.data.permissions.includes('suppliers.view')
    await expect(
      page.getByRole('heading', { level: 1, name: 'Spare Parts & Accessories' }),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add SKU' })).toHaveCount(0)
    await expect(page.getByLabel('Supplier', { exact: true })).toHaveCount(seesSuppliers ? 1 : 0)
    const first = page.getByRole('link', { name: /^brk-/i }).filter({ visible: true }).first()
    if ((await first.count()) > 0) {
      await first.click()
      await expect(page.getByText(/need permission to view suppliers/)).toHaveCount(
        seesSuppliers ? 0 : 1,
      )
      await expect(page.getByRole('button', { name: 'Edit' })).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Archive' })).toHaveCount(0)
    }
    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })
})
