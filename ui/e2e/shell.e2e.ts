import { expect, test } from '@playwright/test'
import { signIn, watchConsole } from './support'

test.describe('application shell (B-03, B-04, B-05)', () => {
  test('dashboard shows live API, database and schema status', async ({ page }) => {
    const problems = watchConsole(page)
    await signIn(page)
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
    await expect(page.getByRole('banner').getByText(/online/)).toBeVisible()
    const status = page.locator('section', {
      has: page.getByRole('heading', { name: 'System status' }),
    })
    await expect(status.getByRole('row', { name: /Database\s+Ready/ })).toBeVisible()
    await expect(status.getByRole('row', { name: /Database schema\s+Ready/ })).toBeVisible()
    expect(problems).toEqual([])
  })

  test('the dashboard shows no invented catalogue data and marks planned modules', async ({
    page,
  }) => {
    await signIn(page)
    const main = page.locator('main')
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
    // Brands, specifications and authorisation must come from the API (C-01/C-02), never the UI.
    await expect(page.getByText(/authori[sz]ed/i)).toHaveCount(0)
    await expect(page.getByText(/Lithium|Telematics|Payload|Dual Battery/i)).toHaveCount(0)
    await expect(page.getByText(/\b3 Brands\b/)).toHaveCount(0)
    // Shortcuts to modules that do not exist yet must say so.
    const shortcut = main.getByRole('link', { name: /Vehicle Units Registry/ })
    await expect(shortcut).toContainText('Planned')
    await expect(shortcut).not.toContainText('Open module')
  })

  test('planned modules are labelled as planned and unknown paths show not-found', async ({
    page,
  }) => {
    await signIn(page, '/procurement/suppliers')
    await expect(page.getByRole('heading', { level: 1, name: 'Suppliers' })).toBeVisible()
    await expect(page.locator('main').getByText('Planned', { exact: true }).first()).toBeVisible()
    await expect(page.getByText(/not implemented yet/)).toBeVisible()

    await page.goto('/no/such/page')
    await expect(page.getByRole('heading', { level: 1, name: 'Page not found' })).toBeVisible()
  })

  test('no page scrolls sideways', async ({ page }) => {
    await signIn(page)
    for (const path of ['/', '/inventory/vehicles', '/no/such/page']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, path).toBeLessThanOrEqual(0)
    }
  })

  test('theme choice persists across reloads', async ({ page }) => {
    await signIn(page)
    await page.getByRole('radio', { name: 'Dark theme' }).click()
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
    await page.reload()
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
    await expect(page.getByRole('radio', { name: 'Dark theme' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    await page.getByRole('radio', { name: 'Light theme' }).click()
    await expect(page.locator('html')).not.toHaveClass(/\bdark\b/)
  })
})

test.describe('navigation on desktop', () => {
  test.skip(({ isMobile }) => isMobile, 'desktop sidebar only')

  test('sidebar links navigate, collapse and keep focus on the new page', async ({ page }) => {
    await signIn(page)
    const nav = page.getByRole('navigation', { name: 'Main' })
    await nav.getByRole('link', { name: /Customers/ }).click()
    await expect(page).toHaveURL(/\/sales\/customers$/)
    await expect(page.locator('#main')).toBeFocused()

    await page.getByRole('button', { name: 'Collapse sidebar' }).click()
    await expect(page.getByRole('button', { name: 'Expand sidebar' })).toBeVisible()
    await nav.getByRole('link', { name: /Dashboard/ }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
    await page.getByRole('button', { name: 'Expand sidebar' }).click()
  })
})

test.describe('navigation on mobile', () => {
  test.skip(({ isMobile }) => !isMobile, 'mobile drawer only')

  test('drawer opens, navigates, closes and the account menu works', async ({ page }) => {
    await signIn(page)
    await page.getByRole('button', { name: 'Open navigation' }).click()
    const drawer = page.getByRole('dialog', { name: 'Navigation' })
    await expect(drawer).toBeVisible()
    await drawer.getByRole('link', { name: /Reports/ }).click()
    await expect(drawer).toBeHidden()
    await expect(page).toHaveURL(/\/reports$/)

    await page.getByRole('button', { name: /^Account: / }).click()
    await expect(
      page.getByRole('dialog', { name: 'Account' }).getByRole('button', { name: 'Sign out' }),
    ).toBeVisible()
    await page.keyboard.press('Escape')
  })
})
