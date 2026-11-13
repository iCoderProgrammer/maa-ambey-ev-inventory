import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Stock movement timeline (E-02) against the real API and PostgreSQL. Placeholder
 * data; throwaway database only. The movements are made the real way: an
 * approved PO, a goods receipt, posting (D-07: one RECEIPT per unit) and one PDI
 * pass (D-06: PDI_PASS). Every expectation is checked against the ledger the API
 * returns — including the per-unit D-08 route, an independent query.
 * PROPOSED grants: E2E_CATALOGUE (owner) as for vehicles.e2e.ts (+ stock.parts.view,
 * goods_receipts.view); E2E_PRICE_APPROVER (receipt/count viewer, no
 * vehicle_units.view / stock.parts.view) and E2E_USERNAME (nothing) are denied;
 * optional E2E_ROLES has vehicle_units.view but not goods_receipts.view (sources
 * hidden), all in the same location.
 */

const UNITS = 3

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function call<T>(
  page: Page,
  method: 'GET' | 'POST' | 'PUT',
  path: string,
  data?: object,
  key?: string,
): Promise<T> {
  const headers: Record<string, string> = {}
  if (method !== 'GET') headers['X-CSRF-Token'] = await csrf(page.request)
  if (key) headers['Idempotency-Key'] = key
  const res = await page.request.fetch(`/api/v1${path}`, { method, headers, data })
  expect(res.ok(), `${method} ${path}: ${await res.text()}`).toBe(true)
  return ((await res.json()) as { data: T }).data
}

type ListOf<T> = { items: T[]; total: number }
interface Movement {
  id: string
  movementType: string
  sourceType: string
  sourceId: string | null
  sourceReference: string | null
  sourceHidden: boolean
  actorId: string
  vehicleUnit: { id: string; unitNumber: string } | null
}

