import { execFileSync } from 'node:child_process'
import { expect, test } from '@playwright/test'
import { hasReceiptFixture, registerUnitThroughReceipt } from './receivedUnit'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Opening stock import PREVIEW (G-06) against the real API and PostgreSQL:
 * template download, a refused XLSX, then a controlled CSV mixing a valid row,
 * a chassis already registered, an in-file duplicate, a launch brand that is
 * pending verification, an unknown brand, an out-of-scope location and a future
 * date. The preview must show every result by line and field, offer no commit,
 * and leave vehicle units, stock movements and balances unchanged (psql counts).
 *
 * THROWAWAY DATABASE ONLY. Needs E2E_CATALOGUE_* with stock.import.preview and
 * vehicle_units.view and one location in scope; E2E_OS_BRAND / E2E_OS_MODEL /
 * E2E_OS_VARIANT naming an ACTIVE catalogue entry (fixture) and E2E_RECON_DB_*.
 * The "already registered" chassis is the test's own unit, registered through a
 * goods receipt posting (./receivedUnit.ts): E2E_CATALOGUE also needs the D-07
 * grants, E2E_PRICE_APPROVER_* approves its PO and E2E_PO_* names the eligible
 * fixture. The row counts are taken after that unit exists.
 * for the row counts. Placeholder identifiers only.
 */

const db = {
  container: process.env.E2E_RECON_DB_CONTAINER ?? '',
  user: process.env.E2E_RECON_DB_USER ?? '',
  name: process.env.E2E_RECON_DB_NAME ?? '',
}
const fixture = {
  brand: process.env.E2E_OS_BRAND ?? '',
  model: process.env.E2E_OS_MODEL ?? '',
  variant: process.env.E2E_OS_VARIANT ?? '',
}

function businessCounts(): string {
  return execFileSync(
    'docker',
    [
      'exec',
      db.container,
      'psql',
      '-U',
      db.user,
      '-d',
      db.name,
      '-Atc',
      `SELECT (SELECT count(*) FROM vehicle_units) || '/' || (SELECT count(*) FROM stock_movements) || '/' ||
        (SELECT count(*) FROM stock_item_balances) || '/' || (SELECT count(*) FROM goods_receipts) || '/' ||
        (SELECT md5(string_agg(t::text, '|' ORDER BY t::text)) FROM vehicle_units t)`,
    ],
    { encoding: 'utf8' },
  ).trim()
}

const header =
  'brand,model_code,variant_code,colour,chassis_number,motor_serial,battery_serial,charger_serial,location_code,' +
  'stock_status,received_date,provenance,supplier,purchase_order_reference,goods_receipt_reference,' +
  'supplier_invoice_reference,acquisition_cost,notes'

function line(
  brand: string,
  model: string,
  variant: string,
  chassis: string,
  location: string,
  date: string,
) {
  return [
    brand,
    model,
    variant,
    '',
    chassis,
    '',
    '',
    '',
    location,
    'AVAILABLE',
    date,
    'E2E stock register',
    '',
    '',
    '',
    '',
    '1000',
    '',
  ].join(',')
}

