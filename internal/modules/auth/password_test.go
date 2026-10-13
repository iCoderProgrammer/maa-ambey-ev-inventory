package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected format: %s", hash)
	}
	if strings.Contains(hash, "correct horse") {
		t.Fatal("hash contains the password")
	}
	ok, err := VerifyPassword(hash, "correct horse battery staple")
	if err != nil || !ok {
		t.Fatalf("verify correct = %v, %v", ok, err)
	}
	ok, err = VerifyPassword(hash, "correct horse battery staplE")
	if err != nil || ok {
		t.Fatalf("verify wrong = %v, %v", ok, err)
	}

	other, _ := HashPassword("correct horse battery staple")
	if other == hash {
		t.Fatal("hashes must be salted")
	}
}

func TestVerifyRejectsMalformedOrAbusiveHashes(t *testing.T) {
	for _, h := range []string{
		"",
		"plaintext",
		"$2a$10$bcrypthashbcrypthashbcrypthashbcrypthashbcrypthash12345",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHQ$",
		"$argon2id$v=19$m=99999999,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaA",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaA",
	} {
		if ok, err := VerifyPassword(h, "x"); ok || err == nil {
			t.Errorf("hash %q: ok=%v err=%v, want rejection", h, ok, err)
		}
	}
}

func TestValidateNewPassword(t *testing.T) {
	tests := []struct {
		password, wantErr string
	}{
		{"short", "at least 12"},
		{strings.Repeat("a", 129), "at most 128"},
		{"asha.verma.01", ""},
		{"Asha.Verma.01", ""},
		{"long enough passphrase", ""},
		{"\xff\xfe invalid utf8 x", "valid text"},
	}
	for _, tt := range tests {
		err := ValidateNewPassword("asha.verma", tt.password)
		if tt.wantErr == "" {
			if err != nil {
				t.Errorf("%q: unexpected %v", tt.password, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%q: err = %v, want %q", tt.password, err, tt.wantErr)
		}
	}
	if err := ValidateNewPassword("asha.verma.01", "Asha.Verma.01"); err == nil {
		t.Error("password equal to the username must be rejected")
	}
}
