import { expect, test } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

test.describe('roles and permissions (B-07)', () => {
  test('a user without grants sees no gated links and is denied on direct access', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page)
    if (!isMobile) {
      const nav = page.getByRole('navigation', { name: 'Main' })
      await expect(nav.getByRole('link', { name: /Catalogue & Brands/ })).toBeVisible()
      await expect(nav.getByRole('link', { name: /Users & Roles/ })).toHaveCount(0)
    }

    await page.goto('/admin/users')
    await expect(page.getByText('Access denied')).toBeVisible()
    await expect(page.getByRole('table')).toHaveCount(0)
    const api = await page.request.get('/api/v1/access/roles')
    expect(api.status()).toBe(403)

    await page.getByRole('button', { name: /^Account: / }).click()
    const menu = page.getByRole('dialog', { name: 'Account' })
    await expect(menu.getByText('No role assigned')).toBeVisible()
    await expect(menu.getByText(/No permissions are granted yet/)).toBeVisible()
    expect(problems).toEqual([])
  })

  test('a user with the permission sees the roles pending owner sign-off', async ({
    page,
    isMobile,
  }) => {
    test.skip(
      !hasAccount('roles'),
      'set E2E_ROLES_USERNAME/PASSWORD for a user holding access.roles.view',
    )
    await signIn(page, '/', 'roles')
    if (!isMobile) {
      await page
        .getByRole('navigation', { name: 'Main' })
        .getByRole('link', { name: /Users & Roles/ })
        .click()
    } else {
      await page.goto('/admin/users')
    }
    await expect(page.getByRole('heading', { level: 1, name: 'Users & Roles' })).toBeVisible()
    await expect(page.getByText(/Permission matrix pending owner sign-off/)).toBeVisible()

    const roles = page.getByRole('table', { name: 'Roles and grants' })
    for (const name of ['Owner', 'Inventory Manager', 'Sales Executive', 'Accountant', 'Viewer']) {
      await expect(roles.getByText(name, { exact: true })).toBeVisible()
    }
    const catalogue = page.getByRole('table', { name: 'Permission catalogue' })
    for (const key of [
      'catalogue.cost_price.view',
      'sales.refund.approve',
      'stock.adjustment.approve',
      'access.roles.manage',
      'settings.invoice.manage',
      'reports.export',
    ]) {
      await expect(catalogue.getByText(key, { exact: true })).toBeVisible()
    }

    // Grant statuses must be readable without scrolling sideways (UI guidelines).
    const viewport = page.viewportSize()?.width ?? 0
    for (const status of await page
      .getByText(/None — pending sign-off|Proposed — not approved/)
      .all()) {
      const box = await status.boundingBox()
      expect(box && box.x + box.width).toBeLessThanOrEqual(viewport)
    }

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(0)
  })
})
