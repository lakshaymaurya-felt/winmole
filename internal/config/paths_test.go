package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetNeverDeletePaths_ContainsCriticalPaths(t *testing.T) {
	paths := GetNeverDeletePaths()

	required := []string{
		`C:\Windows`,
		`C:\Windows\System32`,
		`C:\Windows\SysWOW64`,
		`C:\Users`,
		`C:\ProgramData`,
		`C:\Recovery`,
		`C:\Program Files`,
		`C:\Program Files (x86)`,
		`C:\Boot`,
		`C:\EFI`,
	}

	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[strings.ToLower(filepath.Clean(p))] = true
	}

	for _, req := range required {
		key := strings.ToLower(filepath.Clean(req))
		if !pathSet[key] {
			t.Errorf("GetNeverDeletePaths() MUST contain %q", req)
		}
	}
}

func TestGetNeverDeletePaths_NotEmpty(t *testing.T) {
	paths := GetNeverDeletePaths()
	if len(paths) == 0 {
		t.Fatal("GetNeverDeletePaths() must not return an empty list")
	}
}

func TestGetCleanTargets_AllHaveRequiredFields(t *testing.T) {
	for _, target := range GetCleanTargets() {
		if target.Name == "" {
			t.Error("CleanTarget has empty Name")
		}
		if target.Category == "" {
			t.Errorf("CleanTarget %q has empty Category", target.Name)
		}
		if target.RiskLevel == "" {
			t.Errorf("CleanTarget %q has empty RiskLevel", target.Name)
		}
		validRisk := target.RiskLevel == "low" || target.RiskLevel == "medium" || target.RiskLevel == "high"
		if !validRisk {
			t.Errorf("CleanTarget %q has invalid RiskLevel %q", target.Name, target.RiskLevel)
		}
		// RecycleBin uses shell API with no direct paths — that's expected.
		if target.Name != "RecycleBin" && len(target.Paths) == 0 {
			t.Errorf("CleanTarget %q has no Paths (and is not RecycleBin)", target.Name)
		}
	}
}

func TestGetCleanTargets_NoPaths_OverlapWithNeverDelete(t *testing.T) {
	neverDelete := GetNeverDeletePaths()

	for _, target := range GetCleanTargets() {
		for _, tp := range target.Paths {
			if tp == "" {
				continue
			}
			cleanTP := strings.ToLower(filepath.Clean(tp))

			for _, nd := range neverDelete {
				cleanND := strings.ToLower(filepath.Clean(nd))

				// Clean target must NOT equal a NEVER_DELETE path.
				if cleanTP == cleanND {
					t.Errorf("target %q path %q equals NEVER_DELETE path %q",
						target.Name, tp, nd)
				}

				// Clean target must NOT be a parent of a NEVER_DELETE path.
				// (Deleting a parent would destroy the protected child.)
				prefix := cleanTP + string(filepath.Separator)
				if strings.HasPrefix(cleanND+string(filepath.Separator), prefix) && cleanTP != cleanND {
					t.Errorf("target %q path %q is a parent of NEVER_DELETE path %q",
						target.Name, tp, nd)
				}
			}
		}
	}
}

func TestGetCleanTargets_UniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, target := range GetCleanTargets() {
		if seen[target.Name] {
			t.Errorf("duplicate CleanTarget name: %q", target.Name)
		}
		seen[target.Name] = true
	}
}

func TestGetTargetsByCategory(t *testing.T) {
	categories := []string{"user", "system", "browser", "dev"}
	for _, cat := range categories {
		targets := GetTargetsByCategory(cat)
		if len(targets) == 0 {
			t.Errorf("GetTargetsByCategory(%q) returned no targets", cat)
		}
		for _, tgt := range targets {
			if tgt.Category != cat {
				t.Errorf("target %q has category %q, expected %q", tgt.Name, tgt.Category, cat)
			}
		}
	}
}
