import { z } from 'zod'
import { env } from '@/lib/config/env'
import type { ApiErrorBody, ApiFieldError } from '@/types/api'

/** Client-side error codes in addition to the server's codes. */
export const ClientErrorCode = {
  Network: 'NETWORK_ERROR',
  InvalidResponse: 'INVALID_RESPONSE',
} as const

/** Normalised error for every failed API call. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId?: string
  readonly details: ApiFieldError[]

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.code = body.code
    this.requestId = body.requestId
    this.details = body.details ?? []
  }

  get isNetworkError(): boolean {
    return this.code === ClientErrorCode.Network
  }
}

const errorEnvelopeSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
    requestId: z.string().optional(),
    details: z.array(z.object({ field: z.string(), message: z.string() })).optional(),
  }),
})

const successEnvelopeSchema = z.object({ data: z.unknown() })

const STATUS_FALLBACK: Record<number, Omit<ApiErrorBody, 'requestId'>> = {
  400: { code: 'BAD_REQUEST', message: 'The request was invalid.' },
  401: { code: 'UNAUTHENTICATED', message: 'Please sign in to continue.' },
  403: { code: 'FORBIDDEN', message: 'You do not have permission to do this.' },
  404: { code: 'NOT_FOUND', message: 'The requested resource was not found.' },
  409: {
    code: 'CONFLICT',
    message: 'This record was changed by someone else. Refresh and try again.',
  },
  429: { code: 'RATE_LIMITED', message: 'Too many requests. Please wait a moment.' },
  502: { code: 'BAD_GATEWAY', message: 'The API could not be reached. Try again shortly.' },
  503: { code: 'SERVICE_UNAVAILABLE', message: 'The service is temporarily unavailable.' },
  504: { code: 'GATEWAY_TIMEOUT', message: 'The API did not respond in time. Try again shortly.' },
}

function fallbackError(status: number, requestId?: string): ApiError {
  const fallback = STATUS_FALLBACK[status] ?? {
    code: 'INTERNAL_ERROR',
    message: 'Something went wrong. Please try again.',
  }
  return new ApiError(status, { ...fallback, requestId })
}

async function toApiError(res: Response): Promise<ApiError> {
  const requestId = res.headers.get('X-Request-ID') ?? undefined
  try {
    const parsed = errorEnvelopeSchema.safeParse(await res.json())
    if (parsed.success) {
      return new ApiError(res.status, { requestId, ...parsed.data.error })
    }
  } catch {
    // Body was not JSON; fall through to the status-based message.
  }
  return fallbackError(res.status, requestId)
}

export function buildUrl(path: string, baseUrl: string = env.apiBaseUrl): string {
  return `${baseUrl}/${path.replace(/^\/+/, '')}`
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
  headers?: Record<string, string>
  /**
   * Non-2xx statuses whose success-envelope body is a valid answer, e.g. 503 from
   * /ready ("not_ready"). Any other body for these statuses still throws ApiError.
   */
  acceptStatuses?: number[]
}

/**
 * Calls the API and returns the validated `data` payload of the success
 * envelope. Throws ApiError for HTTP, network and contract failures; abort
 * errors from `signal` are re-thrown unchanged so TanStack Query can ignore them.
 */
export async function apiRequest<S extends z.ZodType>(
  path: string,
  schema: S,
  { method = 'GET', body, signal, headers, acceptStatuses = [] }: RequestOptions = {},
): Promise<z.infer<S>> {
  let res: Response
  try {
    res = await fetch(buildUrl(path), {
      method,
      signal,
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
        ...headers,
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, {
      code: ClientErrorCode.Network,
      message: 'Cannot reach the server. Check your connection and try again.',
    })
  }

  const accepted = !res.ok && acceptStatuses.includes(res.status)
  if (!res.ok && !accepted) throw await toApiError(res)

  const envelope = successEnvelopeSchema.safeParse(await res.json().catch(() => null))
  const parsed = envelope.success ? schema.safeParse(envelope.data.data) : undefined
  if (!parsed?.success) {
    if (accepted) throw fallbackError(res.status, res.headers.get('X-Request-ID') ?? undefined)
    throw new ApiError(res.status, {
      code: ClientErrorCode.InvalidResponse,
      message: 'The server returned an unexpected response.',
      requestId: res.headers.get('X-Request-ID') ?? undefined,
    })
  }
  return parsed.data
}
