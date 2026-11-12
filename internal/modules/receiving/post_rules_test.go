package receiving

import (
	"errors"
	"strings"
	"testing"
)

func TestPostBlockers(t *testing.T) {
	if b := postBlockers(nil); len(b) != 1 || b[0].Field != "lines" {
		t.Fatalf("empty receipt: %+v", b)
	}
	ok := postLine{Index: 0, LineNo: 1, Ordered: 5, Posted: 1, Received: 3, Accepted: 2, Rejected: 1, CapturedAccepted: 2, CapturedRejected: 1}
	if b := postBlockers([]postLine{ok}); len(b) != 0 {
		t.Fatalf("complete line blocked: %+v", b)
	}
	missing := ok
	missing.Index, missing.CapturedRejected = 2, 0
	over := ok
	over.Index, over.Posted = 1, 4
	b := postBlockers([]postLine{missing, over})
	if len(b) != 2 || b[0].Field != "lines[2].units" || b[1].Field != "lines[1].receivedQuantity" || !strings.Contains(b[1].Message, "only 1 left") {
		t.Fatalf("blockers %+v", b)
	}
	covered := over
	q := 3
	covered.OverReceipt = &q
	if b := postBlockers([]postLine{covered}); len(b) != 0 {
		t.Fatalf("authorised over-receipt blocked: %+v", b)
	}
	short := 2
	covered.OverReceipt = &short
	if b := postBlockers([]postLine{covered}); len(b) != 1 {
		t.Fatalf("authorisation below the received quantity accepted: %+v", b)
	}
}

func TestValidatePostAndHash(t *testing.T) {
	v := 3
	loc := "7C9E6679-7425-40DE-944B-E07FC1F90AE7"
	p, err := validatePost("r1", PostInput{Version: &v, LocationID: loc}, "post-key-01")
	if err != nil || p.LocationID != strings.ToLower(loc) || p.Hash == "" {
		t.Fatalf("%+v %v", p, err)
	}
	q, _ := validatePost("r1", PostInput{Version: &v, LocationID: strings.ToLower(loc)}, "other-key-1")
	w := 4
	r, _ := validatePost("r1", PostInput{Version: &w, LocationID: loc}, "post-key-01")
	if q.Hash != p.Hash || r.Hash == p.Hash {
		t.Fatal("hash must follow receipt, version and location only")
	}
	_, err = validatePost("r1", PostInput{LocationID: "x"}, "k")
	var verr *ValidationError
	if !errors.As(err, &verr) || len(verr.Fields) != 3 {
		t.Fatalf("validation %v", err)
	}
	if newPOStatus(true) != "RECEIVED" || newPOStatus(false) != "PARTIALLY_RECEIVED" {
		t.Fatal("PO status")
	}
}
