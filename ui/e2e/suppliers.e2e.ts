import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Supplier master (D-01) against the real API: create with validation and
 * duplicate handling, private document upload and download, identity
 * verification, multi-brand relationships with evidence, contacts kept as
 * history, archive. Placeholder data and a synthetic GSTIN; throwaway database
 * only (PROPOSED grants: the E2E_CATALOGUE user needs suppliers.* and
 * attachments.upload/view; E2E_CATALOGUE_VIEWER needs suppliers.view).
 */

const pdf = Buffer.from('%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n')

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function apiBrand(page: Page, name: string): Promise<string> {
  const res = await page.request.post('/api/v1/brands', {
    headers: { 'X-CSRF-Token': await csrf(page.request) },
    data: { name },
  })
  expect(res.ok(), await res.text()).toBe(true)
  return ((await res.json()) as { data: { id: string } }).data.id
}

function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function syntheticGSTIN(seed: number): string {
  // 2 digits + 5 letters + 4 digits + letter + entity + Z + check (format only; not a real registration).
  const letters = 'ABCDEFGHJKLMNPQRSTUVWXYZ'
  const l = (i: number) => letters[(seed + i) % letters.length] ?? 'A'
  return `27${l(0)}${l(1)}${l(2)}${l(3)}${l(4)}${String(1000 + (seed % 9000))}${l(5)}1Z${l(6)}`
}

