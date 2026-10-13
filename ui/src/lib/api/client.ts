import { z } from 'zod'
import { env } from '@/lib/config/env'
import { CSRF_HEADER, getCsrfToken } from './csrf'
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
  /** Seconds from the `Retry-After` header (429/503), when the server sent one. */
  readonly retryAfterSeconds?: number

  constructor(status: number, body: ApiErrorBody, retryAfterSeconds?: number) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.code = body.code
    this.requestId = body.requestId
    this.details = body.details ?? []
    this.retryAfterSeconds = retryAfterSeconds
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

function fallbackError(status: number, requestId?: string, retryAfter?: number): ApiError {
  const fallback = STATUS_FALLBACK[status] ?? {
    code: 'INTERNAL_ERROR',
    message: 'Something went wrong. Please try again.',
  }
  return new ApiError(status, { ...fallback, requestId }, retryAfter)
}

/** Parses a delta-seconds `Retry-After` value; HTTP dates and junk are ignored. */
function parseRetryAfter(value: string | null): number | undefined {
  if (value === null || !/^\d+$/.test(value.trim())) return undefined
  return Number(value.trim())
}

async function toApiError(res: Response): Promise<ApiError> {
  const requestId = res.headers.get('X-Request-ID') ?? undefined
  const retryAfter = parseRetryAfter(res.headers.get('Retry-After'))
  try {
    const parsed = errorEnvelopeSchema.safeParse(await res.json())
    if (parsed.success) {
      return new ApiError(res.status, { requestId, ...parsed.data.error }, retryAfter)
    }
  } catch {
    // Body was not JSON; fall through to the status-based message.
  }
  return fallbackError(res.status, requestId, retryAfter)
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
  // Cookie-authenticated mutations must prove they come from this app (CSRF).
  const csrfToken = method !== 'GET' ? getCsrfToken() : null
  let res: Response
  try {
    res = await fetch(buildUrl(path), {
      method,
      signal,
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
        ...(csrfToken ? { [CSRF_HEADER]: csrfToken } : {}),
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

/**
 * Resolves an absolute API path returned by the server (e.g. a download link
 * "/api/v1/attachments/…/content?…") against the API's origin.
 */
export function resolveApiPath(path: string, baseUrl: string = env.apiBaseUrl): string {
  const origin = /^https?:\/\//.test(baseUrl) ? new URL(baseUrl).origin : ''
  return `${origin}${path}`
}

export interface UploadOptions {
  signal?: AbortSignal
  /** Called with 0–1 as the request body is sent. */
  onProgress?: (fraction: number) => void
}

function networkError(): ApiError {
  return new ApiError(0, {
    code: ClientErrorCode.Network,
    message: 'Cannot reach the server. Check your connection and try again.',
  })
}

/**
 * Sends multipart form data and returns the validated `data` payload, like
 * apiRequest. Uses XMLHttpRequest because fetch cannot report upload progress.
 */
export function apiUpload<S extends z.ZodType>(
  path: string,
  schema: S,
  form: FormData,
  { signal, onProgress }: UploadOptions = {},
): Promise<z.infer<S>> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', buildUrl(path))
    xhr.withCredentials = true
    xhr.setRequestHeader('Accept', 'application/json')
    const csrfToken = getCsrfToken()
    if (csrfToken) xhr.setRequestHeader(CSRF_HEADER, csrfToken)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(e.loaded / e.total)
    }
    const onAbort = () => {
      xhr.abort()
    }
    signal?.addEventListener('abort', onAbort, { once: true })
    xhr.onabort = () => {
      reject(new DOMException('Upload cancelled', 'AbortError'))
    }
    xhr.onerror = () => {
      reject(networkError())
    }
    xhr.onload = () => {
      signal?.removeEventListener('abort', onAbort)
      const headers = new Headers()
      for (const name of ['X-Request-ID', 'Retry-After', 'Content-Type']) {
        const v = xhr.getResponseHeader(name)
        if (v !== null) headers.set(name, v)
      }
      // Re-use the fetch-based envelope handling.
      const res = new Response(xhr.status === 204 ? null : xhr.responseText, {
        status: xhr.status,
        headers,
      })
      if (!res.ok) {
        void toApiError(res).then(reject)
        return
      }
      void res
        .json()
        .catch(() => null)
        .then((json) => {
          const envelope = successEnvelopeSchema.safeParse(json)
          const parsed = envelope.success ? schema.safeParse(envelope.data.data) : undefined
          if (parsed?.success) resolve(parsed.data)
          else
            reject(
              new ApiError(res.status, {
                code: ClientErrorCode.InvalidResponse,
                message: 'The server returned an unexpected response.',
                requestId: headers.get('X-Request-ID') ?? undefined,
              }),
            )
        })
    }
    if (signal?.aborted) {
      reject(new DOMException('Upload cancelled', 'AbortError'))
      return
    }
    xhr.send(form)
  })
}

/**
 * Fetches a binary resource (e.g. a signed download link) with the session
 * cookie. Errors are normalised to ApiError like apiRequest.
 */
export async function apiFetchBlob(url: string, signal?: AbortSignal): Promise<Blob> {
  let res: Response
  try {
    res = await fetch(url, { credentials: 'include', signal })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw networkError()
  }
  if (!res.ok) throw await toApiError(res)
  return res.blob()
}
