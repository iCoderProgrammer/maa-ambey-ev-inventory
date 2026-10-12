import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/StateViews'
import { PageHeader } from '@/components/layout/PageHeader'
import { SystemStatusCard } from '@/features/system/components/SystemStatusCard'

export function DashboardPage() {
  return (
    <>
      <PageHeader
        title="Dashboard"
        breadcrumbs={[{ label: 'Home' }]}
        description="Showroom overview across all brands you are permitted to see."
      />
      <div className="grid gap-6 lg:grid-cols-3">
        <Card title="Inventory overview" className="lg:col-span-2">
          {/* Real KPIs (saleable, reserved, in PDI/hold, in transit, sold) arrive with backlog task G-01. */}
          <EmptyState title="Stock metrics are not available yet">
            Saleable, reserved, in-PDI and in-transit counts will appear here once the vehicle
            registry and dashboard metrics (backlog G-01) are implemented. No sample figures are
            shown.
          </EmptyState>
        </Card>
        <SystemStatusCard />
      </div>
    </>
  )
}
