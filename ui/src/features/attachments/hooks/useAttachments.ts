import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import {
  fetchAttachment,
  getAttachmentLimits,
  listMyAttachments,
  uploadAttachment,
  type Attachment,
} from '../api/attachments'

export const attachmentsQueryKey = ['attachments'] as const
const mineQueryKey = [...attachmentsQueryKey, 'mine'] as const

export function useAttachmentLimits(enabled: boolean) {
  return useQuery({
    queryKey: [...attachmentsQueryKey, 'limits'],
    queryFn: ({ signal }) => getAttachmentLimits(signal),
    enabled,
    staleTime: 5 * 60_000,
  })
}

export function useMyAttachments(enabled: boolean) {
  return useQuery({
    queryKey: mineQueryKey,
    queryFn: ({ signal }) => listMyAttachments(signal),
    enabled,
  })
}

/** Uploads one file; the list refreshes only after the server confirms. */
export function useUploadAttachment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      file: File
      signal?: AbortSignal
      onProgress?: (fraction: number) => void
    }) => uploadAttachment(vars.file, { signal: vars.signal, onProgress: vars.onProgress }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: mineQueryKey }),
  })
}

/** Downloads a file through a fresh signed link and hands it to the browser. */
export function useDownloadAttachment(attachment: Attachment) {
  return useMutation({
    mutationFn: async () => {
      const blob = await fetchAttachment(attachment.id, 'attachment')
      const url = URL.createObjectURL(blob)
      try {
        const a = document.createElement('a')
        a.href = url
        a.download = attachment.fileName
        a.rel = 'noopener'
        document.body.append(a)
        a.click()
        a.remove()
      } finally {
        // Give the browser a moment to start the download before releasing the blob.
        setTimeout(() => {
          URL.revokeObjectURL(url)
        }, 10_000)
      }
    },
  })
}

/**
 * Loads an image for preview as an in-memory object URL (never a storage URL).
 * The bytes are not cached between openings: each opening is authorised again.
 */
export function useAttachmentPreview(attachment: Attachment, enabled: boolean) {
  const query = useQuery({
    queryKey: [...attachmentsQueryKey, attachment.id, 'preview'],
    queryFn: ({ signal }) => fetchAttachment(attachment.id, 'inline', signal),
    enabled: enabled && attachment.previewable,
    staleTime: 0,
    gcTime: 0,
  })
  const url = useMemo(() => (query.data ? URL.createObjectURL(query.data) : null), [query.data])
  useEffect(
    () => () => {
      if (url) URL.revokeObjectURL(url)
    },
    [url],
  )
  return { ...query, url }
}
