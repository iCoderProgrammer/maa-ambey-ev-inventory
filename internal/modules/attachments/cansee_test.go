package attachments

import (
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Supplier documents (D-01) are visible to holders of attachments.view and
// suppliers.view, goods receipt documents (D-03) to holders of attachments.view
// and goods_receipts.view, PDI evidence (D-06) to holders of attachments.view
// and vehicle_units.view within the file's location; other linked owners stay
// denied until their module adds a check (fail closed).
func TestCanSeeLinkedAttachments(t *testing.T) {
	supplierDoc := Attachment{UploadedBy: "u1", OwnerType: "supplier", OwnerID: "s1"}
	receiptDoc := Attachment{UploadedBy: "u1", OwnerType: "goods_receipt", OwnerID: "g1"}
	other := Attachment{UploadedBy: "u1", OwnerType: "sales_order", OwnerID: "o1"}
	both := access.NewAccess("u2", nil, []string{access.PermAttachmentsView, access.PermSupplierView}, nil, false)
	onlyFiles := access.NewAccess("u2", nil, []string{access.PermAttachmentsView}, nil, false)
	onlySuppliers := access.NewAccess("u2", nil, []string{access.PermSupplierView}, nil, false)
	if !canSee(both, supplierDoc) {
		t.Fatal("supplier viewer with attachments.view cannot see the supplier document")
	}
	if canSee(onlyFiles, supplierDoc) || canSee(onlySuppliers, supplierDoc) {
		t.Fatal("supplier document visible without both permissions")
	}
	if canSee(both, receiptDoc) {
		t.Fatal("goods receipt document visible without goods_receipts.view")
	}
	receiptViewer := access.NewAccess("u2", nil, []string{access.PermAttachmentsView, access.PermGoodsReceiptView}, nil, false)
	onlyReceipts := access.NewAccess("u2", nil, []string{access.PermGoodsReceiptView}, nil, false)
	if !canSee(receiptViewer, receiptDoc) {
		t.Fatal("receipt viewer with attachments.view cannot see the receipt document")
	}
	if canSee(onlyReceipts, receiptDoc) || canSee(receiptViewer, supplierDoc) {
		t.Fatal("document visible without both of its permissions")
	}
	if canSee(both, other) || canSee(receiptViewer, other) {
		t.Fatal("unknown owner type visible")
	}
	pdiDoc := Attachment{UploadedBy: "u1", OwnerType: "pdi_inspection", OwnerID: "p1", LocationID: "loc-1"}
	unitViewer := access.NewAccess("u2", nil, []string{access.PermAttachmentsView, access.PermVehicleUnitView},
		[]access.Location{{ID: "loc-1"}}, false)
	otherLocation := access.NewAccess("u2", nil, []string{access.PermAttachmentsView, access.PermVehicleUnitView},
		[]access.Location{{ID: "loc-2"}}, false)
	onlyUnits := access.NewAccess("u2", nil, []string{access.PermVehicleUnitView, access.PermAllLocations}, nil, false)
	if !canSee(unitViewer, pdiDoc) {
		t.Fatal("unit viewer with attachments.view cannot see PDI evidence at their location")
	}
	if canSee(otherLocation, pdiDoc) || canSee(onlyUnits, pdiDoc) || canSee(receiptViewer, pdiDoc) || canSee(unitViewer, other) {
		t.Fatal("PDI evidence visible outside its location or without both permissions")
	}
	// The uploader no longer has special access once the file is linked.
	uploader := access.NewAccess("u1", nil, []string{access.PermAttachmentsView}, nil, false)
	if canSee(uploader, supplierDoc) || canSee(uploader, receiptDoc) {
		t.Fatal("uploader sees a linked document without its module's view permission")
	}
}

// Opening stock import files (G-06): attachments.view and stock.import.preview,
// and either the uploader or a holder of every location. The generic upload
// allow-list still refuses CSV: import files arrive only through the import route.
func TestCanSeeOpeningStockImportFiles(t *testing.T) {
	file := Attachment{UploadedBy: "u1", OwnerType: "opening_stock_import", OwnerID: "b1", ContentType: "text/csv"}
	perms := []string{access.PermAttachmentsView, access.PermImportPreview}
	if !canSee(access.NewAccess("u1", nil, perms, nil, false), file) {
		t.Fatal("uploader cannot see their import file")
	}
	if canSee(access.NewAccess("u2", nil, perms, nil, false), file) {
		t.Fatal("another importer sees the file")
	}
	if !canSee(access.NewAccess("u2", nil, append(perms, access.PermAllLocations), nil, false), file) {
		t.Fatal("an all-locations importer cannot see the file")
	}
	for _, p := range [][]string{{access.PermAttachmentsView}, {access.PermImportPreview}} {
		if canSee(access.NewAccess("u1", nil, p, nil, false), file) {
			t.Fatalf("visible with only %v", p)
		}
	}
	if _, ok := fileType("text/csv"); ok {
		t.Fatal("generic uploads must not accept CSV")
	}
}
