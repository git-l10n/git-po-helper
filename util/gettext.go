// Package util provides PO file parsing and gettext-related utilities.
package util

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// GettextEntry represents a single PO/JSON entry. Used for parsing, comparison, and output.
// MsgID, MsgStr (forms), MsgIDPlural use PO string format (escape sequences like \n, \t
// stored as literal backslash+char, not decoded). MsgStr holds one element for singular
// msgstr, multiple for msgstr[0], msgstr[1], ... RawLines preserves exact PO format for round-trip.
type GettextEntry struct {
	MsgID         string   `json:"msgid"`
	MsgStr        []string `json:"msgstr,omitempty"` // Always a JSON array; one element = singular, multiple = msgstr[0..]
	MsgIDPlural   string   `json:"msgid_plural,omitempty"`
	Comments      []string `json:"comments,omitempty"`
	Fuzzy         bool     `json:"fuzzy"`
	Obsolete      bool     `json:"obsolete,omitempty"`       // True for #~ obsolete entries
	MsgIDPrevious string   `json:"msgid_previous,omitempty"` // For #~| format (gettext 0.19.8+)
	RawLines      []string `json:"-"`                        // Original PO lines for round-trip; empty when built from JSON
}

// MsgStrSingle returns the first translation form, or "" if none (singular msgstr or msgstr[0]).
func (e *GettextEntry) MsgStrSingle() string {
	if e == nil || len(e.MsgStr) == 0 {
		return ""
	}
	return e.MsgStr[0]
}

