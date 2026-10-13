import { onlineManager } from '@tanstack/react-query'
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppProviders } from '@/app/AppProviders'
import { createQueryClient } from '@/app/queryClient'
import { routes } from '@/app/routes'
import { createStore } from '@/store'
import { renderApp, seedSession, testSession } from '@/test/render'

/** Production query defaults, with a signed-in session already confirmed. */
function signedInClient() {
  const client = createQueryClient()
  seedSession(client, testSession)
  return client
}

function statusCard() {
  return screen.getByRole('heading', { name: 'System status' }).closest('section') ?? document.body
}

afterEach(() => {
  onlineManager.setOnline(true)
})

describe('system status widgets', () => {
  it('shows loading states while the API has not answered', () => {
    vi.spyOn(globalThis, 'fetch').mockReturnValue(new Promise<Response>(() => undefined))
    renderApp('/')

    expect(screen.getByText('Checking API…')).toBeInTheDocument()
    expect(within(statusCard()).getByRole('status')).toHaveTextContent('Contacting API…')
  })

  it('explains an offline browser instead of spinning indefinitely', () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    onlineManager.setOnline(false)
    renderApp('/')

    expect(screen.getByText('Offline')).toBeInTheDocument()
    const card = statusCard()
    expect(within(card).getByRole('alert')).toHaveTextContent(
      'No network connection. The API status cannot be checked.',
    )
    expect(within(card).queryByRole('status')).not.toBeInTheDocument()
    expect(within(card).getByRole('button', { name: 'Refresh' })).toBeDisabled()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('resumes checking the real API when the connection returns', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      return Promise.resolve(
        url.endsWith('/health')
          ? Response.json({
              data: {
                status: 'ok',
                service: 'api',
                version: 'v-test',
                time: '2026-10-12T10:00:00Z',
              },
            })
          : Response.json({ data: { status: 'ready', checks: { database: 'up' } } }),
      )
    })
    onlineManager.setOnline(false)
    renderApp('/')
    expect(fetchMock).not.toHaveBeenCalled()

    act(() => {
      onlineManager.setOnline(true)
    })
    expect(await screen.findByText('v-test')).toBeInTheDocument()
    expect(screen.queryByText('Offline')).not.toBeInTheDocument()
  })

  it('shows the server error message and request ID from the error envelope', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json(
        {
          error: {
            code: 'INTERNAL_ERROR',
            message: 'Something went wrong. Quote the request ID if you contact support.',
            requestId: 'req-12345678',
          },
        },
        { status: 500 },
      ),
    )
    renderApp('/')

    const card = statusCard()
    expect(await within(card).findByText('API unreachable')).toBeInTheDocument()
    expect(within(card).getByText('req-12345678')).toBeInTheDocument()
    expect(screen.getByText('API offline')).toBeInTheDocument()
  })
})

describe('readiness with production query defaults', () => {
  it('shows a database outage from /ready 503 immediately instead of retrying', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      return Promise.resolve(
        url.endsWith('/health')
          ? Response.json({
              data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
            })
          : Response.json(
              { data: { status: 'not_ready', checks: { database: 'down' } } },
              { status: 503 },
            ),
      )
    })
    // Real defaults (retries enabled for transient errors), unlike renderApp.
    const router = createMemoryRouter(routes, { initialEntries: ['/'] })
    render(
      <AppProviders store={createStore()} queryClient={signedInClient()}>
        <RouterProvider router={router} />
      </AppProviders>,
    )

    expect(
      await screen.findByText('Database unavailable', {}, { timeout: 500 }),
    ).toBeInTheDocument()
    const readyCalls = fetchMock.mock.calls.filter(([u]) => (u as string).endsWith('/ready'))
    expect(readyCalls).toHaveLength(1)
  })
})

