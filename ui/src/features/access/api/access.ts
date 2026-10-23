import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'

/**
 * Permissions contract (docs/API_CONVENTIONS.md, "Authorisation"). The server is
 * authoritative: this only decides what to show; every request is checked again.
 */

const locationSchema = z.object({ id: z.string(), code: z.string(), name: z.string() })

export const meSchema = z.object({
  user: z.object({ id: z.string(), username: z.string(), displayName: z.string() }),
  roles: z.array(z.object({ key: z.string(), name: z.string() })),
  /** Effective permission keys. Empty until the owner approves the permission matrix. */
  permissions: z.array(z.string()),
  locationScope: z.object({ all: z.boolean(), locations: z.array(locationSchema) }),
  /** True only in local development when proposed (unapproved) grants are switched on. */
  proposedGrantsActive: z.boolean(),
})
export type Me = z.infer<typeof meSchema>

const grantSchema = z.object({
  permission: z.string(),
  status: z.enum(['PROPOSED', 'APPROVED']),
  approvedBy: z.string().nullable(),
  approvalReference: z.string().nullable(),
  approvedAt: z.string().nullable(),
})

export const roleMatrixSchema = z.object({
  matrixStatus: z.enum(['PENDING_OWNER_SIGN_OFF', 'PARTIALLY_APPROVED']),
  approvedGrantCount: z.number(),
  roles: z.array(
    z.object({
      key: z.string(),
      name: z.string(),
      description: z.string(),
      grants: z.array(grantSchema),
    }),
  ),
  permissions: z.array(
    z.object({ key: z.string(), description: z.string(), sensitive: z.boolean() }),
  ),
})
export type RoleMatrix = z.infer<typeof roleMatrixSchema>

/** Permission keys the UI refers to. Must exist in the server catalogue. */
export const Permission = {
  AccessRolesView: 'access.roles.view',
  AttachmentsUpload: 'attachments.upload',
  AttachmentsView: 'attachments.view',
  CatalogueView: 'catalogue.view',
  CatalogueManage: 'catalogue.manage',
  CatalogueVerify: 'catalogue.verify',
  CatalogueActivate: 'catalogue.activate',
  CatalogueArchive: 'catalogue.archive',
  BrandAuthorise: 'catalogue.brand.authorise',
  BrandActivate: 'catalogue.brand.activate',
  PriceView: 'catalogue.price.view',
  PriceManage: 'catalogue.price.manage',
  PriceApprove: 'catalogue.price.approve',
  CostPriceView: 'catalogue.cost_price.view',
  CostPriceEdit: 'catalogue.cost_price.edit',
  SupplierView: 'suppliers.view',
  SupplierManage: 'suppliers.manage',
  SupplierVerify: 'suppliers.verify',
  SupplierArchive: 'suppliers.archive',
  PurchaseOrderView: 'purchase_orders.view',
  PurchaseOrderManage: 'purchase_orders.manage',
  PurchaseOrderApprove: 'purchase_orders.approve',
  PurchaseOrderCancel: 'purchase_orders.cancel',
  PurchaseOrderSettings: 'purchase_orders.settings.manage',
  GoodsReceiptView: 'goods_receipts.view',
  GoodsReceiptManage: 'goods_receipts.manage',
  GoodsReceiptOverReceipt: 'goods_receipts.over_receipt',
} as const

/** GET /me — the signed-in user's roles, effective permissions and location scope. */
export function getMe(signal?: AbortSignal): Promise<Me> {
  return apiRequest('/me', meSchema, { signal })
}

/** GET /access/roles — role catalogue and grant status. Needs access.roles.view. */
export function getRoleMatrix(signal?: AbortSignal): Promise<RoleMatrix> {
  return apiRequest('/access/roles', roleMatrixSchema, { signal })
}
