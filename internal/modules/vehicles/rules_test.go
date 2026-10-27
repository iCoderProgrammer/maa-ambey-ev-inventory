package vehicles

import (
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/migrations"
)

func TestNormaliseIdentifierTrimsAndUpperCasesOnly(t *testing.T) {
	cases := map[string]string{
		"  ch-0001 ": "CH-0001",
		"ab12cd":     "AB12CD",
		"\tMot 77\n": "MOT 77",
		"CH/0001":    "CH/0001", // separators are kept until OQ-26 says which may be ignored
		"ch 00 01":   "CH 00 01",
		"":           "",
		"   ":        "",
		"ä-chassis ": "Ä-CHASSIS",
	}
	for in, want := range cases {
		if got := NormaliseIdentifier(in); got != want {
			t.Errorf("NormaliseIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}

// The statuses are the approved list (INV-005), in the database CHECK too.
func TestStatusesAreTheApprovedListAndMatchTheDatabase(t *testing.T) {
	approved := []string{"ORDERED", "IN_TRANSIT", "RECEIVED_PENDING_PDI", "QUALITY_HOLD", "AVAILABLE", "DISPLAY", "RESERVED",
		"SERVICE_HOLD", "SOLD", "DELIVERED", "REJECTED", "RETURN_PENDING_INSPECTION", "RETURNED_TO_SUPPLIER"}
	if !slices.Equal(Statuses, approved) {
		t.Fatalf("Statuses = %v, want INV-005 %v", Statuses, approved)
	}
	sql, err := fs.ReadFile(migrations.FS, "000013_create_vehicle_units.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)vehicle_units_status_valid CHECK \(status IN \((.*?)\)\)`).FindSubmatch(sql)
	if m == nil {
		t.Fatal("status CHECK not found")
	}
	var db []string
	for _, v := range strings.Split(string(m[1]), ",") {
		db = append(db, strings.Trim(strings.TrimSpace(v), "'"))
	}
	if !slices.Equal(db, approved) {
		t.Fatalf("database statuses = %v, want %v", db, approved)
	}
}

func TestValidateList(t *testing.T) {
	f, err := validateList(ListQuery{Query: "  ch-00 ", Status: "AVAILABLE", Page: "2", Limit: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Query != "CH-00" || f.Status != "AVAILABLE" || f.Page != 2 || f.Limit != 10 {
		t.Fatalf("filter = %+v", f)
	}
	if f, _ := validateList(ListQuery{}); f.Page != 1 || f.Limit != 25 {
		t.Fatalf("defaults = %+v", f)
	}
	_, err = validateList(ListQuery{Query: strings.Repeat("x", 101), Status: "PARKED", BrandID: "b", VariantID: "v",
		SupplierID: "s", PurchaseOrderID: "p", GoodsReceiptID: "g", LocationID: "l", Page: "0", Limit: "101"})
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v", err)
	}
	var fields []string
	for _, p := range verr.Fields {
		fields = append(fields, p.Field)
	}
	want := []string{"q", "status", "brandId", "variantId", "supplierId", "purchaseOrderId", "goodsReceiptId", "locationId", "page", "limit"}
	if !slices.Equal(fields, want) {
		t.Fatalf("fields = %v, want %v", fields, want)
	}
}

func TestCostIsRemovedWithoutCostPermission(t *testing.T) {
	cost, cur := "95000.00", "INR"
	u := Unit{UnitCost: &cost, CostCurrency: &cur}
	shapeCost(&u, access.NewAccess("u", nil, []string{access.PermVehicleUnitView}, nil, false))
	if u.UnitCost != nil || u.CostCurrency != nil || !u.CostHidden {
		t.Fatalf("view only: %+v", u)
	}
	u = Unit{UnitCost: &cost, CostCurrency: &cur}
	shapeCost(&u, access.NewAccess("u", nil, []string{access.PermVehicleUnitView, access.PermVehicleUnitCostView}, nil, false))
	if u.UnitCost == nil || *u.UnitCost != cost || u.CostHidden {
		t.Fatalf("with cost permission: %+v", u)
	}
	// Other cost permissions do not unlock unit costs.
	u = Unit{UnitCost: &cost, CostCurrency: &cur}
	shapeCost(&u, access.NewAccess("u", nil, []string{access.PermVehicleUnitView, access.PermCostPriceView}, nil, false))
	if u.UnitCost != nil || !u.CostHidden {
		t.Fatalf("catalogue cost permission: %+v", u)
	}
}