describe('readiness with a pending database schema', () => {
  it('names the schema as the reason and explains it without blaming the database', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      return Promise.resolve(
        url.endsWith('/health')
          ? Response.json({
              data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
            })
          : Response.json(
              { data: { status: 'not_ready', checks: { database: 'up', migrations: 'down' } } },
              { status: 503 },
            ),
      )
    })
    renderApp('/')

    expect(await screen.findByText('Schema not ready')).toBeInTheDocument()
    expect(screen.queryByText('Database unavailable')).not.toBeInTheDocument()
    const card = statusCard()
    const schemaRow = within(card).getByText('Database schema').closest('tr')
    expect(schemaRow).not.toBeNull()
    expect(within(schemaRow as HTMLElement).getByText('Unavailable')).toBeInTheDocument()
    const dbRow = within(card).getByText('Database').closest('tr')
    expect(within(dbRow as HTMLElement).getByText('Ready')).toBeInTheDocument()
    expect(within(card).getByText(/migrations pending or a migration failed/)).toBeInTheDocument()
  })
})

describe('readiness with the database down', () => {
  it('does not blame the schema when the database itself is unreachable', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      return Promise.resolve(
        url.endsWith('/health')
          ? Response.json({
              data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
            })
          : Response.json(
              { data: { status: 'not_ready', checks: { database: 'down', migrations: 'down' } } },
              { status: 503 },
            ),
      )
    })
    renderApp('/')

    expect(await screen.findByText('Database unavailable')).toBeInTheDocument()
    expect(screen.queryByText(/migrations pending or/)).not.toBeInTheDocument()
  })
})

describe('readiness check failures', () => {
  function mockApi(readyFails: () => boolean) {
    return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (url.endsWith('/health')) {
        return Promise.resolve(
          Response.json({
            data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
          }),
        )
      }
      return readyFails()
        ? Promise.resolve(new Response('', { status: 502 }))
        : Promise.resolve(Response.json({ data: { status: 'ready', checks: { database: 'up' } } }))
    })
  }

  it('reports unknown status, not a database outage, when /ready cannot be checked', async () => {
    mockApi(() => true)
    renderApp('/')

    expect(await screen.findByText('Status unknown')).toBeInTheDocument()
    expect(screen.queryByText('Database unavailable')).not.toBeInTheDocument()
    expect(within(statusCard()).getByText('Readiness check failed')).toBeInTheDocument()
  })

  it('does not keep showing a cached "Ready" after a failed refresh', async () => {
    let fail = false
    mockApi(() => fail)
    const user = userEvent.setup()
    renderApp('/')
    const card = statusCard()
    expect(await within(card).findByText('Ready')).toBeInTheDocument()

    fail = true
    await user.click(within(card).getByRole('button', { name: 'Refresh' }))
    expect(await within(card).findByText('Readiness check failed')).toBeInTheDocument()
    expect(within(card).queryByText('Ready')).not.toBeInTheDocument()
    expect(screen.getByText('Status unknown')).toBeInTheDocument()
  })
})

describe('retrying after a failed check', () => {
  it('shows "Reconnecting…" instead of the last good result while retries are pending', async () => {
    let down = false
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (down) return Promise.resolve(new Response('', { status: 502 }))
      return Promise.resolve(
        url.endsWith('/health')
          ? Response.json({
              data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
            })
          : Response.json({ data: { status: 'ready', checks: { database: 'up' } } }),
      )
    })
    const user = userEvent.setup()
    // Production defaults: transient failures are retried with backoff.
    const router = createMemoryRouter(routes, { initialEntries: ['/'] })
    render(
      <AppProviders store={createStore()} queryClient={signedInClient()}>
        <RouterProvider router={router} />
      </AppProviders>,
    )
    const card = statusCard()
    expect(await within(card).findByText('Ready')).toBeInTheDocument()
    expect(screen.getByText('online')).toBeInTheDocument()

    down = true
    await user.click(within(card).getByRole('button', { name: 'Refresh' }))
    expect(await screen.findByText('Reconnecting…')).toBeInTheDocument()
    expect(within(card).getByRole('status')).toHaveTextContent(
      'The last check failed. Retrying… Values below may be out of date.',
    )
  })
})
