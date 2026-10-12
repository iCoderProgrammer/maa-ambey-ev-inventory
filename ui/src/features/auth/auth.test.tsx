import { onlineManager } from '@tanstack/react-query'
import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getCsrfToken, setCsrfToken } from '@/lib/api/csrf'
import { renderApp } from '@/test/render'

type Handler = (init: RequestInit | undefined) => Response | Promise<Response>

const sessionBody = {
  data: {
    user: { id: 'u-1', username: 'asha', displayName: 'Asha Verma' },
    csrfToken: 'csrf-from-server',
    expiresAt: '2026-10-12T22:00:00Z',
  },
}

function errorBody(code: string, message: string, requestId = 'req-1') {
  return { error: { code, message, requestId } }
}

/** Fetch mock keyed by "METHOD /path"; health endpoints answer by default. */
function mockApi(handlers: Record<string, Handler>) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const path = url.replace(/^.*\/api\/v1/, '')
    const handler = handlers[`${init?.method ?? 'GET'} ${path}`]
    if (handler) return Promise.resolve(handler(init))
    if (path === '/health') {
      return Promise.resolve(
        Response.json({
          data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
        }),
      )
    }
    if (path === '/ready') {
      return Promise.resolve(
        Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
      )
    }
    return Promise.resolve(Response.json(errorBody('NOT_FOUND', 'Not found'), { status: 404 }))
  })
}

const anonymous = () =>
  Response.json(errorBody('UNAUTHENTICATED', 'Please sign in to continue.'), { status: 401 })

async function fillAndSubmit(username = 'asha', password = 'correct horse battery') {
  const user = userEvent.setup()
  await user.type(await screen.findByLabelText(/Username/), username)
  await user.type(screen.getByLabelText(/^Password/), password)
  await user.click(screen.getByRole('button', { name: 'Sign in' }))
  return user
}

afterEach(() => {
  setCsrfToken(null)
  onlineManager.setOnline(true)
})

