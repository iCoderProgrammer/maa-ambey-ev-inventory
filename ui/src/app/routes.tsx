import type { RouteObject } from 'react-router'
import { AppShell } from '@/components/layout/AppShell'
import { RolesPage } from '@/features/access/RolesPage'
import { DocumentsPage } from '@/features/attachments/DocumentsPage'
import { RequireAuth } from '@/features/auth/components/RequireAuth'
import { LoginPage } from '@/features/auth/LoginPage'
import { CatalogueLayout } from '@/features/catalogue/CatalogueLayout'
import {
  BrandDetailPage,
  ModelDetailPage,
  VariantDetailPage,
} from '@/features/catalogue/pages/DetailPages'
import { BrandsPage, ModelsPage, VariantsPage } from '@/features/catalogue/pages/ListPages'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { SupplierDetailPage, SuppliersPage } from '@/features/suppliers/SupplierPages'
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
              { path: 'admin/users', element: <RolesPage /> },
              { path: 'documents', element: <DocumentsPage /> },
              { path: 'procurement/suppliers', element: <SuppliersPage /> },
              { path: 'procurement/suppliers/:supplierId', element: <SupplierDetailPage /> },
              {
                path: 'admin/catalogue',
                element: <CatalogueLayout />,
                children: [
                  { index: true, element: <BrandsPage /> },
                  { path: 'models', element: <ModelsPage /> },
                  { path: 'variants', element: <VariantsPage /> },
                  { path: 'brands/:brandId', element: <BrandDetailPage /> },
                  { path: 'models/:modelId', element: <ModelDetailPage /> },
                  { path: 'variants/:variantId', element: <VariantDetailPage /> },
                ],
              },
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
