import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Vehicle list and detail (D-08) against the real API and PostgreSQL. Placeholder
 * data; throwaway database only. The vehicles are made the real way: an approved
 * PO, a goods receipt with a private document, posting (D-07) and one PDI pass
 * (D-06). Needs the E2E_PO_* fixture. PROPOSED grants: E2E_CATALOGUE (owner) holds
 * the D-02/D-03/D-07 grants, vehicle_units.view + vehicle_units.cost.view,
 * vehicle_units.pdi.record + checklist.manage, attachments.* and one location;
 * E2E_PRICE_APPROVER approves the PO; E2E_CATALOGUE_VIEWER (accountant) has
 * vehicle_units.view only (no cost, no attachments.view) in the same location;
 * E2E_USERNAME (e2e.active) has nothing.
 */

const UNITS = 30

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

function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

interface Made {
  grnId: string
  grnNumber: string
  poId: string
  poNumber: string
  locationCode: string
  units: { id: string; unitNumber: string; chassisNumber: string }[]
  passed: { id: string; unitNumber: string }
}

/** PO → receipt (30 accepted units, identifiers, one challan) → posting → PDI pass of unit 1. */
async function makeVehicles(page: Page, approver: Page, tag: string): Promise<Made> {
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
  const future = new Date(Date.now() + 7 * 86_400_000).toISOString().slice(0, 10)
  const po = await call<{ id: string; number: string; version: number }>(
    page,
    'POST',
    '/purchase-orders',
    {
      supplierId: s.items[0]?.id,
      expectedDate: future,
      lines: [{ brandId, modelId, variantId, quantity: UNITS, unitCost: '950.00' }],
    },
  )
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
  const units = Array.from({ length: UNITS }, (_, i) => {
    const n = String(i + 1).padStart(2, '0')
    return {
      condition: 'ACCEPTED',
      chassisNumber: `CH-${tag}-${n}`,
      motorSerial: `MOT-${tag}-${n}`,
      batterySerial: `BAT-${tag}-${n}`,
    }
  })
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
        units,
      },
    ],
  })
  // A private delivery challan at the location, attached to the receipt.
  const upload = await page.request.post(`/api/v1/attachments?locationId=${location.id}`, {
    headers: { 'X-CSRF-Token': await csrf(page.request) },
    multipart: {
      file: {
        name: `challan-${tag}.pdf`,
        mimeType: 'application/pdf',
        buffer: Buffer.from('%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF\n'),
      },
    },
  })
  expect(upload.ok(), await upload.text()).toBe(true)
  const attachmentId = ((await upload.json()) as { data: { id: string } }).data.id
  await call(page, 'POST', `/goods-receipts/${d.id}/documents`, {
    attachmentId,
    documentType: 'DELIVERY_CHALLAN',
  })
  d = await call(page, 'GET', `/goods-receipts/${d.id}`)
  const posted = await call<{
    vehicleUnits: { vehicleUnitId: string; unitNumber: string; chassisNumber: string }[]
  }>(
    page,
    'POST',
    `/goods-receipts/${d.id}/post`,
    { version: d.version, locationId: location.id },
    `post-${tag}`,
  )
  const sorted = posted.vehicleUnits
    .map((u) => ({ id: u.vehicleUnitId, unitNumber: u.unitNumber, chassisNumber: u.chassisNumber }))
    .sort((x, y) => x.chassisNumber.localeCompare(y.chassisNumber))
  const first = sorted[0] ?? { id: '', unitNumber: '', chassisNumber: '' }

  // PDI pass of unit 01: every applicable checklist item answered PASS (one is
  // added for the fixture brand if none applies yet).
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
  await call(
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
    poId: po.id,
    poNumber: po.number,
    locationCode: location.code,
    units: sorted,
    passed: first,
  }
}

const visible = (page: Page, text: string | RegExp) =>
  page.getByText(text).filter({ visible: true }).first()
const unitLink = (page: Page, unitNumber: string) =>
  page.getByRole('link', { name: unitNumber, exact: true }).filter({ visible: true })
