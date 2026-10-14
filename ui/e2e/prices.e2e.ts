import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Price history screens (C-04) against the real API, with two people: the
 * drafter (E2E_CATALOGUE) and a different approver (E2E_PRICE_APPROVER).
 * Placeholder amounts and names only; throwaway database only (PROPOSED grants).
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

/** Creates brand → verified, active model → verified, active variant through the API. */
async function catalogue(page: Page, run: string) {
  const token = await csrf(page.request)
  const send = async <T>(method: 'POST', path: string, data: unknown): Promise<T> => {
    const res = await page.request.fetch(`/api/v1${path}`, {
      method,
      headers: { 'X-CSRF-Token': token },
      data,
    })
    expect(res.ok(), `${path}: ${await res.text()}`).toBe(true)
    return ((await res.json()) as { data: T }).data
  }
  type Rec = { id: string; version: number }
  const source = {
    confidence: 'SUPPLIER_CONFIRMED',
    sourceUrl: 'https://example.com/e2e',
    sourceDate: '2026-10-01',
  }
  const brand = await send<Rec>('POST', '/brands', { name: `E2E Price Brand ${run}` })
  let model = await send<Rec>('POST', '/models', {
    brandId: brand.id,
    name: 'E2E Model',
    internalCode: 'PM1',
  })
  model = await send<Rec>('POST', `/models/${model.id}/verify`, {
    version: model.version,
    ...source,
  })
  await send<Rec>('POST', `/models/${model.id}/activate`, { version: model.version })
  let variant = await send<Rec>('POST', '/variants', {
    modelId: model.id,
    name: 'E2E Variant',
    variantCode: 'STD',
  })
  variant = await send<Rec>('POST', `/variants/${variant.id}/verify`, {
    version: variant.version,
    ...source,
  })
  await send<Rec>('POST', `/variants/${variant.id}/activate`, { version: variant.version })
  return { brandId: brand.id, variantId: variant.id }
}

