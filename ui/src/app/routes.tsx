import type { RouteObject } from 'react-router'
import { AppShell } from '@/components/layout/AppShell'
import { RequireAuth } from '@/features/auth/components/RequireAuth'
import { LoginPage } from '@/features/auth/LoginPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { NotFoundPage } from './NotFoundPage'
import { PlannedModulePage } from './PlannedModulePage'
import { RouteErrorPage } from './RouteErrorPage'
import { plannedModules } from './navigation'

export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage />, errorElement: <RouteErrorPage /> },
  {
    // Everything else needs a server-confirmed session (B-06).
    element: <RequireAuth />,
    errorElement: <RouteErrorPage />,
    children: [
      {
        path: '/',
        element: <AppShell />,
        children: [
          {
            // Pathless layout route: page errors render inside the shell, keeping navigation usable.
            errorElement: <RouteErrorPage />,
            children: [
              { index: true, element: <DashboardPage /> },
              ...plannedModules.map(({ section, item }) => ({
                path: item.path.slice(1),
                element: <PlannedModulePage section={section} item={item} />,
              })),
              { path: '*', element: <NotFoundPage /> },
            ],
          },
        ],
      },
    ],
  },
]
