package whitelist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWhitelist_AddAndRemove(t *testing.T) {
	w := &Whitelist{patterns: make([]string, 0)}

	pattern := `C:\Users\test\AppData\Local\keep`
	if err := w.Add(pattern); err != nil {
		t.Fatalf("Add(%q) failed: %v", pattern, err)
	}

	list := w.List()
	if len(list) != 1 || !strings.EqualFold(list[0], pattern) {
		t.Errorf("expected [%q], got %v", pattern, list)
	}

	// Duplicate add should fail.
	if err := w.Add(pattern); err == nil {
		t.Error("Add duplicate pattern should return error")
	}

	if err := w.Remove(pattern); err != nil {
		t.Fatalf("Remove(%q) failed: %v", pattern, err)
	}
	if len(w.List()) != 0 {
		t.Error("list should be empty after remove")
	}

	// Remove non-existent should fail.
	if err := w.Remove(pattern); err == nil {
		t.Error("Remove non-existent pattern should return error")
	}
}

func TestWhitelist_AddRejectsBroadPatterns(t *testing.T) {
	// *, **, C:\, C:\*, D:\* must all be rejected.
	for _, p := range []string{"*", "**", `C:\`, `C:\*`, `D:\*`} {
		w := &Whitelist{patterns: make([]string, 0)}
		if err := w.Add(p); err == nil {
			t.Errorf("Add(%q) should reject broad pattern", p)
		}
	}
}

func TestWhitelist_AddRejectsShallowPatterns(t *testing.T) {
	// Patterns with <2 path separators must be rejected.
	for _, p := range []string{`C:\Users`, `D:\Folder`} {
		w := &Whitelist{patterns: make([]string, 0)}
		if err := w.Add(p); err == nil {
			t.Errorf("Add(%q) should reject pattern with <2 separators", p)
		}
	}
}

func TestWhitelist_IsWhitelisted(t *testing.T) {
	w := &Whitelist{patterns: make([]string, 0)}
	if err := w.Add(`C:\Users\test\AppData\keep`); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Exact match.
	if !w.IsWhitelisted(`C:\Users\test\AppData\keep`) {
		t.Error("exact match should be whitelisted")
	}
	// Case-insensitive.
	if !w.IsWhitelisted(`C:\USERS\TEST\APPDATA\KEEP`) {
		t.Error("case-insensitive match should be whitelisted")
	}
	// Prefix match — subdirectory of whitelisted dir.
	if !w.IsWhitelisted(`C:\Users\test\AppData\keep\SubDir\file.txt`) {
		t.Error("subdirectory of whitelisted path should be whitelisted")
	}
	// Non-matching.
	if w.IsWhitelisted(`C:\Users\test\AppData\other`) {
		t.Error("non-matching path should NOT be whitelisted")
	}
}

func TestWhitelist_IsWhitelistedGlob(t *testing.T) {
	w := &Whitelist{patterns: make([]string, 0)}
	if err := w.Add(`C:\Users\test\AppData\Local\*`); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if !w.IsWhitelisted(`C:\Users\test\AppData\Local\SomeApp`) {
		t.Error("glob * should match single-segment path")
	}
}

func TestWhitelist_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "whitelist.txt")

	// Create, add a pattern, save.
	w := &Whitelist{path: fpath, patterns: make([]string, 0)}
	pattern := `C:\Users\test\AppData\Local\keep`
	if err := w.Add(pattern); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := w.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(fpath); err != nil {
		t.Fatalf("whitelist file not created: %v", err)
	}

	// Load from the same file.
	loaded, err := Load(fpath)
	if err != nil {
		t.Fatalf("Load(%q) failed: %v", fpath, err)
	}

	found := false
	for _, p := range loaded.List() {
		if strings.EqualFold(p, pattern) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("loaded whitelist should contain %q, got: %v", pattern, loaded.List())
	}
}

func TestWhitelist_LoadCreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "new_whitelist.txt")

	// File does not exist — Load should create it with defaults.
	w, err := Load(fpath)
	if err != nil {
		t.Fatalf("Load non-existent file failed: %v", err)
	}
	if len(w.List()) == 0 {
		t.Error("default whitelist should have patterns")
	}
	// File should now exist.
	if _, statErr := os.Stat(fpath); statErr != nil {
		t.Errorf("Load should have created the file: %v", statErr)
	}
}

func TestValidatePattern_RejectsDangerous(t *testing.T) {
	// validatePattern is unexported — test indirectly through Add().
	tests := []struct {
		pattern string
		desc    string
	}{
		{"*", "wildcard-only"},
		{"**", "double-wildcard"},
		{`C:\`, "drive root backslash"},
		{`C:`, "drive root bare"},
		{`C:\*`, "drive root wildcard backslash"},
		{`D:\*`, "drive root wildcard D"},
		{`C:/*`, "drive root wildcard forward slash"},
		{`C:/`, "drive root forward slash"},
		{`C:\Users`, "too few separators"},
		{`D:\Games`, "too few separators D"},
		{"", "empty pattern"},
	}
	for _, tc := range tests {
		w := &Whitelist{patterns: make([]string, 0)}
		if err := w.Add(tc.pattern); err == nil {
			t.Errorf("Add(%q) [%s] should be rejected", tc.pattern, tc.desc)
		}
	}
}
