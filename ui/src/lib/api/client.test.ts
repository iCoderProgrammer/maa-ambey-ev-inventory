import { z } from 'zod'
import { describe, expect, it, vi } from 'vitest'
import { apiRequest, ApiError, buildUrl } from './client'

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}

const schema = z.object({ status: z.string() })

describe('buildUrl', () => {
  it('joins base and path without duplicate slashes', () => {
    expect(buildUrl('/health', '/api/v1')).toBe('/api/v1/health')
    expect(buildUrl('ready', 'http://localhost:8080/api/v1')).toBe(
      'http://localhost:8080/api/v1/ready',
    )
  })
})

describe('apiRequest', () => {
  it('unwraps and validates the data envelope', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(jsonResponse(200, { data: { status: 'ok' } }))

    await expect(apiRequest('/health', schema)).resolves.toEqual({ status: 'ok' })
    const [url, init] = fetchMock.mock.calls[0]!
    expect(url).toBe('/api/v1/health')
    expect(init?.method).toBe('GET')
    expect(init?.credentials).toBe('include')
  })

  it('sends JSON bodies', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(jsonResponse(200, { data: { status: 'ok' } }))
    await apiRequest('/things', schema, { method: 'POST', body: { a: 1 } })
    const init = fetchMock.mock.calls[0]![1]!
    expect(init.body).toBe('{"a":1}')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
  })

  it('parses the server error envelope', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse(422, {
        error: {
          code: 'VALIDATION_FAILED',
          message: 'Check the highlighted fields.',
          requestId: 'req-123',
          details: [{ field: 'slug', message: 'is required' }],
        },
      }),
    )

    const err = await apiRequest('/brands', schema).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({
      status: 422,
      code: 'VALIDATION_FAILED',
      message: 'Check the highlighted fields.',
      requestId: 'req-123',
      details: [{ field: 'slug', message: 'is required' }],
    })
  })

  it('falls back to a status-based message for non-JSON errors', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('<html>Bad gateway</html>', {
        status: 503,
        headers: { 'X-Request-ID': 'abc12345' },
      }),
    )
    await expect(apiRequest('/ready', schema)).rejects.toMatchObject({
      status: 503,
      code: 'SERVICE_UNAVAILABLE',
      requestId: 'abc12345',
    })
  })

  it('maps network failures to NETWORK_ERROR', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'))
    const err = await apiRequest('/health', schema).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).isNetworkError).toBe(true)
  })

  it('rejects responses that do not match the contract', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse(200, { data: { unexpected: true } }),
    )
    await expect(apiRequest('/health', schema)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })

  it('returns the body of an accepted non-2xx status as data', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse(503, { data: { status: 'not_ready' } }),
    )
    await expect(apiRequest('/ready', schema, { acceptStatuses: [503] })).resolves.toEqual({
      status: 'not_ready',
    })
  })

  it('still throws for an accepted status whose body is not a valid answer', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('<html>Service Unavailable</html>', {
        status: 503,
        headers: { 'X-Request-ID': 'lb-1234567' },
      }),
    )
    await expect(apiRequest('/ready', schema, { acceptStatuses: [503] })).rejects.toMatchObject({
      status: 503,
      code: 'SERVICE_UNAVAILABLE',
      requestId: 'lb-1234567',
    })
  })

  it('explains gateway failures when the API behind a proxy is down', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 502 }))
    await expect(apiRequest('/health', schema)).rejects.toMatchObject({
      status: 502,
      code: 'BAD_GATEWAY',
      message: 'The API could not be reached. Try again shortly.',
    })
  })

  it('re-throws aborts unchanged', async () => {
    const abort = new DOMException('aborted', 'AbortError')
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(abort)
    await expect(apiRequest('/health', schema)).rejects.toBe(abort)
  })
})
