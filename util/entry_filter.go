// Package util provides entry state filtering for msg-select and msg-cat.
package util

// EntryStateFilter specifies which entry states to include.
// Used by msg-select and msg-cat to filter entries by translation state.
type EntryStateFilter struct {
	// Translated: msgstr not empty, not fuzzy
	Translated bool
	// Untranslated: msgstr empty
	Untranslated bool
	// Fuzzy: marked fuzzy in comments
	Fuzzy bool
	// WithObsolete: include obsolete entries (default true when no filter flags)
	WithObsolete bool
	// NoObsolete: exclude obsolete entries (overrides WithObsolete)
	NoObsolete bool
	// OnlySame: only entries where msgstr == msgid (mutually exclusive with others)
	OnlySame bool
	// OnlyObsolete: only obsolete entries (mutually exclusive with others)
	OnlyObsolete bool
}

// DefaultFilter returns the default filter: all states including obsolete.
func DefaultFilter() EntryStateFilter {
	return EntryStateFilter{WithObsolete: true}
}

// HasStateFilter returns true if any of --translated, --untranslated, --fuzzy was set.
func (f EntryStateFilter) HasStateFilter() bool {
	return f.Translated || f.Untranslated || f.Fuzzy
}

// IncludeObsolete returns true if obsolete entries should be included.
func (f EntryStateFilter) IncludeObsolete() bool {
	if f.NoObsolete {
		return false
	}
	return f.WithObsolete
}

// FilterGettextEntries filters GettextEntry slice by state.
func FilterGettextEntries(entries []GettextEntry, filter EntryStateFilter) []GettextEntry {
	var result []GettextEntry
	for _, e := range entries {
		if MatchGettextEntryState(e, filter) {
			result = append(result, e)
		}
	}
	return result
}

// MatchGettextEntryState returns true if the entry matches the filter.
func MatchGettextEntryState(e GettextEntry, filter EntryStateFilter) bool {
	if filter.OnlySame {
		return isSameGettextEntry(e) && !e.Obsolete
	}
	if filter.OnlyObsolete {
		return e.Obsolete
	}

	if e.Obsolete {
		return filter.IncludeObsolete()
	}

	if filter.HasStateFilter() {
		matched := false
		if filter.Translated && isTranslatedGettextEntry(e) && !e.Fuzzy {
			matched = true
		}
		if filter.Untranslated && isUntranslatedGettextEntry(e) {
			matched = true
		}
		if filter.Fuzzy && e.Fuzzy {
			matched = true
		}
		return matched
	}

	return true
}

func isTranslatedGettextEntry(e GettextEntry) bool {
	if len(e.MsgStrPlural) > 0 {
		for _, s := range e.MsgStrPlural {
			if s != "" {
				return true
			}
		}
		return false
	}
	return e.MsgStr != ""
}

func isUntranslatedGettextEntry(e GettextEntry) bool {
	if len(e.MsgStrPlural) > 0 {
		for _, s := range e.MsgStrPlural {
			if s != "" {
				return false
			}
		}
		return true
	}
	return e.MsgStr == ""
}

func isSameGettextEntry(e GettextEntry) bool {
	if len(e.MsgStrPlural) > 0 {
		return e.MsgStrPlural[0] == e.MsgID
	}
	return e.MsgStr == e.MsgID
}
