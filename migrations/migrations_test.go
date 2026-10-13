package migrations

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestEmbeddedMigrationsFollowPolicy(t *testing.T) {
	latest, err := Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest < 1 {
		t.Fatalf("latest = %d, want at least 1", latest)
	}
}

func TestValidate(t *testing.T) {
	file := &fstest.MapFile{Data: []byte("SELECT 1;")}
	tests := []struct {
		name    string
		files   []string
		want    uint
		wantErr string
	}{
		{"empty", nil, 0, ""},
		{"paired and sequential", []string{
			"000001_a.up.sql", "000001_a.down.sql", "000002_b_c.up.sql", "000002_b_c.down.sql", "README.md",
		}, 2, ""},
		{"missing down", []string{"000001_a.up.sql"}, 0, "has no .down.sql"},
		{"missing up", []string{"000001_a.down.sql"}, 0, "has no .up.sql"},
		{"gap", []string{"000001_a.up.sql", "000001_a.down.sql", "000003_c.up.sql", "000003_c.down.sql"}, 0, "000002 is missing"},
		{"duplicate version", []string{"000001_a.up.sql", "000001_a.down.sql", "000001_b.up.sql", "000001_b.down.sql"}, 0, "used by two migrations"},
		{"version zero", []string{"000000_a.up.sql", "000000_a.down.sql"}, 0, "versions start at 000001"},
		{"bad name", []string{"1_CreateThings.up.sql"}, 0, "name must look like"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for _, f := range tt.files {
				fsys[f] = file
			}
			got, err := Validate(fsys)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got (%d, %v), want (%d, nil)", got, err, tt.want)
			}
		})
	}
}