describe('route protection', () => {
  it('sends visitors without a session to sign in instead of showing the app', async () => {
    mockApi({ 'GET /auth/session': anonymous })
    const { router } = renderApp('/inventory/vehicles', { session: null })

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    expect(screen.queryByRole('navigation', { name: 'Main' })).not.toBeInTheDocument()
    expect(screen.queryByText(/session has ended/)).not.toBeInTheDocument()
  })

  it('explains that an ended session needs a new sign-in', async () => {
    mockApi({
      'GET /auth/session': () =>
        Response.json(errorBody('SESSION_EXPIRED', 'Your session has ended.'), { status: 401 }),
    })
    renderApp('/', { session: null })

    expect(await screen.findByText(/Your session has ended. Sign in again/)).toBeInTheDocument()
  })

  it('does not guess when the session cannot be checked', async () => {
    mockApi({
      'GET /auth/session': () =>
        Response.json(errorBody('INTERNAL_ERROR', 'Something went wrong.', 'req-500'), {
          status: 500,
        }),
    })
    renderApp('/', { session: null })

    expect(await screen.findByText('Could not check your session')).toBeInTheDocument()
    expect(screen.getByText('req-500')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Sign in' })).not.toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Main' })).not.toBeInTheDocument()
  })

  it('ends the session in the UI when a protected call answers 401', async () => {
    mockApi({
      'GET /health': () =>
        Response.json(errorBody('SESSION_EXPIRED', 'Your session has ended.'), { status: 401 }),
    })
    const { router } = renderApp('/')

    expect(await screen.findByText(/Your session has ended. Sign in again/)).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    expect(getCsrfToken()).toBeNull()
  })

  it('keeps a confirmed session usable when a later session check fails', async () => {
    mockApi({
      'GET /auth/session': () => Promise.reject(new TypeError('Failed to fetch')),
    })
    const { queryClient } = renderApp('/')
    expect(screen.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()

    await act(() => queryClient.refetchQueries({ queryKey: ['auth', 'session'] }))

    expect(screen.queryByText('Could not check your session')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()
  })

  it('shows the sign-in form with a notice instead of spinning while offline', () => {
    const fetchMock = mockApi({})
    onlineManager.setOnline(false)
    renderApp('/login', { session: null })

    expect(screen.queryByText('Checking your session…')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByText(/No network connection/)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('sends a signed-in user away from the sign-in page', async () => {
    mockApi({})
    const { router } = renderApp('/login')

    expect(await screen.findByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })
})

describe('sign in', () => {
  it('signs in with the server session and returns to the requested page', async () => {
    const fetchMock = mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () => Response.json(sessionBody),
    })
    const { router } = renderApp('/inventory/vehicles', { session: null })

    await fillAndSubmit('  asha ', 'correct horse battery')

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Vehicle Units' }),
    ).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/inventory/vehicles')
    const loginCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(JSON.parse(loginCall[1]!.body as string)).toEqual({
      username: 'asha',
      password: 'correct horse battery',
    })
    expect(getCsrfToken()).toBe('csrf-from-server')
    // Only UI preferences (e.g. theme) may live in web storage; never auth data.
    const stored = [window.localStorage, window.sessionStorage]
      .flatMap((store) => Object.keys(store).map((k) => `${k}=${store.getItem(k) ?? ''}`))
      .join('\n')
    expect(stored).not.toMatch(/csrf|session|token|password|asha/i)
  })

  it('validates empty input without calling the API', async () => {
    const fetchMock = mockApi({ 'GET /auth/session': anonymous })
    renderApp('/login', { session: null })
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Sign in' }))

    expect(await screen.findByText('Enter your username.')).toBeInTheDocument()
    expect(screen.getByText('Enter your password.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'POST')).toBe(false)
  })

  it('shows a generic failure for wrong credentials and clears the password', async () => {
    mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () =>
        Response.json(errorBody('INVALID_CREDENTIALS', 'Incorrect username or password.'), {
          status: 401,
        }),
    })
    const { router } = renderApp('/login', { session: null })
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Incorrect username or password')
    expect(screen.getByLabelText(/^Password/)).toHaveValue('')
    expect(screen.getByLabelText(/Username/)).toHaveValue('asha')
    expect(router.state.location.pathname).toBe('/login')
  })

  it('tells a disabled user their account is disabled', async () => {
    mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () =>
        Response.json(
          errorBody('ACCOUNT_DISABLED', 'This account is disabled. Contact your administrator.'),
          { status: 403 },
        ),
    })
    renderApp('/login', { session: null })
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Account disabled')
    expect(alert).toHaveTextContent('Contact your administrator')
  })

  it('explains rate limiting with the wait time from Retry-After', async () => {
    mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () =>
        Response.json(errorBody('RATE_LIMITED', 'Too many sign-in attempts.'), {
          status: 429,
          headers: { 'Retry-After': '120' },
        }),
    })
    renderApp('/login', { session: null })
    await fillAndSubmit()

    expect(await screen.findByRole('alert')).toHaveTextContent('Try again in about 2 minutes.')
  })

  it('reports a network failure without pretending to sign in', async () => {
    mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () => Promise.reject(new TypeError('Failed to fetch')),
    })
    const { router } = renderApp('/login', { session: null })
    await fillAndSubmit()

    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    expect(router.state.location.pathname).toBe('/login')
  })

  it('sends one request even when submitted twice', async () => {
    let release: (r: Response) => void = () => undefined
    const fetchMock = mockApi({
      'GET /auth/session': anonymous,
      'POST /auth/login': () => new Promise<Response>((resolve) => (release = resolve)),
    })
    renderApp('/login', { session: null })
    const user = await fillAndSubmit()

    const pending = await screen.findByRole('button', { name: /Signing in/ })
    expect(pending).toBeDisabled()
    await user.keyboard('{Enter}')
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
    release(Response.json(sessionBody))
    expect(await screen.findByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()
  })

  it('lets the user show and hide the password', async () => {
    mockApi({ 'GET /auth/session': anonymous })
    renderApp('/login', { session: null })
    const user = userEvent.setup()
    const password = await screen.findByLabelText(/^Password/)

    expect(password).toHaveAttribute('type', 'password')
    await user.click(screen.getByRole('button', { name: 'Show password' }))
    expect(password).toHaveAttribute('type', 'text')
    expect(screen.getByRole('button', { name: 'Hide password' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })
})

describe('account menu and sign out', () => {
  it('shows the signed-in user and signs out only after the server confirms', async () => {
    const fetchMock = mockApi({
      'POST /auth/logout': () => Response.json({ data: { signedOut: true } }),
    })
    const { router } = renderApp('/')
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Account: Test User' }))
    const panel = screen.getByRole('dialog', { name: 'Account' })
    expect(within(panel).getByText('test.user')).toBeInTheDocument()
    await user.click(within(panel).getByRole('button', { name: 'Sign out' }))

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
    const logoutCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect((logoutCall[1]!.headers as Record<string, string>)['X-CSRF-Token']).toBe(
      'test-csrf-token',
    )
    expect(getCsrfToken()).toBeNull()
  })

  it('keeps the user signed in and says so when sign-out fails', async () => {
    mockApi({
      'POST /auth/logout': () =>
        Response.json(errorBody('INTERNAL_ERROR', 'Something went wrong.'), { status: 500 }),
    })
    const { router } = renderApp('/')
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Account: Test User' }))
    await user.click(screen.getByRole('button', { name: 'Sign out' }))

    expect(await screen.findByText(/You are still signed in/)).toBeInTheDocument()
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/')
    })
  })
})
