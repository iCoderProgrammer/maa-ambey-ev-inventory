import { expect, type APIRequestContext, type Page } from '@playwright/test'

/**
 * Test-owned vehicle unit, registered through the product's only supported path
 * (D-02 purchase order → second-user approval → D-03 goods receipt → D-07
 * posting), so a test that needs an "already registered" chassis does not
 * depend on whatever the database happens to contain.
 *
 * THROWAWAY DATABASE ONLY. Needs the eligible-PO fixture named in
 * E2E_PO_SUPPLIER and E2E_PO_BRAND (DEVELOPMENT_SETUP "End-to-end browser
 * tests"), a creator page holding purchase_orders.view/manage,
 * goods_receipts.view/manage/post, catalogue.view and suppliers.view with a
 * location in scope, and a different approver page holding
 * purchase_orders.view/approve. Posted stock is append-only, so the unit is not
 * deleted afterwards; its chassis carries the test's tag.
 */

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

type ListOf<T> = { items: T[] }

function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${String(d.getFullYear())}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** Registers one vehicle unit with chassis `chassis` at the creator's first location; returns its unit ID. */
export async function registerUnitThroughReceipt(
  page: Page,
  approver: Page,
  tag: string,
  chassis: string,
): Promise<string> {
  const loc = (
    await call<{ locationScope: { locations: { id: string }[] } }>(page, 'GET', '/me')
  ).locationScope.locations.at(0)
  expect(loc, 'the creator needs a location in scope').toBeTruthy()
  const supplier = (
    await call<ListOf<{ id: string }>>(
      page,
      'GET',
      `/suppliers?q=${encodeURIComponent(process.env.E2E_PO_SUPPLIER ?? '')}`,
    )
  ).items[0]?.id
  const brandId =
    (
      await call<ListOf<{ id: string }>>(
        page,
        'GET',
        `/brands?q=${encodeURIComponent(process.env.E2E_PO_BRAND ?? '')}`,
      )
    ).items[0]?.id ?? ''
  const modelId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/models?brandId=${brandId}`)).items[0]?.id ??
    ''
  const variantId =
    (await call<ListOf<{ id: string }>>(page, 'GET', `/variants?modelId=${modelId}`)).items[0]
      ?.id ?? ''
  expect(
    supplier && brandId && modelId && variantId,
    `E2E_PO_SUPPLIER/E2E_PO_BRAND fixture not found in this database (supplier ${String(Boolean(supplier))}, ` +
      `brand/model/variant ${String(Boolean(brandId && modelId && variantId))}): apply ` +
      'docs/evidence/BROWSER-TEST-2026-11-11/po-fixture.sql to the throwaway database',
  ).toBeTruthy()

  const future = new Date(Date.now() + 7 * 86_400_000).toISOString().slice(0, 10)
  const po = await call<{ id: string; version: number }>(page, 'POST', '/purchase-orders', {
    supplierId: supplier,
    expectedDate: future,
    lines: [{ brandId, modelId, variantId, quantity: 1, unitCost: '950.00' }],
  })
  const sub = await call<{ version: number }>(page, 'POST', `/purchase-orders/${po.id}/submit`, {
    version: po.version,
  })
  await call(approver, 'POST', `/purchase-orders/${po.id}/approve`, { version: sub.version })

  let d = await call<{ id: string; version: number; lines: { purchaseOrderLineId: string }[] }>(
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
        receivedQuantity: 1,
        acceptedQuantity: 1,
        rejectedQuantity: 0,
        units: [{ condition: 'ACCEPTED', chassisNumber: chassis }],
      },
    ],
  })
  const posted = await call<{ vehicleUnits: { vehicleUnitId: string; chassisNumber: string }[] }>(
    page,
    'POST',
    `/goods-receipts/${d.id}/post`,
    { version: d.version, locationId: loc?.id },
    `post-${tag}`,
  )
  const unit = posted.vehicleUnits.find((u) => u.chassisNumber === chassis)
  expect(unit, 'posting did not register the unit').toBeTruthy()
  return unit?.vehicleUnitId ?? ''
}
