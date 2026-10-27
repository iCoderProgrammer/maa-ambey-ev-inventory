import { onlineManager } from '@tanstack/react-query'
import { afterEach, describe, expect, it } from 'vitest'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { createQueryClient } from './queryClient'

afterEach(() => {
  onlineManager.setOnline(true)
})

describe('createQueryClient', () => {
  it('sends a write while offline so it fails at once instead of pausing until reconnect', async () => {
    const queryClient = createQueryClient()
    onlineManager.setOnline(false)
    const mutation = queryClient.getMutationCache().build(queryClient, {
      mutationFn: () =>
        Promise.reject(
          new ApiError(0, { code: ClientErrorCode.Network, message: 'Cannot reach the server.' }),
        ),
    })
    const outcome = await Promise.race([
      mutation.execute(undefined).then(
        () => 'succeeded',
        (e: unknown) =>
          e instanceof ApiError && e.isNetworkError ? 'network error' : 'other error',
      ),
      new Promise((resolve) => {
        setTimeout(() => {
          resolve('paused')
        }, 500)
      }),
    ])
    expect(outcome).toBe('network error')
    expect(mutation.state.isPaused).toBe(false)
  })
})
