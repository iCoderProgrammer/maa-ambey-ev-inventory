import { expect, test, type APIRequestContext, type APIResponse, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Catalogue API (C-02) driven from a signed-in browser session: the session
 * cookie and CSRF token are the real ones the app uses. C-02 has no catalogue
 * screen (that is C-03), so these are API-level checks only.
 *
 * Placeholder data only ("E2E Brand …", example.com sources). Run only against a
 * throwaway database: the grants are PROPOSED and count only with
 * RBAC_ALLOW_PROPOSED_GRANTS=true in development.
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

async function errorOf(res: APIResponse) {
  return ((await res.json()) as { error: { code: string; details?: { field: string }[] } }).error
}

interface Item {
  id: string
  version: number
  brandId?: string
  modelId?: string
  lifecycleStatus: string
}

async function dataOf<T = Item>(res: APIResponse): Promise<T> {
  return ((await res.json()) as { data: T }).data
}

/** Calls the API with the page's session; the CSRF token is read once per session. */
function api(page: Page) {
  let token: Promise<string> | undefined
  const send = async (method: 'POST' | 'PATCH' | 'PUT' | 'DELETE', path: string, body?: unknown) =>
    page.request.fetch(`/api/v1${path}`, {
      method,
      headers: { 'X-CSRF-Token': await (token ??= csrf(page.request)) },
      data: body,
    })
  return {
    get: (path: string) => page.request.get(`/api/v1${path}`),
    post: (path: string, body?: unknown) => send('POST', path, body),
    patch: (path: string, body?: unknown) => send('PATCH', path, body),
    put: (path: string, body?: unknown) => send('PUT', path, body),
    del: (path: string) => send('DELETE', path),
  }
}

const source = {
  confidence: 'SUPPLIER_CONFIRMED',
  sourceUrl: 'https://example.com/e2e-source',
  sourceDate: '2026-10-01',
}

test.describe('catalogue API (C-02)', () => {
  test('deny by default: no session, no grant, no CSRF token', async ({
    page,
    browser,
    isMobile,
  }) => {
    test.skip(isMobile, 'API-level checks run once (desktop)')
    const problems = watchConsole(page)
    const anon = await browser.newContext()
    const base = test.info().project.use.baseURL ?? 'http://localhost:5173'
    const anonRes = await anon.request.get(new URL('/api/v1/brands', base).href)
    await anon.close()
    expect(anonRes.status()).toBe(401)

    await signIn(page, '/', 'active')
    const c = api(page)
    for (const res of [
      await c.get('/brands'),
      await c.get('/models'),
      await c.post('/brands', { name: 'X' }),
    ]) {
      expect(res.status()).toBe(403)
      expect((await errorOf(res)).code).toBe('FORBIDDEN')
    }
    const noCsrf = await page.request.post('/api/v1/brands', { data: { name: 'X' } })
    expect(noCsrf.status()).toBe(403)
    expect((await errorOf(noCsrf)).code).toBe('CSRF_FAILED')
    expect(problems).toEqual([])
  })

  test('relationships, duplicates, activation, archive and versions are enforced', async ({
    page,
    browser,
    isMobile,
  }) => {
    test.skip(isMobile, 'API-level checks run once (desktop)')
    test.skip(
      !hasAccount('catalogue') || !hasAccount('catalogueViewer'),
      'needs E2E_CATALOGUE_* and E2E_CATALOGUE_VIEWER_* accounts (throwaway database only)',
    )
    test.setTimeout(90_000)
    const problems = watchConsole(page)
    await signIn(page, '/', 'catalogue')
    const c = api(page)
    const run = Date.now().toString(36)

    // Create two brands; names are unique case-insensitively.
    const brandA = await c.post('/brands', { name: `E2E Brand A ${run}` })
    expect(brandA.status()).toBe(201)
    const a = await dataOf<Item & { operational: boolean }>(brandA)
    expect(a.lifecycleStatus).toBe('PENDING_VERIFICATION')
    expect(a.operational).toBe(false)
    const b = await dataOf(await c.post('/brands', { name: `E2E Brand B ${run}` }))
    const dupBrand = await c.post('/brands', { name: `e2e brand a ${run}`, slug: `other-${run}` })
    expect(dupBrand.status()).toBe(409)
    expect((await errorOf(dupBrand)).code).toBe('DUPLICATE')

    // Server-owned fields cannot be sent.
    const forged = await c.post('/brands', { name: `Forged ${run}`, lifecycleStatus: 'ACTIVE' })
    expect(forged.status()).toBe(422)
    expect((await errorOf(forged)).details?.[0]?.field).toBe('lifecycleStatus')

    // Codes are unique within a brand, not across brands.
    let m = await dataOf(
      await c.post('/models', { brandId: a.id, name: 'E2E Model', internalCode: 'e2e1' }),
    )
    expect(
      (
        await c.post('/models', { brandId: b.id, name: 'E2E Model', internalCode: 'E2E1' })
      ).status(),
    ).toBe(201)
    const dupCode = await c.post('/models', { brandId: a.id, name: 'Other', internalCode: 'E2E1' })
    expect(dupCode.status()).toBe(409)
    expect((await errorOf(dupCode)).details?.[0]?.field).toBe('internalCode')

    // Cross-brand: a variant cannot claim another brand; a model cannot move.
    const cross = await c.post('/variants', {
      modelId: m.id,
      brandId: b.id,
      name: 'V',
      variantCode: 'STD',
    })
    expect(cross.status()).toBe(422)
    expect((await errorOf(cross)).details?.[0]?.field).toBe('brandId')
    expect((await c.patch(`/models/${m.id}`, { version: m.version, brandId: b.id })).status()).toBe(
      422,
    )
    let v = await dataOf(
      await c.post('/variants', { modelId: m.id, name: 'E2E Variant', variantCode: 'std' }),
    )
    expect(v.brandId).toBe(a.id)

    // Activation order: verify first, model before variant.
    const unverified = await c.post(`/models/${m.id}/activate`, { version: m.version })
    expect(unverified.status()).toBe(409)
    expect((await errorOf(unverified)).code).toBe('ACTIVATION_BLOCKED')
    m = await dataOf(await c.post(`/models/${m.id}/verify`, { version: m.version, ...source }))
    v = await dataOf(await c.post(`/variants/${v.id}/verify`, { version: v.version, ...source }))
    const early = await c.post(`/variants/${v.id}/activate`, { version: v.version })
    expect((await errorOf(early)).details?.map((d) => d.field)).toEqual(['parent'])
    const stale = await c.post(`/models/${m.id}/activate`, { version: m.version - 1 })
    expect(stale.status()).toBe(409)
    expect((await errorOf(stale)).code).toBe('VERSION_CONFLICT')
    m = await dataOf(await c.post(`/models/${m.id}/activate`, { version: m.version }))
    v = await dataOf(await c.post(`/variants/${v.id}/activate`, { version: v.version }))
    expect(v.lifecycleStatus).toBe('ACTIVE')
    const editActive = await c.patch(`/variants/${v.id}`, { version: v.version, name: 'X' })
    expect((await errorOf(editActive)).code).toBe('INVALID_STATE')

    // Brand activation stays blocked (evidence missing; later-task prerequisites pending).
    const noEvidence = await c.put(`/brands/${a.id}/authorisation`, {
      version: a.version,
      authorisationStatus: 'AUTHORISED',
    })
    expect(noEvidence.status()).toBe(422)
    const readiness = await dataOf<{ ready: boolean; checks: { key: string; met: boolean }[] }>(
      await c.get(`/brands/${a.id}/activation-readiness`),
    )
    expect(readiness.ready).toBe(false)
    expect(readiness.checks.find((k) => k.key === 'verified_catalogue')?.met).toBe(true)
    const blocked = await c.post(`/brands/${a.id}/activate`, { version: a.version })
    expect(blocked.status()).toBe(409)
    expect((await errorOf(blocked)).details?.map((d) => d.field)).toContain('approved_prices')

    // Archive keeps history: the variant still points at its model and brand; nothing new is added.
    m = await dataOf(await c.post(`/models/${m.id}/archive`, { version: m.version, reason: 'e2e' }))
    expect(m.lifecycleStatus).toBe('ARCHIVED')
    const kept = await dataOf(await c.get(`/variants/${v.id}`))
    expect([kept.modelId, kept.brandId]).toEqual([m.id, a.id])
    const underArchived = await c.post('/variants', {
      modelId: m.id,
      name: 'New',
      variantCode: 'NEW',
    })
    expect((await errorOf(underArchived)).code).toBe('INVALID_STATE')
    expect((await c.del(`/models/${m.id}`)).status()).toBe(405)

    // Lists page and filter by brand.
    const list = await dataOf<{ items: Item[]; total: number; limit: number }>(
      await c.get(`/models?brandId=${a.id}&limit=1`),
    )
    expect(list.total).toBe(1)
    expect(list.items).toHaveLength(1)
    expect((await c.get('/models?limit=500')).status()).toBe(422)

    // A viewer reads but cannot change anything.
    const other = await browser.newContext()
    const viewerPage = await other.newPage()
    await signIn(viewerPage, '/', 'catalogueViewer')
    const viewer = api(viewerPage)
    expect((await viewer.get(`/brands/${a.id}`)).status()).toBe(200)
    expect((await viewer.post(`/brands/${a.id}/archive`, { version: a.version })).status()).toBe(
      403,
    )
    expect((await viewer.post('/brands', { name: `Viewer ${run}` })).status()).toBe(403)
    await other.close()

    expect(problems).toEqual([])
  })
})
