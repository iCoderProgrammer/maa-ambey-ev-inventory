import { execFileSync } from 'node:child_process'
import { expect, test, type APIRequestContext, type Page, type Request } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Stock ledger reconciliation (E-06) against the real API and PostgreSQL:
 * consistent stock reconciles; an injected drift is reported with the ledger's
 * expected and the record's actual value; a rerun gives the same finding; the
 * run changes nothing; a user without the report permission is refused.
 *
 * THROWAWAY DATABASE ONLY. The drift is injected with psql inside the database
 * container (E2E_RECON_DB_CONTAINER, E2E_RECON_DB_USER, E2E_RECON_DB_NAME) by
 * switching the balance table's user triggers off for one statement — the only
 * way a record can disagree with the ledger — and is undone at the end.
 * PROPOSED grants: E2E_CATALOGUE also needs stock.reconciliation.view,
 * vehicle_units.view, stock.parts.view/receive, stock_items.view/manage and
 * suppliers.view/manage, plus a location in its scope; E2E_CATALOGUE_VIEWER must
 * not hold stock.reconciliation.view. Placeholder data only.
 */

const db = {
  container: process.env.E2E_RECON_DB_CONTAINER ?? '',
  user: process.env.E2E_RECON_DB_USER ?? '',
  name: process.env.E2E_RECON_DB_NAME ?? '',
}
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

function sql(statement: string): string {
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
      '-v',
      'ON_ERROR_STOP=1',
      '-Atc',
      statement,
    ],
    { encoding: 'utf8' },
  ).trim()
}

/** Sets a balance without a movement (test-only drift). IDs come from the API and are checked. */
function setBalance(itemId: string, locationId: string, quantity: number) {
  if (!uuid.test(itemId) || !uuid.test(locationId) || !Number.isInteger(quantity))
    throw new Error('bad input')
  sql(`BEGIN; ALTER TABLE stock_item_balances DISABLE TRIGGER USER;
    UPDATE stock_item_balances SET quantity = ${quantity} WHERE stock_item_id = '${itemId}' AND location_id = '${locationId}';
    ALTER TABLE stock_item_balances ENABLE TRIGGER USER; COMMIT;`)
}

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function post(page: Page, path: string, data: object, key?: string): Promise<string> {
  const headers: Record<string, string> = { 'X-CSRF-Token': await csrf(page.request) }
  if (key) headers['Idempotency-Key'] = key
  const res = await page.request.post(`/api/v1${path}`, { headers, data })
  expect(res.ok(), await res.text()).toBe(true)
  return ((await res.json()) as { data: { id: string } }).data.id
}

async function ledgerState(page: Page, itemId: string) {
  const b = await page.request.get(`/api/v1/stock-items/${itemId}/balances`)
  const m = await page.request.get(`/api/v1/stock-items/${itemId}/stock-movements`)
  return {
    balances: JSON.stringify(((await b.json()) as { data: unknown }).data),
    movements: ((await m.json()) as { data: { total: number } }).data.total,
  }
}

