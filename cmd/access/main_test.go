package main

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"roles", []string{"roles"}, ""},
		{"proposed grant", []string{"grant", "--role", "viewer", "--permission", "access.roles.view"}, ""},
		{"approved grant", []string{"grant", "--role", "owner", "--permission", "reports.export", "--approved-by", "Owner", "--approval-ref", "Email 2026-10-12"}, ""},
		{"approval without reference", []string{"grant", "--role", "owner", "--permission", "reports.export", "--approved-by", "Owner"}, "both --approved-by and --approval-ref"},
		{"reference without approver", []string{"grant", "--role", "owner", "--permission", "reports.export", "--approval-ref", "x"}, "both --approved-by and --approval-ref"},
		{"grant without role", []string{"grant", "--permission", "x.y"}, "needs --role"},
		{"assign", []string{"assign", "--username", "Asha", "--role", "viewer"}, ""},
		{"assign without role", []string{"assign", "--username", "asha"}, "needs --role"},
		{"location", []string{"location-create", "--code", "sr1", "--name", "Showroom"}, ""},
		{"bad location code", []string{"location-create", "--code", "S", "--name", "Showroom"}, "--code must be"},
		{"location without name", []string{"location-create", "--code", "SR1"}, "--name is required"},
		{"scope", []string{"scope-add", "--username", "asha", "--location", "SR1"}, ""},
		{"unknown", []string{"grant-all"}, "unknown command"},
		{"empty", nil, "usage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := parse(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q (parsed %+v)", err, tt.wantErr, r)
			}
		})
	}
	if r, _ := parse([]string{"location-create", "--code", "sr1", "--name", "X"}); r.location != "SR1" {
		t.Errorf("location codes are stored upper-case: %q", r.location)
	}
}
