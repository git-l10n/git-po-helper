package util

import (
	"reflect"
	"strings"
	"testing"
)

func TestCheckTyposInPoEntry_InitialLetterCase(t *testing.T) {
	setTestEnv(t, "GITHUB_ACTIONS", "")
	setTestEnv(t, "GITHUB_EVENT_NAME", "")
	for _, test := range []struct {
		name, msgID, msgStr string
		mismatch            bool
	}{
		{"additional attribute", "unable to add additional attribute", "Kunde inte lägga till ytterligare attribut", true},
		{"todo list", "could not generate todo list", "Kunde inte skapa attgöra-lista", true},
		{"commit graph", "unable to open commit-graph chain file", "Kunde inte öppna incheckningsgrafkedjefilen", true},
		{"uppercase to lowercase", "Unable to open", "kunde inte öppna", true},
		{"both uppercase", "Unable to open", "Kunde inte öppna", false},
		{"both lowercase", "unable to open", "kunde inte öppna", false},
		{"Chinese", "unable to open", "无法打开", false},
		{"accented", "unable to open", "Öppning misslyckades", false},
		{"empty source", "", "Kunde inte öppna", false},
		{"untranslated", "unable to open", "", false},
		{"leading space", " unable to open", "Kunde inte öppna", false},
		{"leading punctuation", "unable to open", "(Kunde inte öppna)", false},
		{"leading digit", "1 file", "En fil", false},
		{"leading placeholder", "%s has failed", "Kunde inte öppna %s", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			msgs, ok := checkTyposInPoEntry("sv", test.msgID, test.msgStr)
			var want []string
			if test.mismatch {
				want = []string{"initial letter case mismatch", ">> msgid: " + test.msgID, ">> msgstr: " + test.msgStr, ""}
			}
			if !reflect.DeepEqual(msgs, want) || ok == test.mismatch {
				t.Errorf("got (%q, %t), want (%q, %t)", msgs, ok, want, !test.mismatch)
			}
		})
	}
}

func TestCheckTyposInPoEntry_CombinedDiagnostics(t *testing.T) {
	msgs, _ := checkTyposInPoEntry("sv", "use --force", "Använd --forced")
	want := []string{
		"initial letter case mismatch",
		"mismatched patterns: --force, --forced",
		">> msgid: use --force",
		">> msgstr: Använd --forced",
		"",
	}
	if !reflect.DeepEqual(msgs, want) {
		t.Errorf("got %q, want %q", msgs, want)
	}
}

func TestCheckTyposInPo_PluralInitialLetterCase(t *testing.T) {
	po := &GettextPO{Entries: []GettextEntry{{
		MsgID:       "One file",
		MsgIDPlural: "several files",
		MsgStr:      []string{"en fil", "Flera filer", "många filer"},
	}}}
	msgs, _ := checkTyposInPo("sv", po)
	want := []string{
		"initial letter case mismatch", ">> msgid: One file", ">> msgstr: en fil", "",
		"initial letter case mismatch", ">> msgid: several files", ">> msgstr: Flera filer", "",
	}
	if !reflect.DeepEqual(msgs, want) {
		t.Errorf("got %q, want %q", msgs, want)
	}
}

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
