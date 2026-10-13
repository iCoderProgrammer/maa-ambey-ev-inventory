import { describe, expect, it } from 'vitest'
import { checkLabel, notReadyReason } from './readinessChecks'

describe('readiness check helpers', () => {
  it('labels known checks and keeps unknown ones visible', () => {
    expect(checkLabel('database')).toBe('Database')
    expect(checkLabel('migrations')).toBe('Database schema')
    expect(checkLabel('cache')).toBe('cache')
  })

  it('reports the most fundamental failure first', () => {
    expect(notReadyReason({ database: 'down', migrations: 'down' })).toBe('Database unavailable')
    expect(notReadyReason({ database: 'up', migrations: 'down' })).toBe('Schema not ready')
    expect(notReadyReason({ database: 'up', cache: 'down' })).toBe('Not ready')
  })
})