/**
 * Wait until the URL carries a filter and the page has re-rendered with it (the
 * controlled field shows it): one change at a time, as a person makes them.
 */
async function inUrl(page: Page, key: string, value?: string, field?: string) {
  await expect
    .poll(() => {
      const v = new URL(page.url()).searchParams.get(key)
      return value === undefined ? v !== null : v === value
    })
    .toBe(true)
  if (field) {
    const control = page.getByLabel(field)
    if (value === undefined) await expect(control).not.toHaveValue('')
    else await expect(control).toHaveValue(value)
  }
}
const listParams = (page: Page) =>
  page.waitForResponse(
    (r) => new URL(r.url()).pathname === '/api/v1/vehicle-units' && r.request().method() === 'GET',
  )

test.describe('vehicle list and detail (D-08)', () => {
  test('search, combined filters, paging, detail, history, documents and permissions', async ({
    page,
    browser,
  }, info) => {
    test.skip(
      !hasAccount('catalogue') ||
        !hasAccount('priceApprover') ||
        !hasAccount('catalogueViewer') ||
        !hasAccount('active') ||
        !process.env.E2E_PO_SUPPLIER,
      'needs E2E_CATALOGUE_*, E2E_PRICE_APPROVER_*, E2E_CATALOGUE_VIEWER_*, E2E_USERNAME and E2E_PO_* (throwaway database only)',
    )
    test.setTimeout(240_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/', 'catalogue')
    const actx = await browser.newContext({ ...info.project.use })
    const approver = await actx.newPage()
    await signIn(approver, '/', 'priceApprover')
    const made = await makeVehicles(page, approver, tag)
    await actx.close()
    const mobile = Boolean(info.project.use.isMobile)

    // ── Scenario A: listing, search, filters, paging, clear ──
    await page.goto('/inventory/vehicles')
    await expect(page.getByRole('heading', { level: 1, name: 'Vehicle Units' })).toBeVisible()
    await expect(visible(page, /^1–25 of \d+ vehicles$/)).toBeVisible()
    const search = page.getByLabel('Search unit number, chassis, motor or battery')
    // Chassis search
    let res = listParams(page)
    await search.fill(`ch-${tag}-07`)
    await res
    await expect(visible(page, '1–1 of 1 vehicle')).toBeVisible()
    await expect(unitLink(page, made.units[6]?.unitNumber ?? '')).toBeVisible()
    // Motor and battery search (stored identifiers)
    res = listParams(page)
    await search.fill(`MOT-${tag}-12`)
    await res
    await expect(unitLink(page, made.units[11]?.unitNumber ?? '')).toBeVisible()
    await expect(visible(page, '1–1 of 1 vehicle')).toBeVisible()
    res = listParams(page)
    await search.fill(`bat-${tag}-30`)
    await res
    await expect(unitLink(page, made.units[29]?.unitNumber ?? '')).toBeVisible()
    // Search + status + location + supplier + received date: the 29 units still awaiting PDI.
    await search.fill(tag)
    await inUrl(page, 'q', tag, 'Search unit number, chassis, motor or battery')
    await expect(visible(page, `1–25 of ${String(UNITS)} vehicles`)).toBeVisible()
    await page.getByLabel('Status').selectOption('RECEIVED_PENDING_PDI')
    await inUrl(page, 'status', 'RECEIVED_PENDING_PDI', 'Status')
    await page
      .getByLabel('Location')
      .selectOption({ label: `${made.locationCode} — BT Test Showroom` })
    await inUrl(page, 'locationId', undefined, 'Location')
    await page.getByLabel('Supplier').selectOption({ label: process.env.E2E_PO_SUPPLIER ?? '' })
    await inUrl(page, 'supplierId', undefined, 'Supplier')
    await page.getByLabel('Received from').fill(today())
    await inUrl(page, 'receivedFrom', today(), 'Received from')
    await expect(visible(page, `1–25 of ${String(UNITS - 1)} vehicles`)).toBeVisible()
    await expect(page.getByText('Not saleable').filter({ visible: true }).first()).toBeVisible()
    await expect(unitLink(page, made.passed.unitNumber)).toHaveCount(0)
    // Brand → model cascade + Available: only the unit that passed PDI.
    await page.getByLabel('Status').selectOption('AVAILABLE')
    await inUrl(page, 'status', 'AVAILABLE', 'Status')
    await page.getByLabel('Brand').selectOption({ label: process.env.E2E_PO_BRAND ?? '' })
    await inUrl(page, 'brandId', undefined, 'Brand')
    await expect(page.getByLabel('Model')).toBeEnabled()
    await page.getByLabel('Model').selectOption({ index: 1 })
    await inUrl(page, 'modelId', undefined, 'Model')
    await expect(visible(page, '1–1 of 1 vehicle')).toBeVisible()
    await expect(unitLink(page, made.passed.unitNumber)).toBeVisible()
    expect(new URL(page.url()).searchParams.get('status')).toBe('AVAILABLE')
    // Age: the units were received today, so "at least 1 day" finds none.
    await page.getByLabel('Min. days in stock').fill('1')
    await page.getByLabel('Min. days in stock').press('Enter')
    await expect(visible(page, 'No vehicles match these filters')).toBeVisible()
    // Clear: back to the full list.
    await page.getByRole('button', { name: 'Clear filters' }).click()
    await expect(page).toHaveURL(/\/inventory\/vehicles$/)
    await expect(search).toHaveValue('')
    // Paging on the server: 30 tagged units, sorted by unit number.
    await search.fill(tag)
    await inUrl(page, 'q', tag, 'Search unit number, chassis, motor or battery')
    await expect(visible(page, `1–25 of ${String(UNITS)} vehicles`)).toBeVisible()
    await page.getByLabel('Sort by').selectOption('unitNumber')
    await inUrl(page, 'sort', 'unitNumber', 'Sort by')
    await expect(visible(page, `1–25 of ${String(UNITS)} vehicles`)).toBeVisible()
    const page2 = page.waitForResponse((r) => {
      const u = new URL(r.url())
      return u.pathname === '/api/v1/vehicle-units' && u.searchParams.get('page') === '2'
    })
    await page.getByRole('button', { name: /next/i }).click()
    expect(new URL((await page2).url()).searchParams.get('sort')).toBe('unitNumber')
    await expect(visible(page, `26–30 of ${String(UNITS)} vehicles`)).toBeVisible()
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)
    if (mobile) {
      await expect(page.getByRole('list', { name: 'Vehicle units' })).toBeVisible()
    } else {
      await expect(page.getByRole('table', { name: 'Vehicle units' })).toBeVisible()
    }

    // ── Scenario B: detail of the passed unit ──
    await page.goto(`/inventory/vehicles?q=${encodeURIComponent(made.passed.unitNumber)}`)
    await unitLink(page, made.passed.unitNumber).click()
    await expect(
      page.getByRole('heading', { level: 1, name: made.passed.unitNumber }),
    ).toBeVisible()
    await expect(visible(page, 'Available — this vehicle is saleable stock.')).toBeVisible()
    await expect(visible(page, `CH-${tag}-01`)).toBeVisible()
    await expect(visible(page, `MOT-${tag}-01`)).toBeVisible()
    await expect(visible(page, `BAT-${tag}-01`)).toBeVisible()
    await expect(visible(page, `${made.locationCode} — BT Test Showroom`)).toBeVisible()
    await expect(visible(page, made.passed.id)).toBeVisible() // internal ID
    await expect(visible(page, '₹950.00')).toBeVisible() // cost permission
    await expect(page.getByRole('link', { name: made.poNumber })).toHaveAttribute(
      'href',
      `/procurement/purchase-orders/${made.poId}`,
    )
    // Documents: the receipt's challan, downloadable through the private attachment API.
    const docs = page.getByRole('list', { name: 'Vehicle documents' })
    await expect(docs.getByText(`challan-${tag}.pdf`)).toBeVisible()
    const download = page.waitForEvent('download')
    await docs.getByRole('button', { name: `Download challan-${tag}.pdf` }).click()
    expect((await download).suggestedFilename()).toBe(`challan-${tag}.pdf`)
    // PDI and movement history, oldest first.
    const inspections = page.getByRole('list', { name: 'Inspections' })
    await expect(inspections.getByText('Pass', { exact: true })).toBeVisible()
    const history = page.getByRole('list', { name: 'Stock movements' })
    const entries = history.getByRole('listitem')
    await expect(entries).toHaveCount(2)
    await expect(entries.nth(0).getByText('Received into stock')).toBeVisible()
    await expect(entries.nth(1).getByText('Passed PDI')).toBeVisible()
    await expect(entries.nth(1).getByRole('link', { name: /^PDI-\d+$/ })).toBeVisible()
    // Back keeps the list search.
    await page.getByRole('link', { name: 'Back to vehicle units' }).click()
    await expect(page).toHaveURL(new RegExp(`q=${made.passed.unitNumber}`))
    // Related records: the receipt from the movement history.
    await unitLink(page, made.passed.unitNumber).click()
    await history.getByRole('link', { name: made.grnNumber }).click()
    await expect(page.getByRole('heading', { level: 1, name: made.grnNumber })).toBeVisible()
    // And PDI from the detail page.
    await page.goBack()
    await page.getByRole('link', { name: 'Open PDI' }).click()
    await expect(page).toHaveURL(new RegExp(`/procurement/pdi/${made.passed.id}$`))
    expect(problems).toEqual([])

    // ── Scenario C: permissions ──
    // A viewer without cost or attachment permission: no amount, no documents, API agrees.
    const vctx = await browser.newContext({ ...info.project.use })
    const viewer = await vctx.newPage()
    const viewerProblems = watchConsole(viewer)
    await signIn(viewer, `/inventory/vehicles/${made.passed.id}`, 'catalogueViewer')
    await expect(
      viewer.getByRole('heading', { level: 1, name: made.passed.unitNumber }),
    ).toBeVisible()
    await expect(visible(viewer, 'Recorded (not visible to you)')).toBeVisible()
    await expect(viewer.getByText('₹950.00')).toHaveCount(0)
    await expect(
      visible(viewer, 'You do not have permission to open documents, so none are listed.'),
    ).toBeVisible()
    const vd = await call<{
      unitCost: string | null
      costHidden: boolean
      documents: unknown[]
      documentsHidden: boolean
    }>(viewer, 'GET', `/vehicle-units/${made.passed.id}`)
    expect(vd).toMatchObject({
      unitCost: null,
      costHidden: true,
      documents: [],
      documentsHidden: true,
    })
    const owner = await call<{ documents: { attachmentId: string }[] }>(
      page,
      'GET',
      `/vehicle-units/${made.passed.id}`,
    )
    const docId = owner.documents[0]?.attachmentId ?? ''
    const link = await viewer.request.get(
      `/api/v1/attachments/${docId}/link?disposition=attachment`,
    )
    expect([403, 404]).toContain(link.status())
    await vctx.close()
    expect(viewerProblems).toEqual([])
    // A user without vehicle_units.view: denied on the list, a direct URL and the API.
    const nctx = await browser.newContext({ ...info.project.use })
    const nobody = await nctx.newPage()
    await signIn(nobody, '/inventory/vehicles', 'active')
    await expect(visible(nobody, 'Access denied')).toBeVisible()
    await nobody.goto(`/inventory/vehicles/${made.passed.id}`)
    await expect(visible(nobody, 'Access denied')).toBeVisible()
    await expect(nobody.getByText(made.passed.unitNumber)).toHaveCount(0)
    for (const p of [
      '/vehicle-units',
      `/vehicle-units/${made.passed.id}`,
      `/vehicle-units/${made.passed.id}/stock-movements`,
    ]) {
      expect((await nobody.request.get(`/api/v1${p}`)).status()).toBe(403)
    }
    await nctx.close()
  })
})