test.describe('stock ledger reconciliation', () => {
  test('consistent stock reconciles, injected drift is reported, reruns repeat, nothing changes', async ({
    page,
  }, info) => {
    test.skip(!hasAccount('catalogue'), 'Set E2E_CATALOGUE_* (see DEVELOPMENT_SETUP)')
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
    test.setTimeout(120_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/', 'catalogue')

    // Disposable stock through the real API: a SKU with 10 received at the user's location.
    const meRes = await page.request.get('/api/v1/me')
    const me = (await meRes.json()) as {
      data: { locationScope: { locations: { id: string; code: string }[] } }
    }
    const loc = me.data.locationScope.locations.at(0)
    expect(loc, 'E2E_CATALOGUE needs a location in scope').toBeTruthy()
    if (!loc) return
    const categoryId = await post(page, '/stock-item-categories', { name: `Recon test ${tag}` })
    const itemId = await post(page, '/stock-items', {
      sku: `REC-${tag}`,
      name: 'Reconciliation test part',
      categoryId,
      uom: 'PCS',
      trackingMode: 'QUANTITY',
    })
    const supplierId = await post(page, '/suppliers', { legalName: `Recon Test Supplier ${tag}` })
    await post(
      page,
      `/stock-items/${itemId}/receipts`,
      { locationId: loc.id, quantity: '10', uom: 'PCS', supplierId, sourceReference: `DC-${tag}` },
      `recon-rcpt-${tag}`,
    )

    // Every request the page makes from here on: reconciliation must only GET.
    const sent: Request[] = []
    page.on('request', (r) => {
      if (r.url().includes('/api/v1/')) sent.push(r)
    })

    // 1. Consistent data: the run reconciles.
    await page.goto('/inventory/reconciliation')
    await expect(page.getByRole('heading', { name: 'Stock Reconciliation' })).toBeVisible()
    await expect(page.getByText('Reconciliation has not been run')).toBeVisible()
    expect(sent.filter((r) => r.url().includes('/stock-reconciliation'))).toHaveLength(0)
    await page.getByRole('button', { name: 'Run reconciliation' }).click()
    await expect(page.getByText('Ledger agrees')).toBeVisible()
    await expect(page.getByText('No discrepancies', { exact: true })).toBeVisible()
    const before = await ledgerState(page, itemId)

    // 2. Drift: the balance says 13 while the ledger says 10.
    setBalance(itemId, loc.id, 13)
    try {
      await page.getByRole('button', { name: 'Run again' }).click()
      const row = page
        .getByRole('row')
        .or(page.getByRole('listitem'))
        .filter({ hasText: `REC-${tag}` })
        .filter({ visible: true })
        .first()
      await expect(row).toBeVisible()
      await expect(row).toContainText('Parts balance mismatch')
      await expect(row).toContainText('High severity')
      await expect(row).toContainText('10 PCS')
      await expect(row).toContainText('13 PCS')
      const reference = `PARTS_BALANCE_MISMATCH:${itemId}:${loc.id}`
      await expect(row).toContainText(reference)
      await expect(page.getByText('1 discrepancy', { exact: true })).toBeVisible()
      await page.screenshot({
        path: info.outputPath(`reconciliation-drift-${info.project.name}.png`),
        fullPage: true,
      })

      // 3. Rerun without changes: the same finding.
      const runs = sent.filter((r) => r.url().includes('/stock-reconciliation')).length
      await page.getByRole('button', { name: 'Run again' }).click()
      await expect
        .poll(() => sent.filter((r) => r.url().includes('/stock-reconciliation')).length)
        .toBe(runs + 1)
      await expect(row).toContainText(reference)
      await expect(page.getByText('1 discrepancy', { exact: true })).toBeVisible()

      // 4. Filters reach the server; the summary still describes the whole run.
      await page.getByLabel('Severity').selectOption('MEDIUM')
      await expect(page.getByText('No discrepancies match these filters')).toBeVisible()
      await page.getByLabel('Severity').selectOption('HIGH')
      await expect(row).toBeVisible()

      // 5. Nothing changed: balances (still the drifted 13) and history are as before the runs.
      const after = await ledgerState(page, itemId)
      expect(after.movements).toBe(before.movements)
      expect(after.balances).not.toBe(before.balances) // the injected drift, not a repair
      expect(after.balances).toContain('13')
      expect(sent.filter((r) => r.method() !== 'GET')).toEqual([])

      // Phone width: no sideways scroll.
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow).toBeLessThanOrEqual(1)
    } finally {
      setBalance(itemId, loc.id, 10)
    }
    await page.getByRole('button', { name: 'Run again' }).click()
    await expect(page.getByText('Ledger agrees')).toBeVisible()
    expect(problems).toEqual([])
  })

  test('a user without the report permission is refused', async ({ page }) => {
    test.skip(!hasAccount('catalogueViewer'), 'Set E2E_CATALOGUE_VIEWER_* (see DEVELOPMENT_SETUP)')
    await signIn(page, '/inventory/reconciliation', 'catalogueViewer')
    await expect(page.getByText(/do not have permission/i)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Run reconciliation' })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Stock Reconciliation' })).toHaveCount(0)
    const res = await page.request.get('/api/v1/stock-reconciliation')
    expect(res.status()).toBe(403)
  })
})
