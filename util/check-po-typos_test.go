package util

import (
	"strings"
	"testing"
)

func TestFindMismatchedVariables_IgnoresShortConfigLike(t *testing.T) {
	// Spanish/Portuguese "p.e." (por ejemplo) looks like a git config variable
	// but has only 2 letters; must not be reported as a mismatch.
	mismatched := findMismatchedVariables("es",
		"for example use color.ui",
		"p.e. use color.ui")
	for _, m := range mismatched {
		if strings.HasPrefix(m, "p.e") || m == "p.e" {
			t.Errorf("short config-like %q should be ignored; got mismatched: %v", m, mismatched)
		}
	}
	if len(mismatched) != 0 {
		t.Errorf("expected no mismatches when only short abbr differs; got %v", mismatched)
	}

	// Real config variables with >= 6 letters are still checked.
	mismatched = findMismatchedVariables("zh_CN",
		"set color.ui to auto",
		"set color.UI to auto")
	if len(mismatched) == 0 {
		t.Fatal("expected mismatch for color.ui vs color.UI")
	}
	found := false
	for _, m := range mismatched {
		if strings.Contains(strings.ToLower(m), "color.ui") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected color.ui-related mismatch; got %v", mismatched)
	}
}

func TestCheckTyposInPoEntry_ShortConfigLikeNoReport(t *testing.T) {
	msgs, ok := checkTyposInPoEntry("es",
		"See the manual for details.",
		"Véase el manual, p.e. la sección de configuración.")
	if !ok {
		t.Fatalf("expected ok=true for short abbr only; msgs=%v", msgs)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no typo messages for p.e.; got %v", msgs)
	}
}
