// Package access implements role-based access control and location scope (B-07).
//
// Deny by default: a request may do something only if the signed-in user holds
// a role with an APPROVED grant for the permission (PROPOSED grants count only
// when RBAC_ALLOW_PROPOSED_GRANTS is on, accepted only with APP_ENV=development). The
// permission matrix is pending owner sign-off, so out of the box nobody holds
// any permission.
//
// Routes: protect them with Handler.Require(access.PermX). Services: check
// objects with Access.Has and Access.Scope (Allows / Restrict / Check) — a
// route guard alone does not stop cross-location access to a specific record.
package access

import (
	"errors"
	"slices"
)

// Permission keys. Every key must exist in the permissions table (migrations
// 000004, 000006, 000008, 000009, 000010, 000011, 000012, 000013, 000015); TestPermissionCatalogueMatchesDatabase keeps them in step.
const (
	PermRolesView         = "access.roles.view"
	PermRolesManage       = "access.roles.manage"
	PermAuditView         = "audit.events.view"
	PermAllLocations      = "locations.all"
	PermCostPriceView     = "catalogue.cost_price.view"
	PermCostPriceEdit     = "catalogue.cost_price.edit"
	PermDiscountApprove   = "sales.discount.approve"
	PermRefundCreate      = "sales.refund.create"
	PermRefundApprove     = "sales.refund.approve"
	PermAdjustmentCreate  = "stock.adjustment.create"
	PermAdjustmentApprove = "stock.adjustment.approve"
	PermInvoiceSettings   = "settings.invoice.manage"
	PermTaxSettings       = "settings.tax.manage"
	PermReportsExport     = "reports.export"
	PermCustomersExport   = "customers.export"
	// Private attachments (B-09, migration 000006).
	PermAttachmentsUpload = "attachments.upload"
	PermAttachmentsView   = "attachments.view"
	// Catalogue API (C-02, migration 000008).
	PermCatalogueView     = "catalogue.view"
	PermCatalogueManage   = "catalogue.manage"
	PermCatalogueVerify   = "catalogue.verify"
	PermCatalogueActivate = "catalogue.activate"
	PermCatalogueArchive  = "catalogue.archive"
	PermBrandAuthorise    = "catalogue.brand.authorise"
	PermBrandActivate     = "catalogue.brand.activate"
	// Price history (C-04, migration 000009). Cost prices use PermCostPrice*.
	PermPriceView    = "catalogue.price.view"
	PermPriceManage  = "catalogue.price.manage"
	PermPriceApprove = "catalogue.price.approve"
	// Supplier master (D-01, migration 000010).
	PermSupplierView    = "suppliers.view"
	PermSupplierManage  = "suppliers.manage"
	PermSupplierVerify  = "suppliers.verify"
	PermSupplierArchive = "suppliers.archive"
	// Purchase orders (D-02, migration 000011).
	PermPurchaseOrderView     = "purchase_orders.view"
	PermPurchaseOrderManage   = "purchase_orders.manage"
	PermPurchaseOrderApprove  = "purchase_orders.approve"
	PermPurchaseOrderCancel   = "purchase_orders.cancel"
	PermPurchaseOrderSettings = "purchase_orders.settings.manage"
	// Goods receipt drafts (D-03, migration 000012).
	PermGoodsReceiptView        = "goods_receipts.view"
	PermGoodsReceiptManage      = "goods_receipts.manage"
	PermGoodsReceiptOverReceipt = "goods_receipts.over_receipt"
	// Vehicle unit registry (D-04, migration 000013).
	PermVehicleUnitView     = "vehicle_units.view"
	PermVehicleUnitCostView = "vehicle_units.cost.view"
	// Stock movements (E-01, migration 000015): AVAILABLE <-> DISPLAY.
	PermVehicleUnitDisplayManage = "vehicle_units.display.manage"
)

