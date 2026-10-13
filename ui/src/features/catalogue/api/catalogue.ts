import { apiRequest, ApiError } from '@/lib/api/client'
import {
  brandReadinessSchema,
  brandSchema,
  pageSchema,
  variantColourSchema,
  variantSchema,
  vehicleModelSchema,
  type AuthorisationStatus,
  type Brand,
  type BrandReadiness,
  type Confidence,
  type LifecycleStatus,
  type Page,
  type Variant,
  type VariantColour,
  type VehicleModel,
} from './contracts'

/**
 * Catalogue API client (C-02; docs/openapi, docs/API_CONVENTIONS.md
 * "Catalogue API"). Every call goes through apiRequest, which sends the session
 * cookie and CSRF token and validates the response against the Zod contracts.
 * The server decides permissions, brand relationships and lifecycle rules;
 * nothing here assumes a call will succeed.
 */

/** Item kinds that share the verify / activate / archive lifecycle. */
export type CatalogueItemKind = 'models' | 'variants' | 'colours'

export interface ItemByKind {
  models: VehicleModel
  variants: Variant
  colours: VariantColour
}

const itemSchemas = {
  models: vehicleModelSchema,
  variants: variantSchema,
  colours: variantColourSchema,
} as const

// ── Requests ────────────────────────────────────────────────────────────────

export interface ListParams {
  q?: string
  status?: LifecycleStatus
  page?: number
  /** 1–100; the server defaults to 25. */
  limit?: number
}
export interface BrandListParams extends ListParams {
  authorisationStatus?: AuthorisationStatus
}
export interface ModelListParams extends ListParams {
  brandId?: string
  confidence?: Confidence
}
export interface VariantListParams extends ModelListParams {
  modelId?: string
}
export interface ColourListParams extends ModelListParams {
  variantId?: string
}

export interface BrandCreate {
  name: string
  /** Derived from the name by the server when omitted. */
  slug?: string
}
/** Fields left out are unchanged. `version` is the version last read. */
export interface BrandUpdate {
  version: number
  name?: string
  slug?: string
}
export interface BrandAuthorisation {
  version: number
  authorisationStatus: AuthorisationStatus
  manufacturerLegalName?: string | null
  authorisationReference?: string | null
  /** Private attachment ID (B-09). Required with AUTHORISED. */
  authorisationEvidenceId?: string | null
}

export interface ModelCreate {
  brandId: string
  name: string
  internalCode: string
  manufacturerCode?: string | null
  vehicleType?: string | null
}
/** null clears an optional field; the brand never changes. */
export interface ModelUpdate {
  version: number
  name?: string
  internalCode?: string
  manufacturerCode?: string | null
  vehicleType?: string | null
}
export interface VariantCreate {
  modelId: string
  /** Optional; when sent the server checks it is the model's brand. */
  brandId?: string
  name: string
  variantCode: string
  manufacturerCode?: string | null
}
export interface VariantUpdate {
  version: number
  name?: string
  variantCode?: string
  manufacturerCode?: string | null
}
export interface ColourCreate {
  variantId: string
  brandId?: string
  name: string
  colourCode?: string | null
}
export interface ColourUpdate {
  version: number
  name?: string
  colourCode?: string | null
}

/** Source and verification of a pending item. The verifier is the signed-in user (server-side). */
export interface VerifyInput {
  version: number
  confidence: Exclude<Confidence, 'UNVERIFIED'>
  sourceUrl?: string | null
  sourceDocumentId?: string | null
  /** YYYY-MM-DD, not in the future. */
  sourceDate: string
}
export interface ArchiveInput {
  version: number
  reason?: string
}

// ── Helpers ─────────────────────────────────────────────────────────────────

export function queryString(params: object): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value as string | number))
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}

const id = (value: string) => encodeURIComponent(value)

// ── Brands ──────────────────────────────────────────────────────────────────

const brandPage = pageSchema(brandSchema)

/** GET /brands — needs catalogue.view. */
export function listBrands(
  params: BrandListParams = {},
  signal?: AbortSignal,
): Promise<Page<Brand>> {
  return apiRequest(`/brands${queryString(params)}`, brandPage, { signal })
}

/** GET /brands/{id} — needs catalogue.view. */
export function getBrand(brandId: string, signal?: AbortSignal): Promise<Brand> {
  return apiRequest(`/brands/${id(brandId)}`, brandSchema, { signal })
}

/** POST /brands — needs catalogue.manage. New brands start PENDING_VERIFICATION / UNVERIFIED. */
export function createBrand(input: BrandCreate): Promise<Brand> {
  return apiRequest('/brands', brandSchema, { method: 'POST', body: input })
}

/** PATCH /brands/{id} — needs catalogue.manage. */
export function updateBrand(brandId: string, input: BrandUpdate): Promise<Brand> {
  return apiRequest(`/brands/${id(brandId)}`, brandSchema, { method: 'PATCH', body: input })
}

/** PUT /brands/{id}/authorisation — needs catalogue.brand.authorise. */
export function recordBrandAuthorisation(
  brandId: string,
  input: BrandAuthorisation,
): Promise<Brand> {
  return apiRequest(`/brands/${id(brandId)}/authorisation`, brandSchema, {
    method: 'PUT',
    body: input,
  })
}

/** GET /brands/{id}/activation-readiness — needs catalogue.view. */
export function getBrandReadiness(brandId: string, signal?: AbortSignal): Promise<BrandReadiness> {
  return apiRequest(`/brands/${id(brandId)}/activation-readiness`, brandReadinessSchema, { signal })
}

