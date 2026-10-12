import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadEnv() {
  vi.resetModules()
  const mod = await import('./env')
  return mod.env
}

describe('env', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('defaults the API base URL to the same-origin /api/v1 proxy path', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '')
    expect((await loadEnv()).apiBaseUrl).toBe('/api/v1')
  })

  it('uses VITE_API_BASE_URL without trailing slashes', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.test/api/v1//')
    expect((await loadEnv()).apiBaseUrl).toBe('https://api.example.test/api/v1')
  })

  it('exposes only non-secret settings', async () => {
    expect(Object.keys(await loadEnv())).toEqual(['apiBaseUrl'])
  })
})
