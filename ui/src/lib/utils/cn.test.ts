import { describe, expect, it } from 'vitest'
import { cn } from './cn'

describe('cn', () => {
  it('drops falsy values', () => {
    expect(cn('a', false, null, undefined, 'b')).toBe('a b')
  })

  it('lets later classes override conflicting earlier ones', () => {
    expect(cn('inline-flex', 'hidden lg:inline-flex')).toBe('hidden lg:inline-flex')
    expect(cn('h-8 px-3', 'h-10')).toBe('px-3 h-10')
    expect(cn('bg-surface', 'bg-primary')).toBe('bg-primary')
  })

  it('knows the custom radius tokens', () => {
    expect(cn('rounded-control', 'rounded-full')).toBe('rounded-full')
    expect(cn('rounded-full', 'rounded-card')).toBe('rounded-card')
  })

  it('keeps font size and custom text colour together', () => {
    expect(cn('text-xs text-muted')).toBe('text-xs text-muted')
  })
})
