package util

import (
	"strings"
	"testing"
)

func TestGetPrettyLocaleName(t *testing.T) {
	tests := []struct {
		name      string
		locale    string
		wantName  string
		wantErrs  []string // substrings to match in error messages
		wantNoErr []string // substrings that must NOT appear in any error
	}{
		{
			name:     "language only",
			locale:   "zh",
			wantName: "Chinese",
			wantErrs: nil,
		},
		{
			name:     "language with ISO 3166 zone, correct case",
			locale:   "zh_CN",
			wantName: "Chinese - China",
			wantErrs: nil,
		},
		{
			name:     "language with ISO 3166 zone, wrong case",
			locale:   "zh_cn",
			wantName: "Chinese - China",
			wantErrs: []string{"should be", "CN", "ISO 3166"},
		},
		{
			name:     "language with ISO 15924 script, correct case",
			locale:   "zh_Hans",
			wantName: "Chinese - Han (Simplified variant)",
			wantErrs: nil,
		},
		{
			name:     "language with ISO 15924 script, wrong case",
			locale:   "zh_hans",
			wantName: "Chinese - Han (Simplified variant)",
			wantErrs: []string{"should be", "Hans", "ISO 15924"},
		},
		{
			name:     "language with Latin script",
			locale:   "en_Latn",
			wantName: "English - Latin",
			wantErrs: nil,
		},
		{
			name:     "invalid language",
			locale:   "xx",
			wantName: "",
			wantErrs: []string{"invalid language code", "ISO 639"},
		},
		{
			name:     "invalid zone",
			locale:   "zh_XX",
			wantName: "Chinese", // language part still returned when zone invalid
			wantErrs: []string{"invalid region/territory or script", "ISO 3166", "ISO 15924"},
		},
		{
			name:     "language case error",
			locale:   "ZH_cn",
			wantName: "Chinese - China",
			wantErrs: []string{"language code", "lowercase"},
		},
		{
			name:     "invalid language and zone",
			locale:   "xx_YY",
			wantName: "",
			wantErrs: []string{"invalid language"},
		},
		{
			name:     "ISO 3166 alpha-3",
			locale:   "en_USA",
			wantName: "English - United States of America",
			wantErrs: nil,
		},
		{
			name:     "ISO 15924 numeric script",
			locale:   "zh_501",
			wantName: "Chinese - Han (Simplified variant)",
			wantErrs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotErrs := GetPrettyLocaleName(tt.locale)
			if gotName != tt.wantName {
				t.Errorf("GetPrettyLocaleName(%q) name = %q, want %q", tt.locale, gotName, tt.wantName)
			}
			errStrs := errSliceToStrings(gotErrs)
			for _, want := range tt.wantErrs {
				if !containsAny(errStrs, want) {
					t.Errorf("GetPrettyLocaleName(%q) errs = %v, want substring %q", tt.locale, errStrs, want)
				}
			}
			for _, noWant := range tt.wantNoErr {
				if containsAny(errStrs, noWant) {
					t.Errorf("GetPrettyLocaleName(%q) errs = %v, must not contain %q", tt.locale, errStrs, noWant)
				}
			}
		})
	}
}

func errSliceToStrings(errs []error) []string {
	s := make([]string, len(errs))
	for i, e := range errs {
		s[i] = e.Error()
	}
	return s
}

func containsAny(strs []string, sub string) bool {
	for _, s := range strs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TestGetPrettyLocaleName_ErrorCount(t *testing.T) {
	// Invalid language only: 1 error
	_, errs := GetPrettyLocaleName("xx")
	if len(errs) != 1 {
		t.Errorf("GetPrettyLocaleName(\"xx\") got %d errs, want 1: %v", len(errs), errs)
	}

	// Invalid language + invalid zone: at least 1 (language), zone may add another
	_, errs = GetPrettyLocaleName("xx_YY")
	if len(errs) < 1 {
		t.Errorf("GetPrettyLocaleName(\"xx_YY\") got %d errs, want >= 1", len(errs))
	}

	// Valid but wrong case: 1 error
	_, errs = GetPrettyLocaleName("zh_cn")
	if len(errs) != 1 {
		t.Errorf("GetPrettyLocaleName(\"zh_cn\") got %d errs, want 1: %v", len(errs), errs)
	}

	// Language case + zone case: 2 errors
	_, errs = GetPrettyLocaleName("ZH_cn")
	if len(errs) != 2 {
		t.Errorf("GetPrettyLocaleName(\"ZH_cn\") got %d errs, want 2: %v", len(errs), errs)
	}
}
