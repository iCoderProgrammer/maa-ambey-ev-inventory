import { z } from 'zod'
import { todayISO } from './presentation'

/**
 * Client-side form rules. They mirror the server's validation (C-02,
 * internal/modules/catalogue/validate.go) so most mistakes are caught early;
 * the server checks everything again and its answer wins.
 */

const text = (label: string, max: number) =>
  z
    .string()
    .trim()
    .min(1, `${label} is required`)
    .max(max, `${label} must be at most ${String(max)} characters`)

const optionalText = (label: string, max: number) =>
  z
    .string()
    .trim()
    .max(max, `${label} must be at most ${String(max)} characters`)

const code = (label: string) =>
  z
    .string()
    .trim()
    .min(1, `${label} is required`)
    .transform((v) => v.toUpperCase())
    .pipe(
      z
        .string()
        .regex(
          /^[A-Z0-9][A-Z0-9._-]{0,39}$/,
          'Use 1–40 letters, digits, ".", "_" or "-", starting with a letter or digit',
        ),
    )

export const brandFormSchema = z.object({
  name: text('Name', 120),
  slug: z
    .string()
    .trim()
    .max(80, 'Slug must be at most 80 characters')
    .refine((v) => v === '' || /^[a-z0-9]+(-[a-z0-9]+)*$/.test(v), {
      message: 'Use lowercase letters, digits and single hyphens',
    }),
})
export type BrandFormValues = z.input<typeof brandFormSchema>

export const modelFormSchema = z.object({
  brandId: z.string().min(1, 'Choose a brand'),
  name: text('Name', 120),
  internalCode: code('Internal code'),
  manufacturerCode: optionalText('Manufacturer code', 60),
  vehicleType: z
    .string()
    .trim()
    .transform((v) => v.toUpperCase())
    .refine((v) => v === '' || /^[A-Z][A-Z0-9_]{0,39}$/.test(v), {
      message: 'Use an upper-case identifier such as SCOOTER (letters, digits, "_")',
    }),
})
export type ModelFormValues = z.input<typeof modelFormSchema>

export const variantFormSchema = z.object({
  brandId: z.string().min(1, 'Choose a brand'),
  modelId: z.string().min(1, 'Choose a model'),
  name: text('Name', 120),
  variantCode: code('Variant code'),
  manufacturerCode: optionalText('Manufacturer code', 60),
})
export type VariantFormValues = z.input<typeof variantFormSchema>

export const colourFormSchema = z.object({
  name: text('Name', 80),
  colourCode: optionalText('Colour code', 40),
})
export type ColourFormValues = z.input<typeof colourFormSchema>

export const verifyFormSchema = z
  .object({
    confidence: z.enum(['SUPPLIER_CONFIRMED', 'CLIENT_APPROVED'], {
      message: 'Choose who confirmed the data',
    }),
    sourceUrl: z
      .string()
      .trim()
      .max(2000, 'URL is too long')
      .refine((v) => v === '' || /^https?:\/\/\S+$/i.test(v), {
        message: 'Use an http or https link',
      }),
    sourceDocumentId: z.string(),
    sourceDate: z
      .string()
      .min(1, 'Source date is required')
      .refine((v) => v <= todayISO(), { message: 'Source date cannot be in the future' }),
  })
  .refine((v) => v.sourceUrl !== '' || v.sourceDocumentId !== '', {
    message: 'Give a source link or choose a source document',
    path: ['sourceUrl'],
  })
export type VerifyFormValues = z.input<typeof verifyFormSchema>

export const authorisationFormSchema = z
  .object({
    authorisationStatus: z.enum(['UNVERIFIED', 'AUTHORISED', 'NOT_AUTHORISED', 'REVOKED']),
    manufacturerLegalName: optionalText('Legal name', 200),
    authorisationReference: optionalText('Reference', 200),
    authorisationEvidenceId: z.string(),
  })
  .superRefine((v, ctx) => {
    if (v.authorisationStatus !== 'AUTHORISED') return
    if (!v.manufacturerLegalName) {
      ctx.addIssue({
        code: 'custom',
        path: ['manufacturerLegalName'],
        message: 'Required for Authorised',
      })
    }
    if (!v.authorisationReference) {
      ctx.addIssue({
        code: 'custom',
        path: ['authorisationReference'],
        message: 'Required for Authorised',
      })
    }
    if (!v.authorisationEvidenceId) {
      ctx.addIssue({
        code: 'custom',
        path: ['authorisationEvidenceId'],
        message: 'Attach the authorisation document',
      })
    }
  })
export type AuthorisationFormValues = z.input<typeof authorisationFormSchema>

export const archiveFormSchema = z.object({
  reason: optionalText('Reason', 500),
})
export type ArchiveFormValues = z.input<typeof archiveFormSchema>

/** Empty string → null, for optional PATCH/POST fields. */
export function orNull(v: string | undefined): string | null {
  const t = v?.trim() ?? ''
  return t === '' ? null : t
}
