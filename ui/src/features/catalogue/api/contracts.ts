import { z } from 'zod'

/**
 * Catalogue domain contracts (C-01, C-02). They mirror the database schema
 * (migration 000007) and the catalogue API responses (docs/openapi,
 * docs/API_CONVENTIONS.md "Catalogue API"). Fetching lives in ./catalogue.ts.
 *
 * Brands, models, variants, colours and specification keys are data from the
 * API: there is no brand list, model list or spec key in the UI code. Only the
 * status value sets, which the database CHECK constraints define, are listed.
 */

/** Catalogue lifecycle (BR-001, BR-008). Only ACTIVE items may be used operationally (C-02). */
export const lifecycleStatusSchema = z.enum(['PENDING_VERIFICATION', 'ACTIVE', 'ARCHIVED'])
export type LifecycleStatus = z.infer<typeof lifecycleStatusSchema>

/** Dealership authorisation, separate from lifecycle (BR-002). Value set pending sign-off (OQ-1). */
export const authorisationStatusSchema = z.enum([
  'UNVERIFIED',
  'AUTHORISED',
  'NOT_AUTHORISED',
  'REVOKED',
])
export type AuthorisationStatus = z.infer<typeof authorisationStatusSchema>

/** Source confidence (BR-004). Unverified data must not be shown as a definitive claim. */
export const confidenceSchema = z.enum(['UNVERIFIED', 'SUPPLIER_CONFIRMED', 'CLIENT_APPROVED'])
export type Confidence = z.infer<typeof confidenceSchema>

/** Where a catalogue value came from and who checked it (BR-004). */
export const provenanceSchema = z.object({
  confidence: confidenceSchema,
  sourceUrl: z.url().nullable(),
  /** Private attachment ID (B-09); open it through the attachments API, never a URL. */
  sourceDocumentId: z.string().nullable(),
  sourceDate: z.iso.date().nullable(),
  verifiedBy: z.string().nullable(),
  verifiedAt: z.iso.datetime({ offset: true }).nullable(),
})
export type Provenance = z.infer<typeof provenanceSchema>

const timestamps = {
  /** Optimistic-concurrency token: send it back on edits (C-02). */
  version: z.number().int().positive(),
  createdAt: z.iso.datetime({ offset: true }),
  updatedAt: z.iso.datetime({ offset: true }),
}

export const brandSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  lifecycleStatus: lifecycleStatusSchema,
  authorisationStatus: authorisationStatusSchema,
  manufacturerLegalName: z.string().nullable(),
  authorisationReference: z.string().nullable(),
  authorisationEvidenceId: z.string().nullable(),
  verifiedBy: z.string().nullable(),
  verifiedAt: z.iso.datetime({ offset: true }).nullable(),
  /** Server-derived: ACTIVE and AUTHORISED, i.e. usable for purchase/sales. Never computed in the UI. */
  operational: z.boolean(),
  ...timestamps,
})
export type Brand = z.infer<typeof brandSchema>

export const vehicleModelSchema = z.object({
  id: z.string(),
  brandId: z.string(),
  name: z.string(),
  internalCode: z.string(),
  manufacturerCode: z.string().nullable(),
  /** Free classification until the client confirms a value set (OQ-11). */
  vehicleType: z.string().nullable(),
  lifecycleStatus: lifecycleStatusSchema,
  ...provenanceSchema.shape,
  ...timestamps,
})
export type VehicleModel = z.infer<typeof vehicleModelSchema>

export const variantSchema = z.object({
  id: z.string(),
  brandId: z.string(),
  modelId: z.string(),
  name: z.string(),
  variantCode: z.string(),
  manufacturerCode: z.string().nullable(),
  lifecycleStatus: lifecycleStatusSchema,
  ...provenanceSchema.shape,
  ...timestamps,
})
export type Variant = z.infer<typeof variantSchema>

export const variantColourSchema = z.object({
  id: z.string(),
  brandId: z.string(),
  variantId: z.string(),
  name: z.string(),
  colourCode: z.string().nullable(),
  lifecycleStatus: lifecycleStatusSchema,
  ...provenanceSchema.shape,
  ...timestamps,
})
export type VariantColour = z.infer<typeof variantColourSchema>

/**
 * One specification value effective from a date. Exactly one of valueText /
 * valueNumeric is set. Verified rows never change; a change is a new row.
 */
export const variantSpecSchema = z
  .object({
    id: z.string(),
    brandId: z.string(),
    variantId: z.string(),
    specKey: z.string().regex(/^[a-z][a-z0-9_]{0,59}$/),
    valueText: z.string().nullable(),
    valueNumeric: z.number().nullable(),
    unit: z.string().nullable(),
    effectiveFrom: z.iso.date(),
    ...provenanceSchema.shape,
    createdAt: z.iso.datetime({ offset: true }),
    updatedAt: z.iso.datetime({ offset: true }),
  })
  .refine((s) => (s.valueText === null) !== (s.valueNumeric === null), {
    message: 'exactly one of valueText or valueNumeric must be set',
  })
export type VariantSpec = z.infer<typeof variantSpecSchema>

/** Whether a value may be shown as a definitive claim (BR-004). */
export function isVerified(p: Pick<Provenance, 'confidence'>): boolean {
  return p.confidence !== 'UNVERIFIED'
}

/** One page of a catalogue list. */
export function pageSchema<T extends z.ZodType>(item: T) {
  return z.object({
    items: z.array(item),
    page: z.number().int().positive(),
    limit: z.number().int().positive(),
    total: z.number().int().nonnegative(),
  })
}
export interface Page<T> {
  items: T[]
  page: number
  limit: number
  total: number
}

/**
 * One prerequisite for enabling a brand for purchase/sales (BR-002). `pending`
 * checks belong to features not built yet and are never met (fail closed).
 */
export const readinessCheckSchema = z.object({
  key: z.string(),
  met: z.boolean(),
  pending: z.boolean(),
  message: z.string(),
})
export type ReadinessCheck = z.infer<typeof readinessCheckSchema>

export const brandReadinessSchema = z.object({
  brandId: z.string(),
  ready: z.boolean(),
  checks: z.array(readinessCheckSchema),
})
export type BrandReadiness = z.infer<typeof brandReadinessSchema>