function today(offset = 0): string {
  const d = new Date(Date.now() + offset * 86_400_000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

interface Made {
  grnId: string
  grnNumber: string
  locationId: string
  locationCode: string
  units: { id: string; unitNumber: string; chassisNumber: string }[]
  passed: { id: string; unitNumber: string }
  pdiNumber: string
}

/** PO → receipt (3 accepted units) → posting → PDI pass of unit 1. */
async function makeMovements(page: Page, approver: Page, tag: string): Promise<Made> {
  const loc = (
    await call<{ locationScope: { locations: { id: string; code: string }[] } }>(page, 'GET', '/me')
  ).locationScope.locations[0]
  expect(loc, 'E2E_CATALOGUE needs a location in scope').toBeTruthy()
  const location = loc
  const s = await call<ListOf<{ id: string }>>(
    page,
    'GET',
    `/suppliers?q=${encodeURIComponent(process.env.E2E_PO_SUPPLIER ?? '')}`,
  )
  const b = await call<ListOf<{ id: string }>>(
    page,
    'GET',
    `/brands?q=${encodeURIComponent(process.env.E2E_PO_BRAND ?? '')}`,
  )
  const brandId = b.items[0]?.id ?? ''
  const modelId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/models?brandId=${brandId}`)).items[0]?.id ??
    ''
  const variantId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/variants?modelId=${modelId}`)).items[0]
      ?.id ?? ''
  const po = await call<{ id: string; version: number }>(page, 'POST', '/purchase-orders', {
    supplierId: s.items[0]?.id,
    expectedDate: today(7),
    lines: [{ brandId, modelId, variantId, quantity: UNITS, unitCost: '950.00' }],
  })
  const sub = await call<{ version: number }>(page, 'POST', `/purchase-orders/${po.id}/submit`, {
    version: po.version,
  })
  await call(approver, 'POST', `/purchase-orders/${po.id}/approve`, { version: sub.version })
  let d = await call<{
    id: string
    number: string
    version: number
    lines: { purchaseOrderLineId: string }[]
  }>(
    page,
    'POST',
    '/goods-receipts',
    { purchaseOrderId: po.id, receivedDate: today(), supplierReference: `DC-${tag}` },
    `grn-${tag}`,
  )
  d = await call(page, 'PUT', `/goods-receipts/${d.id}`, {
    version: d.version,
    receivedDate: today(),
    supplierReference: `DC-${tag}`,
    lines: [
      {
        purchaseOrderLineId: d.lines[0]?.purchaseOrderLineId,
        receivedQuantity: UNITS,
        acceptedQuantity: UNITS,
        rejectedQuantity: 0,
        units: Array.from({ length: UNITS }, (_, i) => ({
          condition: 'ACCEPTED',
          chassisNumber: `CH-${tag}-0${String(i + 1)}`,
        })),
      },
    ],
  })
  const posted = await call<{
    vehicleUnits: { vehicleUnitId: string; unitNumber: string; chassisNumber: string }[]
  }>(
    page,
    'POST',
    `/goods-receipts/${d.id}/post`,
    { version: d.version, locationId: location.id },
    `post-${tag}`,
  )
  const units = posted.vehicleUnits
    .map((u) => ({ id: u.vehicleUnitId, unitNumber: u.unitNumber, chassisNumber: u.chassisNumber }))
    .sort((x, y) => x.chassisNumber.localeCompare(y.chassisNumber))
  const first = units[0] ?? { id: '', unitNumber: '', chassisNumber: '' }
  let ov = await call<{ checklist: { id: string }[]; unit: { version: number } }>(
    page,
    'GET',
    `/vehicle-units/${first.id}/pdi`,
  )
  if (ov.checklist.length === 0) {
    await call(page, 'POST', '/pdi-checklist-items', {
      code: `E2E_${tag}`,
      label: 'Placeholder check (E2E)',
      brandId,
    })
    ov = await call(page, 'GET', `/vehicle-units/${first.id}/pdi`)
  }
  const inspection = await call<{ inspection: { inspectionNumber: string } }>(
    page,
    'POST',
    `/vehicle-units/${first.id}/pdi`,
    {
      version: ov.unit.version,
      disposition: 'PASS',
      items: ov.checklist.map((c) => ({ checklistItemId: c.id, outcome: 'PASS' })),
      attachmentIds: [],
    },
    `pdi-${tag}`,
  )
  return {
    grnId: d.id,
    grnNumber: d.number,
    locationId: location.id,
    locationCode: location.code,
    units,
    passed: first,
    pdiNumber: inspection.inspection.inspectionNumber,
  }
}

const visible = (page: Page, text: string | RegExp) =>
  page.getByText(text).filter({ visible: true }).first()
/** The rows (desktop table) or cards (phone list) of the timeline. */
const entries = (page: Page, mobile: boolean) =>
  mobile
    ? page.getByRole('list', { name: 'Stock movements' }).getByRole('listitem')
    : page
        .getByRole('table', { name: 'Stock movements' })
        .getByRole('row')
        .filter({
          hasNot: page.getByRole('columnheader'),
        })
/** Wait until the URL carries the filter and the page has re-rendered with it. */
async function inUrl(page: Page, key: string, value: string | null, field?: string) {
  await expect.poll(() => new URL(page.url()).searchParams.get(key)).toBe(value)
  if (field)
    await expect(page.getByLabel(field, { exact: true })).not.toHaveValue(value === null ? 'x' : '')
}
const urlParams = (page: Page) => Object.fromEntries(new URL(page.url()).searchParams)

