import { Download, Eye, FileText, Image as ImageIcon } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/Button'
import { ApiError } from '@/lib/api/client'
import { formatBytes, type Attachment } from '../api/attachments'
import { downloadErrorMessage, typeLabel } from '../messages'
import { useDownloadAttachment } from '../hooks/useAttachments'
import { ImagePreview } from './ImagePreview'

/**
 * One private file: name, type, size, upload date, and Download / Preview
 * actions. Both actions are authorised by the server each time they are used.
 */
export function DocumentRow({ attachment }: { attachment: Attachment }) {
  const download = useDownloadAttachment(attachment)
  const [previewOpen, setPreviewOpen] = useState(false)
  const Icon = attachment.contentType.startsWith('image/') ? ImageIcon : FileText
  const created = new Date(attachment.createdAt)

  return (
    <li className="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center sm:gap-4">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <span
          className="grid size-9 shrink-0 place-items-center rounded-control bg-surface-muted text-muted"
          aria-hidden="true"
        >
          <Icon className="size-4" />
        </span>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-ink" title={attachment.fileName}>
            {attachment.fileName}
          </p>
          <p className="text-xs text-muted">
            {typeLabel(attachment.contentType)} · {formatBytes(attachment.sizeBytes)} · uploaded{' '}
            <time dateTime={attachment.createdAt}>
              {Number.isNaN(created.getTime()) ? attachment.createdAt : created.toLocaleString()}
            </time>
          </p>
          {download.isError && (
            <p role="alert" className="mt-1 text-xs text-danger">
              {downloadErrorMessage(download.error)}
              {download.error instanceof ApiError && download.error.requestId && (
                <span className="ml-1 font-mono text-muted">
                  Reference: {download.error.requestId}
                </span>
              )}
            </p>
          )}
        </div>
      </div>
      <div className="flex shrink-0 gap-2 sm:justify-end">
        {attachment.previewable && (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              setPreviewOpen(true)
            }}
            aria-label={`Preview ${attachment.fileName}`}
          >
            <Eye className="size-3.5" aria-hidden="true" />
            Preview
          </Button>
        )}
        <Button
          variant="secondary"
          size="sm"
          loading={download.isPending}
          onClick={() => {
            download.mutate()
          }}
          aria-label={`Download ${attachment.fileName}`}
        >
          {!download.isPending && <Download className="size-3.5" aria-hidden="true" />}
          {download.isError ? 'Try again' : 'Download'}
        </Button>
      </div>
      {attachment.previewable && (
        <ImagePreview
          attachment={attachment}
          open={previewOpen}
          onClose={() => {
            setPreviewOpen(false)
          }}
        />
      )}
    </li>
  )
}