// UnmarshalJSON accepts msgstr as either a JSON string (singular) or a JSON array
// of strings (singular or plural forms), normalizing to MsgStr []string.
func (e *GettextEntry) UnmarshalJSON(data []byte) error {
	var aux struct {
		MsgID         string          `json:"msgid"`
		MsgStrRaw     json.RawMessage `json:"msgstr"`
		MsgIDPlural   string          `json:"msgid_plural,omitempty"`
		Comments      []string        `json:"comments,omitempty"`
		Fuzzy         bool            `json:"fuzzy"`
		Obsolete      bool            `json:"obsolete,omitempty"`
		MsgIDPrevious string          `json:"msgid_previous,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.MsgID = aux.MsgID
	e.MsgIDPlural = aux.MsgIDPlural
	e.Comments = aux.Comments
	e.Fuzzy = aux.Fuzzy
	e.Obsolete = aux.Obsolete
	e.MsgIDPrevious = aux.MsgIDPrevious
	e.MsgStr = nil
	if len(aux.MsgStrRaw) == 0 || string(aux.MsgStrRaw) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(aux.MsgStrRaw, &s); err == nil {
		e.MsgStr = []string{s}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(aux.MsgStrRaw, &arr); err == nil {
		e.MsgStr = arr
		return nil
	}
	return fmt.Errorf("gettext entry msgstr: want string or array of strings")
}

// commentHasFuzzyFlag returns true if the line is a flag comment (e.g. "#, fuzzy" or "#, fuzzy, c-format")
// that includes the "fuzzy" flag.
func commentHasFuzzyFlag(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#,") {
		return false
	}
	flags := strings.TrimPrefix(trimmed, "#,")
	for _, f := range strings.Split(flags, ",") {
		if strings.TrimSpace(f) == "fuzzy" {
			return true
		}
	}
	return false
}

// StripFuzzyFromCommentLine removes the "fuzzy" flag from a "#," comment line.
// If the line is "#, fuzzy" only, returns "". If the line is "#, fuzzy, c-format" or similar,
// returns "#, c-format" (other flags preserved). Non-flag lines are returned unchanged.
func StripFuzzyFromCommentLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#,") {
		return line
	}
	flagsStr := strings.TrimPrefix(trimmed, "#,")
	var rest []string
	for _, f := range strings.Split(flagsStr, ",") {
		if strings.TrimSpace(f) != "fuzzy" {
			rest = append(rest, strings.TrimSpace(f))
		}
	}
	if len(rest) == 0 {
		return ""
	}
	return "#, " + strings.Join(rest, ", ")
}

// StripFuzzyFromFlagLine removes "fuzzy" from a "#," flag line.
// Returns the line with fuzzy stripped, or empty string if no flags remain.
func StripFuzzyFromFlagLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#,") {
		return line
	}
	flagsStr := strings.TrimPrefix(trimmed, "#,")
	var flags []string
	for _, f := range strings.Split(flagsStr, ",") {
		s := strings.TrimSpace(f)
		if s != "" && s != "fuzzy" {
			flags = append(flags, s)
		}
	}
	if len(flags) == 0 {
		return ""
	}
	return "#, " + strings.Join(flags, ", ")
}

// MergeFuzzyIntoFlagLine returns a "#," flag line with "fuzzy" prepended to existing flags.
// If addFuzzy is false, returns line unchanged. If addFuzzy is true, any existing "fuzzy"
// in the line is not duplicated (input may be "#, c-format" or legacy "#, fuzzy").
func MergeFuzzyIntoFlagLine(line string, addFuzzy bool) string {
	if !addFuzzy {
		return line
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#,") {
		return line
	}
	flagsStr := strings.TrimPrefix(trimmed, "#,")
	var flags []string
	for _, f := range strings.Split(flagsStr, ",") {
		s := strings.TrimSpace(f)
		if s != "" && s != "fuzzy" {
			flags = append(flags, s)
		}
	}
	out := "#, fuzzy"
	if len(flags) > 0 {
		out += ", " + strings.Join(flags, ", ")
	}
	return out
}

// poParsedToPoFormat converts PO-parsed string to GettextEntry storage format.
// PO file uses escape sequences (\\, \n, \t, etc.); we unescape then convert
// newline/tab/cr to backslash+n/t/r for consistent PO format storage.
func poParsedToPoFormat(s string) string {
	return jsonDecodedToPoFormat(poUnescape(s))
}

// jsonDecodedToPoFormat converts JSON-decoded string to PO format for GettextEntry storage.
// Matches RFC 8259 / Python json.loads: \n→newline, \t→tab, \r→cr, \"→quote, \\→backslash,
// \uXXXX→codepoint. We store as PO escape sequences: newline→\n, tab→\t, cr→\r, quote→\",
// backslash→\\. When JSON has \\n (literal backslash+n), we get \ and n; output as-is.
// Standalone backslash (not part of \n,\t,\r,\",\\) → output \\.
func jsonDecodedToPoFormat(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			if i+1 < len(s) {
				switch s[i+1] {
				case 'n', 't', 'r', '"', '\\':
					b.WriteByte('\\')
					b.WriteByte(s[i+1])
					i++
					continue
				}
			}
			b.WriteString(`\\`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// poUnescape decodes PO escape sequences in s into real characters.
// PO uses \n (newline), \t (tab), \r (carriage return), \" (quote), \\ (backslash).
func poUnescape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
			case 't':
				b.WriteByte('\t')
				i++
			case 'r':
				b.WriteByte('\r')
				i++
			case '"':
				b.WriteByte('"')
				i++
			case '\\':
				b.WriteByte('\\')
				i++
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// strDeQuote removes one quote character from each end of s if both ends have a quote.
// Returns s unchanged otherwise.
func strDeQuote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// poLineKind is the syntactic kind of a PO line (after trimming).
// Used to dispatch parsing and to add new formats (e.g. msgctxt, #=) in one place.
type poLineKind int

const (
	poLineBlank poLineKind = iota
	poLineComment
	poLineFlagHashComma    // #,
	poLineFlagHashEq       // #= (sticky flags)
	poLineCommentRef       // #: reference (file:line)
	poLineCommentExtracted // #. extracted
	poLineCommentPrev      // #| previous untranslated
	poLineObsoletePrefix   // #~
	poLineObsoletePrev     // #~|
	poLineMsgid
	poLineMsgidPlural
	poLineMsgstr
	poLineMsgstrN
	poLineQuotedString
	poLineUnknown
)

// classifyPoLine returns the kind of line for trimmed (TrimSpace of original).
// Caller handles #~ / #~| and passes the rest as trimmed when appropriate.
func classifyPoLine(trimmed string) poLineKind {
	if trimmed == "" {
		return poLineBlank
	}
	if strings.HasPrefix(trimmed, "#~ ") {
		return poLineObsoletePrefix
	}
	if strings.HasPrefix(trimmed, "#~| ") {
		return poLineObsoletePrev
	}
	if strings.HasPrefix(trimmed, "msgid ") {
		return poLineMsgid
	}
	if strings.HasPrefix(trimmed, "msgid_plural ") {
		return poLineMsgidPlural
	}
	if strings.HasPrefix(trimmed, "msgstr[") {
		return poLineMsgstrN
	}
	if strings.HasPrefix(trimmed, "msgstr ") {
		return poLineMsgstr
	}
	if strings.HasPrefix(trimmed, `"`) {
		return poLineQuotedString
	}
	if strings.HasPrefix(trimmed, "#,") {
		return poLineFlagHashComma
	}
	if strings.HasPrefix(trimmed, "#=") {
		return poLineFlagHashEq
	}
	if strings.HasPrefix(trimmed, "#:") {
		return poLineCommentRef
	}
	if strings.HasPrefix(trimmed, "#.") {
		return poLineCommentExtracted
	}
	if strings.HasPrefix(trimmed, "#|") {
		return poLineCommentPrev
	}
	if strings.HasPrefix(trimmed, "#") {
		return poLineComment
	}
	return poLineUnknown
}

