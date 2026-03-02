package util

import (
	"testing"
)

func TestFilterGettextEntries_Default(t *testing.T) {
	entries := []GettextEntry{
		{MsgID: "a", MsgStr: "A", Obsolete: false},
		{MsgID: "b", MsgStr: "", Obsolete: false},
		{MsgID: "c", MsgStr: "c", Fuzzy: false, Obsolete: false},
		{MsgID: "d", MsgStr: "", Obsolete: true},
	}
	f := DefaultFilter()
	got := FilterGettextEntries(entries, f)
	if len(got) != 4 {
		t.Errorf("default filter: expected 4 entries, got %d", len(got))
	}
}

func TestFilterGettextEntries_NoObsolete(t *testing.T) {
	entries := []GettextEntry{
		{MsgID: "a", MsgStr: "A", Obsolete: false},
		{MsgID: "d", MsgStr: "", Obsolete: true},
	}
	f := EntryStateFilter{NoObsolete: true}
	got := FilterGettextEntries(entries, f)
	if len(got) != 1 {
		t.Errorf("no-obsolete: expected 1 entry, got %d", len(got))
	}
	if len(got) > 0 && got[0].MsgID != "a" {
		t.Errorf("expected entry a, got %s", got[0].MsgID)
	}
}

func TestFilterGettextEntries_OnlyTranslated(t *testing.T) {
	entries := []GettextEntry{
		{MsgID: "a", MsgStr: "A", Obsolete: false},
		{MsgID: "b", MsgStr: "", Obsolete: false},
		{MsgID: "c", MsgStr: "c", Fuzzy: false, Obsolete: false},
	}
	f := EntryStateFilter{Translated: true, WithObsolete: false}
	got := FilterGettextEntries(entries, f)
	// Translated: a (A), c (same). b is untranslated.
	if len(got) != 2 {
		t.Errorf("translated only: expected 2 entries, got %d", len(got))
	}
}

func TestFilterGettextEntries_OnlyObsolete(t *testing.T) {
	entries := []GettextEntry{
		{MsgID: "a", MsgStr: "A", Obsolete: false},
		{MsgID: "d", MsgStr: "x", Obsolete: true},
	}
	f := EntryStateFilter{OnlyObsolete: true}
	got := FilterGettextEntries(entries, f)
	if len(got) != 1 {
		t.Errorf("only-obsolete: expected 1 entry, got %d", len(got))
	}
	if len(got) > 0 && got[0].MsgID != "d" {
		t.Errorf("expected entry d (obsolete), got %s", got[0].MsgID)
	}
}

func TestFilterGettextEntries_OnlySame(t *testing.T) {
	entries := []GettextEntry{
		{MsgID: "a", MsgStr: "A", Obsolete: false},
		{MsgID: "b", MsgStr: "b", Obsolete: false},
	}
	f := EntryStateFilter{OnlySame: true}
	got := FilterGettextEntries(entries, f)
	if len(got) != 1 {
		t.Errorf("only-same: expected 1 entry, got %d", len(got))
	}
	if len(got) > 0 && got[0].MsgID != "b" {
		t.Errorf("expected entry b (same), got %s", got[0].MsgID)
	}
}
