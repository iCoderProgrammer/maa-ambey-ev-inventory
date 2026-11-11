package pdi

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)
	codePattern        = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,39}$`)
)

type checker struct{ fields []FieldProblem }

func (c *checker) add(field, msg string) { c.fields = append(c.fields, FieldProblem{field, msg}) }

func (c *checker) err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: c.fields}
}

// text trims an optional text; blank or too long (or with control characters
// other than line breaks) is a problem.
func (c *checker) text(field string, s *string, max int) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	switch {
	case t == "":
		c.add(field, "must not be blank")
	case utf8.RuneCountInString(t) > max:
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	case strings.ContainsFunc(t, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' }):
		c.add(field, "must not contain control characters")
	}
	return &t
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil && len(s) == 36
}

// normalisedRecord is a RecordInput after static validation.
type normalisedRecord struct {
	Version       int
	Disposition   string
	Items         []ItemOutcomeInput // notes trimmed
	Notes         *string
	AttachmentIDs []string
}

// validateRecord checks everything that needs no database. The checklist
// itself (which items apply) and the attachments are checked against the
// database under the unit lock.
func validateRecord(in RecordInput, key string) (normalisedRecord, error) {
	var c checker
	out := normalisedRecord{Disposition: strings.TrimSpace(in.Disposition)}
	if !idempotencyPattern.MatchString(key) {
		c.add("Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'")
	}
	switch {
	case in.Version == nil:
		c.add("version", "is required")
	case *in.Version < 1:
		c.add("version", "must be 1 or more")
	default:
		out.Version = *in.Version
	}
	if !slices.Contains(Dispositions, out.Disposition) {
		c.add("disposition", "must be PASS, HOLD or REJECT")
	}
	if len(in.Items) > maxItems {
		c.add("items", "must have at most "+strconv.Itoa(maxItems)+" entries")
	}
	seen := map[string]bool{}
	failed := false
	for i, it := range in.Items {
		f := "items[" + strconv.Itoa(i) + "]"
		id := strings.ToLower(strings.TrimSpace(it.ChecklistItemID))
		switch {
		case !isUUID(id):
			c.add(f+".checklistItemId", "must be a checklist item ID")
		case seen[id]:
			c.add(f+".checklistItemId", "is listed twice")
		}
		seen[id] = true
		outcome := strings.TrimSpace(it.Outcome)
		if outcome != OutcomePass && outcome != OutcomeFail {
			c.add(f+".outcome", "must be PASS or FAIL")
		}
		note := c.text(f+".note", it.Note, maxItemNote)
		if outcome == OutcomeFail {
			failed = true
			if note == nil {
				c.add(f+".note", "is required for a failed item")
			}
		}
		out.Items = append(out.Items, ItemOutcomeInput{ChecklistItemID: id, Outcome: outcome, Note: note})
	}
	out.Notes = c.text("notes", in.Notes, maxNotes)
	if len(in.AttachmentIDs) > maxAttachments {
		c.add("attachmentIds", "must have at most "+strconv.Itoa(maxAttachments)+" entries")
	}
	seenAtt := map[string]bool{}
	for i, a := range in.AttachmentIDs {
		id := strings.ToLower(strings.TrimSpace(a))
		f := "attachmentIds[" + strconv.Itoa(i) + "]"
		switch {
		case !isUUID(id):
			c.add(f, "must be an attachment ID")
		case seenAtt[id]:
			c.add(f, "is listed twice")
		}
		seenAtt[id] = true
		out.AttachmentIDs = append(out.AttachmentIDs, id)
	}
	switch out.Disposition {
	case DispositionPass:
		if failed {
			c.add("disposition", "cannot be PASS when a checklist item failed")
		}
	case DispositionHold, DispositionReject:
		// INV-008: a failed PDI records issue notes and evidence.
		if out.Notes == nil && !slices.ContainsFunc(c.fields, func(p FieldProblem) bool { return p.Field == "notes" }) {
			c.add("notes", "are required for a hold or rejection: describe the issue")
		}
		if len(in.AttachmentIDs) == 0 {
			c.add("attachmentIds", "need at least one photo or document as evidence for a hold or rejection")
		}
	}
	return out, c.err()
}

// requestHash identifies one inspection request (unit, version, disposition,
// outcomes, notes, evidence), independent of the order of items and attachments.
func requestHash(unitID string, r normalisedRecord) string {
	items := make([]string, len(r.Items))
	for i, it := range r.Items {
		note := ""
		if it.Note != nil {
			note = *it.Note
		}
		items[i] = it.ChecklistItemID + "\x1f" + it.Outcome + "\x1f" + note
	}
	slices.Sort(items)
	atts := slices.Clone(r.AttachmentIDs)
	slices.Sort(atts)
	notes := ""
	if r.Notes != nil {
		notes = *r.Notes
	}
	parts := []string{strings.ToLower(unitID), strconv.Itoa(r.Version), r.Disposition, notes, strings.Join(items, "\x1e"),
		strings.Join(atts, "\x1e")}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1d")))
	return hex.EncodeToString(sum[:])
}

// normalisedItem is a validated checklist create request.
type normalisedItem struct {
	Code, Label string
	Description *string
	BrandID     *string
	SortOrder   int
}

func validateChecklistCreate(in ChecklistCreateInput) (normalisedItem, error) {
	var c checker
	out := normalisedItem{Code: strings.ToUpper(strings.TrimSpace(in.Code))}
	if !codePattern.MatchString(out.Code) {
		c.add("code", "must be 2-40 characters: a letter, then letters, digits or _")
	}
	label := c.text("label", &in.Label, maxLabel)
	if label != nil {
		out.Label = *label
	}
	out.Description = c.text("description", in.Description, maxDescription)
	if in.BrandID != nil {
		id := strings.ToLower(strings.TrimSpace(*in.BrandID))
		if !isUUID(id) {
			c.add("brandId", "must be a brand ID")
		}
		out.BrandID = &id
	}
	if in.SortOrder != nil {
		if *in.SortOrder < 0 || *in.SortOrder > 9999 {
			c.add("sortOrder", "must be from 0 to 9999")
		}
		out.SortOrder = *in.SortOrder
	}
	return out, c.err()
}

func validateRetire(in ChecklistRetireInput) (int, string, error) {
	var c checker
	v := 0
	switch {
	case in.Version == nil:
		c.add("version", "is required")
	case *in.Version < 1:
		c.add("version", "must be 1 or more")
	default:
		v = *in.Version
	}
	reason := c.text("reason", &in.Reason, maxReason)
	r := ""
	if reason != nil {
		r = *reason
	}
	return v, r, c.err()
}

// checklistFilter is a validated checklist list request.
type checklistFilter struct {
	Status, BrandID string
	Page, Limit     int
}

func validateChecklistQuery(q ChecklistQuery) (checklistFilter, error) {
	var c checker
	f := checklistFilter{Status: strings.TrimSpace(q.Status), BrandID: strings.ToLower(strings.TrimSpace(q.BrandID)), Page: 1, Limit: 50}
	if f.Status != "" && f.Status != ItemActive && f.Status != ItemRetired {
		c.add("status", "must be ACTIVE or RETIRED")
	}
	if f.BrandID != "" && !isUUID(f.BrandID) {
		c.add("brandId", "must be a brand ID")
	}
	if q.Page != "" {
		if n, err := strconv.Atoi(q.Page); err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			f.Page = n
		}
	}
	if q.Limit != "" {
		if n, err := strconv.Atoi(q.Limit); err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			f.Limit = n
		}
	}
	return f, c.err()
}