/** POST /brands/{id}/activate — needs catalogue.brand.activate; 409 ACTIVATION_BLOCKED until every check is met. */
export function activateBrand(brandId: string, version: number): Promise<Brand> {
  return apiRequest(`/brands/${id(brandId)}/activate`, brandSchema, {
    method: 'POST',
    body: { version },
  })
}

/** POST /brands/{id}/archive — needs catalogue.archive. */
export function archiveBrand(brandId: string, input: ArchiveInput): Promise<Brand> {
  return apiRequest(`/brands/${id(brandId)}/archive`, brandSchema, { method: 'POST', body: input })
}

// ── Models, variants, colours ───────────────────────────────────────────────

const modelPage = pageSchema(vehicleModelSchema)
const variantPage = pageSchema(variantSchema)
const colourPage = pageSchema(variantColourSchema)

/** GET /models — needs catalogue.view. */
export function listModels(
  params: ModelListParams = {},
  signal?: AbortSignal,
): Promise<Page<VehicleModel>> {
  return apiRequest(`/models${queryString(params)}`, modelPage, { signal })
}

/** GET /variants — needs catalogue.view. */
export function listVariants(
  params: VariantListParams = {},
  signal?: AbortSignal,
): Promise<Page<Variant>> {
  return apiRequest(`/variants${queryString(params)}`, variantPage, { signal })
}

/** GET /colours — needs catalogue.view. */
export function listColours(
  params: ColourListParams = {},
  signal?: AbortSignal,
): Promise<Page<VariantColour>> {
  return apiRequest(`/colours${queryString(params)}`, colourPage, { signal })
}

/** GET /{kind}/{id} — needs catalogue.view. */
export function getItem<K extends CatalogueItemKind>(
  kind: K,
  itemId: string,
  signal?: AbortSignal,
): Promise<ItemByKind[K]> {
  return apiRequest(`/${kind}/${id(itemId)}`, itemSchemas[kind], { signal }) as Promise<
    ItemByKind[K]
  >
}

/** POST /models — needs catalogue.manage. */
export function createModel(input: ModelCreate): Promise<VehicleModel> {
  return apiRequest('/models', vehicleModelSchema, { method: 'POST', body: input })
}

/** POST /variants — needs catalogue.manage. The brand comes from the model. */
export function createVariant(input: VariantCreate): Promise<Variant> {
  return apiRequest('/variants', variantSchema, { method: 'POST', body: input })
}

/** POST /colours — needs catalogue.manage. The brand comes from the variant. */
export function createColour(input: ColourCreate): Promise<VariantColour> {
  return apiRequest('/colours', variantColourSchema, { method: 'POST', body: input })
}

export interface UpdateByKind {
  models: ModelUpdate
  variants: VariantUpdate
  colours: ColourUpdate
}

/** PATCH /{kind}/{id} — needs catalogue.manage. Only pending items can be edited. */
export function updateItem<K extends CatalogueItemKind>(
  kind: K,
  itemId: string,
  input: UpdateByKind[K],
): Promise<ItemByKind[K]> {
  return apiRequest(`/${kind}/${id(itemId)}`, itemSchemas[kind], {
    method: 'PATCH',
    body: input,
  }) as Promise<ItemByKind[K]>
}

/** POST /{kind}/{id}/verify — needs catalogue.verify. */
export function verifyItem<K extends CatalogueItemKind>(
  kind: K,
  itemId: string,
  input: VerifyInput,
): Promise<ItemByKind[K]> {
  return apiRequest(`/${kind}/${id(itemId)}/verify`, itemSchemas[kind], {
    method: 'POST',
    body: input,
  }) as Promise<ItemByKind[K]>
}

/** POST /{kind}/{id}/activate — needs catalogue.activate; 409 ACTIVATION_BLOCKED if unverified or the parent is not active. */
export function activateItem<K extends CatalogueItemKind>(
  kind: K,
  itemId: string,
  version: number,
): Promise<ItemByKind[K]> {
  return apiRequest(`/${kind}/${id(itemId)}/activate`, itemSchemas[kind], {
    method: 'POST',
    body: { version },
  }) as Promise<ItemByKind[K]>
}

/** POST /{kind}/{id}/archive — needs catalogue.archive. Never deletes. */
export function archiveItem<K extends CatalogueItemKind>(
  kind: K,
  itemId: string,
  input: ArchiveInput,
): Promise<ItemByKind[K]> {
  return apiRequest(`/${kind}/${id(itemId)}/archive`, itemSchemas[kind], {
    method: 'POST',
    body: input,
  }) as Promise<ItemByKind[K]>
}

// ── Error helpers ───────────────────────────────────────────────────────────

/** Catalogue error codes (all 409) in addition to the shared ones. */
export const CatalogueErrorCode = {
  Duplicate: 'DUPLICATE',
  VersionConflict: 'VERSION_CONFLICT',
  InvalidState: 'INVALID_STATE',
  ActivationBlocked: 'ACTIVATION_BLOCKED',
} as const

/** The unmet prerequisites of a refused activation, or null for any other error. */
export function activationBlockers(error: unknown): { key: string; message: string }[] | null {
  if (!(error instanceof ApiError) || error.code !== CatalogueErrorCode.ActivationBlocked)
    return null
  return error.details.map((d) => ({ key: d.field, message: d.message }))
}

/** True when the item changed since it was read: reload before retrying. */
export function isVersionConflict(error: unknown): boolean {
  return error instanceof ApiError && error.code === CatalogueErrorCode.VersionConflict
}
