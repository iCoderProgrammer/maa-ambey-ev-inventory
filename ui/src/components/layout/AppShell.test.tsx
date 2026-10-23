import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AppProviders } from '@/app/AppProviders'
import { createQueryClient } from '@/app/queryClient'
import { RouteErrorPage } from '@/app/RouteErrorPage'
import { createStore } from '@/store'
import { renderApp, seedSession, testSession } from '@/test/render'
import { AppShell } from './AppShell'

beforeEach(() => {
  // The shell polls /health and /ready; these tests do not depend on the result.
  vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'))
})

describe('mobile navigation drawer', () => {
  it('opens as a modal dialog, traps focus and returns focus on Escape', async () => {
    const user = userEvent.setup()
    renderApp('/')

    const opener = screen.getByRole('button', { name: 'Open navigation' })
    await user.click(opener)
    const dialog = screen.getByRole('dialog', { name: 'Navigation' })
    expect(opener).toHaveAttribute('aria-expanded', 'true')
    const close = within(dialog).getByRole('button', { name: 'Close navigation' })
    expect(close).toHaveFocus()
    // Modal: the rest of the app is hidden from assistive technology while open.
    expect(screen.queryByRole('main')).not.toBeInTheDocument()

    // Shift+Tab from the first control wraps to the last link inside the drawer.
    await user.tab({ shift: true })
    const links = within(dialog).getAllByRole('link')
    expect(links[links.length - 1]).toHaveFocus()
    await user.tab()
    expect(close).toHaveFocus()

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', { name: 'Navigation' })).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
    expect(screen.getByRole('main')).toBeInTheDocument()
  })

  it('closes after navigating and moves focus to the new page', async () => {
    const user = userEvent.setup()
    const { router } = renderApp('/')

    await user.click(screen.getByRole('button', { name: 'Open navigation' }))
    const dialog = screen.getByRole('dialog', { name: 'Navigation' })
    await user.click(within(dialog).getByRole('link', { name: /Customers/ }))

    expect(router.state.location.pathname).toBe('/sales/customers')
    expect(screen.queryByRole('dialog', { name: 'Navigation' })).not.toBeInTheDocument()
    expect(screen.getByRole('main')).toHaveFocus()
    expect(screen.getByRole('heading', { level: 1, name: 'Customers' })).toBeInTheDocument()
  })
})

describe('sidebar', () => {
  it('keeps planned modules labelled as planned when collapsed', async () => {
    const user = userEvent.setup()
    renderApp('/')

    await user.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    const nav = screen.getByRole('navigation', { name: 'Main' })
    const vehicles = within(nav).getByRole('link', { name: /Vehicle Units/ })
    expect(vehicles).toHaveAccessibleName('Vehicle Units, planned')
    const dashboard = within(nav).getByRole('link', { name: 'Dashboard' })
    expect(dashboard).toHaveAttribute('aria-current', 'page')

    // Regression: the tooltip Slot once stringified NavLink's className function.
    expect(vehicles.className).not.toContain('=>')
    expect(vehicles).toHaveClass('flex', 'justify-center')
    expect(dashboard).toHaveClass('bg-white/10')

    // Keyboard focus shows the visual label as a tooltip (title attributes do not).
    vehicles.focus()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Vehicle Units · Planned')
  })

  it('shows no tooltips while the sidebar is expanded', () => {
    renderApp('/')
    within(screen.getByRole('navigation', { name: 'Main' }))
      .getByRole('link', { name: /Vehicle Units/ })
      .focus()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('moves focus to the page content after keyboard navigation', async () => {
    const user = userEvent.setup()
    renderApp('/')

    const link = within(screen.getByRole('navigation', { name: 'Main' })).getByRole('link', {
      name: /Goods Receipts/,
    })
    link.focus()
    await user.keyboard('{Enter}')
    expect(
      screen.getByRole('heading', { level: 1, name: 'Goods Receipts & PDI' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('main')).toHaveFocus()
    expect(document.title).toBe('Goods Receipts & PDI · Maa Ambey Inventory')
  })
})

describe('route errors', () => {
  it('renders a safe error page inside the shell when a page throws', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    function Broken(): never {
      throw new Error('secret stack detail')
    }
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppShell />,
          children: [
            { errorElement: <RouteErrorPage />, children: [{ index: true, element: <Broken /> }] },
          ],
        },
      ],
      { initialEntries: ['/'] },
    )
    const queryClient = createQueryClient()
    queryClient.setDefaultOptions({ queries: { retry: false, refetchInterval: false } })
    seedSession(queryClient, testSession)
    render(
      <AppProviders store={createStore()} queryClient={queryClient}>
        <RouterProvider router={router} />
      </AppProviders>,
    )

    expect(screen.getByRole('heading', { level: 1, name: 'Page error' })).toBeInTheDocument()
    expect(screen.getByText('This page could not be displayed')).toBeInTheDocument()
    expect(screen.queryByText(/secret stack detail/)).not.toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Main' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })
})
