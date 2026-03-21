package util

import (
	"strings"
	"testing"
)

func TestValidateLocale(t *testing.T) {
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
			gotName := FormatLocaleName(tt.locale)
			gotErrs := ValidateLocale(tt.locale)
			if gotName != tt.wantName {
				t.Errorf("FormatLocaleName(%q) = %q, want %q", tt.locale, gotName, tt.wantName)
			}
			errStrs := errSliceToStrings(gotErrs)
			for _, want := range tt.wantErrs {
				if !containsAny(errStrs, want) {
					t.Errorf("ValidateLocale(%q) errs = %v, want substring %q", tt.locale, errStrs, want)
				}
			}
			for _, noWant := range tt.wantNoErr {
				if containsAny(errStrs, noWant) {
					t.Errorf("ValidateLocale(%q) errs = %v, must not contain %q", tt.locale, errStrs, noWant)
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

func TestValidateLocale_ErrorCount(t *testing.T) {
	// Invalid language only: 1 error
	errs := ValidateLocale("xx")
	if len(errs) != 1 {
		t.Errorf("ValidateLocale(\"xx\") got %d errs, want 1: %v", len(errs), errs)
	}

	// Invalid language + invalid zone: at least 1 (language), zone may add another
	errs = ValidateLocale("xx_YY")
	if len(errs) < 1 {
		t.Errorf("ValidateLocale(\"xx_YY\") got %d errs, want >= 1", len(errs))
	}

	// Valid but wrong case: 1 error
	errs = ValidateLocale("zh_cn")
	if len(errs) != 1 {
		t.Errorf("ValidateLocale(\"zh_cn\") got %d errs, want 1: %v", len(errs), errs)
	}

	// Language case + zone case: 2 errors
	errs = ValidateLocale("ZH_cn")
	if len(errs) != 2 {
		t.Errorf("ValidateLocale(\"ZH_cn\") got %d errs, want 2: %v", len(errs), errs)
	}
}

func TestFormatLocaleName(t *testing.T) {
	if got := FormatLocaleName("zh_CN"); got != "Chinese - China" {
		t.Errorf("FormatLocaleName(\"zh_CN\") = %q, want \"Chinese - China\"", got)
	}
	if got := FormatLocaleName("zh"); got != "Chinese" {
		t.Errorf("FormatLocaleName(\"zh\") = %q, want \"Chinese\"", got)
	}
}
