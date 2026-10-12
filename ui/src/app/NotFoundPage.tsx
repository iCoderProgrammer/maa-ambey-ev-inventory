import { ArrowLeft, SearchX } from 'lucide-react'
import { Link } from 'react-router'
import { PageHeader } from '@/components/layout/PageHeader'
import { buttonVariants } from '@/components/ui/buttonVariants'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/StateViews'

export function NotFoundPage() {
  return (
    <>
      <PageHeader
        title="Page not found"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Not found' }]}
      />
      <Card>
        <EmptyState
          title="We couldn't find that page"
          icon={<SearchX className="size-6" />}
          action={
            <Link to="/" className={buttonVariants({ variant: 'primary', size: 'sm' })}>
              <ArrowLeft className="size-4" aria-hidden="true" />
              Go to dashboard
            </Link>
          }
        >
          The address may be mistyped, or the page may have moved. Use the navigation to find what
          you need.
        </EmptyState>
      </Card>
    </>
  )
}
