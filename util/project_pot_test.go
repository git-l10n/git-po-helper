package util

import (
	"testing"

	"github.com/git-l10n/git-po-helper/config"
)

func TestParseDefaultPotAction(t *testing.T) {
	tests := []struct {
		s    string
		want DefaultPotAction
	}{
		{"auto", DefaultPotActionAuto},
		{"no", DefaultPotActionNo},
		{"false", DefaultPotActionNo},
		{"build", DefaultPotActionBuild},
		{"download", DefaultPotActionDownload},
		{"use_if_exist", DefaultPotActionUseIfExist},
		{"", DefaultPotActionNo},
		{"unknown", DefaultPotActionNo},
	}
	for _, tt := range tests {
		got := ParseDefaultPotAction(tt.s)
		if got != tt.want {
			t.Errorf("ParseDefaultPotAction(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestMergeProjectPotOverlays_OverrideBuiltin(t *testing.T) {
	base := []*ProjectPotConfig{
		{
			ProjectName:       "Git",
			MinGettextVersion: "0.14",
			DefaultAction:     DefaultPotActionAuto,
		},
	}
	overlays := []map[string]config.PotProjectEntry{
		{
			"Git": config.PotProjectEntry{MinGettextVersion: "0.16"},
		},
	}
	merged := mergeProjectPotOverlays(base, overlays)
	if len(merged) != 1 {
		t.Fatalf("expected 1 project, got %d", len(merged))
	}
	if merged[0].ProjectName != "Git" {
		t.Fatalf("ProjectName: got %q", merged[0].ProjectName)
	}
	if merged[0].MinGettextVersion != "0.16" {
		t.Fatalf("expected overlay min_gettext_version 0.16, got %q", merged[0].MinGettextVersion)
	}
	if merged[0].DefaultAction != DefaultPotActionAuto {
		t.Fatalf("expected unchanged DefaultAction auto, got %v", merged[0].DefaultAction)
	}
}

func TestMergeProjectPotOverlays_CaseInsensitiveOverride(t *testing.T) {
	base := []*ProjectPotConfig{
		{ProjectName: "Git", MinGettextVersion: "0.14"},
	}
	overlays := []map[string]config.PotProjectEntry{
		{"git": config.PotProjectEntry{MinGettextVersion: "0.15"}},
	}
	merged := mergeProjectPotOverlays(base, overlays)
	if len(merged) != 1 {
		t.Fatalf("expected 1 project, got %d", len(merged))
	}
	if merged[0].MinGettextVersion != "0.15" {
		t.Fatalf("expected git overlay to match Git, got min_gettext_version %q", merged[0].MinGettextVersion)
	}
}

func TestMergeProjectPotOverlays_NewProject(t *testing.T) {
	base := []*ProjectPotConfig{
		{ProjectName: "Git"},
	}
	overlays := []map[string]config.PotProjectEntry{
		{
			"MyProject": config.PotProjectEntry{
				DownloadURL: "https://example.com/po/my.pot",
				BuildCmd:    []string{"make", "pot"},
			},
		},
	}
	merged := mergeProjectPotOverlays(base, overlays)
	if len(merged) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(merged))
	}
	var my *ProjectPotConfig
	for _, c := range merged {
		if c.ProjectName == "MyProject" {
			my = c
			break
		}
	}
	if my == nil {
		t.Fatal("expected MyProject in merged list")
	}
	if my.DownloadURL != "https://example.com/po/my.pot" {
		t.Fatalf("MyProject DownloadURL: got %q", my.DownloadURL)
	}
	if my.DefaultAction != DefaultPotActionNo {
		t.Fatalf("new project without default_action should get DefaultPotActionNo, got %v", my.DefaultAction)
	}
}

func TestMergeProjectPotOverlays_BaseNotMutated(t *testing.T) {
	base := []*ProjectPotConfig{
		{ProjectName: "Git", MinGettextVersion: "0.14"},
	}
	overlays := []map[string]config.PotProjectEntry{
		{"Git": config.PotProjectEntry{MinGettextVersion: "0.16"}},
	}
	_ = mergeProjectPotOverlays(base, overlays)
	if base[0].MinGettextVersion != "0.14" {
		t.Fatalf("base should not be mutated, got %q", base[0].MinGettextVersion)
	}
}
