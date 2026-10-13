import { z } from 'zod'
import { apiFetchBlob, apiRequest, apiUpload, resolveApiPath } from '@/lib/api/client'

/**
 * Private attachments (B-09, docs/API_CONVENTIONS.md "Attachments"). Files are
 * never addressed by a public URL: the UI asks for a short-lived link signed for
 * the signed-in user and fetches the bytes with the session cookie.
 */

export const attachmentSchema = z.object({
  id: z.string(),
  fileName: z.string(),
  contentType: z.string(),
  sizeBytes: z.number(),
  sha256: z.string(),
  locationId: z.string().nullable(),
  linked: z.boolean(),
  uploadedBy: z.string(),
  createdAt: z.string(),
  previewable: z.boolean(),
})
export type Attachment = z.infer<typeof attachmentSchema>

export const limitsSchema = z.object({
  maxBytes: z.number(),
  allowedTypes: z.array(
    z.object({
      contentType: z.string(),
      extensions: z.array(z.string()),
      previewable: z.boolean(),
    }),
  ),
})
export type AttachmentLimits = z.infer<typeof limitsSchema>

export const linkSchema = z.object({
  url: z.string(),
  expiresAt: z.string(),
  disposition: z.enum(['attachment', 'inline']),
})
export type AttachmentLink = z.infer<typeof linkSchema>

export type Disposition = AttachmentLink['disposition']

/** GET /attachments/limits — size limit and accepted types. Needs attachments.upload. */
export function getAttachmentLimits(signal?: AbortSignal): Promise<AttachmentLimits> {
  return apiRequest('/attachments/limits', limitsSchema, { signal })
}

/** GET /attachments — the caller's unlinked uploads. Needs attachments.view. */
export function listMyAttachments(signal?: AbortSignal): Promise<Attachment[]> {
  return apiRequest('/attachments', z.array(attachmentSchema), { signal })
}

/** POST /attachments (multipart). Needs attachments.upload. */
export function uploadAttachment(
  file: File,
  options: { signal?: AbortSignal; onProgress?: (fraction: number) => void } = {},
): Promise<Attachment> {
  const form = new FormData()
  form.append('file', file, file.name)
  return apiUpload('/attachments', attachmentSchema, form, options)
}

/** GET /attachments/{id}/link — a short-lived download link. */
export function getAttachmentLink(
  id: string,
  disposition: Disposition,
  signal?: AbortSignal,
): Promise<AttachmentLink> {
  return apiRequest(
    `/attachments/${encodeURIComponent(id)}/link?disposition=${disposition}`,
    linkSchema,
    { signal },
  )
}

/** Authorises, then fetches the file's bytes. Each call gets a fresh link. */
export async function fetchAttachment(
  id: string,
  disposition: Disposition,
  signal?: AbortSignal,
): Promise<Blob> {
  const link = await getAttachmentLink(id, disposition, signal)
  return apiFetchBlob(resolveApiPath(link.url), signal)
}

/** Client-side check against the server's limits; the server checks again. */
export function checkFile(file: File, limits: AttachmentLimits): string | null {
  if (file.size === 0) return 'The file is empty.'
  if (file.size > limits.maxBytes) {
    return `The file is ${formatBytes(file.size)}; the limit is ${formatBytes(limits.maxBytes)}.`
  }
  const dot = file.name.lastIndexOf('.')
  const ext = dot >= 0 ? file.name.slice(dot).toLowerCase() : ''
  const type = limits.allowedTypes.find((t) => t.extensions.includes(ext))
  if (!type) return `This file type is not accepted. Allowed: ${allowedLabel(limits)}.`
  if (file.type && file.type !== type.contentType) {
    return `The file's type (${file.type}) does not match its name.`
  }
  return null
}

export function allowedLabel(limits: AttachmentLimits): string {
  return limits.allowedTypes
    .flatMap((t) => t.extensions[0] ?? [])
    .map((e) => e.slice(1).toUpperCase())
    .join(', ')
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(n < 10 * 1024 ? 1 : 0)} KB`
  return `${(n / (1024 * 1024)).toFixed(1).replace(/\.0$/, '')} MB`
}
