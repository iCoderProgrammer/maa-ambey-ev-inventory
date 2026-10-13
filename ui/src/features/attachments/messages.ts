import { ApiError } from '@/lib/api/client'

const typeLabels: Record<string, string> = {
  'image/jpeg': 'JPEG image',
  'image/png': 'PNG image',
  'image/webp': 'WebP image',
  'application/pdf': 'PDF document',
}

export function typeLabel(contentType: string): string {
  return typeLabels[contentType] ?? contentType
}

export function downloadErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.code === 'FORBIDDEN') return 'You no longer have permission to open this file.'
    if (error.code === 'NOT_FOUND') return 'This file is no longer available to you.'
    return error.message
  }
  return 'The file could not be opened. Please try again.'
}
