import { describe, expect, it, vi } from 'vitest'
import {
  describeGRNError,
  DUPLICATE_IDENTIFIER_FIELD_MESSAGE,
} from '@/features/receiving/lib/labels'
import { ApiError } from '@/lib/api/client'
import { isIdentifierPath, normaliseChassis } from './lib/identifiers'

// Placeholder identifiers only ("CH-0001"): nothing real.

describe('normaliseChassis (D-05)', () => {
  it('trims surrounding spaces and upper-cases, nothing else', () => {
    expect(normaliseChassis('  ch-0001 ')).toBe('CH-0001')
    expect(normaliseChassis('Ch-0001')).toBe('CH-0001')
    expect(normaliseChassis('')).toBe('')
    expect(normaliseChassis('   ')).toBe('')
  })

  it('keeps inner spaces and separators (OQ-26)', () => {
    const forms = ['CH 0001', 'CH/0001', 'CH0001', 'CH-0001', 'CH  0001']
    expect(new Set(forms.map(normaliseChassis)).size).toBe(forms.length)
    expect(normaliseChassis(' ch/00 01 ')).toBe('CH/00 01')
  })
})

describe('isIdentifierPath', () => {
  it('accepts the four identifier fields only', () => {
    expect(isIdentifierPath('lines.0.units.1.chassisNumber')).toBe(true)
    expect(isIdentifierPath('lines.0.units.1.motorSerial')).toBe(true)
    expect(isIdentifierPath('lines.0.units.1.note')).toBe(false)
    expect(isIdentifierPath('lines.0.receivedQuantity')).toBe(false)
    expect(isIdentifierPath('chassisNumberX')).toBe(false)
  })
})

const dup = (details: { field: string; message: string }[], status = 409) =>
  new ApiError(status, {
    code: 'DUPLICATE_IDENTIFIER',
    message: 'This identifier is already registered to another vehicle.',
    requestId: 'req-1',
    details,
  })

describe('describeGRNError with DUPLICATE_IDENTIFIER', () => {
  const known = new Set(['lines.0.units.1.chassisNumber', 'lines.1.units.0.chassisNumber'])
  const fields = () => ({ setError: vi.fn(), isField: (p: string) => known.has(p) })

  it('places the conflict on the nested unit field with its own message', () => {
    const f = fields()
    const r = describeGRNError(
      dup([{ field: 'lines[0].units[1].chassisNumber', message: 'is already registered' }]),
      f,
      [1, 2],
    )
    expect(f.setError).toHaveBeenCalledTimes(1)
    expect(f.setError).toHaveBeenCalledWith('lines.0.units.1.chassisNumber', {
      message: DUPLICATE_IDENTIFIER_FIELD_MESSAGE,
    })
    expect(r.blockers).toEqual([
      {
        key: 'Line 1, unit 2: chassis/frame number',
        message: 'already registered to another vehicle',
      },
    ])
    expect(r.message).toMatch(/Nothing was saved; your entries are kept/)
    expect(r.stale).toBeUndefined()
    expect(r.requestId).toBe('req-1')
  })

  it('uses the line numbers of the form for the label', () => {
    const r = describeGRNError(
      dup([{ field: 'lines[1].units[0].chassisNumber', message: 'x' }]),
      fields(),
      [3, 7],
    )
    expect(r.blockers?.[0]?.key).toBe('Line 7, unit 1: chassis/frame number')
  })

  it.each([
    'lines[5].units[0].chassisNumber',
    'lines[0].units[9].chassisNumber',
    'lines[0].units[1].note',
    'lines[0].receivedQuantity',
    '__proto__',
    'constructor.prototype',
    '',
  ])('falls back to a general message for the unsafe or unknown path %j', (field) => {
    const f = fields()
    const r = describeGRNError(dup([{ field, message: 'is already registered' }]), f)
    expect(f.setError).not.toHaveBeenCalled()
    expect(r.blockers).toBeUndefined()
    expect(r.message).toMatch(/already registered to another vehicle/)
    expect(r.message).not.toContain(field || '\u0000')
  })

  it('falls back when there are no details or no form', () => {
    expect(describeGRNError(dup([])).blockers).toBeUndefined()
    expect(
      describeGRNError(dup([{ field: 'lines[0].units[1].chassisNumber', message: 'x' }])).blockers,
    ).toBeUndefined()
  })

  it('never repeats what the server message or details say about other records', () => {
    const r = describeGRNError(
      dup([{ field: 'lines[0].units[1].chassisNumber', message: 'unit VU-000042 at Shop 2' }]),
      fields(),
    )
    expect(JSON.stringify(r)).not.toMatch(/VU-000042|Shop 2/)
  })

  it('keeps other 409s apart: a version conflict is stale, not a duplicate', () => {
    const f = fields()
    const r = describeGRNError(
      new ApiError(409, { code: 'VERSION_CONFLICT', message: 'Changed.' }),
      f,
    )
    expect(f.setError).not.toHaveBeenCalled()
    expect(r.message).not.toMatch(/already registered/)
    const fallback = describeGRNError(
      new ApiError(409, { code: 'CONFLICT', message: 'Changed.' }),
      f,
    )
    expect(fallback.message).not.toMatch(/already registered/)
  })

  it.each([
    [401, 'UNAUTHENTICATED'],
    [403, 'FORBIDDEN'],
    [404, 'NOT_FOUND'],
    [422, 'UNPROCESSABLE'],
    [0, 'NETWORK_ERROR'],
  ])('leaves %i %s to the usual error handling', (status, code) => {
    const f = fields()
    const r = describeGRNError(
      new ApiError(status, {
        code,
        message: 'Refused.',
        details: [{ field: 'lines[0].units[1].chassisNumber', message: 'x' }],
      }),
      f,
    )
    expect(f.setError).not.toHaveBeenCalled()
    expect(r.message).not.toMatch(/already registered/)
  })
})
