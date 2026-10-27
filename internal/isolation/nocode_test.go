package isolation

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// rosterNames reads the brand names from the seed, so this test does not
// itself hardcode any brand.
func rosterNames(t *testing.T, root string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "seeds", "0001_initial_brands.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range regexp.MustCompile(`\('([^']+)',\s*'[^']+'\)`).FindAllStringSubmatch(string(b), -1) {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("no brand names found in the seed")
	}
	return names
}

// TestNoBrandNameInApplicationCode: brands are data. No Go or UI source (tests
// excluded) may name a brand, so adding or renaming one needs no code change
// and no brand gets special treatment in code (C-06, BR-002).
func TestNoBrandNameInApplicationCode(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	names := rosterNames(t, root)
	// Also forbid the related name the client asked not to assume equivalent.
	names = append(names, "Saffron")
	checked := 0
	for _, dir := range []string{"cmd", "internal", filepath.Join("ui", "src")} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".sql" {
				return nil
			}
			base := filepath.Base(path)
			if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			checked++
			lower := strings.ToLower(string(b))
			for _, n := range names {
				if strings.Contains(lower, strings.ToLower(n)) {
					t.Errorf("%s names brand %q; brands must come from data", strings.TrimPrefix(path, root+"/"), n)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if checked < 50 {
		t.Fatalf("only %d files scanned; the walk is broken", checked)
	}
}