test.describe('supplier master (D-01)', () => {
  test('without suppliers.view there is no link and the page is denied', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page, '/', 'active')
    if (!isMobile) {
      await expect(
        page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: /Suppliers/ }),
      ).toHaveCount(0)
    }
    await page.goto('/procurement/suppliers')
    await expect(page.getByText('Access denied')).toBeVisible()
    expect((await page.request.get('/api/v1/suppliers')).status()).toBe(403)
    // A denied detail link keeps its heading and breadcrumbs.
    await page.goto('/procurement/suppliers/00000000-0000-0000-0000-000000000000')
    await expect(page.getByText('Access denied')).toBeVisible()
    await expect(page.getByRole('heading', { level: 1, name: 'Supplier' })).toBeVisible()
    expect(problems).toEqual([])
  })

  test('dialogs return focus to their opener; an offline save explains itself and sends nothing later', async ({
    page,
    context,
  }) => {
    test.skip(!hasAccount('catalogue'), 'needs E2E_CATALOGUE with supplier grants')
    const problems = watchConsole(page)
    await signIn(page, '/procurement/suppliers', 'catalogue')
    const opener = page.getByRole('button', { name: 'Add supplier' }).first()
    const dialog = page.getByRole('dialog', { name: 'Add supplier' })
    await opener.click()
    await expect(dialog).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(opener).toBeFocused()

    const name = `E2E Offline Supplier ${Date.now().toString(36)}`
    await opener.click()
    await dialog.getByLabel(/Legal name/).fill(name)
    await context.setOffline(true)
    await dialog.getByRole('button', { name: 'Add supplier' }).click()
    await expect(dialog.getByRole('alert')).toContainText('Cannot reach the server')
    await expect(dialog.getByLabel(/Legal name/)).toHaveValue(name)
    await dialog.getByRole('button', { name: 'Cancel' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(opener).toBeFocused()
    await context.setOffline(false)
    await page.waitForTimeout(1500)
    const res = await page.request.get(`/api/v1/suppliers?q=${encodeURIComponent(name)}`)
    expect(((await res.json()) as { data: { total: number } }).data.total).toBe(0)
    expect(problems).toEqual([])
  })

  test('create, document, verify, relate to brands, contacts and archive', async ({
    page,
    browser,
    isMobile,
  }, testInfo) => {
    test.skip(
      !hasAccount('catalogue') || !hasAccount('catalogueViewer'),
      'needs E2E_CATALOGUE_* with supplier grants (throwaway database only)',
    )
    test.setTimeout(150_000)
    const problems = watchConsole(page)
    const run = `${testInfo.project.name} ${Date.now().toString(36)}`
    const name = `E2E Supplier ${run}`
    const gstin = syntheticGSTIN((Date.now() % 100000) + (isMobile ? 7 : 0))
    await signIn(page, '/', 'catalogue')
    await apiBrand(page, `E2E Supplied A ${run}`)
    await apiBrand(page, `E2E Supplied B ${run}`)

    if (isMobile) await page.goto('/procurement/suppliers')
    else
      await page
        .getByRole('navigation', { name: 'Main' })
        .getByRole('link', { name: /Suppliers/ })
        .click()
    await expect(page.getByRole('heading', { level: 1, name: 'Suppliers' })).toBeVisible()
    await expect(page.getByRole('note').first()).toContainText(
      'not proof of dealership authorisation',
    )

    // Create: client validation, then the server.
    await page.getByRole('button', { name: 'Add supplier' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'Add supplier' })
    await dialog.getByLabel(/Legal name/).fill(name)
    await dialog.getByLabel(/GSTIN/).fill('12345')
    await dialog.getByRole('button', { name: 'Add supplier' }).click()
    await expect(dialog.getByText(/Enter a 15-character GSTIN/)).toBeVisible()
    await dialog.getByLabel(/GSTIN/).fill(gstin.toLowerCase())
    await dialog.getByLabel(/PIN code/).fill('411001')
    await dialog.getByRole('button', { name: 'Add supplier' }).click()
    await expect(page.getByRole('heading', { level: 1, name })).toBeVisible()
    await expect(page.getByText(gstin)).toBeVisible()
    const supplierUrl = page.url()

    // Duplicate legal name (case-insensitive) is refused on the field.
    await page.goto('/procurement/suppliers')
    await page.getByRole('button', { name: 'Add supplier' }).first().click()
    dialog = page.getByRole('dialog', { name: 'Add supplier' })
    await dialog.getByLabel(/Legal name/).fill(name.toUpperCase())
    await dialog.getByRole('button', { name: 'Add supplier' }).click()
    await expect(dialog.getByText('Already used — choose another')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Attach a private document (uploaded in the dialog).
    await page.goto(supplierUrl)
    await page.getByRole('button', { name: 'Attach document' }).click()
    dialog = page.getByRole('dialog', { name: 'Attach document' })
    await dialog.getByLabel(/Document type/).selectOption('AUTHORISATION')
    await dialog
      .locator('input[type="file"]')
      .setInputFiles({ name: 'authorisation.pdf', mimeType: 'application/pdf', buffer: pdf })
    // Visible only: the uploaded file also appears as an <option> of the uploads select.
    await expect(
      dialog
        .getByText(/Uploaded|authorisation\.pdf/)
        .filter({ visible: true })
        .first(),
    ).toBeVisible()
    await dialog.getByRole('button', { name: 'Attach document' }).click()
    await expect(dialog).toHaveCount(0)
    const documents = page.getByRole(isMobile ? 'list' : 'table', { name: 'Documents' })
    await expect(documents.getByText('authorisation.pdf')).toBeVisible()

    // Download through a short-lived link.
    const download = page.waitForEvent('download')
    await documents.getByRole('button', { name: 'Download authorisation.pdf' }).click()
    expect((await download).suggestedFilename()).toBe('authorisation.pdf')

    // Verify the supplier's identity with that document.
    await page.getByRole('button', { name: 'Verify', exact: true }).first().click()
    dialog = page.getByRole('dialog', { name: 'Verify supplier' })
    await dialog.getByLabel(/Evidence document/).selectOption({ index: 1 })
    await dialog.getByLabel(/Reference/).fill('GST certificate checked')
    await dialog.getByRole('button', { name: 'Verify supplier' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('Verified', { exact: true }).first()).toBeVisible()

    // Two brands; verify one relationship with evidence.
    for (const b of [`E2E Supplied A ${run}`, `E2E Supplied B ${run}`]) {
      await page.getByRole('button', { name: 'Add brand' }).click()
      dialog = page.getByRole('dialog', { name: 'Add brand relationship' })
      await dialog.getByLabel(/Brand/).selectOption({ label: b })
      await dialog.getByRole('button', { name: 'Add relationship' }).click()
      await expect(dialog).toHaveCount(0)
    }
    const links = page.getByRole(isMobile ? 'list' : 'table', { name: 'Brand relationships' })
    await expect(links.getByText('Unverified')).toHaveCount(2)
    await links.getByRole('button', { name: 'Verify' }).first().click()
    dialog = page.getByRole('dialog', { name: /Verify relationship with/ })
    await dialog.getByLabel(/Authorisation document/).selectOption({ index: 1 })
    await dialog.getByLabel(/Authorisation reference/).fill('DLR-E2E-1')
    await dialog.getByLabel(/Valid from/).fill(today())
    await dialog.getByRole('button', { name: 'Verify relationship' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(links.getByText('Verified · current')).toBeVisible()

    // The brand page shows the relationship and the readiness check is met.
    const verifiedBrandRow = links
      .getByRole(isMobile ? 'listitem' : 'row')
      .filter({ hasText: 'Verified · current' })
    await verifiedBrandRow.getByRole('link').click()
    await expect(
      page.getByRole('list', { name: 'Supplier relationships' }).getByRole('link', { name }),
    ).toBeVisible()
    await expect(page.getByText('Verified supplier relationship — met')).toBeAttached()
    await page.goto(supplierUrl)

    // Contacts: add, remove (kept as history).
    await page.getByRole('button', { name: 'Add contact' }).click()
    dialog = page.getByRole('dialog', { name: 'Add contact' })
    await dialog.getByLabel(/Name/).fill('E2E Contact')
    await dialog.getByRole('button', { name: 'Add contact' }).click()
    await expect(dialog.getByText('Give a phone number or an email')).toBeVisible()
    await dialog.getByLabel(/Phone/).fill('9000000000')
    await dialog.getByRole('button', { name: 'Add contact' }).click()
    await expect(dialog).toHaveCount(0)
    const contacts = page.getByRole(isMobile ? 'list' : 'table', { name: 'Contacts' })
    await contacts.getByRole('button', { name: 'Remove' }).click()
    await expect(contacts.getByText('Removed')).toBeVisible()

    // End the verified relationship (history kept).
    await links.getByRole('button', { name: 'End' }).first().click()
    dialog = page.getByRole('dialog', { name: /End relationship with/ })
    await dialog.getByLabel(/Reason/).fill('E2E test end')
    await dialog.getByRole('button', { name: 'End relationship' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(links.getByText('Ended', { exact: true })).toBeVisible()

    // A read-only user sees the supplier but cannot change it.
    const other = await browser.newContext({
      baseURL: testInfo.project.use.baseURL,
      viewport: testInfo.project.use.viewport,
      isMobile: testInfo.project.use.isMobile,
      hasTouch: testInfo.project.use.hasTouch,
    })
    const viewer = await other.newPage()
    await signIn(viewer, '/', 'catalogueViewer')
    await viewer.goto(supplierUrl)
    await expect(viewer.getByRole('heading', { level: 1, name })).toBeVisible()
    for (const label of ['Edit', 'Archive', 'Add contact', 'Attach document', 'Add brand']) {
      await expect(viewer.getByRole('button', { name: label, exact: true })).toHaveCount(0)
    }
    await other.close()

    // Archive: confirmation, history stays, no more edits.
    await page.getByRole('button', { name: 'Archive', exact: true }).click()
    dialog = page.getByRole('dialog', { name: 'Archive supplier' })
    await expect(dialog.getByText(/Nothing is deleted/)).toBeVisible()
    await dialog.getByLabel(/Reason/).fill('E2E archive')
    await dialog.getByRole('button', { name: 'Archive supplier' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText(/This supplier is archived/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit', exact: true })).toHaveCount(0)
    await expect(links.getByText('Ended', { exact: true })).toBeVisible()

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    expect(problems).toEqual([])
  })
})
