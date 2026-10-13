package access

import (
	"errors"
	"slices"
	"testing"
)

var (
	sr1 = Location{ID: "loc-1", Code: "SR1", Name: "Showroom 1"}
	sr2 = Location{ID: "loc-2", Code: "SR2", Name: "Showroom 2"}
)

func TestAccessHasOnlyGrantedPermissions(t *testing.T) {
	a := NewAccess("u", nil, []string{PermRolesView}, nil, false)
	if !a.Has(PermRolesView) {
		t.Error("granted permission missing")
	}
	for _, p := range []string{PermRolesManage, PermCostPriceView, "", "access.roles.view ", "ACCESS.ROLES.VIEW"} {
		if a.Has(p) {
			t.Errorf("%q must not be granted", p)
		}
	}
	if none := NewAccess("u", nil, nil, nil, false); len(none.Permissions()) != 0 || none.Scope.All {
		t.Error("no grants must mean no permissions and no locations")
	}
}

func TestScopeAllows(t *testing.T) {
	scoped := NewAccess("u", nil, nil, []Location{sr1}, false).Scope
	if !scoped.Allows("loc-1") || scoped.Allows("loc-2") || scoped.Allows("") {
		t.Errorf("scoped user: %+v", scoped)
	}
	if !errors.Is(scoped.Check("loc-2"), ErrOutOfScope) || scoped.Check("loc-1") != nil {
		t.Error("Check must reject cross-location access")
	}
	unscoped := NewAccess("u", nil, nil, nil, false).Scope
	if unscoped.Allows("loc-1") {
		t.Error("a user with no locations reaches none")
	}
	everywhere := NewAccess("u", nil, []string{PermAllLocations}, nil, false).Scope
	if !everywhere.All || !everywhere.Allows("loc-9") {
		t.Error("locations.all must grant every location")
	}
}

func TestScopeRestrict(t *testing.T) {
	scoped := Scope{Locations: []Location{sr1, sr2}}
	ids, all, err := scoped.Restrict(nil)
	if err != nil || all || !slices.Equal(ids, []string{"loc-1", "loc-2"}) {
		t.Errorf("no filter: %v %v %v", ids, all, err)
	}
	if ids, _, err := scoped.Restrict([]string{"loc-2"}); err != nil || !slices.Equal(ids, []string{"loc-2"}) {
		t.Errorf("in-scope filter: %v %v", ids, err)
	}
	// Asking for another location's data is refused, not silently narrowed.
	if _, _, err := scoped.Restrict([]string{"loc-2", "loc-3"}); !errors.Is(err, ErrOutOfScope) {
		t.Errorf("cross-location filter: %v", err)
	}
	if ids, _, err := (Scope{}).Restrict(nil); err != nil || len(ids) != 0 {
		t.Errorf("empty scope must yield no locations: %v %v", ids, err)
	}
	if _, all, _ := (Scope{All: true}).Restrict(nil); !all {
		t.Error("all-locations user without a filter sees everything")
	}
}
