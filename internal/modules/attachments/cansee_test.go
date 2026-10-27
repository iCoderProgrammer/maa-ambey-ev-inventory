package attachments

import (
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Supplier documents (D-01) are visible to holders of attachments.view and
// suppliers.view, goods receipt documents (D-03) to holders of attachments.view
// and goods_receipts.view; other linked owners stay denied until their module
// adds a check (fail closed).
func TestCanSeeLinkedAttachments(t *testing.T) {
	supplierDoc := Attachment{UploadedBy: "u1", OwnerType: "supplier", OwnerID: "s1"}
	receiptDoc := Attachment{UploadedBy: "u1", OwnerType: "goods_receipt", OwnerID: "g1"}
	other := Attachment{UploadedBy: "u1", OwnerType: "pdi_inspection", OwnerID: "p1"}
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
	// The uploader no longer has special access once the file is linked.
	uploader := access.NewAccess("u1", nil, []string{access.PermAttachmentsView}, nil, false)
	if canSee(uploader, supplierDoc) || canSee(uploader, receiptDoc) {
		t.Fatal("uploader sees a linked document without its module's view permission")
	}
}