/** datetime-local value for now + offset (local time). */
function localInput(offsetMs: number): string {
  const d = new Date(Date.now() + offsetMs)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const HOUR = 3_600_000
const DAY = 24 * HOUR

async function priceTable(page: Page, isMobile: boolean) {
  const t = page.getByRole(isMobile ? 'list' : 'table', { name: 'Price history' })
  await expect(t).toBeVisible()
  return t
}

async function lookUp(page: Page, at: string) {
  await page.getByLabel(/Price valid at/).fill(at)
  await page.getByRole('button', { name: /Look up/ }).click()
}

test.describe('price history (C-04)', () => {
  test('draft, approve by someone else, supersede, and historical lookups stay stable', async ({
    page,
    browser,
    isMobile,
  }, testInfo) => {
    test.skip(
      !hasAccount('catalogue') || !hasAccount('priceApprover'),
      'needs E2E_CATALOGUE_* (with price draft grants) and E2E_PRICE_APPROVER_* (throwaway database only)',
    )
    test.setTimeout(150_000)
    const problems = watchConsole(page)
    const run = `${testInfo.project.name} ${Date.now().toString(36)}`
    await signIn(page, '/', 'catalogue')
    const { brandId, variantId } = await catalogue(page, run)

    // Drafter: create a price list on the brand page.
    await page.goto(`/admin/catalogue/brands/${brandId}`)
    await page.getByRole('button', { name: 'Add price list' }).click()
    let dialog = page.getByRole('dialog', { name: 'Add price list' })
    await dialog.getByLabel(/Name/).fill('Showroom')
    await dialog.getByRole('button', { name: 'Add price list' }).click()
    await expect(dialog).toHaveCount(0)

    // Draft a selling price starting in two hours.
    await page.goto(`/admin/catalogue/variants/${variantId}`)
    await expect(page.getByText(/never changes existing quotations/).first()).toBeVisible()
    await page.getByRole('button', { name: 'Draft price' }).click()
    dialog = page.getByRole('dialog', { name: 'Draft a selling price' })
    await dialog.getByLabel(/Amount/).fill('1000')
    const start1 = localInput(2 * HOUR)
    await dialog.getByLabel(/Effective from/).fill(start1)
    await dialog.getByRole('button', { name: 'Save as draft' }).click()
    await expect(dialog).toHaveCount(0)
    let table = await priceTable(page, isMobile)
    await expect(table.getByText('Draft — not approved')).toBeVisible()
    // The drafter cannot approve (no permission, and never their own draft).
    await expect(table.getByRole('button', { name: 'Approve' })).toHaveCount(0)

    // Approver (another person) approves.
    const u = testInfo.project.use
    const approverCtx = await browser.newContext({
      baseURL: u.baseURL,
      viewport: u.viewport,
      isMobile: u.isMobile,
      hasTouch: u.hasTouch,
      deviceScaleFactor: u.deviceScaleFactor,
      userAgent: u.userAgent,
    })
    const approver = await approverCtx.newPage()
    const approverProblems = watchConsole(approver)
    await signIn(approver, '/', 'priceApprover')
    await approver.goto(`/admin/catalogue/variants/${variantId}`)
    let atable = await priceTable(approver, isMobile)
    await atable.getByRole('button', { name: 'Approve' }).click()
    dialog = approver.getByRole('dialog', { name: 'Approve price' })
    await expect(dialog.getByText(/never changes existing quotations/)).toBeVisible()
    await dialog.getByLabel(/Approval reference/).fill('E2E-CIRCULAR-1')
    await dialog.getByRole('button', { name: 'Approve price' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(atable.getByText('Future')).toBeVisible()
    await expect(atable.getByText('Ref: E2E-CIRCULAR-1')).toBeVisible()

    // Drafter: a second price overlapping without supersession is refused at approval.
    await page.reload()
    await page.getByRole('button', { name: 'Draft price' }).click()
    dialog = page.getByRole('dialog', { name: 'Draft a selling price' })
    await dialog.getByLabel(/Amount/).fill('999')
    await dialog.getByLabel(/Effective from/).fill(localInput(3 * HOUR))
    await dialog.getByRole('button', { name: 'Save as draft' }).click()
    await expect(dialog).toHaveCount(0)
    await approver.reload()
    atable = await priceTable(approver, isMobile)
    await atable.getByRole('button', { name: 'Approve' }).click()
    dialog = approver.getByRole('dialog', { name: 'Approve price' })
    await dialog.getByRole('button', { name: 'Approve price' }).click()
    await expect(dialog.getByText(/already covers part of this period/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Drafter cancels that draft and drafts a controlled change from tomorrow.
    await page.reload()
    table = await priceTable(page, isMobile)
    await table.getByRole('button', { name: 'Cancel' }).click()
    dialog = page.getByRole('dialog', { name: 'Cancel draft price' })
    await dialog.getByRole('button', { name: 'Cancel draft' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(table.getByText('Cancelled draft')).toBeVisible()
    await table.getByRole('button', { name: /Change price/ }).click()
    dialog = page.getByRole('dialog', { name: 'Draft a replacement price' })
    await expect(dialog.getByText(/Replaces the approved price of ₹1,000.00/)).toBeVisible()
    await dialog.getByLabel(/Amount/).fill('1200')
    await dialog.getByLabel(/Effective from/).fill(localInput(DAY))
    await dialog.getByRole('button', { name: 'Save as draft' }).click()
    await expect(dialog).toHaveCount(0)

    await approver.reload()
    atable = await priceTable(approver, isMobile)
    await atable.getByRole('button', { name: 'Approve' }).click()
    dialog = approver.getByRole('dialog', { name: 'Approve price' })
    await expect(dialog.getByText(/price it replaces will end/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Approve price' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(atable.getByText('Superseded')).toBeVisible()

    // Lookups: the old instant still answers ₹1,000 (old quotations stay stable); later ₹1,200.
    await lookUp(approver, localInput(3 * HOUR))
    await expect(approver.getByText(/₹1,000.00 — approved price valid from/)).toBeVisible()
    await lookUp(approver, localInput(2 * DAY))
    await expect(approver.getByText(/₹1,200.00 — approved price valid from/)).toBeVisible()
    await lookUp(approver, localInput(-DAY))
    await expect(approver.getByText('No approved price is valid at that time.')).toBeVisible()

    // No retroactive approval: a draft starting yesterday is refused.
    await page.reload()
    await page.getByRole('button', { name: 'Draft price' }).click()
    dialog = page.getByRole('dialog', { name: 'Draft a selling price' })
    await dialog.getByLabel(/Amount/).fill('500')
    await dialog.getByLabel(/Effective from/).fill(localInput(-DAY))
    await dialog.getByLabel(/Effective until/).fill(localInput(-HOUR))
    await dialog.getByRole('button', { name: 'Save as draft' }).click()
    await expect(dialog).toHaveCount(0)
    await approver.reload()
    atable = await priceTable(approver, isMobile)
    await atable.getByRole('button', { name: 'Approve' }).click()
    dialog = approver.getByRole('dialog', { name: 'Approve price' })
    await dialog.getByRole('button', { name: 'Approve price' }).click()
    await expect(dialog.getByText(/effectiveFrom is in the past/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    // Cost prices: the drafter (cost viewer) has a type switch; the approver (no cost permission) does not.
    await page.reload()
    await expect(page.getByLabel('Price type')).toBeVisible()
    await approver.reload()
    await priceTable(approver, isMobile)
    await expect(approver.getByLabel('Price type')).toHaveCount(0)

    const overflow = await approver.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    await approverCtx.close()
    expect(problems).toEqual([])
    expect(approverProblems).toEqual([])
  })
})
