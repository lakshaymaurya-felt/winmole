package envutil

import (
	"testing"
)

func TestExpandWindowsEnv_PercentVars(t *testing.T) {
	t.Setenv("TEST_WM_VAR", "hello")
	result := ExpandWindowsEnv(`%TEST_WM_VAR%\path`)
	expected := `hello\path`
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestExpandWindowsEnv_DollarVars(t *testing.T) {
	t.Setenv("TEST_WM_DOLLAR", "world")

	// $VAR syntax.
	r1 := ExpandWindowsEnv("$TEST_WM_DOLLAR")
	if r1 != "world" {
		t.Errorf("$VAR: got %q, want %q", r1, "world")
	}

	// ${VAR} syntax.
	r2 := ExpandWindowsEnv("${TEST_WM_DOLLAR}")
	if r2 != "world" {
		t.Errorf("${VAR}: got %q, want %q", r2, "world")
	}

	// $VAR with trailing path.
	r3 := ExpandWindowsEnv(`$TEST_WM_DOLLAR\path`)
	if r3 != `world\path` {
		t.Errorf("$VAR\\path: got %q, want %q", r3, `world\path`)
	}
}

func TestExpandWindowsEnv_MixedVars(t *testing.T) {
	t.Setenv("WM_MIX1", "alpha")
	t.Setenv("WM_MIX2", "beta")

	// %VAR1%\$VAR2 should expand both.
	result := ExpandWindowsEnv(`%WM_MIX1%\$WM_MIX2`)
	expected := `alpha\beta`
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestExpandWindowsEnv_UnsetVar(t *testing.T) {
	// Use a name that is guaranteed not to be set.
	result := ExpandWindowsEnv("%WINMOLE_TRULY_NONEXISTENT_VAR_XYZ123%")
	if result != "" {
		t.Errorf("unset %%VAR%% should expand to empty string, got %q", result)
	}
}

func TestExpandWindowsEnv_EmptyPercent(t *testing.T) {
	// %% should collapse to a single %.
	result := ExpandWindowsEnv("%%")
	if result != "%" {
		t.Errorf("%%%% should collapse to single %%, got %q", result)
	}
}

func TestExpandWindowsEnv_NoVars(t *testing.T) {
	// Plain string without any variables should pass through unchanged.
	input := `C:\plain\path\no\vars`
	result := ExpandWindowsEnv(input)
	if result != input {
		t.Errorf("plain string should be unchanged, got %q", result)
	}
}

func TestExpandWindowsEnv_EmptyString(t *testing.T) {
	result := ExpandWindowsEnv("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}
