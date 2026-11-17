import { execFileSync } from 'node:child_process'
import { expect, request, test, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Opening stock RECONCILIATION PREFLIGHT (H-08) against the real API and
 * PostgreSQL: upload a controlled file, open the preflight from the preview,
 * check the integrity result, the categories, the quantities beside the
 * (unrecorded) physical count and the blockers; filter and download the
 * exceptions; deactivate a location after the preview and run again (the row
 * is now reported as changed); and prove with psql that nothing in the stock,
 * catalogue or import tables changed. No approve or commit action exists on the
 * page and only GET requests are sent from it.
 *
 * THROWAWAY DATABASE ONLY. Needs E2E_CATALOGUE_* holding stock.import.preview
 * with two locations in scope, E2E_CATALOGUE_VIEWER_* without import
 * permissions, E2E_OS_BRAND/MODEL/VARIANT naming an ACTIVE catalogue fixture,
 * optionally E2E_OS_PENDING_BRAND/MODEL/VARIANT naming a catalogue entry pending
 * verification, optionally E2E_H08_APPROVER_* holding stock.import.approve and
 * every location (test grants), and E2E_RECON_DB_* for psql. Run with
 * OPENING_STOCK_APPROVED_STATUSES empty (the default). Placeholder identifiers only.
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
const pending = {
  brand: process.env.E2E_OS_PENDING_BRAND ?? '',
  model: process.env.E2E_OS_PENDING_MODEL ?? '',
  variant: process.env.E2E_OS_PENDING_VARIANT ?? '',
}
const approver = {
  username: process.env.E2E_H08_APPROVER_USERNAME ?? '',
  password: process.env.E2E_H08_APPROVER_PASSWORD ?? '',
}

function sql(query: string): string {
  return execFileSync(
    'docker',
    ['exec', db.container, 'psql', '-U', db.user, '-d', db.name, '-Atc', query],
    { encoding: 'utf8' },
  ).trim()
}

/** Every table the preflight must leave unchanged (stock, catalogue, import staging). */
function fingerprint(): string {
  return [
    'vehicle_units',
    'vehicle_unit_events',
    'stock_movements',
    'stock_item_balances',
    'goods_receipts',
    'brands',
    'models',
    'variants',
    'opening_stock_import_batches',
    'opening_stock_import_rows',
    'opening_stock_import_approvals',
    'opening_stock_import_commits',
  ]
    .map(
      (t) =>
        `${t}=${sql(`SELECT count(*) || ':' || coalesce(md5(string_agg(x::text, '|' ORDER BY x::text)), '') FROM ${t} x`)}`,
    )
    .join(' ')
}

const header =
  'brand,model_code,variant_code,colour,chassis_number,motor_serial,battery_serial,charger_serial,location_code,' +
  'stock_status,received_date,provenance,supplier,purchase_order_reference,goods_receipt_reference,' +
  'supplier_invoice_reference,acquisition_cost,notes'

const line = (
  c: { brand: string; model: string; variant: string },
  chassis: string,
  location: string,
  status = 'AVAILABLE',
  cost = '1000',
) =>
  [
    c.brand,
    c.model,
    c.variant,
    '',
    chassis,
    '',
    '',
    '',
    location,
    status,
    '2026-01-15',
    'E2E stock register',
    '',
    '',
    '',
    '',
    cost,
    '',
  ].join(',')

async function upload(page: Page, name: string, lines: string[]) {
  await page.goto('/inventory/opening-stock')
  await page.getByLabel('Choose the CSV file').setInputFiles({
    name,
    mimeType: 'text/csv',
    buffer: Buffer.from(`#template:OS-IMPORT-1\n${header}\n${lines.join('\n')}\n`),
  })
  await page.getByRole('button', { name: 'Upload and validate' }).click()
  await expect(page.getByRole('heading', { level: 1, name: 'Import preview' })).toBeVisible()
  return page.url().split('/').at(-1) ?? ''
}

const tile = (page: Page, label: string) =>
  page.getByLabel('Preflight summary').getByText(label, { exact: true }).locator('xpath=..')

test.describe('opening stock reconciliation preflight (H-08)', () => {
  test('read-only re-check: categories, quantities vs physical count, exceptions, changes since preview', async ({
    page,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') || !fixture.brand,
      'Set E2E_CATALOGUE_* and E2E_OS_* (throwaway fixture)',
    )
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/opening-stock', 'catalogue')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { locationScope: { locations: { code: string }[] } }
    }
    const [locA, locB] = me.data.locationScope.locations.map((l) => l.code)
    expect(locA && locB, 'E2E_CATALOGUE needs two locations in scope').toBeTruthy()
    const a = locA ?? ''
    const b = locB ?? ''

    const lines = [
      line(fixture, `PF-${tag}-1`, a), // 3 valid
      line(fixture, `PF-${tag}-2`, b, 'DISPLAY'), // 4 valid (location deactivated later)
      line(fixture, `PF-${tag}-3`, a, 'AVAILABLE', ''), // 5 warning: no cost
      line(fixture, `PF-${tag}-D`, a), // 6 duplicate in file
      line(fixture, `pf-${tag}-d`, a), // 7 duplicate in file
      line(fixture, '', a), // 8 missing chassis
    ]
    if (pending.brand) lines.push(line(pending, `PF-${tag}-P`, a)) // 9 unverified catalogue
    const id = await upload(page, `preflight-${tag}.csv`, lines)

    const methods: string[] = []
    page.on('request', (r) => {
      if (r.url().includes('/api/v1/opening-stock-imports')) methods.push(r.method())
    })
    const before = fingerprint()
    await page.getByRole('link', { name: 'Run the reconciliation preflight' }).click()
    await expect(
      page.getByRole('heading', { level: 1, name: 'Reconciliation preflight' }),
    ).toBeVisible()
    await expect(page.getByText(/No inventory records were created or changed/)).toBeVisible()
    await expect(page.getByText('Verified', { exact: true })).toBeVisible()
    await expect(page.getByRole('list', { name: 'Source file checks' })).toContainText(
      'SHA-256 matches the upload: passed',
    )
    const total = pending.brand ? '7' : '6'
    await expect(tile(page, 'Total records')).toContainText(total)
    await expect(tile(page, 'Valid')).toContainText('2')
    await expect(tile(page, 'Warnings')).toContainText('1')
    await expect(tile(page, 'Blocking errors')).toContainText('3')
    await expect(tile(page, 'Unverified catalogue')).toContainText(pending.brand ? '1' : '0')
    await expect(tile(page, 'Missing chassis')).toContainText('1')
    const blockers = page.getByRole('list', { name: 'Blockers' })
    await expect(blockers).toContainText('Pending: OQ-47')
    await expect(blockers).toContainText('Pending: OQ-49')
    await expect(blockers).toContainText('Pending: H-08')
    await expect(page.getByText('Not recorded').first()).toBeVisible()

    // Exceptions: filter and download (built from the server's answer).
    await page.getByLabel('Category').selectOption('BLOCKING_ERROR')
    const exceptions = page
      .getByRole('table', { name: 'Records needing attention' })
      .or(page.getByRole('list', { name: 'Records needing attention' }))
      .filter({ visible: true })
    await expect(exceptions).toContainText('Line 6')
    await expect(exceptions).toContainText('Line 8')
    await expect(exceptions).not.toContainText('Line 5')
    const download = page.waitForEvent('download')
    await page.getByRole('button', { name: /Download exceptions/ }).click()
    const file = await download
    expect(file.suggestedFilename()).toMatch(/^OSI-\d+-preflight-exceptions\.csv$/)
    await page.screenshot({
      path: info.outputPath(`preflight-${info.project.name}.png`),
      fullPage: true,
    })
    expect(fingerprint()).toBe(before)

    // After the preview, location B is closed: the preflight reports line 4 as changed.
    sql(`UPDATE locations SET active = false WHERE code = '${b}'`)
    try {
      await page.getByRole('button', { name: 'Run again' }).click()
      await expect(tile(page, 'Changed since preview')).toContainText('1')
      await expect(tile(page, 'Valid')).toContainText('1')
      await page.getByLabel('Category').selectOption('BLOCKING_ERROR')
      await expect(exceptions).toContainText('Line 4')
      await expect(exceptions).toContainText(/Valid → Error/)
    } finally {
      sql(`UPDATE locations SET active = true WHERE code = '${b}'`)
    }
    // The staged preview itself is unchanged: line 4 is still VALID there.
    const staged = (await (
      await page.request.get(`/api/v1/opening-stock-imports/${id}/rows?status=VALID&limit=50`)
    ).json()) as { data: { items: { rowNumber: number }[] } }
    expect(staged.data.items.map((r) => r.rowNumber)).toContain(4)

    expect(fingerprint()).toBe(before)
    expect(methods.every((m) => m === 'GET')).toBe(true)
    await expect(page.getByRole('button', { name: /approve|commit/i })).toHaveCount(0)
    await expect(page.getByRole('link', { name: /approve|commit/i })).toHaveCount(0)
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    expect(problems).toEqual([])
  })

  test('after a recorded approval the signed count reference is shown, still no quantities', async ({
    page,
    baseURL,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') || !fixture.brand || !approver.username,
      'Set E2E_CATALOGUE_*, E2E_OS_* and E2E_H08_APPROVER_* (throwaway fixture)',
    )
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/opening-stock', 'catalogue')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { locationScope: { locations: { code: string }[] } }
    }
    const loc = me.data.locationScope.locations.at(0)?.code ?? ''
    const id = await upload(page, `clean-${tag}.csv`, [
      line(fixture, `PA-${tag}-1`, loc),
      line(fixture, `PA-${tag}-2`, loc),
    ])
    // The approval is the separate G-07 step, recorded by another (test) user.
    const ctx = await request.newContext({ baseURL })
    const login = await ctx.post('/api/v1/auth/login', { data: approver })
    expect(login.status()).toBe(200)
    const csrf = ((await login.json()) as { data: { csrfToken: string } }).data.csrfToken
    const ap = await ctx.post(`/api/v1/opening-stock-imports/${id}/approval`, {
      headers: { 'X-CSRF-Token': csrf },
      data: {
        physicalCountReference: `PC-E2E-${tag}`,
        physicalCountDate: '2026-11-16',
        mappingsReviewed: true,
        duplicatesReviewed: true,
        receivedDatesApproved: false,
      },
    })
    expect(ap.status()).toBe(201)
    await ctx.dispose()
    const before = fingerprint()

    await page.goto(`/inventory/opening-stock/${id}/preflight`)
    await expect(page.getByRole('note')).toContainText(`signed count PC-E2E-${tag} of 2026-11-16`)
    await expect(page.getByText('No exceptions')).toBeVisible()
    const blockers = page.getByRole('list', { name: 'Blockers' })
    await expect(blockers).not.toContainText('No physical-count approval')
    await expect(blockers).toContainText('Pending: OQ-47')
    await expect(blockers).toContainText('Pending: H-08')
    expect(fingerprint()).toBe(before)
    expect(sql(`SELECT count(*) FROM vehicle_units WHERE source_type = 'OPENING_STOCK'`)).toBe('0')
  })

  test('a user without an import permission is refused', async ({ page }) => {
    test.skip(!hasAccount('catalogueViewer'), 'Set E2E_CATALOGUE_VIEWER_*')
    await signIn(
      page,
      '/inventory/opening-stock/00000000-0000-0000-0000-000000000000/preflight',
      'catalogueViewer',
    )
    await expect(page.getByText('Access denied')).toBeVisible()
    expect(
      (
        await page.request.get(
          '/api/v1/opening-stock-imports/00000000-0000-0000-0000-000000000000/preflight',
        )
      ).status(),
    ).toBe(403)
  })
})