test.describe('opening stock import preview', () => {
  test('template, refused xlsx, row-level preview, no commit, no stock change', async ({
    page,
    browser,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') || !fixture.brand,
      'Set E2E_CATALOGUE_* and E2E_OS_* (throwaway fixture)',
    )
    test.skip(
      !hasAccount('priceApprover') || !hasReceiptFixture(),
      'Set E2E_PRICE_APPROVER_* and E2E_PO_* (throwaway fixture for the registered unit)',
    )
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/opening-stock', 'catalogue')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { locationScope: { locations: { code: string }[] } }
    }
    const loc = me.data.locationScope.locations.at(0)?.code ?? ''
    expect(loc, 'E2E_CATALOGUE needs a location in scope').not.toBe('')
    // A registered vehicle unit owned by this test (D-07 posting), for line 4.
    const taken = `OS-${tag}-REG`
    const approverContext = await browser.newContext()
    const approver = await approverContext.newPage()
    await signIn(approver, '/', 'priceApprover')
    await registerUnitThroughReceipt(page, approver, tag, taken)
    await approverContext.close()
    await page.goto('/inventory/opening-stock')
    const before = businessCounts()

    // Step 1: template.
    await expect(
      page.getByRole('heading', { level: 1, name: 'Opening Stock Import' }),
    ).toBeVisible()
    await expect(page.getByText(/OS-IMPORT-1/).first()).toBeVisible()
    const download = page.waitForEvent('download')
    await page.getByRole('button', { name: /Download template/ }).click()
    const tpl = await download
    expect(tpl.suggestedFilename()).toBe('opening-stock-template-OS-IMPORT-1.csv')

    // Step 2: an XLSX is refused before sending (and would be refused by the server).
    const input = page.getByLabel('Choose the CSV file')
    await input.setInputFiles({
      name: 'stock.xlsx',
      mimeType: 'application/zip',
      buffer: Buffer.from('PK\x03\x04xx'),
    })
    await expect(page.getByRole('alert').filter({ hasText: 'CSV UTF-8' })).toBeVisible()
    const xlsx = await page.request.post('/api/v1/opening-stock-imports', {
      headers: {
        'X-CSRF-Token': (
          (await (await page.request.get('/api/v1/auth/session')).json()) as {
            data: { csrfToken: string }
          }
        ).data.csrfToken,
        'Idempotency-Key': `e2e-xlsx-${tag}`,
      },
      multipart: {
        file: {
          name: 'stock.xlsx',
          mimeType: 'application/zip',
          buffer: Buffer.from('PK\x03\x04xx'),
        },
      },
    })
    expect(xlsx.status()).toBe(415)

    // Steps 3-5: a controlled CSV.
    const csv = [
      '#template:OS-IMPORT-1',
      header,
      line(fixture.brand, fixture.model, fixture.variant, `OS-${tag}-1`, loc, '2026-01-15'), // 3 valid
      line(fixture.brand, fixture.model, fixture.variant, taken, loc, '2026-01-15'), // 4 already registered
      line(fixture.brand, fixture.model, fixture.variant, `OS-${tag}-TWIN`, loc, '2026-01-15'), // 5 twin
      line(fixture.brand, fixture.model, fixture.variant, `os-${tag}-twin`, loc, '2026-01-15'), // 6 twin
      line('Lectrix EV', 'X1', 'STD', `OS-${tag}-7`, loc, '2026-01-15'), // 7 pending launch brand
      line('Brand Unknown', 'X1', 'STD', `OS-${tag}-8`, loc, '2026-01-15'), // 8 unknown brand
      line(fixture.brand, fixture.model, fixture.variant, `OS-${tag}-9`, 'NOWHERE', '2026-01-15'), // 9 unknown location
      line(fixture.brand, fixture.model, fixture.variant, `OS-${tag}-10`, loc, '2999-01-01'), // 10 future date
      '',
    ].join('\n')
    await input.setInputFiles({
      name: `opening-${tag}.csv`,
      mimeType: 'text/csv',
      buffer: Buffer.from(csv),
    })
    await page.getByRole('button', { name: 'Upload and validate' }).click()
    await expect(page).toHaveURL(/\/inventory\/opening-stock\/[0-9a-f-]{36}$/)
    await expect(
      page.getByText('Preview completed. No inventory records have been created or modified.'),
    ).toBeVisible()
    const summary = page.getByLabel('Preview summary')
    await expect(summary.getByText('Total rows').locator('..')).toContainText('8')
    await expect(summary.getByText('Valid', { exact: true }).locator('..')).toContainText('1')
    await expect(summary.getByText('Duplicate identifiers').locator('..')).toContainText('3')
    await expect(page.getByRole('button', { name: /commit/i })).toHaveCount(0)

    const rowFor = (n: number) =>
      page
        .getByRole('row')
        .or(page.getByRole('listitem'))
        .filter({ visible: true })
        .filter({ has: page.getByText(String(n), { exact: true }) })
        .first()
    await expect(rowFor(3)).toContainText('Valid')
    await expect(rowFor(4)).toContainText('already registered to a vehicle unit')
    await expect(rowFor(5)).toContainText('also appears on line(s) 6')
    await expect(rowFor(7)).toContainText('exists but is not verified and activated')
    await expect(rowFor(8)).toContainText('No brand named "Brand Unknown"')
    await expect(rowFor(9)).toContainText('No location has code "NOWHERE"')
    await expect(rowFor(10)).toContainText("after today's business date")

    // Filters (server-side) and row details.
    await page.getByLabel('Duplicates').selectOption('true')
    await expect(page).toHaveURL(/duplicate=true/)
    await expect(rowFor(3)).toHaveCount(0)
    await expect(rowFor(5)).toBeVisible()
    await page.getByLabel('Duplicates').selectOption('')
    await page.getByLabel('Catalogue mapping').selectOption('true')
    await expect(rowFor(7)).toBeVisible()
    await expect(rowFor(3)).toHaveCount(0)
    await page
      .getByRole('button', { name: 'Details of line 7' })
      .filter({ visible: true })
      .first()
      .click()
    const dialog = page.getByRole('dialog', { name: 'Line 7' })
    await expect(dialog.getByRole('cell', { name: 'Lectrix EV' })).toBeVisible()
    await page.keyboard.press('Escape')
    await page.screenshot({
      path: info.outputPath(`opening-stock-preview-${info.project.name}.png`),
      fullPage: true,
    })

    // Nothing in the stock or receipt tables changed.
    expect(businessCounts()).toBe(before)
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    expect(problems).toEqual([])
  })

  test('a user without the permission is refused', async ({ page }) => {
    test.skip(!hasAccount('catalogueViewer'), 'Set E2E_CATALOGUE_VIEWER_*')
    await signIn(page, '/inventory/opening-stock', 'catalogueViewer')
    await expect(page.getByText(/do not have permission/i)).toBeVisible()
    expect((await page.request.get('/api/v1/opening-stock-imports/template')).status()).toBe(403)
  })
})
