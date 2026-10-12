import { describe, expect, it } from 'vitest'
import { safeDestination } from './redirect'

describe('safeDestination', () => {
  it('keeps in-app paths with query and hash', () => {
    expect(safeDestination('/inventory/vehicles?brand=x#top')).toBe(
      '/inventory/vehicles?brand=x#top',
    )
  })

  it.each([
    undefined,
    null,
    42,
    '',
    'https://evil.example',
    '//evil.example',
    '/\\evil.example',
    '/login',
    '/login?x=1',
  ])('falls back to the dashboard for %s', (from) => {
    expect(safeDestination(from)).toBe('/')
  })
})
