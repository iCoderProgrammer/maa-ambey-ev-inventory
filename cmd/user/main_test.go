package main

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    request
		wantErr string
	}{
		{"list", []string{"list"}, request{command: "list"}, ""},
		{"create normalises username", []string{"create", "--username", " Asha.Verma ", "--name", " Asha Verma "},
			request{command: "create", username: "asha.verma", displayName: "Asha Verma"}, ""},
		{"disable", []string{"disable", "--username", "asha"}, request{command: "disable", username: "asha"}, ""},
		{"no command", nil, request{}, "usage"},
		{"unknown command", []string{"delete", "--username", "asha"}, request{}, "unknown command"},
		{"password flag refused", []string{"create", "--username", "asha", "--name", "A", "--password", "x"}, request{}, "flag provided but not defined"},
		{"bad username", []string{"create", "--username", "a b", "--name", "A"}, request{}, "--username must be"},
		{"short username", []string{"enable", "--username", "ab"}, request{}, "--username must be"},
		{"missing name", []string{"create", "--username", "asha"}, request{}, "--name is required"},
		{"bad email", []string{"create", "--username", "asha", "--name", "A", "--email", "nope"}, request{}, "--email"},
		{"name on disable", []string{"disable", "--username", "asha", "--name", "A"}, request{}, "only valid with create"},
		{"stray argument", []string{"list", "extra"}, request{}, "unexpected argument"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got.command != tt.want.command || got.username != tt.want.username || got.displayName != tt.want.displayName {
				t.Fatalf("got (%+v, %v), want %+v", got, err, tt.want)
			}
		})
	}
}

func TestNewPasswordHashFromStdin(t *testing.T) {
	var out strings.Builder
	hash, err := newPasswordHash("asha", strings.NewReader("long enough passphrase\n"), &out)
	if err != nil || !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash=%q err=%v", hash, err)
	}
	if _, err := newPasswordHash("asha", strings.NewReader("short\n"), &out); err == nil {
		t.Error("policy must reject short passwords")
	}
	if strings.Contains(out.String(), "passphrase") {
		t.Error("password echoed to output")
	}
}
