import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * Brand isolation (C-06) in the browser and through the real API: stale
 * cross-brand filters are dropped, cross-brand requests fail, brand filters
 * stay scoped and arbitrary brand names work. Placeholder data; throwaway
 * database only.
 */

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  return ((await res.json()) as { data: { csrfToken: string } }).data.csrfToken
}

type Rec = { id: string; version: number; brandId?: string }

function api(page: Page) {
  let token: Promise<string> | undefined
  return {
    post: async <T = Rec>(path: string, data: unknown): Promise<T> => {
      const res = await page.request.post(`/api/v1${path}`, {
        headers: { 'X-CSRF-Token': await (token ??= csrf(page.request)) },
        data,
      })
      expect(res.ok(), `${path}: ${await res.text()}`).toBe(true)
      return ((await res.json()) as { data: T }).data
    },
    get: (path: string) => page.request.get(`/api/v1${path}`),
  }
}

const source = {
  confidence: 'SUPPLIER_CONFIRMED',
  sourceUrl: 'https://example.com/iso',
  sourceDate: '2026-10-01',
}

async function brandWithVariant(page: Page, name: string) {
  const c = api(page)
  const brand = await c.post('/brands', { name })
  let model = await c.post('/models', {
    brandId: brand.id,
    name: `${name} Model`,
    internalCode: 'IM1',
  })
  model = await c.post(`/models/${model.id}/verify`, { version: model.version, ...source })
  await c.post(`/models/${model.id}/activate`, { version: model.version })
  const variant = await c.post('/variants', {
    modelId: model.id,
    name: `${name} Variant`,
    variantCode: 'STD',
  })
  const list = await c.post('/price-lists', { brandId: brand.id, name: 'Showroom' })
  return { brand: brand.id, model: model.id, variant: variant.id, list: list.id }
}

test.describe('brand isolation (C-06)', () => {
  test('cross-brand filters and requests are refused; stale links are cleaned', async ({
    page,
    isMobile,
  }, testInfo) => {
    test.skip(
      !hasAccount('catalogue'),
      'needs E2E_CATALOGUE_* with price draft grants (throwaway database only)',
    )
    test.setTimeout(90_000)
    const problems = watchConsole(page)
    const run = `${testInfo.project.name} ${Date.now().toString(36)}`
    await signIn(page, '/', 'catalogue')
    // Arbitrary brand names, unknown to any code.
    const a = await brandWithVariant(page, `Isolation Aster ${run}`)
    const b = await brandWithVariant(page, `Isolation Borealis ${run}`)
    const c = api(page)

    // A stale link pairing brand B with brand A's model: the UI drops the model filter, no error shown.
    await page.goto(`/admin/catalogue/variants?brandId=${b.brand}&modelId=${a.model}`)
    await expect(page).toHaveURL(new RegExp(`/admin/catalogue/variants\\?brandId=${b.brand}$`))
    const list = page.getByRole(isMobile ? 'list' : 'table', { name: 'Variants' })
    await expect(
      list.getByRole('link', { name: `Isolation Borealis ${run} Variant` }),
    ).toBeVisible()
    await expect(list.getByRole('link', { name: `Isolation Aster ${run} Variant` })).toHaveCount(0)
    await expect(page.getByText(/Could not load variants/)).toHaveCount(0)

    // The model filter offers only brand B's models.
    if (!isMobile) {
      const options = await page.getByLabel('Model').locator('option').allTextContents()
      expect(options).toEqual(['All models of this brand', `Isolation Borealis ${run} Model (IM1)`])
    }

    // The API refuses cross-brand pairs from the signed-in browser session.
    for (const path of [
      `/variants?brandId=${b.brand}&modelId=${a.model}`,
      `/price-items?priceListId=${a.list}&variantId=${b.variant}`,
      `/prices/lookup?priceListId=${a.list}&variantId=${b.variant}&priceType=SELLING`,
    ]) {
      const res = await c.get(path)
      expect(res.status(), path).toBe(422)
    }
    const write = await page.request.post('/api/v1/variants', {
      headers: { 'X-CSRF-Token': await csrf(page.request) },
      data: { modelId: a.model, brandId: b.brand, name: 'X', variantCode: 'X' },
    })
    expect(write.status()).toBe(422)

    // Brand filters return only that brand's rows.
    for (const s of [a, b]) {
      for (const path of [
        `/models?brandId=${s.brand}`,
        `/variants?brandId=${s.brand}`,
        `/price-lists?brandId=${s.brand}`,
      ]) {
        const body = (await (await c.get(path)).json()) as {
          data: { items: { brandId: string }[] }
        }
        expect(body.data.items.length, path).toBe(1)
        expect(
          body.data.items.every((i) => i.brandId === s.brand),
          path,
        ).toBe(true)
      }
    }
    expect(problems).toEqual([])
  })
})
