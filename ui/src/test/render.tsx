import { render } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AppProviders } from '@/app/AppProviders'
import { routes } from '@/app/routes'
import { createQueryClient } from '@/app/queryClient'
import { createStore } from '@/store'

/** Renders the full app at `path` with fresh store and query client. */
export function renderApp(path = '/') {
  const queryClient = createQueryClient()
  queryClient.setDefaultOptions({ queries: { retry: false, refetchInterval: false } })
  const router = createMemoryRouter(routes, { initialEntries: [path] })
  return {
    router,
    ...render(
      <AppProviders store={createStore()} queryClient={queryClient}>
        <RouterProvider router={router} />
      </AppProviders>,
    ),
  }
}
