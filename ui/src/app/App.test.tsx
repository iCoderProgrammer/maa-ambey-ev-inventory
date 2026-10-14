import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderApp } from '@/test/render'

function mockApi({ dbUp = true }: { dbUp?: boolean } = {}) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    if (url.endsWith('/health')) {
      return Promise.resolve(
        Response.json({
          data: {
            status: 'ok',
            service: 'maa-ambey-inventory-api',
            version: 'abc123',
            time: '2026-10-12T10:00:00Z',
          },
        }),
      )
    }
    if (url.endsWith('/ready')) {
      return Promise.resolve(
        Response.json(
          {
            data: {
              status: dbUp ? 'ready' : 'not_ready',
              checks: { database: dbUp ? 'up' : 'down' },
            },
          },
          { status: dbUp ? 200 : 503 },
        ),
      )
    }
    return Promise.resolve(new Response(null, { status: 404 }))
  })
}

describe('application shell', () => {
  it('renders navigation, header and dashboard', () => {
    mockApi()
    renderApp('/')

    expect(screen.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()
    const nav = screen.getByRole('navigation', { name: 'Main' })
    for (const label of [
      'Vehicle Units',
      'Purchase Orders',
      'Customers',
      'Reports',
      'Quotations',
    ]) {
      expect(within(nav).getByRole('link', { name: new RegExp(label) })).toBeInTheDocument()
    }
    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main')
  })

  it('shows live API status from the health endpoints', async () => {
    const fetchMock = mockApi()
    renderApp('/')

    expect(await screen.findByText('abc123')).toBeInTheDocument()
    expect(screen.getByText('Ready')).toBeInTheDocument()
    expect(screen.getAllByText(/online/i).length).toBeGreaterThan(0)
    const urls = fetchMock.mock.calls.map(([u]) => u as string)
    expect(urls).toContain('/api/v1/health')
    expect(urls).toContain('/api/v1/ready')
  })

  it('reports an unavailable database when /ready returns 503', async () => {
    mockApi({ dbUp: false })
    renderApp('/')

    expect(await screen.findByText('Unavailable')).toBeInTheDocument()
    expect(screen.getByText('Database unavailable')).toBeInTheDocument()
  })

  it('shows an error with retry when the API is unreachable', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'))
    renderApp('/')

    expect(await screen.findByText('API unreachable')).toBeInTheDocument()
    expect(screen.getByText('API offline')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('marks planned modules honestly instead of faking workflows', () => {
    mockApi()
    renderApp('/inventory/vehicles')

    expect(screen.getByRole('heading', { level: 1, name: 'Vehicle Units' })).toBeInTheDocument()
    expect(screen.getByText('This module is not implemented yet.')).toBeInTheDocument()
    const crumbs = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(within(crumbs).getByText('Inventory')).toBeInTheDocument()
    expect(
      within(screen.getByRole('list', { name: 'Backlog tasks' })).getByText('D-04'),
    ).toBeInTheDocument()
  })

  it('navigates via the sidebar and renders a not-found page', async () => {
    mockApi()
    const user = userEvent.setup()
    const { router } = renderApp('/nope')

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument()
    await user.click(screen.getByRole('link', { name: /Purchase Orders/ }))
    expect(router.state.location.pathname).toBe('/procurement/purchase-orders')
  })

  it('switches between light and dark themes', async () => {
    mockApi()
    const user = userEvent.setup()
    renderApp('/')

    await user.click(screen.getByRole('radio', { name: 'Dark theme' }))
    expect(document.documentElement).toHaveClass('dark')
    await user.click(screen.getByRole('radio', { name: 'Light theme' }))
    expect(document.documentElement).not.toHaveClass('dark')
  })
})
