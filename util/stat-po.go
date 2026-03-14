// Package util provides PO file report statistics.
package util

import (
	"fmt"
	"os"
	"strings"
)

// PoStats holds statistics for a PO file.
type PoStats struct {
	Translated   int // Entries with non-empty translation, not fuzzy, not same as msgid
	Untranslated int // Entries with empty msgstr
	Same         int // Entries where msgstr equals msgid (suspect untranslated)
	Fuzzy        int // Entries with fuzzy flag
	Obsolete     int // Obsolete entries (#~ format)
}

// Total returns the sum of Translated, Untranslated, and Fuzzy (excludes Obsolete and Same).
func (s *PoStats) Total() int {
	return s.Translated + s.Untranslated + s.Fuzzy
}

// getPoStatsFromGettextJSON computes PoReportStats from GettextJSON entries.
func getPoStatsFromGettextJSON(j *GettextJSON) *PoStats {
	stats := &PoStats{}
	if j == nil {
		return stats
	}
	for _, e := range j.Entries {
		if e.MsgID == "" {
			continue
		}
		if e.Obsolete {
			stats.Obsolete++
			continue
		}
		hasTranslation := false
		msgstrValue := ""
		for _, s := range e.MsgStr {
			if s != "" {
				hasTranslation = true
				break
			}
		}
		if len(e.MsgStr) > 0 {
			msgstrValue = e.MsgStr[0]
		}
		if e.Fuzzy {
			stats.Fuzzy++
			continue
		}
		if !hasTranslation {
			stats.Untranslated++
			continue
		}
		if msgstrValue == e.MsgID {
			stats.Same++
			stats.Translated++
			continue
		}
		stats.Translated++
	}
	return stats
}

// GetPoStats reads a PO or gettext JSON file and returns entry statistics.
// Uses LoadFileToGettextJSON for unified loading (same interface as msg-select, msg-cat, compare).
func GetPoStats(file string) (*PoStats, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", file, err)
	}
	j, err := LoadFileToGettextJSON(data, file)
	if err != nil {
		return nil, err
	}
	return getPoStatsFromGettextJSON(j), nil
}

// FormatMsgfmtStatistics formats stats to match msgfmt --statistics output.
// For compatibility, "same" (msgstr == msgid) is counted as translated.
func FormatMsgfmtStatistics(stats *PoStats) string {
	translated := stats.Translated + stats.Same
	var parts []string
	if translated > 0 {
		if translated == 1 {
			parts = append(parts, "1 translated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d translated messages", translated))
		}
	}
	if stats.Fuzzy > 0 {
		if stats.Fuzzy == 1 {
			parts = append(parts, "1 fuzzy translation")
		} else {
			parts = append(parts, fmt.Sprintf("%d fuzzy translations", stats.Fuzzy))
		}
	}
	if stats.Untranslated > 0 {
		if stats.Untranslated == 1 {
			parts = append(parts, "1 untranslated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d untranslated messages", stats.Untranslated))
		}
	}
	if len(parts) == 0 {
		return "0 translated messages.\n"
	}
	return strings.Join(parts, ", ") + ".\n"
}

// FormatStatLine formats stats in one line, similar to msgfmt --statistics,
// but also includes same and obsolete. Only non-zero categories are shown.
func FormatStatLine(stats *PoStats) string {
	var parts []string
	if stats.Translated > 0 {
		if stats.Translated == 1 {
			parts = append(parts, "1 translated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d translated messages", stats.Translated))
		}
	}
	if stats.Fuzzy > 0 {
		if stats.Fuzzy == 1 {
			parts = append(parts, "1 fuzzy translation")
		} else {
			parts = append(parts, fmt.Sprintf("%d fuzzy translations", stats.Fuzzy))
		}
	}
	if stats.Untranslated > 0 {
		if stats.Untranslated == 1 {
			parts = append(parts, "1 untranslated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d untranslated messages", stats.Untranslated))
		}
	}
	if stats.Same > 0 {
		if stats.Same == 1 {
			parts = append(parts, "1 same message")
		} else {
			parts = append(parts, fmt.Sprintf("%d same messages", stats.Same))
		}
	}
	if stats.Obsolete > 0 {
		if stats.Obsolete == 1 {
			parts = append(parts, "1 obsolete entry")
		} else {
			parts = append(parts, fmt.Sprintf("%d obsolete entries", stats.Obsolete))
		}
	}
	if len(parts) == 0 {
		return "0 translated messages.\n"
	}
	return strings.Join(parts, ", ") + ".\n"
}
