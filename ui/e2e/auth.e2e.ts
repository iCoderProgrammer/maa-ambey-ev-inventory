import { expect, test } from '@playwright/test'
import { account, fillLogin, signIn, watchConsole } from './support'

test.describe('authentication (B-06)', () => {
  test('a deep link without a session goes to sign-in and returns after it', async ({ page }) => {
    const problems = watchConsole(page)
    await page.goto('/sales/quotations?view=open#top')
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(0)

    const { username, password } = account('active')
    await fillLogin(page, `  ${username.toUpperCase()} `, password)
    await expect(page).toHaveURL(/\/sales\/quotations\?view=open#top$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Quotations' })).toBeVisible()
    expect(problems).toEqual([])
  })

  test('empty submit is validated without calling the API', async ({ page }) => {
    await page.goto('/login')
    let posted = false
    page.on('request', (r) => {
      if (r.url().includes('/auth/login')) posted = true
    })
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByText('Enter your username.')).toBeVisible()
    await expect(page.getByText('Enter your password.')).toBeVisible()
    expect(posted).toBe(false)
  })

  test('a wrong password gives a generic error and clears the password', async ({ page }) => {
    await page.goto('/login')
    await fillLogin(page, account('active').username, 'definitely-not-the-password')
    await expect(page.getByRole('alert')).toContainText('Incorrect username or password')
    await expect(page.getByLabel('Password', { exact: false }).first()).toHaveValue('')
    await expect(page.getByLabel('Password', { exact: false }).first()).toBeFocused()
    await expect(page).toHaveURL(/\/login$/)
  })

  test('a disabled account cannot sign in', async ({ page }) => {
    const { username, password } = account('disabled')
    await page.goto('/login')
    await fillLogin(page, username, password)
    await expect(page.getByRole('alert')).toContainText('Account disabled')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('the session survives a reload and stays out of reach of scripts', async ({
    page,
    context,
  }) => {
    await signIn(page)
    await page.reload()
    await expect(page.getByRole('button', { name: /^Account: / })).toBeVisible()

    const cookie = (await context.cookies()).find((c) => c.name.endsWith('maa_session'))
    expect(cookie).toBeDefined()
    expect(cookie?.httpOnly).toBe(true)
    expect(cookie?.sameSite).toBe('Strict')
    expect(cookie?.expires).toBe(-1) // browser-session cookie

    const exposed = await page.evaluate(() => ({
      cookie: document.cookie.includes('maa_session'),
      storage: [localStorage, sessionStorage]
        .flatMap((s) => Object.keys(s).map((k) => `${k}=${s.getItem(k) ?? ''}`))
        .join('\n'),
    }))
    expect(exposed.cookie).toBe(false)
    expect(exposed.storage).not.toMatch(/csrf|session|token|password/i)
  })

  test('the sign-in page sends a signed-in user to the app', async ({ page }) => {
    await signIn(page)
    await page.goto('/login')
    await expect(page).toHaveURL(/\/$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
  })

  test('sign-out ends the session on the server', async ({ page }) => {
    await signIn(page)
    // A protected page in the history, so Back after sign-out has somewhere to go.
    await page.goto('/inventory/vehicles')
    await expect(page.getByRole('heading', { level: 1, name: 'Vehicle Units' })).toBeVisible()
    await page.getByRole('button', { name: /^Account: / }).click()
    await page.getByRole('button', { name: 'Sign out' }).click()
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByText(/Your session has ended/)).toHaveCount(0)

    await page.goBack()
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
    await expect(page.getByRole('heading', { level: 1, name: 'Vehicle Units' })).toHaveCount(0)
    const res = await page.request.get('/api/v1/auth/session')
    expect(res.status()).toBe(401)
  })

  test('a session the server no longer accepts sends the user to sign in', async ({
    page,
    context,
  }) => {
    await signIn(page, '/inventory/vehicles')
    const cookie = (await context.cookies()).find((c) => c.name.endsWith('maa_session'))
    if (!cookie) throw new Error('no session cookie')
    await context.addCookies([{ ...cookie, value: 'revoked-or-forged-token' }])

    await page.reload()
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByText('Your session has ended. Sign in again to continue.')).toBeVisible()
    // Signing in again returns to where the user was.
    await fillLogin(page, account('active').username, account('active').password)
    await expect(page).toHaveURL(/\/inventory\/vehicles$/)
  })

  test('mutations need a session and the CSRF token', async ({ page }) => {
    const anon = await page.request.post('/api/v1/auth/logout')
    expect(anon.status()).toBe(401)

    await signIn(page)
    const noCsrf = await page.request.post('/api/v1/auth/logout')
    expect(noCsrf.status()).toBe(403)
    const body = (await noCsrf.json()) as { error: { code: string } }
    expect(body.error.code).toBe('CSRF_FAILED')
    // Still signed in: the forged request changed nothing.
    expect((await page.request.get('/api/v1/auth/session')).status()).toBe(200)
  })

  test('a temporary API outage does not throw a signed-in user out', async ({ page }) => {
    await page.clock.install()
    await signIn(page)
    await expect(page.getByRole('banner').getByText(/online/)).toBeVisible()

    await page.route('**/api/v1/**', (route) => route.abort('connectionrefused'))
    // Past the periodic session re-check (5 min) and the retries.
    await page.clock.runFor('06:00')
    await page.clock.runFor('01:00')

    await expect(page.getByText('Could not check your session')).toHaveCount(0)
    await expect(page.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeVisible()
    await expect(page.getByRole('banner').getByText('API offline')).toBeVisible()

    await page.unroute('**/api/v1/**')
    await page.clock.runFor('01:00')
    await expect(page.getByRole('banner').getByText(/online/)).toBeVisible()
  })

  test('repeated failures are throttled with a wait time', async ({ page }) => {
    const username = `throttle-${String(Date.now())}`
    await page.goto('/login')
    for (let i = 0; i < 5; i++) {
      await fillLogin(page, username, 'wrong-password-123')
      await expect(page.getByRole('alert')).toContainText('Incorrect username or password')
    }
    await fillLogin(page, username, 'wrong-password-123')
    await expect(page.getByRole('alert')).toContainText('Too many sign-in attempts')
    await expect(page.getByRole('alert')).toContainText(/Try again in about \d+ minutes/)
  })
})
