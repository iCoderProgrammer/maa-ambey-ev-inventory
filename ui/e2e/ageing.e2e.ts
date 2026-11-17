import { execFileSync } from 'node:child_process'
import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Stock ageing (E-07) against the real API and PostgreSQL.
 *
 * Scenario A/D: goods receipts with received dates 0, 30, 31, 60, 61, 90 and 91
 * days before the business date (Asia/Kolkata) are created, captured and POSTED
 * through the real API; the list, detail and ageing summary must show exactly
 * those ages and bands (LED-008 proposed set: a band covers min <= age <= max).
 * Scenario C: one unit's received date is made to disagree with its receipt
 * (psql, user triggers off for one statement — the only way to get an
 * unreliable origin); it must show "No reliable origin date", no age, and be
 * counted as missing; it is restored at the end.
 *
 * THROWAWAY DATABASE ONLY. Needs E2E_AGEING_PO (an APPROVED purchase order with
 * one line, quantity >= 7, inserted as a fixture because brand activation is
 * not possible through the product — audit H-1) and E2E_RECON_DB_* for psql.
 * E2E_CATALOGUE needs goods_receipts.view/manage/post and vehicle_units.view
 * and a location in scope (PROPOSED grants). Placeholder data only.
 */

const db = {
  container: process.env.E2E_RECON_DB_CONTAINER ?? '',
  user: process.env.E2E_RECON_DB_USER ?? '',
  name: process.env.E2E_RECON_DB_NAME ?? '',
}
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

function shiftReceivedDate(unitId: string, days: number) {
  if (!uuid.test(unitId) || !Number.isInteger(days)) throw new Error('bad input')
  execFileSync(
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
      `BEGIN; ALTER TABLE vehicle_units DISABLE TRIGGER USER;
       UPDATE vehicle_units SET received_date = received_date + ${String(days)} WHERE id = '${unitId}';
       ALTER TABLE vehicle_units ENABLE TRIGGER USER; COMMIT;`,
    ],
    { encoding: 'utf8' },
  )
}

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
) {
  const headers: Record<string, string> = {}
  if (method !== 'GET') headers['X-CSRF-Token'] = await csrf(page.request)
  if (key) headers['Idempotency-Key'] = key
  const res = await page.request.fetch(`/api/v1${path}`, { method, headers, data })
  expect(res.ok(), `${method} ${path}: ${await res.text()}`).toBe(true)
  return ((await res.json()) as { data: T }).data
}

/** The business date (Asia/Kolkata) minus days, as YYYY-MM-DD. */
function businessDay(minusDays: number): string {
  const today = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Kolkata' }).format(new Date())
  const [y, m, d] = today.split('-').map(Number) as [number, number, number]
  const date = new Date(Date.UTC(y, m - 1, d - minusDays))
  return date.toISOString().slice(0, 10)
}

interface Unit {
  id: string
  unitNumber: string
  ageDays: number | null
  ageing: {
    status: string
    ageDays: number | null
    bandKey: string | null
    bandLabel: string | null
  }
}

const ages = [0, 30, 31, 60, 61, 90, 91]
const band: Record<number, string> = {
  0: '0–30 days',
  30: '0–30 days',
  31: '31–60 days',
  60: '31–60 days',
  61: '61–90 days',
  90: '61–90 days',
  91: 'Over 90 days',
}

