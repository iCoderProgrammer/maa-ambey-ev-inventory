import { expect, type Page } from '@playwright/test'

/**
 * Test accounts come from the environment, never from the repository. Create
 * them with `make user ARGS="create ..."` on a non-production database.
 */
export type AccountKind =
  | 'active'
  | 'disabled'
  | 'roles'
  | 'attachments'
  | 'attachmentsViewer'
  | 'catalogue'
  | 'catalogueViewer'

const prefixes: Record<AccountKind, string> = {
  active: 'E2E',
  disabled: 'E2E_DISABLED',
  // Optional: a user holding access.roles.view (see DEVELOPMENT_SETUP "End-to-end browser tests").
  roles: 'E2E_ROLES',
  // Optional (B-09): a user holding attachments.upload + attachments.view, and a second user
  // holding attachments.view only (see DEVELOPMENT_SETUP "End-to-end browser tests").
  attachments: 'E2E_ATTACH',
  attachmentsViewer: 'E2E_ATTACH_VIEWER',
  // Optional (C-02): a user holding every catalogue.* key, and one holding catalogue.view only.
  catalogue: 'E2E_CATALOGUE',
  catalogueViewer: 'E2E_CATALOGUE_VIEWER',
}

export function hasAccount(kind: AccountKind): boolean {
  return Boolean(
    process.env[`${prefixes[kind]}_USERNAME`] && process.env[`${prefixes[kind]}_PASSWORD`],
  )
}

export function account(kind: AccountKind): { username: string; password: string } {
  const prefix = prefixes[kind]
  const username = process.env[`${prefix}_USERNAME`]
  const password = process.env[`${prefix}_PASSWORD`]
  if (!username || !password) {
    throw new Error(`Set ${prefix}_USERNAME and ${prefix}_PASSWORD for the end-to-end tests.`)
  }
  return { username, password }
}

export async function fillLogin(page: Page, username: string, password: string) {
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password', { exact: false }).first().fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
}

/** Signs in through the real form and waits for the app shell. */
export async function signIn(page: Page, path = '/', kind: AccountKind = 'active') {
  const { username, password } = account(kind)
  await page.goto(path)
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  await fillLogin(page, username, password)
  await expect(page.getByRole('button', { name: /^Account: / })).toBeVisible()
}

/** Fails the test on uncaught page errors and console errors other than expected HTTP statuses. */
export function watchConsole(page: Page): string[] {
  const problems: string[] = []
  page.on('pageerror', (e) => problems.push(`pageerror: ${e.message}`))
  page.on('console', (m) => {
    // Chrome logs every non-2xx fetch as "Failed to load resource"; those are asserted by the tests.
    if (m.type() === 'error' && !m.text().includes('Failed to load resource')) {
      problems.push(`console: ${m.text()}`)
    }
  })
  return problems
}
