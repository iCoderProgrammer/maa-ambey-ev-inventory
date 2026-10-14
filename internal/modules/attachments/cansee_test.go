package attachments

import (
	"testing"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Supplier documents (D-01) are visible to holders of attachments.view and
// suppliers.view; other linked owners stay denied until their module adds a
// check (fail closed).
func TestCanSeeLinkedAttachments(t *testing.T) {
	supplierDoc := Attachment{UploadedBy: "u1", OwnerType: "supplier", OwnerID: "s1"}
	other := Attachment{UploadedBy: "u1", OwnerType: "goods_receipt", OwnerID: "g1"}
	both := access.NewAccess("u2", nil, []string{access.PermAttachmentsView, access.PermSupplierView}, nil, false)
	onlyFiles := access.NewAccess("u2", nil, []string{access.PermAttachmentsView}, nil, false)
	onlySuppliers := access.NewAccess("u2", nil, []string{access.PermSupplierView}, nil, false)
	if !canSee(both, supplierDoc) {
		t.Fatal("supplier viewer with attachments.view cannot see the supplier document")
	}
	if canSee(onlyFiles, supplierDoc) || canSee(onlySuppliers, supplierDoc) {
		t.Fatal("supplier document visible without both permissions")
	}
	if canSee(both, other) {
		t.Fatal("unknown owner type visible")
	}
	// The uploader no longer has special access once the file is linked.
	uploader := access.NewAccess("u1", nil, []string{access.PermAttachmentsView}, nil, false)
	if canSee(uploader, supplierDoc) {
		t.Fatal("uploader sees a linked supplier document without suppliers.view")
	}
}