// poParseState holds mutable state during ParsePoEntries.
// Centralizing state makes finishCurrentEntry and startNewEntry consistent.
type poParseState struct {
	inHeader           bool
	hasSeenHeaderBlock bool
	headerLines        []string

	currentEntry       *GettextEntry
	entryLines         []string
	msgidValue         strings.Builder
	msgstrValue        strings.Builder
	msgidPluralValue   strings.Builder
	msgstrPluralValues []strings.Builder
	inMsgid            bool
	inMsgstr           bool
	inMsgidPlural      bool
	currentPluralIndex int
	inObsolete         bool
}

// finishCurrentEntry writes the current entry's collected msgid/msgstr into
// currentEntry, sets RawLines/Fuzzy/Obsolete, and appends to entries if the
// entry has content. Call before starting a new entry or on blank line.
func finishCurrentEntry(st *poParseState, entries *[]*GettextEntry) {
	if st.currentEntry == nil {
		return
	}
	if st.msgidValue.Len() == 0 && st.msgstrValue.Len() == 0 {
		return
	}
	st.currentEntry.MsgID = poParsedToPoFormat(st.msgidValue.String())
	if st.msgidPluralValue.Len() > 0 {
		st.currentEntry.MsgIDPlural = poParsedToPoFormat(st.msgidPluralValue.String())
		st.currentEntry.MsgStr = make([]string, len(st.msgstrPluralValues))
		for i, b := range st.msgstrPluralValues {
			st.currentEntry.MsgStr[i] = poParsedToPoFormat(b.String())
		}
	} else {
		st.currentEntry.MsgStr = []string{poParsedToPoFormat(st.msgstrValue.String())}
	}
	st.currentEntry.RawLines = st.entryLines
	st.currentEntry.Fuzzy = entryHasFuzzyFlag(st.currentEntry.Comments)
	st.currentEntry.Obsolete = st.inObsolete
	*entries = append(*entries, st.currentEntry)
}

// resetEntryContent resets only the string builders and inMsgid/inMsgstr flags.
// Use when keeping currentEntry and entryLines (e.g. entry had only comments).
func resetEntryContent(st *poParseState) {
	st.msgidValue.Reset()
	st.msgstrValue.Reset()
	st.msgidPluralValue.Reset()
	st.msgstrPluralValues = nil
	st.inMsgid = true
	st.inMsgstr = false
	st.inMsgidPlural = false
	st.currentPluralIndex = -1
}

