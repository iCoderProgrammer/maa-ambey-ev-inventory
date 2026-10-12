import type { RouteObject } from 'react-router'
import { AppShell } from '@/components/layout/AppShell'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { NotFoundPage } from './NotFoundPage'
import { PlannedModulePage } from './PlannedModulePage'
import { RouteErrorPage } from './RouteErrorPage'
import { plannedModules } from './navigation'

export const routes: RouteObject[] = [
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
]
