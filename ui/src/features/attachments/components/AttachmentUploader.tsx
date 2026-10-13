import { AlertTriangle, CheckCircle2, UploadCloud, X } from 'lucide-react'
import { useId, useRef, useState, type DragEvent } from 'react'
import { Button } from '@/components/ui/Button'
import { ApiError } from '@/lib/api/client'
import { cn } from '@/lib/utils/cn'
import {
  allowedLabel,
  checkFile,
  formatBytes,
  type Attachment,
  type AttachmentLimits,
} from '../api/attachments'
import { useUploadAttachment } from '../hooks/useAttachments'

type ItemStatus = 'invalid' | 'uploading' | 'done' | 'error' | 'cancelled'

interface Item {
  key: string
  file: File
  status: ItemStatus
  progress: number
  message?: string
  requestId?: string
  /** Retrying cannot help (invalid file, too large, wrong type, no permission). */
  final?: boolean
}

export interface AttachmentUploaderProps {
  limits: AttachmentLimits
  onUploaded?: (attachment: Attachment) => void
}

/** Errors where sending the same file again would fail the same way. */
const FINAL_CODES = new Set([
  'FILE_TOO_LARGE',
  'UNSUPPORTED_MEDIA_TYPE',
  'VALIDATION_FAILED',
  'FORBIDDEN',
])

let nextKey = 0

/**
 * Picks or drops files, checks them against the server's limits, uploads each
 * with progress and shows the server's verdict. Success is shown only after the
 * server has stored the file; failures can be retried when retrying can help.
 */