// AllPermissions lists every key the code knows.
var AllPermissions = []string{
	PermRolesView, PermRolesManage, PermAuditView, PermAllLocations,
	PermCostPriceView, PermCostPriceEdit, PermDiscountApprove, PermRefundCreate, PermRefundApprove,
	PermAdjustmentCreate, PermAdjustmentApprove, PermInvoiceSettings, PermTaxSettings,
	PermReportsExport, PermCustomersExport, PermAttachmentsUpload, PermAttachmentsView,
	PermCatalogueView, PermCatalogueManage, PermCatalogueVerify, PermCatalogueActivate, PermCatalogueArchive,
	PermBrandAuthorise, PermBrandActivate, PermPriceView, PermPriceManage, PermPriceApprove,
	PermSupplierView, PermSupplierManage, PermSupplierVerify, PermSupplierArchive,
	PermPurchaseOrderView, PermPurchaseOrderManage, PermPurchaseOrderApprove, PermPurchaseOrderCancel, PermPurchaseOrderSettings,
	PermGoodsReceiptView, PermGoodsReceiptManage, PermGoodsReceiptOverReceipt,
	PermVehicleUnitView, PermVehicleUnitCostView, PermVehicleUnitDisplayManage,
}

// ErrOutOfScope means a location outside the user's assigned scope was requested.
var ErrOutOfScope = errors.New("location outside the user's scope")

// Role is a role the user holds.
type Role struct {
	Key  string `json:"key" example:"viewer"`
	Name string `json:"name" example:"Viewer"`
} //@name AccessRole

// Location is a location in the user's scope.
type Location struct {
	ID   string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Code string `json:"code" example:"SR1"`
	Name string `json:"name" example:"Main showroom"`
} //@name AccessLocation

// Access is everything the server decided a user may do. Build it with
// Store.LoadAccess; never from client input.
type Access struct {
	UserID               string
	Roles                []Role
	permissions          map[string]struct{}
	Scope                Scope
	ProposedGrantsActive bool
}

// NewAccess builds an Access; the scope's All flag follows the locations.all permission.
func NewAccess(userID string, roles []Role, permissions []string, locations []Location, proposedActive bool) Access {
	set := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		set[p] = struct{}{}
	}
	_, all := set[PermAllLocations]
	return Access{
		UserID: userID, Roles: roles, permissions: set, ProposedGrantsActive: proposedActive,
		Scope: Scope{All: all, Locations: locations},
	}
}

// Has reports whether the permission is granted. Unknown keys are never granted.
func (a Access) Has(permission string) bool {
	_, ok := a.permissions[permission]
	return ok
}

// Permissions returns the granted keys, sorted.
func (a Access) Permissions() []string {
	out := make([]string, 0, len(a.permissions))
	for p := range a.permissions {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}

// Scope is the set of locations a user may work in.
type Scope struct {
	All       bool
	Locations []Location
}

// Allows reports whether the location is in scope.
func (s Scope) Allows(locationID string) bool {
	if s.All {
		return true
	}
	return slices.ContainsFunc(s.Locations, func(l Location) bool { return l.ID == locationID })
}

// Check returns ErrOutOfScope unless the location is in scope. Use it before
// reading or changing a record that belongs to a location.
func (s Scope) Check(locationID string) error {
	if !s.Allows(locationID) {
		return ErrOutOfScope
	}
	return nil
}

// Restrict turns a client-supplied location filter into the locations a query
// may use: nil/empty means "everything in scope". Any requested location
// outside the scope is an error rather than being silently dropped. When the
// user has every location, the returned all flag is true and ids is the
// (possibly empty) requested filter.
func (s Scope) Restrict(requested []string) (ids []string, all bool, err error) {
	if s.All {
		return requested, len(requested) == 0, nil
	}
	if len(requested) == 0 {
		ids = make([]string, len(s.Locations))
		for i, l := range s.Locations {
			ids[i] = l.ID
		}
		return ids, false, nil
	}
	for _, id := range requested {
		if !s.Allows(id) {
			return nil, false, ErrOutOfScope
		}
	}
	return requested, false, nil
}
