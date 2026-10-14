import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'
import { pageSchema, type Page } from '@/features/catalogue/api/contracts'
import { queryString } from '@/features/catalogue/api/catalogue'

/**
 * Supplier master (D-01; docs/openapi, API_CONVENTIONS "Suppliers"). A supplier
 * record is not proof of dealership authorisation: suppliers and their brand
 * relationships are verified separately, with evidence. Documents are private
 * attachments opened through the attachments API.
 */

const stamp = z.iso.datetime({ offset: true })

export const supplierStatusSchema = z.enum(['PENDING_VERIFICATION', 'VERIFIED', 'ARCHIVED'])
export type SupplierStatus = z.infer<typeof supplierStatusSchema>
export const linkStatusSchema = z.enum(['UNVERIFIED', 'VERIFIED', 'ENDED'])
export type LinkStatus = z.infer<typeof linkStatusSchema>
/** Proposed value set (OQ-19); mirrors the database CHECK. */
export const documentTypes = ['GST_REGISTRATION', 'AUTHORISATION', 'AGREEMENT', 'OTHER'] as const
export type DocumentType = (typeof documentTypes)[number]

export const supplierSchema = z.object({
  id: z.string(),
  legalName: z.string(),
  tradeName: z.string().nullable(),
  gstin: z.string().nullable(),
  addressLine1: z.string().nullable(),
  addressLine2: z.string().nullable(),
  city: z.string().nullable(),
  state: z.string().nullable(),
  postalCode: z.string().nullable(),
  status: supplierStatusSchema,
  verificationReference: z.string().nullable(),
  verificationDocumentId: z.string().nullable(),
  verifiedBy: z.string().nullable(),
  verifiedAt: stamp.nullable(),
  notes: z.string().nullable(),
  createdBy: z.string(),
  brandCount: z.number().int().nonnegative(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type Supplier = z.infer<typeof supplierSchema>

export const contactSchema = z.object({
  id: z.string(),
  supplierId: z.string(),
  name: z.string(),
  role: z.string().nullable(),
  phone: z.string().nullable(),
  email: z.string().nullable(),
  isPrimary: z.boolean(),
  active: z.boolean(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type Contact = z.infer<typeof contactSchema>

export const documentSchema = z.object({
  id: z.string(),
  supplierId: z.string(),
  attachmentId: z.string(),
  documentType: z.enum(documentTypes),
  note: z.string().nullable(),
  fileName: z.string(),
  contentType: z.string(),
  addedBy: z.string(),
  createdAt: stamp,
})
export type SupplierDocument = z.infer<typeof documentSchema>

export const brandLinkSchema = z.object({
  id: z.string(),
  supplierId: z.string(),
  supplierName: z.string(),
  supplierStatus: supplierStatusSchema,
  brandId: z.string(),
  brandName: z.string(),
  status: linkStatusSchema,
  authorisationReference: z.string().nullable(),
  evidenceDocumentId: z.string().nullable(),
  effectiveFrom: z.iso.date().nullable(),
  effectiveTo: z.iso.date().nullable(),
  verifiedBy: z.string().nullable(),
  verifiedAt: stamp.nullable(),
  note: z.string().nullable(),
  endedReason: z.string().nullable(),
  /** Server-derived: verified, supplier verified and today within the dates. */
  current: z.boolean(),
  version: z.number().int().positive(),
  createdAt: stamp,
  updatedAt: stamp,
})
export type BrandLink = z.infer<typeof brandLinkSchema>

export const supplierDetailSchema = supplierSchema.extend({
  contacts: z.array(contactSchema),
  documents: z.array(documentSchema),
  brandLinks: z.array(brandLinkSchema),
})
export type SupplierDetail = z.infer<typeof supplierDetailSchema>

export interface SupplierListParams {
  q?: string
  status?: SupplierStatus
  brandId?: string
  page?: number
  limit?: number
}
export interface SupplierFields {
  legalName: string
  tradeName?: string | null
  gstin?: string | null
  addressLine1?: string | null
  addressLine2?: string | null
  city?: string | null
  state?: string | null
  postalCode?: string | null
  notes?: string | null
}
export interface ContactFields {
  name: string
  role?: string | null
  phone?: string | null
  email?: string | null
  isPrimary: boolean
}

const page = pageSchema(supplierSchema)
const id = (v: string) => encodeURIComponent(v)

export function listSuppliers(
  params: SupplierListParams = {},
  signal?: AbortSignal,
): Promise<Page<Supplier>> {
  return apiRequest(`/suppliers${queryString(params)}`, page, { signal })
}
export function getSupplier(supplierId: string, signal?: AbortSignal): Promise<SupplierDetail> {
  return apiRequest(`/suppliers/${id(supplierId)}`, supplierDetailSchema, { signal })
}
export function createSupplier(input: SupplierFields): Promise<Supplier> {
  return apiRequest('/suppliers', supplierSchema, { method: 'POST', body: input })
}
export function updateSupplier(
  supplierId: string,
  input: Partial<SupplierFields> & { version: number },
): Promise<Supplier> {
  return apiRequest(`/suppliers/${id(supplierId)}`, supplierSchema, {
    method: 'PATCH',
    body: input,
  })
}
export function verifySupplier(
  supplierId: string,
  input: { version: number; reference: string; documentId: string },
): Promise<Supplier> {
  return apiRequest(`/suppliers/${id(supplierId)}/verify`, supplierSchema, {
    method: 'POST',
    body: input,
  })
}
export function archiveSupplier(
  supplierId: string,
  input: { version: number; reason?: string },
): Promise<Supplier> {
  return apiRequest(`/suppliers/${id(supplierId)}/archive`, supplierSchema, {
    method: 'POST',
    body: input,
  })
}
export function addContact(supplierId: string, input: ContactFields): Promise<Contact> {
  return apiRequest(`/suppliers/${id(supplierId)}/contacts`, contactSchema, {
    method: 'POST',
    body: input,
  })
}
export function updateContact(
  supplierId: string,
  contactId: string,
  input: Partial<ContactFields> & { version: number },
): Promise<Contact> {
  return apiRequest(`/suppliers/${id(supplierId)}/contacts/${id(contactId)}`, contactSchema, {
    method: 'PATCH',
    body: input,
  })
}
export function removeContact(
  supplierId: string,
  contactId: string,
  version: number,
): Promise<Contact> {
  return apiRequest(
    `/suppliers/${id(supplierId)}/contacts/${id(contactId)}/remove`,
    contactSchema,
    {
      method: 'POST',
      body: { version },
    },
  )
}
export function addDocument(
  supplierId: string,
  input: { attachmentId: string; documentType: DocumentType; note?: string },
): Promise<SupplierDocument> {
  return apiRequest(`/suppliers/${id(supplierId)}/documents`, documentSchema, {
    method: 'POST',
    body: input,
  })
}
export function listBrandLinks(
  params: { brandId?: string; supplierId?: string },
  signal?: AbortSignal,
): Promise<BrandLink[]> {
  return apiRequest(`/supplier-brand-links${queryString(params)}`, z.array(brandLinkSchema), {
    signal,
  })
}
export function createBrandLink(
  supplierId: string,
  input: { brandId: string; note?: string },
): Promise<BrandLink> {
  return apiRequest(`/suppliers/${id(supplierId)}/brand-links`, brandLinkSchema, {
    method: 'POST',
    body: input,
  })
}
export function verifyBrandLink(
  linkId: string,
  input: {
    version: number
    authorisationReference: string
    evidenceDocumentId: string
    effectiveFrom: string
    effectiveTo?: string | null
  },
): Promise<BrandLink> {
  return apiRequest(`/supplier-brand-links/${id(linkId)}/verify`, brandLinkSchema, {
    method: 'POST',
    body: input,
  })
}
export function endBrandLink(
  linkId: string,
  input: { version: number; reason: string; effectiveTo?: string | null },
): Promise<BrandLink> {
  return apiRequest(`/supplier-brand-links/${id(linkId)}/end`, brandLinkSchema, {
    method: 'POST',
    body: input,
  })
}
