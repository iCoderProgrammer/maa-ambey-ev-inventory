import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { ErrorState } from '@/components/ui/StateViews'
import { ApiError } from '@/lib/api/client'
import { renderApp, testMe } from '@/test/render'
import type { Me, RoleMatrix } from './api/access'

const withRolesView: Me = { ...testMe, permissions: ['access.roles.view'] }

const pendingMatrix: RoleMatrix = {
  matrixStatus: 'PENDING_OWNER_SIGN_OFF',
  approvedGrantCount: 0,
  roles: [
    { key: 'owner', name: 'Owner', description: 'Business owner.', grants: [] },
    { key: 'viewer', name: 'Viewer', description: 'Read-only access.', grants: [] },
  ],
  permissions: [
    { key: 'catalogue.cost_price.view', description: 'View cost prices', sensitive: true },
    { key: 'access.roles.view', description: 'View roles and grants', sensitive: false },
  ],
}

function mockApi(roles: () => Response) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    if (url.endsWith('/access/roles')) return Promise.resolve(roles())
    if (url.endsWith('/health')) {
      return Promise.resolve(
        Response.json({
          data: { status: 'ok', service: 'api', version: 'v', time: '2026-10-12T10:00:00Z' },
        }),
      )
    }
    return Promise.resolve(
      Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
    )
  })
}

describe('navigation from the permissions contract', () => {
  it('hides links to pages the server has not granted', () => {
    mockApi(() => Response.json({ data: pendingMatrix }))
    renderApp('/')
    const nav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(nav).queryByRole('link', { name: /Users & Roles/ })).not.toBeInTheDocument()
    expect(within(nav).queryByRole('link', { name: /Catalogue & Brands/ })).not.toBeInTheDocument()
    // Ungated (planned) modules stay listed.
    expect(within(nav).getByRole('link', { name: /Suppliers/ })).toBeInTheDocument()
  })

  it('shows the link once the permission is granted', () => {
    mockApi(() => Response.json({ data: pendingMatrix }))
    renderApp('/', { me: withRolesView })
    const nav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(nav).getByRole('link', { name: /Users & Roles/ })).toBeInTheDocument()
  })
})

