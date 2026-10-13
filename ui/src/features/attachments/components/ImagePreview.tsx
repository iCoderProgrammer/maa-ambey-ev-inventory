import { Dialog } from '@/components/ui/Dialog'
import { Button } from '@/components/ui/Button'
import { LoadingState } from '@/components/ui/StateViews'
import { ApiError } from '@/lib/api/client'
import type { Attachment } from '../api/attachments'
import { useAttachmentPreview } from '../hooks/useAttachments'
import { downloadErrorMessage } from '../messages'

export interface ImagePreviewProps {
  attachment: Attachment
  open: boolean
  onClose: () => void
}

/**
 * Shows an image attachment in a dialog. The image is fetched through a fresh
 * signed link only while the dialog is open and displayed from memory.
 */
export function ImagePreview({ attachment, open, onClose }: ImagePreviewProps) {
  const preview = useAttachmentPreview(attachment, open)
  return (
    <Dialog open={open} onClose={onClose} title={attachment.fileName}>
      {preview.isPending && preview.fetchStatus !== 'idle' && (
        <LoadingState label="Loading preview…" />
      )}
      {preview.isError && (
        <div role="alert" className="flex flex-col items-start gap-3 text-sm">
          <p className="text-danger">{downloadErrorMessage(preview.error)}</p>
          {preview.error instanceof ApiError && preview.error.requestId && (
            <p className="font-mono text-xs text-muted">Reference: {preview.error.requestId}</p>
          )}
          <Button variant="secondary" size="sm" onClick={() => void preview.refetch()}>
            Try again
          </Button>
        </div>
      )}
      {preview.url && (
        <img
          src={preview.url}
          alt={`Preview of ${attachment.fileName}`}
          className="mx-auto max-h-[60dvh] w-auto max-w-full rounded-control object-contain"
        />
      )}
    </Dialog>
  )
}
