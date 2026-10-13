import { FileLock2, Hourglass } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Card } from '@/components/ui/Card'
import { AccessDeniedState, EmptyState, ErrorState, LoadingState } from '@/components/ui/StateViews'
import { Permission } from '@/features/access/api/access'
import { useMe } from '@/features/access/hooks/useAccess'
import { AttachmentUploader } from './components/AttachmentUploader'
import { DocumentRow } from './components/DocumentRow'
import { useAttachmentLimits, useMyAttachments } from './hooks/useAttachments'

function UploadCard() {
  const limits = useAttachmentLimits(true)
  return (
    <Card
      title="Upload"
      description="Files are checked by the server (type and size) and stored privately. Nothing gets a public link."
    >
      {limits.isPending && <LoadingState label="Loading upload limits…" />}
      {limits.isError && (
        <ErrorState
          error={limits.error}
          title="Could not load upload limits"
          onRetry={() => void limits.refetch()}
        />
      )}
      {limits.data && <AttachmentUploader limits={limits.data} />}
    </Card>
  )
}

function MyUploadsCard({ canUpload }: { canUpload: boolean }) {
  const list = useMyAttachments(true)
  return (
    <Card
      title="My uploads"
      description="Files you uploaded that are not attached to a record yet. Only you can open them."
      badge={
        list.data ? (
          <span className="text-xs text-muted">
            {list.data.length} {list.data.length === 1 ? 'file' : 'files'}
          </span>
        ) : undefined
      }
      contentClassName="p-0"
    >
      {list.isPending && (
        <div className="px-5">
          <LoadingState label="Loading your uploads…" />
        </div>
      )}
      {list.isError && (
        <div className="p-5">
          <ErrorState
            error={list.error}
            title="Could not load your uploads"
            onRetry={() => void list.refetch()}
          />
        </div>
      )}
      {list.data?.length === 0 && (
        <EmptyState title="No uploads yet" icon={<FileLock2 className="size-6 text-muted/80" />}>
          {canUpload
            ? 'Files you upload above appear here.'
            : 'You can open files you have uploaded; you cannot upload new ones.'}
        </EmptyState>
      )}
      {list.data && list.data.length > 0 && (
        <ul className="divide-y divide-border" aria-label="My uploads">
          {list.data.map((a) => (
            <DocumentRow key={a.id} attachment={a} />
          ))}
        </ul>
      )}
    </Card>
  )
}

/**
 * Private attachments (B-09): upload with server-side checks and open your own
 * uploads through short-lived links. Linking files to receipts, PDI and sales
 * records comes with those modules.
 */
export function DocumentsPage() {
  const me = useMe()
  const perms = me.data?.permissions ?? []
  const canUpload = perms.includes(Permission.AttachmentsUpload)
  const canView = perms.includes(Permission.AttachmentsView)

  let body
  if (me.isPending) body = <LoadingState label="Checking your permissions…" />
  else if (me.isError && !me.data) {
    body = (
      <ErrorState
        error={me.error}
        title="Could not check your permissions"
        onRetry={() => void me.refetch()}
      />
    )
  } else if (!canUpload && !canView) body = <AccessDeniedState what="documents" />
  else {
    body = (
      <div className="flex flex-col gap-6">
        <div
          role="status"
          className="flex items-start gap-3 rounded-card border border-info/30 bg-info-subtle p-4 text-sm text-ink"
        >
          <Hourglass className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
          <p>
            Accepted types and the size limit are proposals pending owner confirmation. Attaching
            files to goods receipts, PDI and sales records arrives with those modules.
          </p>
        </div>
        {canUpload && <UploadCard />}
        {canView ? (
          <MyUploadsCard canUpload={canUpload} />
        ) : (
          <p role="status" className="text-sm text-muted">
            You can upload files but not open them. Ask the owner or your administrator if you need
            to view documents.
          </p>
        )}
      </div>
    )
  }

  return (
    <>
      <PageHeader
        title="Documents"
        breadcrumbs={[{ label: 'Home', to: '/' }, { label: 'Documents' }]}
        description="Private files such as PDI photos and supplier bills. Every upload and download is checked by the server and recorded in the audit log."
      />
      {body}
    </>
  )
}
