import { describe, expect, it } from 'vitest'
import {
  brandSchema,
  isVerified,
  variantSchema,
  variantSpecSchema,
  vehicleModelSchema,
} from './api/contracts'

// Placeholder values only: no real brand data is invented.
const stamps = {
  version: 1,
  createdAt: '2026-10-13T09:00:00Z',
  updatedAt: '2026-10-13T09:00:00Z',
}
const unverified = {
  confidence: 'UNVERIFIED',
  sourceUrl: null,
  sourceDocumentId: null,
  sourceDate: null,
  verifiedBy: null,
  verifiedAt: null,
} as const

describe('catalogue contracts', () => {
  it('accept any brand from the API, not a fixed list', () => {
    for (const name of ['Brand A', 'A brand added next year']) {
      const parsed = brandSchema.parse({
        id: 'b1',
        name,
        slug: 'brand-a',
        lifecycleStatus: 'PENDING_VERIFICATION',
        authorisationStatus: 'UNVERIFIED',
        manufacturerLegalName: null,
        authorisationReference: null,
        authorisationEvidenceId: null,
        verifiedBy: null,
        verifiedAt: null,
        ...stamps,
      })
      expect(parsed.name).toBe(name)
    }
  })

  it('reject statuses the database does not allow', () => {
    const model = {
      id: 'm1',
      brandId: 'b1',
      name: 'Model 1',
      internalCode: 'M1',
      manufacturerCode: null,
      vehicleType: null,
      lifecycleStatus: 'ACTIVE',
      ...unverified,
      ...stamps,
    }
    expect(vehicleModelSchema.safeParse(model).success).toBe(true)
    expect(vehicleModelSchema.safeParse({ ...model, lifecycleStatus: 'LIVE' }).success).toBe(false)
    expect(vehicleModelSchema.safeParse({ ...model, confidence: 'TRUSTED' }).success).toBe(false)
  })

  it('keep the brand on every child record', () => {
    const variant = {
      id: 'v1',
      modelId: 'm1',
      name: 'Variant 1',
      variantCode: 'STD',
      manufacturerCode: null,
      lifecycleStatus: 'PENDING_VERIFICATION',
      ...unverified,
      ...stamps,
    }
    expect(variantSchema.safeParse(variant).success).toBe(false)
    expect(variantSchema.safeParse({ ...variant, brandId: 'b1' }).success).toBe(true)
  })

  it('require exactly one spec value and treat only verified data as definitive', () => {
    const spec = {
      id: 's1',
      brandId: 'b1',
      variantId: 'v1',
      specKey: 'attribute_a',
      valueText: null,
      valueNumeric: 1.5,
      unit: 'u',
      effectiveFrom: '2026-10-01',
      ...unverified,
      createdAt: stamps.createdAt,
      updatedAt: stamps.updatedAt,
    }
    expect(variantSpecSchema.safeParse(spec).success).toBe(true)
    expect(variantSpecSchema.safeParse({ ...spec, valueText: 'x' }).success).toBe(false)
    expect(variantSpecSchema.safeParse({ ...spec, valueNumeric: null }).success).toBe(false)
    expect(variantSpecSchema.safeParse({ ...spec, specKey: 'Range KM' }).success).toBe(false)
    expect(isVerified(spec)).toBe(false)
    expect(isVerified({ confidence: 'SUPPLIER_CONFIRMED' })).toBe(true)
  })

  it('the UI source names no brand from the roster', () => {
    const sources = import.meta.glob<string>(['/src/**/*.{ts,tsx}', '!/src/**/*.test.{ts,tsx}'], {
      query: '?raw',
      import: 'default',
      eager: true,
    })
    expect(Object.keys(sources).length).toBeGreaterThan(20)
    const roster = [/lectrix/i, /ceeon/i, /ujagar/i]
    const offenders = Object.entries(sources)
      .filter(([, text]) => roster.some((r) => r.test(text)))
      .map(([path]) => path)
    expect(offenders).toEqual([])
  })
})