test.describe('stock ageing', () => {
  test('posted goods receipts age by business date and band; a missing origin is not aged; UI equals API', async ({
    page,
  }, info) => {
    const po = process.env.E2E_AGEING_PO ?? ''
    test.skip(
      !hasAccount('catalogue') || !uuid.test(po),
      'Set E2E_CATALOGUE_* and E2E_AGEING_PO (throwaway fixture)',
    )
    test.skip(!db.container || !db.user || !db.name, 'Set E2E_RECON_DB_* (throwaway database only)')
    test.setTimeout(180_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/', 'catalogue')
    const me = await call<{ locationScope: { locations: { id: string }[] } }>(page, 'GET', '/me')
    const loc = me.locationScope.locations.at(0)
    expect(loc, 'E2E_CATALOGUE needs a location in scope').toBeTruthy()
    if (!loc) return

    // Scenario A: one receipt per age, through the real goods receipt workflow, then posted.
    const byAge = new Map<number, string>()
    for (const age of ages) {
      let grn = await call<{
        id: string
        version: number
        lines: { purchaseOrderLineId: string }[]
      }>(
        page,
        'POST',
        '/goods-receipts',
        {
          purchaseOrderId: po,
          receivedDate: businessDay(age),
          supplierReference: `DC-${tag}-${String(age)}`,
        },
        `age-grn-${tag}-${String(age)}`,
      )
      grn = await call(page, 'PUT', `/goods-receipts/${grn.id}`, {
        version: grn.version,
        receivedDate: businessDay(age),
        supplierReference: `DC-${tag}-${String(age)}`,
        lines: [
          {
            purchaseOrderLineId: grn.lines[0]?.purchaseOrderLineId,
            receivedQuantity: 1,
            acceptedQuantity: 1,
            rejectedQuantity: 0,
            units: [{ condition: 'ACCEPTED', chassisNumber: `AGE-${tag}-${String(age)}` }],
          },
        ],
      })
      const posted = await call<{ vehicleUnits: { vehicleUnitId: string }[] }>(
        page,
        'POST',
        `/goods-receipts/${grn.id}/post`,
        { version: grn.version, locationId: loc.id },
        `age-post-${tag}-${String(age)}`,
      )
      byAge.set(age, posted.vehicleUnits[0]?.vehicleUnitId ?? '')
    }

    // The API: exact ages and bands.
    const listed = await call<{ items: Unit[] }>(
      page,
      'GET',
      `/vehicle-units?q=AGE-${tag}&limit=100`,
    )
    expect(listed.items).toHaveLength(ages.length)
    for (const [age, id] of byAge) {
      const u = listed.items.find((x) => x.id === id)
      expect(u?.ageing.status, `age ${String(age)}`).toBe('AGED')
      expect(u?.ageing.ageDays).toBe(age)
      expect(u?.ageDays).toBe(age)
      expect(u?.ageing.bandLabel).toBe(band[age])
    }

    // The screen shows the same values (Scenario D: both sides of every boundary).
    await page.goto(`/inventory/vehicles?q=AGE-${tag}`)
    const rows = page.getByRole('row').or(page.getByRole('listitem')).filter({ visible: true })
    for (const [age, id] of byAge) {
      const unit = listed.items.find((x) => x.id === id)
      const row = rows.filter({ hasText: unit?.unitNumber ?? '' }).first()
      await expect(row).toContainText(`${String(age)} ${age === 1 ? 'day' : 'days'}`)
      await expect(row).toContainText(band[age] ?? '')
    }
    const panel = page.getByRole('region', { name: 'Stock ageing' })
    await expect(panel).toContainText(`${String(ages.length)} vehicles as of`)
    const tiles = panel.getByRole('list', { name: 'Vehicles by ageing band' })
    await expect(tiles.getByRole('button', { name: /0–30 days\s*2/ })).toBeVisible()
    await expect(tiles.getByRole('button', { name: /31–60 days\s*2/ })).toBeVisible()
    await expect(tiles.getByRole('button', { name: /61–90 days\s*2/ })).toBeVisible()
    await expect(tiles.getByRole('button', { name: /Over 90 days\s*1/ })).toBeVisible()
    await expect(tiles.getByRole('button', { name: /No reliable origin date\s*0/ })).toBeVisible()

    // Scenario E: a band filter (server-side) shows only that band; summary and rows agree.
    await tiles.getByRole('button', { name: /31–60 days/ }).click()
    await expect(page).toHaveURL(/ageingBand=31-60/)
    await expect(panel).toContainText('2 vehicles as of')
    await expect(
      page
        .getByText('1–2 of 2 vehicles')
        .or(page.getByText(/of 2 vehicle/))
        .first(),
    ).toBeVisible()

    // Scenario C: the 60-day unit's date no longer matches its receipt -> no age, counted as missing.
    const drifted = byAge.get(60) ?? ''
    shiftReceivedDate(drifted, 1)
    try {
      await page.goto(`/inventory/vehicles?q=AGE-${tag}`)
      await expect(tiles.getByRole('button', { name: /No reliable origin date\s*1/ })).toBeVisible()
      await expect(tiles.getByRole('button', { name: /31–60 days\s*1/ })).toBeVisible()
      const unit = listed.items.find((x) => x.id === drifted)
      const row = rows.filter({ hasText: unit?.unitNumber ?? '' }).first()
      await expect(row).toContainText('No reliable origin date')
      await expect(row).toContainText('Date does not match goods receipt')
      await expect(row).not.toContainText('60 days')
      const api = await call<Unit>(page, 'GET', `/vehicle-units/${drifted}`)
      expect(api.ageDays).toBeNull()
      expect(api.ageing.bandKey).toBeNull()
      await tiles.getByRole('button', { name: /No reliable origin date/ }).click()
      await expect(page).toHaveURL(/originStatus=MISSING/)
      await expect(panel).toContainText('1 vehicle as of')
      await page.goto(`/inventory/vehicles/${drifted}`)
      await expect(page.getByText('Why no age')).toBeVisible()
      await page.screenshot({
        path: info.outputPath(`ageing-missing-${info.project.name}.png`),
        fullPage: true,
      })
    } finally {
      shiftReceivedDate(drifted, -1)
    }
    await page.goto(`/inventory/vehicles/${byAge.get(91) ?? ''}`)
    await expect(page.getByText(/91 days \(as of/)).toBeVisible()
    await expect(page.getByText('Over 90 days').first()).toBeVisible()

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    expect(problems).toEqual([])
  })
})