describe('roles page', () => {
  it('denies access without the permission and does not call the API', () => {
    const fetchMock = mockApi(() => Response.json({ data: pendingMatrix }))
    renderApp('/admin/users')

    expect(screen.getByText('Access denied')).toBeInTheDocument()
    expect(screen.getByText(/role and permission matrix/)).toBeInTheDocument()
    expect(
      fetchMock.mock.calls.some(([u]) => typeof u === 'string' && u.endsWith('/access/roles')),
    ).toBe(false)
  })

  it('shows confirmed roles with grants clearly pending owner sign-off', async () => {
    mockApi(() => Response.json({ data: pendingMatrix }))
    renderApp('/admin/users', { me: withRolesView })

    expect(await screen.findByText(/Permission matrix pending owner sign-off/)).toBeInTheDocument()
    const roles = screen.getByRole('table', { name: 'Roles and grants' })
    expect(within(roles).getByText('Owner')).toBeInTheDocument()
    expect(within(roles).getAllByText('None — pending sign-off')).toHaveLength(2)
    const catalogue = screen.getByRole('table', { name: 'Permission catalogue' })
    expect(within(catalogue).getByText('catalogue.cost_price.view')).toBeInTheDocument()
    expect(within(catalogue).getAllByText('Sensitive').length).toBeGreaterThan(0)
    expect(screen.queryByText(/Development mode/)).not.toBeInTheDocument()
  })

  it('labels proposed grants as not approved and says when they are active', async () => {
    mockApi(() =>
      Response.json({
        data: {
          ...pendingMatrix,
          roles: [
            {
              key: 'owner',
              name: 'Owner',
              description: 'Business owner.',
              grants: [
                {
                  permission: 'access.roles.view',
                  status: 'PROPOSED',
                  approvedBy: null,
                  approvalReference: null,
                  approvedAt: null,
                },
              ],
            },
          ],
        },
      }),
    )
    renderApp('/admin/users', { me: { ...withRolesView, proposedGrantsActive: true } })

    expect(await screen.findByText('Proposed — not approved')).toBeInTheDocument()
    expect(screen.getByText(/proposed \(unapproved\) grants are switched on/)).toBeInTheDocument()
  })

  it('shows owner-approved grants as approved and reports partial approval', async () => {
    mockApi(() =>
      Response.json({
        data: {
          ...pendingMatrix,
          matrixStatus: 'PARTIALLY_APPROVED',
          approvedGrantCount: 1,
          roles: [
            {
              key: 'owner',
              name: 'Owner',
              description: 'Business owner.',
              grants: [
                {
                  permission: 'reports.export',
                  status: 'APPROVED',
                  approvedBy: 'Owner',
                  approvalReference: 'Signed matrix',
                  approvedAt: '2026-10-20T10:00:00Z',
                },
              ],
            },
          ],
        },
      }),
    )
    renderApp('/admin/users', { me: withRolesView })

    expect(await screen.findByText(/Partially approved/)).toBeInTheDocument()
    expect(screen.getByText(/1 grant\(s\) are approved/)).toBeInTheDocument()
    expect(screen.getByText(/^Approved/)).toBeInTheDocument()
    expect(screen.queryByText(/Proposed — not approved/)).not.toBeInTheDocument()
  })

  it('shows access denied when the server refuses, even if the UI thought it was allowed', async () => {
    mockApi(() =>
      Response.json(
        {
          error: {
            code: 'FORBIDDEN',
            message: 'You do not have permission to do this.',
            requestId: 'req-403',
          },
        },
        { status: 403 },
      ),
    )
    renderApp('/admin/users', { me: withRolesView })

    expect(await screen.findByText('Access denied')).toBeInTheDocument()
    expect(screen.getByText('Reference: req-403')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument()
  })
})

describe('permission revoked on the server', () => {
  it('a 403 refreshes the permissions contract so navigation stops offering the page', async () => {
    let meCalls = 0
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (url.endsWith('/access/roles')) {
        return Promise.resolve(
          Response.json(
            { error: { code: 'FORBIDDEN', message: 'You do not have permission to do this.' } },
            { status: 403 },
          ),
        )
      }
      if (url.endsWith('/me')) {
        meCalls++
        return Promise.resolve(Response.json({ data: testMe })) // the grant is gone
      }
      return Promise.resolve(
        Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
      )
    })
    // The UI still believes the permission is granted (cached /me).
    renderApp('/admin/users', { me: withRolesView })

    await waitFor(() => {
      expect(meCalls).toBeGreaterThan(0)
    })
    const nav = screen.getByRole('navigation', { name: 'Main' })
    await waitFor(() => {
      expect(within(nav).queryByRole('link', { name: /Users & Roles/ })).not.toBeInTheDocument()
    })
    expect(screen.getByText('Access denied')).toBeInTheDocument()
  })
})

describe('account menu', () => {
  it('shows roles, location scope and that nothing is granted yet', async () => {
    mockApi(() => Response.json({ data: pendingMatrix }))
    renderApp('/', {
      me: {
        ...testMe,
        roles: [{ key: 'viewer', name: 'Viewer' }],
        locationScope: {
          all: false,
          locations: [{ id: 'l1', code: 'SR1', name: 'Main Showroom' }],
        },
      },
    })
    await userEvent.setup().click(screen.getByRole('button', { name: 'Account: Test User' }))
    const panel = screen.getByRole('dialog', { name: 'Account' })
    expect(within(panel).getByText('Viewer')).toBeInTheDocument()
    expect(within(panel).getByText('Main Showroom')).toBeInTheDocument()
    expect(within(panel).getByText(/pending the owner/)).toBeInTheDocument()
  })
})

describe('error state for 403 responses', () => {
  it('treats only FORBIDDEN as access denied', () => {
    const { rerender } = render(
      <ErrorState error={new ApiError(403, { code: 'FORBIDDEN', message: 'No.' })} />,
    )
    expect(screen.getByText('Access denied')).toBeInTheDocument()

    rerender(
      <ErrorState
        error={new ApiError(403, { code: 'CSRF_FAILED', message: 'Reload the page.' })}
        title="Save failed"
      />,
    )
    expect(screen.queryByText('Access denied')).not.toBeInTheDocument()
    expect(screen.getByText('Reload the page.')).toBeInTheDocument()
  })
})