test.describe('stock movement timeline (E-02)', () => {
  test('ledger listing, filters, detail, links, permissions and no writes', async ({
    page,
    browser,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') ||
        !hasAccount('priceApprover') ||
        !hasAccount('active') ||
        !process.env.E2E_PO_SUPPLIER,
      'needs E2E_CATALOGUE_*, E2E_PRICE_APPROVER_*, E2E_USERNAME and E2E_PO_* (throwaway database only)',
    )
    test.setTimeout(240_000)
    const problems = watchConsole(page)
    const mobile = Boolean(info.project.use.isMobile)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/', 'catalogue')
    const actx = await browser.newContext({ ...info.project.use })
    const approver = await actx.newPage()
    await signIn(approver, '/', 'priceApprover')
    const made = await makeMovements(page, approver, tag)
    const meId = (await call<{ user: { id: string } }>(page, 'GET', '/me')).user.id
    // No movement is ever written from the timeline: watch every request from here on.
    const writes: string[] = []
    page.on('request', (r) => {
      if (r.method() !== 'GET' && r.url().includes('/api/v1/') && !r.url().includes('/auth/'))
        writes.push(`${r.method()} ${r.url()}`)
    })

    // ── Scenario A: listing matches the ledger ──
    const ledger: Movement[] = []
    for (const u of made.units) {
      // D-08's per-unit route is an independent read of the same ledger.
      ledger.push(
        ...(await call<ListOf<Movement>>(page, 'GET', `/vehicle-units/${u.id}/stock-movements`))
          .items,
      )
    }
    expect(ledger.map((m) => m.movementType).sort()).toEqual([
      'PDI_PASS',
      'RECEIPT',
      'RECEIPT',
      'RECEIPT',
    ])
    const api = await call<ListOf<Movement>>(page, 'GET', `/stock-movements?q=${tag}&limit=100`)
    expect(api.items.map((m) => m.id).sort()).toEqual(ledger.map((m) => m.id).sort())
    expect(api.items[0]?.movementType).toBe('PDI_PASS') // newest first

    await page.goto('/inventory/movements')
    await expect(page.getByRole('heading', { level: 1, name: 'Stock Movements' })).toBeVisible()
    await expect(visible(page, /^1–25 of \d+ movements$/)).toBeVisible()
    // Server paging.
    await page.getByRole('button', { name: /next/i }).click()
    await inUrl(page, 'page', '2')
    await expect(visible(page, /^26–50 of \d+ movements$/)).toBeVisible()
    // This run's movements, newest first, as the ledger holds them.
    const search = page.getByLabel('Search unit number, chassis or SKU', { exact: true })
    await search.fill(tag)
    await inUrl(page, 'q', tag)
    expect(urlParams(page).page).toBeUndefined() // a filter change returns to page 1
    await expect(visible(page, '1–4 of 4 movements')).toBeVisible()
    const rows = entries(page, mobile)
    await expect(rows).toHaveCount(4)
    await expect(rows.nth(0)).toContainText('Passed PDI')
    await expect(rows.nth(0)).toContainText('Awaiting PDI')
    await expect(rows.nth(0)).toContainText('Available')
    await expect(rows.nth(0)).toContainText('You')
    await expect(rows.nth(0)).toContainText(made.pdiNumber)
    for (const i of [1, 2, 3]) {
      await expect(rows.nth(i)).toContainText('Received into stock')
      await expect(rows.nth(i)).toContainText('Not in stock')
      await expect(rows.nth(i)).toContainText(made.grnNumber)
    }
    // Empty result.
    await page.getByLabel('Movement', { exact: true }).selectOption('REJECT')
    await inUrl(page, 'movementType', 'REJECT', 'Movement')
    await expect(visible(page, 'No movements match these filters')).toBeVisible()

    // ── Scenario B: filters combine, live in the URL and do not overwrite each other ──
    await page.getByLabel('Movement', { exact: true }).selectOption('PDI_PASS')
    await inUrl(page, 'movementType', 'PDI_PASS', 'Movement')
    await expect(visible(page, '1–1 of 1 movement')).toBeVisible()
    await page.getByLabel('Recorded by', { exact: true }).selectOption(meId)
    await inUrl(page, 'actorId', meId, 'Recorded by')
    await page
      .getByLabel('Location', { exact: true })
      .selectOption({ label: `${made.locationCode} — BT Test Showroom` })
    await inUrl(page, 'locationId', made.locationId, 'Location')
    await page.getByLabel('From date', { exact: true }).fill(today())
    await inUrl(page, 'from', today(), 'From date')
    await page.getByLabel('To date', { exact: true }).fill(today())
    await inUrl(page, 'to', today(), 'To date')
    await page.getByLabel('Source', { exact: true }).selectOption('PDI_INSPECTION')
    await inUrl(page, 'sourceType', 'PDI_INSPECTION', 'Source')
    await expect(visible(page, '1–1 of 1 movement')).toBeVisible()
    expect(urlParams(page)).toEqual({
      q: tag,
      movementType: 'PDI_PASS',
      actorId: meId,
      locationId: made.locationId,
      from: today(),
      to: today(),
      sourceType: 'PDI_INSPECTION',
    })
    // A range that ends before it starts is refused by the server, field by field.
    await page.getByLabel('To date', { exact: true }).fill(today(-1))
    await inUrl(page, 'to', today(-1), 'To date')
    await expect(page.getByRole('alert')).toContainText('To date must be after from')
    // A valid day with none of them.
    await page.getByLabel('From date', { exact: true }).fill(today(-1))
    await inUrl(page, 'from', today(-1), 'From date')
    await expect(visible(page, 'No movements match these filters')).toBeVisible()
    // Reset.
    await page.getByRole('button', { name: 'Clear filters' }).click()
    await expect(page).toHaveURL(/\/inventory\/movements$/)
    await expect(search).toHaveValue('')
    // Unit (chassis) and source reference.
    await search.fill(made.units[1]?.chassisNumber.toLowerCase() ?? '')
    await inUrl(page, 'q', made.units[1]?.chassisNumber.toLowerCase() ?? '')
    await expect(visible(page, '1–1 of 1 movement')).toBeVisible()
    await expect(rows.nth(0)).toContainText(made.units[1]?.unitNumber ?? '')
    await page.getByRole('button', { name: 'Clear filters' }).click()
    await page.getByLabel('Source reference', { exact: true }).fill(made.grnNumber.toLowerCase())
    await page.getByLabel('Source reference', { exact: true }).press('Enter')
    await inUrl(page, 'sourceReference', made.grnNumber.toLowerCase())
    await expect(visible(page, `1–${String(UNITS)} of ${String(UNITS)} movements`)).toBeVisible()
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    if (mobile) await expect(page.getByRole('list', { name: 'Stock movements' })).toBeVisible()
    else await expect(page.getByRole('table', { name: 'Stock movements' })).toBeVisible()

    // ── Scenario C: detail and related records ──
    await page.goto(`/inventory/movements?q=${tag}&movementType=PDI_PASS`)
    await entries(page, mobile)
      .nth(0)
      .getByRole('link', { name: 'Passed PDI', exact: true })
      .click()
    await expect(page.getByRole('heading', { level: 1, name: 'Passed PDI' })).toBeVisible()
    const card = page.getByRole('region', { name: 'Movement' }).or(page.locator('main'))
    await expect(card.getByText('Awaiting PDI').first()).toBeVisible()
    await expect(card.getByText('Available').first()).toBeVisible()
    await expect(card.getByText(`${made.locationCode} — BT Test Showroom`).first()).toBeVisible()
    await expect(card.getByText('You', { exact: true }).first()).toBeVisible()
    await expect(
      page.getByRole('button', { name: /edit|delete|reverse|correct|undo/i }),
    ).toHaveCount(0)
    await page.getByRole('link', { name: made.pdiNumber, exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`/procurement/pdi/${made.passed.id}$`))
    await page.goBack()
    await page.getByRole('link', { name: made.passed.unitNumber, exact: true }).first().click()
    await expect(
      page.getByRole('heading', { level: 1, name: made.passed.unitNumber }),
    ).toBeVisible()
    // D-08 regression: the vehicle's own history still shows the same two movements.
    const history = page.getByRole('list', { name: 'Stock movements' })
    await expect(history.getByRole('listitem')).toHaveCount(2)
    await page.goBack()
    await page.getByRole('link', { name: 'Back to stock movements' }).click()
    await expect(page).toHaveURL(/movementType=PDI_PASS/)
    // A receipt: its goods receipt opens.
    await page.goto(`/inventory/movements?q=${made.units[2]?.chassisNumber ?? ''}`)
    await entries(page, mobile)
      .nth(0)
      .getByRole('link', { name: 'Received into stock', exact: true })
      .click()
    await page.getByRole('link', { name: made.grnNumber, exact: true }).first().click()
    await expect(page.getByRole('heading', { level: 1, name: made.grnNumber })).toBeVisible()
    expect(writes).toEqual([])
    expect(problems).toEqual([])
    await actx.close()

    // ── Scenario D: permissions ──
    // Receipt and count permissions alone do not open the ledger.
    const sctx = await browser.newContext({ ...info.project.use })
    const sales = await sctx.newPage()
    await signIn(sales, '/inventory/movements', 'priceApprover')
    await expect(visible(sales, 'Access denied')).toBeVisible()
    await expect(sales.getByRole('link', { name: /Stock Movements/ })).toHaveCount(0)
    const anyId = ledger[0]?.id ?? ''
    await sales.goto(`/inventory/movements/${anyId}`)
    await expect(visible(sales, 'Access denied')).toBeVisible()
    for (const p of ['/stock-movements', `/stock-movements/${anyId}`])
      expect((await sales.request.get(`/api/v1${p}`)).status()).toBe(403)
    await sctx.close()
    const nctx = await browser.newContext({ ...info.project.use })
    const nobody = await nctx.newPage()
    await signIn(nobody, `/inventory/movements/${anyId}`, 'active')
    await expect(visible(nobody, 'Access denied')).toBeVisible()
    await expect(nobody.getByText(made.grnNumber)).toHaveCount(0)
    expect((await nobody.request.get('/api/v1/stock-movements')).status()).toBe(403)
    await nctx.close()
    // Vehicles but not goods receipts: the receipt rows show, their source does not.
    if (hasAccount('roles')) {
      const vctx = await browser.newContext({ ...info.project.use })
      const viewer = await vctx.newPage()
      const viewerProblems = watchConsole(viewer)
      await signIn(viewer, `/inventory/movements?q=${tag}`, 'roles')
      await expect(visible(viewer, '1–4 of 4 movements')).toBeVisible()
      await expect(viewer.getByText(made.grnNumber)).toHaveCount(0)
      await expect(visible(viewer, 'Not visible to you')).toBeVisible()
      await expect(visible(viewer, made.pdiNumber)).toBeVisible() // PDI is readable with vehicle_units.view
      const seen = await call<ListOf<Movement>>(viewer, 'GET', `/stock-movements?q=${tag}`)
      for (const m of seen.items.filter((x) => x.sourceType === 'GOODS_RECEIPT'))
        expect(m).toMatchObject({ sourceHidden: true, sourceId: null, sourceReference: null })
      // A hidden reference cannot be confirmed by searching for it.
      const probe = await call<ListOf<Movement>>(
        viewer,
        'GET',
        `/stock-movements?sourceReference=${made.grnNumber}`,
      )
      expect(probe.total).toBe(0)
      // No parts rows without stock.parts.view.
      const parts = await call<ListOf<Movement>>(
        viewer,
        'GET',
        '/stock-movements?itemType=STOCK_ITEM',
      )
      expect(parts.total).toBe(0)
      const body = await (await viewer.request.get(`/api/v1/stock-movements?q=${tag}`)).text()
      for (const leak of ['cost', 'storage', 'idempotency', 'requestHash'])
        expect(body).not.toContain(leak)
      await vctx.close()
      expect(viewerProblems).toEqual([])
    }
  })
})