// startNewEntry resets entry-related state for a new entry. If the current
// entry had content it was already appended by finishCurrentEntry. Reuses or
// allocates currentEntry and resets string builders and flags.
func startNewEntry(st *poParseState) {
	st.currentEntry = &GettextEntry{}
	st.entryLines = nil
	resetEntryContent(st)
}

// ParsePoEntries parses PO file entries and returns entries and header.
// The header includes comments, the empty msgid/msgstr block, and any continuation lines.
// Entries are 1-based for content (header entry with empty msgid is not included).
func ParsePoEntries(data []byte) (entries []*GettextEntry, header []string, err error) {
	lines := strings.Split(string(data), "\n")
	st := &poParseState{
		inHeader:           true,
		hasSeenHeaderBlock: false,
		headerLines:        nil,
		currentPluralIndex: -1,
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Obsolete entry format: #~ msgid, #~ msgstr, #~| msgid (check first, before header/comment)
		if strings.HasPrefix(trimmed, "#~ ") {
			rest := trimmed[3:]
			restTrimmed := strings.TrimSpace(rest)
			if strings.HasPrefix(restTrimmed, `"`) || strings.HasPrefix(restTrimmed, "msgstr") {
				st.inObsolete = true
			}
			if strings.HasPrefix(restTrimmed, `"`) && (st.inMsgid || st.inMsgstr || st.inMsgidPlural) {
				value := strDeQuote(restTrimmed)
				if st.inMsgid {
					st.msgidValue.WriteString(value)
				} else if st.inMsgidPlural {
					st.msgidPluralValue.WriteString(value)
				} else if st.inMsgstr {
					if st.currentPluralIndex >= 0 {
						st.msgstrPluralValues[st.currentPluralIndex].WriteString(value)
					} else {
						st.msgstrValue.WriteString(value)
					}
				}
				st.entryLines = append(st.entryLines, line)
				continue
			}
			trimmed = rest
		} else if strings.HasPrefix(trimmed, "#~| ") {
			rest := trimmed[4:]
			if strings.HasPrefix(rest, "msgid ") {
				value := strings.TrimPrefix(rest, "msgid ")
				value = strings.TrimSpace(value)
				value = strDeQuote(value)
				finishCurrentEntry(st, &entries)
				if st.currentEntry == nil || st.msgidValue.Len() > 0 || st.msgstrValue.Len() > 0 {
					startNewEntry(st)
				} else {
					resetEntryContent(st)
				}
				st.currentEntry.MsgIDPrevious = poParsedToPoFormat(value)
				st.currentEntry.Obsolete = true
				st.inObsolete = true
				st.entryLines = append(st.entryLines, line)
				continue
			}
		}

		// Header: first msgid "" starts the header block
		if !st.hasSeenHeaderBlock && strings.HasPrefix(trimmed, "msgid ") {
			value := strings.TrimPrefix(trimmed, "msgid ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			if value == "" {
				st.hasSeenHeaderBlock = true
				st.headerLines = append(st.headerLines, line)
				st.entryLines = append(st.entryLines, line)
				continue
			}
		}

		// Collect header lines until we leave the header
		if st.inHeader {
			if strings.HasPrefix(trimmed, "msgstr ") {
				value := strings.TrimPrefix(trimmed, "msgstr ")
				value = strings.TrimSpace(value)
				value = strDeQuote(value)
				if st.msgidValue.Len() == 0 && value == "" {
					st.headerLines = append(st.headerLines, line)
					continue
				}
			}
			if strings.HasPrefix(trimmed, `"`) {
				if st.currentEntry != nil || st.inMsgid || st.inMsgstr || st.inMsgidPlural {
					// Continuation of an entry, not header; fall through to entry parsing
				} else {
					st.headerLines = append(st.headerLines, trimmed)
					continue
				}
			}
			if trimmed == "" {
				if !st.hasSeenHeaderBlock {
					st.headerLines = append(st.headerLines, line)
					continue
				}
				st.inHeader = false
				st.msgidValue.Reset()
				st.msgstrValue.Reset()
				continue
			}
			if strings.HasPrefix(trimmed, "msgid ") {
				st.inHeader = false
				st.msgidValue.Reset()
				st.msgstrValue.Reset()
				// Fall through to entry parsing with this msgid line
			} else {
				st.headerLines = append(st.headerLines, line)
				continue
			}
		}

		// Entry parsing: dispatch by line kind
		kind := classifyPoLine(trimmed)
		switch kind {
		case poLineComment, poLineFlagHashComma, poLineFlagHashEq, poLineCommentRef, poLineCommentExtracted, poLineCommentPrev:
			if st.currentEntry == nil {
				st.currentEntry = &GettextEntry{}
				st.entryLines = nil
			}
			st.currentEntry.Comments = append(st.currentEntry.Comments, line)
			st.entryLines = append(st.entryLines, line)

		case poLineMsgid:
			finishCurrentEntry(st, &entries)
			if st.currentEntry == nil || st.msgidValue.Len() > 0 || st.msgstrValue.Len() > 0 {
				startNewEntry(st)
			} else {
				resetEntryContent(st)
			}
			if strings.HasPrefix(strings.TrimSpace(line), "#~ ") {
				st.inObsolete = true
			}
			value := strings.TrimPrefix(trimmed, "msgid ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			st.msgidValue.WriteString(value)
			st.entryLines = append(st.entryLines, line)

		case poLineMsgidPlural:
			st.inMsgid = false
			st.inMsgidPlural = true
			value := strings.TrimPrefix(trimmed, "msgid_plural ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			st.msgidPluralValue.WriteString(value)
			st.entryLines = append(st.entryLines, line)

		case poLineMsgstrN:
			st.inMsgid = false
			st.inMsgidPlural = false
			st.inMsgstr = true
			idxStr := strings.TrimPrefix(trimmed, "msgstr[")
			idxStr = strings.Split(idxStr, "]")[0]
			var idx int
			_, _ = fmt.Sscanf(idxStr, "%d", &idx)
			for len(st.msgstrPluralValues) <= idx {
				st.msgstrPluralValues = append(st.msgstrPluralValues, strings.Builder{})
			}
			st.currentPluralIndex = idx
			value := strings.TrimPrefix(trimmed, fmt.Sprintf("msgstr[%d] ", idx))
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			st.msgstrPluralValues[idx].WriteString(value)
			st.entryLines = append(st.entryLines, line)

		case poLineMsgstr:
			st.inMsgid = false
			st.inMsgidPlural = false
			st.inMsgstr = true
			value := strings.TrimPrefix(trimmed, "msgstr ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			st.msgstrValue.WriteString(value)
			st.entryLines = append(st.entryLines, line)

		case poLineQuotedString:
			if st.inMsgid || st.inMsgstr || st.inMsgidPlural {
				value := strDeQuote(trimmed)
				if st.inMsgid {
					st.msgidValue.WriteString(value)
				} else if st.inMsgidPlural {
					st.msgidPluralValue.WriteString(value)
				} else if st.inMsgstr {
					if st.currentPluralIndex >= 0 {
						st.msgstrPluralValues[st.currentPluralIndex].WriteString(value)
					} else {
						st.msgstrValue.WriteString(value)
					}
				}
				st.entryLines = append(st.entryLines, line)
			} else {
				if st.currentEntry != nil {
					st.entryLines = append(st.entryLines, line)
				} else if !st.inHeader {
					st.entryLines = append(st.entryLines, line)
				}
			}

		case poLineBlank:
			finishCurrentEntry(st, &entries)
			st.currentEntry = nil
			st.entryLines = nil
			st.msgidValue.Reset()
			st.msgstrValue.Reset()
			st.msgidPluralValue.Reset()
			st.msgstrPluralValues = nil
			st.inMsgid = false
			st.inMsgstr = false
			st.inMsgidPlural = false
			st.currentPluralIndex = -1
			st.inObsolete = false

		default:
			if st.currentEntry != nil {
				st.entryLines = append(st.entryLines, line)
			} else if !st.inHeader {
				st.entryLines = append(st.entryLines, line)
			}
		}
	}

	finishCurrentEntry(st, &entries)
	return entries, st.headerLines, nil
}

// entryHasFuzzyFlag returns true if any comment in the entry has the fuzzy flag.
func entryHasFuzzyFlag(comments []string) bool {
	for _, c := range comments {
		if commentHasFuzzyFlag(c) {
			return true
		}
	}
	return false
}

// BuildPoContent builds PO file content from header and entries.
// It is the inverse of ParsePoEntries: the output can be parsed back to produce the same header and entries.
// When header is nil or empty, no header block is written (only content entries).
// Entries with RawLines use them for round-trip; otherwise content is built from the entry.
func BuildPoContent(header []string, entries []*GettextEntry) []byte {
	var b strings.Builder
	if len(entries) > 0 && len(header) > 0 {
		for _, line := range header {
			b.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	for i, entry := range entries {
		if len(entry.RawLines) > 0 {
			for _, line := range entry.RawLines {
				b.WriteString(line)
				b.WriteString("\n")
			}
		} else {
			_ = writeGettextEntryToPO(&b, *entry)
		}
		// Add blank line between entries, but not after the last one
		if i < len(entries)-1 {
			b.WriteString("\n")
		}
	}
	return []byte(b.String())
}

// ParseEntryRange parses a range specification like "3,5,9-13", "-5", or "50-"
// into a set of entry indices. Entry 0 (header) is handled by MsgSelect; this
// returns only content entry indices (1 to maxEntry). Returns indices in
// ascending order, deduplicated.
// Empty spec selects all entries (equivalent to "1-").
// Range formats:
//   - N-M: entries N through M
//   - -N: entries 1 through N (omit start)
//   - N-: entries N through last (omit end)
//   - ~N: last N entries (equivalent to "<total-N+1>-<total>")
func ParseEntryRange(spec string, maxEntry int) ([]int, error) {
	if spec == "" {
		// Select all entries (1 through maxEntry)
		spec = "1-"
	}

	seen := make(map[int]bool)

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// ~N: last N entries (from maxEntry-N+1 to maxEntry)
		if strings.HasPrefix(part, "~") {
			nStr := strings.TrimSpace(part[1:])
			if nStr == "" {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			n, err := strconv.Atoi(nStr)
			if err != nil {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			if n <= 0 {
				continue
			}
			start := maxEntry - n + 1
			if start < 1 {
				start = 1
			}
			for i := start; i <= maxEntry; i++ {
				seen[i] = true
			}
			continue
		}

		if strings.Contains(part, "-") {
			// Range: N-M, -N (1 to N), or N- (N to last)
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			startStr := strings.TrimSpace(rangeParts[0])
			endStr := strings.TrimSpace(rangeParts[1])

			var start, end int
			if startStr == "" {
				// -N: from 1 to N
				if endStr == "" {
					return nil, fmt.Errorf("invalid range: %s", part)
				}
				var err error
				end, err = strconv.Atoi(endStr)
				if err != nil {
					return nil, fmt.Errorf("invalid range end: %s", endStr)
				}
				start = 1
			} else if endStr == "" {
				// N-: from N to last entry
				var err error
				start, err = strconv.Atoi(startStr)
				if err != nil {
					return nil, fmt.Errorf("invalid range start: %s", startStr)
				}
				end = maxEntry
			} else {
				// N-M: from N to M
				var err error
				start, err = strconv.Atoi(startStr)
				if err != nil {
					return nil, fmt.Errorf("invalid range start: %s", startStr)
				}
				end, err = strconv.Atoi(endStr)
				if err != nil {
					return nil, fmt.Errorf("invalid range end: %s", endStr)
				}
				if start > end {
					return nil, fmt.Errorf("invalid range: start %d > end %d", start, end)
				}
			}
			for i := start; i <= end; i++ {
				if i > 0 && i <= maxEntry {
					seen[i] = true
				}
			}
		} else {
			// Single number
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid entry number: %s", part)
			}
			if n > 0 && n <= maxEntry {
				seen[n] = true
			}
		}
	}

	// Build result in ascending order (1, 2, ...)
	var result []int
	for i := 1; i <= maxEntry; i++ {
		if seen[i] {
			result = append(result, i)
		}
	}
	return result, nil
}

// MsgSelect reads a PO/POT file, selects entries by state filter and range,
// and writes the result to w. Entry 0 (header) is included when content entries
// are selected, unless noHeader is true. If filter is nil, DefaultFilter() is used.
// Range applies to the filtered entry list (1 = first matching, etc.).
func MsgSelect(poFile, rangeSpec string, w io.Writer, noHeader bool, filter *EntryStateFilter) error {
	data, err := os.ReadFile(poFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", poFile, err)
	}

	entries, header, err := ParsePoEntries(data)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", poFile, err)
	}

	f := DefaultFilter()
	if filter != nil {
		f = *filter
	}

	// Filter by state first
	entriesSlice := make([]GettextEntry, len(entries))
	for i, e := range entries {
		entriesSlice[i] = *e
	}
	filtered := FilterGettextEntries(entriesSlice, f)
	maxEntry := len(filtered)
	indices, err := ParseEntryRange(rangeSpec, maxEntry)
	if err != nil {
		return fmt.Errorf("invalid range %q: %w", rangeSpec, err)
	}

	// Map range indices to filtered entries
	var selected []*GettextEntry
	for _, idx := range indices {
		if idx > 0 && idx <= len(filtered) {
			selected = append(selected, &filtered[idx-1])
		}
	}

	// If no content entries, output empty
	if len(selected) == 0 {
		return nil
	}

	// Write header (unless skipped)
	if !noHeader {
		for _, line := range header {
			if _, err := io.WriteString(w, line); err != nil {
				return err
			}
			if !strings.HasSuffix(line, "\n") {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}

	// Write selected entries
	for _, entry := range selected {
		for _, line := range entry.RawLines {
			if _, err := io.WriteString(w, line); err != nil {
				return err
			}
			if !strings.HasSuffix(line, "\n") {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}

	return nil
}

// WriteGettextJSONFromPOFile reads a PO/POT file, selects entries by state filter and range,
// and writes a single JSON object to w. If filter is nil, DefaultFilter() is used.
func WriteGettextJSONFromPOFile(poFile, rangeSpec string, w io.Writer, filter *EntryStateFilter) error {
	data, err := os.ReadFile(poFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", poFile, err)
	}
	entries, header, err := ParsePoEntries(data)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", poFile, err)
	}
	headerComment, headerMeta, err := SplitHeader(header)
	if err != nil {
		return fmt.Errorf("split header: %w", err)
	}
	f := DefaultFilter()
	if filter != nil {
		f = *filter
	}
	entriesSlice := make([]GettextEntry, len(entries))
	for i, e := range entries {
		entriesSlice[i] = *e
	}
	filtered := FilterGettextEntries(entriesSlice, f)
	maxEntry := len(filtered)
	indices, err := ParseEntryRange(rangeSpec, maxEntry)
	if err != nil {
		return fmt.Errorf("invalid range %q: %w", rangeSpec, err)
	}
	var selected []*GettextEntry
	for _, idx := range indices {
		if idx > 0 && idx <= len(filtered) {
			selected = append(selected, &filtered[idx-1])
		}
	}
	return BuildGettextJSON(headerComment, headerMeta, selected, w)
}
