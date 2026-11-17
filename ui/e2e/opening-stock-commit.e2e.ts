import { execFileSync } from 'node:child_process'
import { expect, test, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Opening stock REVIEW → APPROVAL → COMMIT (G-07) against the real API and
 * PostgreSQL: upload a controlled file, approve it with typed confirmations,
 * commit it while the first response is lost (the page must ask the server and
 * show the commit instead of sending again), reload, and check in the database
 * that exactly one unit and one OPENING_BALANCE movement exist per line. A line
 * with a status the test configuration does not approve is refused with nothing
 * created.
 *
 * THROWAWAY DATABASE ONLY: needs E2E_CATALOGUE_* holding stock.import.preview,
 * stock.import.approve, stock.import.commit and vehicle_units.view with one
 * location in scope (test grants), E2E_CATALOGUE_VIEWER_* without them,
 * E2E_OS_BRAND/MODEL/VARIANT naming an ACTIVE catalogue fixture, E2E_RECON_DB_*
 * for the counts, and an API started with OPENING_STOCK_APPROVED_STATUSES
 * containing AVAILABLE but not QUALITY_HOLD (test configuration, not an owner
 * decision). Placeholder identifiers only.
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

function sql(query: string): string {
  return execFileSync(
    'docker',
    ['exec', db.container, 'psql', '-U', db.user, '-d', db.name, '-Atc', query],
    {
      encoding: 'utf8',
    },
  ).trim()
}

const header =
  'brand,model_code,variant_code,colour,chassis_number,motor_serial,battery_serial,charger_serial,location_code,' +
  'stock_status,received_date,provenance,supplier,purchase_order_reference,goods_receipt_reference,' +
  'supplier_invoice_reference,acquisition_cost,notes'

const line = (chassis: string, location: string, status: string) =>
  [
    fixture.brand,
    fixture.model,
    fixture.variant,
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
    '1000',
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
  await expect(
    page.getByText('Preview completed. No inventory records have been created or modified.'),
  ).toBeVisible()
  // The preview itself never commits.
  await expect(page.getByRole('button', { name: /commit/i })).toHaveCount(0)
  await page.getByRole('link', { name: 'Review, approve and commit' }).click()
  await expect(page.getByRole('heading', { level: 1, name: 'Review and commit' })).toBeVisible()
  return new URL(page.url()).pathname.split('/').at(-2) ?? ''
}

async function approve(page: Page, reference: string) {
  await page.getByLabel(/Physical count reference/).fill(reference)
  await page.getByLabel(/Physical count date/).fill('2026-11-15')
  await page.getByRole('checkbox', { name: /reviewed the brand, model/ }).check()
  await page.getByRole('checkbox', { name: /reviewed the duplicate/ }).check()
  await page.getByRole('radio', { name: /^Yes/ }).check()
  await page.getByRole('button', { name: 'Review the approval…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Record this approval?' })
  await expect(dialog.getByText(reference)).toBeVisible()
  await dialog.getByRole('button', { name: 'Record approval' }).click()
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible()
  await expect(page.getByText(reference)).toBeVisible()
}

async function openCommit(page: Page) {
  await page.getByRole('button', { name: 'Commit this batch…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Commit this batch into stock?' })
  await dialog.getByRole('checkbox').check()
  return dialog
}

test.describe('opening stock commit (G-07)', () => {
  test.beforeEach(() => {
    test.skip(
      !hasAccount('catalogue') || !fixture.brand,
      'Set E2E_CATALOGUE_* and E2E_OS_* (throwaway fixture)',
    )
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
  })

  test('approve, commit once through a lost response, reload, provenance in the database', async ({
    page,
  }, info) => {
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/opening-stock', 'catalogue')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { locationScope: { locations: { code: string }[] } }
    }
    const loc = me.data.locationScope.locations.at(0)?.code ?? ''
    const id = await upload(page, `commit-${tag}.csv`, [
      line(`G7-${tag}-1`, loc, 'AVAILABLE'),
      line(`G7-${tag}-2`, loc, 'RECEIVED_PENDING_PDI'),
    ])
    expect(id).toMatch(/^[0-9a-f-]{36}$/)
    await expect(page.getByText(id)).toBeVisible()
    // Nothing is preselected; the commit waits for the approval.
    await expect(
      page.getByRole('checkbox', { name: /reviewed the brand, model/ }),
    ).not.toBeChecked()
    await expect(page.getByRole('button', { name: 'Commit this batch…' })).toBeDisabled()
    await approve(page, `PC-E2E-${tag}`)

    const before = sql(`SELECT count(*) FROM vehicle_units WHERE opening_stock_batch_id = '${id}'`)
    expect(before).toBe('0')
    // The first commit response is lost after the server committed.
    let posts = 0
    await page.route(`**/api/v1/opening-stock-imports/${id}/commit`, async (route) => {
      if (route.request().method() !== 'POST') return route.continue()
      posts++
      await route.fetch()
      await route.abort('connectionreset')
    })
    const dialog = await openCommit(page)
    await dialog.getByRole('button', { name: /Commit 2 line/ }).click()
    await expect(page.getByRole('heading', { name: /Committed · OSC-/ })).toBeVisible()
    expect(posts).toBe(1)
    await page.unroute(`**/api/v1/opening-stock-imports/${id}/commit`)
    await expect(page.getByText(/2 vehicle unit\(s\) were created/)).toBeVisible()
    await expect(page.getByRole('link', { name: /^MAE-VEH-/ })).toHaveCount(2)

    // Reload: the server's commit is shown again; nothing is offered.
    await page.reload()
    await expect(page.getByRole('heading', { name: /Committed · OSC-/ })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Commit this batch…' })).toHaveCount(0)

    // Database: once, with provenance.
    expect(sql(`SELECT count(*) FROM opening_stock_import_commits WHERE batch_id = '${id}'`)).toBe(
      '1',
    )
    expect(
      sql(`SELECT count(*) FROM vehicle_units v JOIN stock_movements m ON m.vehicle_unit_id = v.id
        WHERE v.opening_stock_batch_id = '${id}' AND v.source_type = 'OPENING_STOCK'
          AND m.movement_type = 'OPENING_BALANCE' AND m.source_id = '${id}'`),
    ).toBe('2')

    // The vehicle shows where it came from.
    await page
      .getByRole('link', { name: /^MAE-VEH-/ })
      .first()
      .click()
    await expect(page.getByText(/^OSI-\d+ · line 3$/)).toBeVisible()
    const width = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    )
    expect(width).toBeLessThanOrEqual(1)
    expect(problems).toEqual([])
  })

  test('a line with an unapproved status is refused and nothing is created', async ({
    page,
  }, info) => {
    test.setTimeout(120_000)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/inventory/opening-stock', 'catalogue')
    const me = (await (await page.request.get('/api/v1/me')).json()) as {
      data: { locationScope: { locations: { code: string }[] } }
    }
    const loc = me.data.locationScope.locations.at(0)?.code ?? ''
    const id = await upload(page, `hold-${tag}.csv`, [line(`G7-${tag}-H`, loc, 'QUALITY_HOLD')])
    await approve(page, `PC-E2E-${tag}`)
    const units = sql('SELECT count(*) FROM vehicle_units')
    const dialog = await openCommit(page)
    await dialog.getByRole('button', { name: /Commit 1 line/ }).click()
    await expect(page.getByText('Nothing was committed.')).toBeVisible()
    await expect(page.getByText(/STATUS_NOT_APPROVED/)).toBeVisible()
    expect(sql('SELECT count(*) FROM vehicle_units')).toBe(units)
    expect(sql(`SELECT count(*) FROM opening_stock_import_commits WHERE batch_id = '${id}'`)).toBe(
      '0',
    )
  })

  test('a user without the import permissions is refused', async ({ page }) => {
    test.skip(!hasAccount('catalogueViewer'), 'Set E2E_CATALOGUE_VIEWER_*')
    await signIn(
      page,
      '/inventory/opening-stock/00000000-0000-0000-0000-000000000000/commit',
      'catalogueViewer',
    )
    await expect(page.getByText(/do not have permission/i)).toBeVisible()
  })
})
