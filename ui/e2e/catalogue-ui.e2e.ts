import { expect, test, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Catalogue & brand management screens (C-03) against the real API. Creates
 * placeholder records ("E2E UI Brand …", example.com sources): throwaway
 * database only — grants are PROPOSED and count only with
 * RBAC_ALLOW_PROPOSED_GRANTS=true in development.
 */

const needsAccounts = () => !hasAccount('catalogue') || !hasAccount('catalogueViewer')

async function noSidewaysScroll(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  )
  expect(overflow).toBeLessThanOrEqual(1)
}

/** Goes to a catalogue tab; on phones the sidebar is a drawer, so navigate by URL. */
async function openCatalogue(page: Page, isMobile: boolean, path = '/admin/catalogue') {
  if (isMobile || path !== '/admin/catalogue') {
    await page.goto(path)
  } else {
    await page
      .getByRole('navigation', { name: 'Main' })
      .getByRole('link', { name: /Catalogue & Brands/ })
      .click()
  }
  await expect(page.getByRole('heading', { level: 1, name: 'Catalogue & Brands' })).toBeVisible()
}

test.describe('catalogue screens (C-03)', () => {
  test('without catalogue.view there is no link and the page is denied', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page, '/', 'active')
    if (!isMobile) {
      await expect(
        page
          .getByRole('navigation', { name: 'Main' })
          .getByRole('link', { name: /Catalogue & Brands/ }),
      ).toHaveCount(0)
    }
    await page.goto('/admin/catalogue')
    await expect(page.getByText('Access denied')).toBeVisible()
    await expect(page.getByRole('table')).toHaveCount(0)
    expect(problems).toEqual([])
  })

  test('a viewer sees the real brand roster, statuses and no edit actions', async ({
    page,
    isMobile,
  }) => {
    test.skip(needsAccounts(), 'needs E2E_CATALOGUE_* accounts (throwaway database only)')
    const problems = watchConsole(page)
    await signIn(page, '/', 'catalogueViewer')
    await openCatalogue(page, isMobile)
    // Table on desktop, stacked list on phones; both are visible only at their size.
    const table = page.getByRole(isMobile ? 'list' : 'table', { name: 'Brands' })
    await expect(table).toBeVisible()
    // The seeded roster is listed from the API, pending and not authorised.
    expect(await table.getByRole('link').count()).toBeGreaterThanOrEqual(3)
    await expect(table.getByText('Authorisation unverified').first()).toBeVisible()
    await expect(table.getByText('Not enabled for purchase & sales').first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add brand' })).toHaveCount(0)
    await expect(page.getByRole('note').first()).toContainText('not confirmed facts')
    await noSidewaysScroll(page)

    await table.getByRole('link').first().click()
    await expect(page.getByRole('heading', { name: 'Activation readiness' })).toBeVisible()
    await expect(page.getByText(/supplier master, D-01/)).toBeVisible()
    for (const name of ['Edit', 'Archive', 'Record', 'Enable…', 'Add model']) {
      await expect(page.getByRole('button', { name, exact: true })).toHaveCount(0)
    }
    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })

  test('an administrator manages brand, model and variant through the real API', async ({
    page,
    browser,
    isMobile,
  }, testInfo) => {
    test.skip(needsAccounts(), 'needs E2E_CATALOGUE_* accounts (throwaway database only)')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const run = `${testInfo.project.name} ${Date.now().toString(36)}`
    const brandName = `E2E UI Brand ${run}`
    await signIn(page, '/', 'catalogue')
    await openCatalogue(page, isMobile)

    // Create a brand: client validation first, then the server.
    await page.getByRole('button', { name: 'Add brand' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'Add brand' })
    await dialog.getByRole('button', { name: 'Add brand' }).click()
    await expect(dialog.getByText('Name is required')).toBeVisible()
    await dialog.getByLabel(/Brand name/).fill(brandName)
    await dialog.getByRole('button', { name: 'Add brand' }).click()
    await expect(page.getByRole('heading', { level: 1, name: brandName })).toBeVisible()
    await expect(page.getByText('Authorisation not verified.')).toBeVisible()
    const brandUrl = page.url()

    // Duplicate name (case-insensitive) is refused by the server and shown on the field.
    await page.goto('/admin/catalogue')
    await page.getByRole('button', { name: 'Add brand' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add brand' })
    await dialog.getByLabel(/Brand name/).fill(brandName.toLowerCase())
    await dialog.getByRole('button', { name: 'Add brand' }).click()
    await expect(dialog.getByText('Already used — choose another')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Add a model from the brand page.
    await page.goto(brandUrl)
    await page.getByRole('button', { name: 'Add model' }).click()
    dialog = page.getByRole('dialog', { name: 'Add model' })
    await dialog.getByLabel(/Model name/).fill(`E2E Model ${run}`)
    await dialog.getByLabel(/Internal code/).fill('e2e-m1')
    await dialog.getByRole('button', { name: 'Add model' }).click()
    await expect(dialog).toHaveCount(0)
    await page.getByRole('link', { name: `E2E Model ${run}` }).click()
    await expect(page.getByRole('heading', { level: 1, name: `E2E Model ${run}` })).toBeVisible()
    await expect(page.getByText('E2E-M1')).toBeVisible()

    // Activation before verification: confirmed, then refused by the server with the reason.
    await page.getByRole('button', { name: 'Activate', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Activate model' })
    await dialog.getByRole('button', { name: 'Activate model' }).click()
    await expect(dialog.getByText('Source and verification recorded')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Verify, then activate.
    await page.getByRole('button', { name: 'Verify', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Record model verification' })
    await dialog.getByLabel(/Source link/).fill('https://example.com/e2e-source')
    await dialog.getByLabel(/Source date/).fill('2026-10-01')
    await dialog.getByRole('button', { name: 'Record verification' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('Supplier confirmed').first()).toBeVisible()
    await page.getByRole('button', { name: 'Activate', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Activate model' })
    await dialog.getByRole('button', { name: 'Activate model' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('Active', { exact: true }).first()).toBeVisible()
    // Active items cannot be edited.
    await expect(page.getByRole('button', { name: 'Edit', exact: true })).toHaveCount(0)
    const modelUrl = page.url()

    // Add a variant from the Variants tab: the model list holds only the chosen brand's models.
    await openCatalogue(page, isMobile, '/admin/catalogue/variants')
    await page.getByRole('button', { name: 'Add variant' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add variant' })
    const modelSelect = dialog.getByLabel(/^Model/)
    await expect(modelSelect).toBeDisabled()
    await dialog.getByLabel(/^Brand/).selectOption({ label: brandName })
    await expect(modelSelect).toBeEnabled()
    const offered = await modelSelect.locator('option').allTextContents()
    expect(offered).toEqual(['Choose a model', `E2E Model ${run} (E2E-M1)`])
    await modelSelect.selectOption({ label: `E2E Model ${run} (E2E-M1)` })
    await dialog.getByLabel(/Variant name/).fill('E2E Variant')
    await dialog.getByLabel(/Variant code/).fill('std')
    await dialog.getByRole('button', { name: 'Add variant' }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'E2E Variant' })).toBeVisible()
    await expect(page.getByText('STD', { exact: true })).toBeVisible()
    await expect(page.getByText(/No price is shown or implied/)).toBeVisible()

    // Stale version: another session changes the variant first.
    const other = await browser.newContext()
    const otherPage = await other.newPage()
    await signIn(otherPage, '/', 'catalogue')
    const csrf = (
      (await (await otherPage.request.get('/api/v1/auth/session')).json()) as {
        data: { csrfToken: string }
      }
    ).data.csrfToken
    const variantId = page.url().split('/').pop() ?? ''
    const current = (
      (await (await otherPage.request.get(`/api/v1/variants/${variantId}`)).json()) as {
        data: { version: number }
      }
    ).data
    const changed = await otherPage.request.patch(`/api/v1/variants/${variantId}`, {
      headers: { 'X-CSRF-Token': csrf },
      data: { version: current.version, name: 'E2E Variant (changed elsewhere)' },
    })
    expect(changed.status()).toBe(200)
    await other.close()
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Edit variant' })
    await dialog.getByLabel(/Variant name/).fill('E2E Variant renamed here')
    await dialog.getByRole('button', { name: 'Save variant' }).click()
    await expect(dialog.getByText(/Someone else changed this record/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Reload latest version' }).click()
    await expect(
      page.getByRole('heading', { level: 1, name: 'E2E Variant (changed elsewhere)' }),
    ).toBeVisible()

    // Archive the model with a reason; history stays, nothing new can be added.
    await page.goto(modelUrl)
    await page.getByRole('button', { name: 'Archive', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Archive model' })
    await expect(dialog.getByText(/Nothing is deleted/)).toBeVisible()
    await dialog.getByLabel(/Reason/).fill('E2E archive')
    await dialog.getByRole('button', { name: 'Archive model' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText(/This model is archived/)).toBeVisible()
    await expect(page.getByRole('link', { name: 'E2E Variant (changed elsewhere)' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add variant' })).toHaveCount(0)

    // Brand activation stays impossible: readiness lists pending prerequisites.
    await page.goto(brandUrl)
    await page.getByRole('button', { name: 'Enable…' }).click()
    dialog = page.getByRole('dialog', { name: 'Enable brand for purchase and sales' })
    await expect(dialog.getByRole('button', { name: 'Enable for purchase & sales' })).toBeDisabled()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Refresh keeps state from the server.
    await page.reload()
    await expect(page.getByRole('heading', { level: 1, name: brandName })).toBeVisible()
    await noSidewaysScroll(page)
    expect(problems).toEqual([])
  })
})
