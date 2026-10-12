import { Link, isRouteErrorResponse, useRouteError } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { Card } from '@/components/ui/Card'
import { ErrorState } from '@/components/ui/StateViews'

/**
 * Rendered inside the shell when a page throws while rendering or loading.
 * Shows a safe message only; details stay in the browser console for developers.
 */
export function RouteErrorPage() {
  const error = useRouteError()
  const title = isRouteErrorResponse(error) ? `Error ${String(error.status)}` : 'Page error'

  return (
    <>
      <PageHeader title={title} breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Error' }]} />
      <Card>
        <ErrorState
          error={error}
          title="This page could not be displayed"
          onRetry={() => {
            window.location.reload()
          }}
        />
        <p className="mt-4 text-sm text-muted">
          Nothing was saved or changed. Reload the page, or go back to the{' '}
          <Link to="/" className="text-primary underline">
            dashboard
          </Link>
          .
        </p>
      </Card>
    </>
  )
}
