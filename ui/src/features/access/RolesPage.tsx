import { FlaskConical, Hourglass } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Table } from '@/components/ui/Table'
import { Permission, type RoleMatrix } from './api/access'
import { RequirePermission } from './components/RequirePermission'
import { useMe, useRoleMatrix } from './hooks/useAccess'

type Role = RoleMatrix['roles'][number]
type PermissionInfo = RoleMatrix['permissions'][number]

function formatDate(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString()
}

function MatrixView() {
  const matrix = useRoleMatrix()
  const me = useMe()

  if (matrix.isPending) return <LoadingState label="Loading roles and permissions…" />
  if (matrix.isError) {
    return (
      <ErrorState
        error={matrix.error}
        title="Could not load roles"
        onRetry={() => void matrix.refetch()}
      />
    )
  }
  const { data } = matrix

  return (
    <div className="flex flex-col gap-6">
      <div
        role="status"
        className="flex items-start gap-3 rounded-card border border-warning/30 bg-warning-subtle p-4 text-sm text-ink"
      >
        <Hourglass className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
        {data.matrixStatus === 'PENDING_OWNER_SIGN_OFF' ? (
          <p>
            <span className="font-semibold">Permission matrix pending owner sign-off.</span> The
            five roles are confirmed, but no permission is approved for any role yet, so every
            protected action is denied until the owner approves the matrix (decision RB-5).
          </p>
        ) : (
          <p>
            <span className="font-semibold">Partially approved.</span> {data.approvedGrantCount}{' '}
            grant(s) are approved by the owner; everything else stays denied until it is approved.
          </p>
        )}
      </div>

      {me.data?.proposedGrantsActive && (
        <div
          role="status"
          className="flex items-start gap-3 rounded-card border border-info/30 bg-info-subtle p-4 text-sm text-ink"
        >
          <FlaskConical className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
          <p>
            <span className="font-semibold">Development mode:</span> proposed (unapproved) grants
            are switched on for testing. Staging and production ignore them.
          </p>
        </div>
      )}

      <Card title="Roles" description="Client-confirmed role identities and their current grants.">
        <Table<Role>
          caption="Roles and grants"
          rows={data.roles}
          rowKey={(r) => r.key}
          columns={[
            {
              key: 'role',
              header: 'Role',
              render: (r) => (
                <div className="sm:min-w-40">
                  <p className="font-medium text-ink">{r.name}</p>
                  <p className="hidden text-xs text-muted sm:block">{r.description}</p>
                </div>
              ),
            },
            {
              key: 'grants',
              header: 'Grants',
              render: (r) =>
                r.grants.length === 0 ? (
                  <Badge tone="warning">None — pending sign-off</Badge>
                ) : (
                  <ul className="flex flex-col gap-1.5">
                    {r.grants.map((g) => (
                      <li key={g.permission} className="flex flex-wrap items-center gap-2">
                        <code className="font-mono text-xs break-all text-ink">{g.permission}</code>
                        {g.status === 'APPROVED' ? (
                          <Badge tone="success">
                            Approved{g.approvedAt ? ` ${formatDate(g.approvedAt)}` : ''}
                          </Badge>
                        ) : (
                          <Badge tone="warning">Proposed — not approved</Badge>
                        )}
                      </li>
                    ))}
                  </ul>
                ),
            },
          ]}
        />
      </Card>

      <Card
        title="Permission catalogue"
        description="Every action the system can protect. A permission is granted only when the owner approves it for a role."
      >
        <Table<PermissionInfo>
          caption="Permission catalogue"
          rows={data.permissions}
          rowKey={(p) => p.key}
          columns={[
            {
              key: 'key',
              header: 'Permission',
              render: (p) => (
                <div className="flex flex-col items-start gap-1">
                  <code className="font-mono text-xs break-all text-ink">{p.key}</code>
                  {/* Narrow screens: sensitivity sits under the key instead of in its own column. */}
                  <span className="sm:hidden">
                    {p.sensitive ? <Badge tone="danger">Sensitive</Badge> : <Badge>Standard</Badge>}
                  </span>
                </div>
              ),
            },
            { key: 'description', header: 'Allows', render: (p) => p.description },
            {
              key: 'sensitive',
              header: 'Sensitivity',
              className: 'hidden sm:table-cell',
              render: (p) =>
                p.sensitive ? <Badge tone="danger">Sensitive</Badge> : <Badge>Standard</Badge>,
            },
          ]}
        />
      </Card>
    </div>
  )
}

/** Read-only view of roles, grants and their approval status (B-07). */
export function RolesPage() {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Users & Roles"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Admin' }, { label: 'Users & Roles' }]}
        description="Roles, permission grants and their approval status. Read-only: grants and accounts are changed by an administrator with the audited command-line tools."
      />
      <RequirePermission
        permission={Permission.AccessRolesView}
        what="the role and permission matrix"
      >
        <MatrixView />
      </RequirePermission>
    </div>
  )
}