export function AttachmentUploader({ limits, onUploaded }: AttachmentUploaderProps) {
  const inputId = useId()
  const hintId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const controllers = useRef(new Map<string, AbortController>())
  const [items, setItems] = useState<Item[]>([])
  const [dragging, setDragging] = useState(false)
  const upload = useUploadAttachment()

  const patch = (key: string, change: Partial<Item>) => {
    setItems((list) => list.map((it) => (it.key === key ? { ...it, ...change } : it)))
  }

  const start = (key: string, file: File) => {
    const controller = new AbortController()
    controllers.current.set(key, controller)
    patch(key, { status: 'uploading', progress: 0, message: undefined, requestId: undefined })
    upload
      .mutateAsync({
        file,
        signal: controller.signal,
        onProgress: (p) => {
          patch(key, { progress: p })
        },
      })
      .then((att) => {
        patch(key, { status: 'done', progress: 1 })
        onUploaded?.(att)
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === 'AbortError') {
          patch(key, { status: 'cancelled', message: 'Upload cancelled.' })
          return
        }
        const api = err instanceof ApiError ? err : undefined
        patch(key, {
          status: 'error',
          message: api?.message ?? 'The upload failed. Please try again.',
          requestId: api?.requestId,
          final: api ? FINAL_CODES.has(api.code) : false,
        })
      })
      .finally(() => controllers.current.delete(key))
  }

  const addFiles = (files: FileList | null) => {
    if (!files) return
    const added: Item[] = Array.from(files).map((file) => {
      const problem = checkFile(file, limits)
      nextKey += 1
      return {
        key: `f${String(nextKey)}`,
        file,
        status: problem ? 'invalid' : 'uploading',
        progress: 0,
        message: problem ?? undefined,
        final: Boolean(problem),
      }
    })
    setItems((list) => [...added, ...list])
    for (const it of added) if (it.status === 'uploading') start(it.key, it.file)
    if (inputRef.current) inputRef.current.value = ''
  }

  const remove = (key: string) => {
    controllers.current.get(key)?.abort()
    setItems((list) => list.filter((it) => it.key !== key))
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    addFiles(e.dataTransfer.files)
  }

  const accept = limits.allowedTypes.flatMap((t) => [t.contentType, ...t.extensions]).join(',')

  return (
    <div className="flex flex-col gap-4">
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => {
          setDragging(false)
        }}
        onDrop={onDrop}
        className={cn(
          'flex flex-col items-center gap-3 rounded-card border-2 border-dashed px-4 py-8 text-center transition-colors duration-150',
          dragging ? 'border-primary bg-primary/5' : 'border-border bg-surface-muted/40',
        )}
      >
        <UploadCloud className="size-8 text-muted" aria-hidden="true" />
        <div className="flex flex-col gap-1">
          <label htmlFor={inputId} className="text-sm font-medium text-ink">
            Choose files to upload
          </label>
          <p id={hintId} className="text-xs text-muted">
            {allowedLabel(limits)} · up to {formatBytes(limits.maxBytes)} each · or drop files here
          </p>
        </div>
        <input
          ref={inputRef}
          id={inputId}
          type="file"
          multiple
          accept={accept}
          tabIndex={-1}
          onChange={(e) => {
            addFiles(e.target.files)
          }}
          className="sr-only"
        />
        <Button
          variant="secondary"
          size="sm"
          aria-describedby={hintId}
          onClick={() => inputRef.current?.click()}
        >
          Browse files
        </Button>
      </div>

      {items.length > 0 && (
        <ul className="flex flex-col gap-2" aria-label="Uploads">
          {items.map((it) => (
            <UploadRow
              key={it.key}
              item={it}
              onCancel={() => controllers.current.get(it.key)?.abort()}
              onRetry={() => {
                start(it.key, it.file)
              }}
              onRemove={() => {
                remove(it.key)
              }}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

function UploadRow({
  item,
  onCancel,
  onRetry,
  onRemove,
}: {
  item: Item
  onCancel: () => void
  onRetry: () => void
  onRemove: () => void
}) {
  const pct = Math.round(item.progress * 100)
  const failed = item.status === 'invalid' || item.status === 'error' || item.status === 'cancelled'
  return (
    <li
      className={cn(
        'flex flex-col gap-2 rounded-control border px-3 py-2.5 text-sm',
        failed ? 'border-danger/30 bg-danger-subtle' : 'border-border bg-surface',
      )}
    >
      <div className="flex items-center gap-3">
        {item.status === 'done' ? (
          <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
        ) : failed ? (
          <AlertTriangle className="size-4 shrink-0 text-danger" aria-hidden="true" />
        ) : (
          <UploadCloud className="size-4 shrink-0 text-primary" aria-hidden="true" />
        )}
        <span className="min-w-0 flex-1 truncate font-medium text-ink" title={item.file.name}>
          {item.file.name}
        </span>
        <span className="shrink-0 text-xs text-muted">{formatBytes(item.file.size)}</span>
        {item.status === 'uploading' && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onCancel}
            aria-label={`Cancel upload of ${item.file.name}`}
          >
            Cancel
          </Button>
        )}
        {(item.status === 'error' || item.status === 'cancelled') && !item.final && (
          <Button
            variant="secondary"
            size="sm"
            onClick={onRetry}
            aria-label={`Retry upload of ${item.file.name}`}
          >
            Retry
          </Button>
        )}
        {item.status !== 'uploading' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onRemove}
            aria-label={`Dismiss ${item.file.name}`}
          >
            <X className="size-4" aria-hidden="true" />
          </Button>
        )}
      </div>
      {item.status === 'uploading' && (
        <div
          role="progressbar"
          aria-label={`Uploading ${item.file.name}`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={pct}
          className="h-1.5 overflow-hidden rounded-full bg-surface-muted"
        >
          <div
            className="h-full bg-primary transition-[width] duration-150"
            style={{ width: `${String(pct)}%` }}
          />
        </div>
      )}
      {item.status === 'done' && (
        <p role="status" className="text-xs text-success">
          Uploaded and stored privately.
        </p>
      )}
      {failed && item.message && (
        <p role="alert" className="text-xs text-danger">
          {item.message}
          {item.requestId && (
            <span className="ml-1 font-mono text-muted">Reference: {item.requestId}</span>
          )}
        </p>
      )}
    </li>
  )
}
