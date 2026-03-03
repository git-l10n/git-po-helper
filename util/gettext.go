// Package util provides PO file parsing and gettext-related utilities.
package util

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// GettextEntry represents a single PO/JSON entry. Used for parsing, comparison, and output.
// MsgID/MsgStr/MsgIDPlural/MsgStrPlural use PO string format (escape sequences like \n, \t
// stored as literal backslash+char, not decoded). RawLines preserves exact PO format for round-trip.
type GettextEntry struct {
	MsgID         string   `json:"msgid"`
	MsgStr        string   `json:"msgstr"`
	MsgIDPlural   string   `json:"msgid_plural,omitempty"`
	MsgStrPlural  []string `json:"msgstr_plural,omitempty"`
	Comments      []string `json:"comments,omitempty"`
	Fuzzy         bool     `json:"fuzzy"`
	Obsolete      bool     `json:"obsolete,omitempty"`       // True for #~ obsolete entries
	MsgIDPrevious string   `json:"msgid_previous,omitempty"` // For #~| format (gettext 0.19.8+)
	RawLines      []string `json:"-"`                        // Original PO lines for round-trip; empty when built from JSON
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

// ParsePoEntries parses PO file entries and returns entries and header.
// The header includes comments, the empty msgid/msgstr block, and any continuation lines.
// Entries are 1-based for content (header entry with empty msgid is not included).
func ParsePoEntries(data []byte) (entries []*GettextEntry, header []string, err error) {
	lines := strings.Split(string(data), "\n")
	var currentEntry *GettextEntry
	var inHeader = true
	var hasSeenHeaderBlock bool // true after we've seen msgid "" (the header block)
	var headerLines []string
	var entryLines []string
	var msgidValue strings.Builder
	var msgstrValue strings.Builder
	var msgidPluralValue strings.Builder
	var msgstrPluralValues []strings.Builder
	var inMsgid, inMsgstr, inMsgidPlural bool
	var currentPluralIndex int = -1
	var inObsolete bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Obsolete entry format: #~ msgid, #~ msgstr, #~| msgid (check first, before header/comment)
		if strings.HasPrefix(trimmed, "#~ ") {
			rest := trimmed[3:] // "#~ " = 3 chars
			// Only set inObsolete for continuation lines; #~ msgid starts new entry, save previous first
			if strings.HasPrefix(strings.TrimSpace(rest), `"`) || strings.HasPrefix(strings.TrimSpace(rest), "msgstr") {
				inObsolete = true
			}
			if strings.HasPrefix(strings.TrimSpace(rest), `"`) && (inMsgid || inMsgstr || inMsgidPlural) {
				value := strDeQuote(strings.TrimSpace(rest))
				if inMsgid {
					msgidValue.WriteString(value)
				} else if inMsgidPlural {
					msgidPluralValue.WriteString(value)
				} else if inMsgstr {
					if currentPluralIndex >= 0 {
						msgstrPluralValues[currentPluralIndex].WriteString(value)
					} else {
						msgstrValue.WriteString(value)
					}
				}
				entryLines = append(entryLines, line)
				continue
			}
			trimmed = rest
		} else if strings.HasPrefix(trimmed, "#~| ") {
			rest := trimmed[4:] // "#~| " = 4 chars
			if strings.HasPrefix(rest, "msgid ") {
				value := strings.TrimPrefix(rest, "msgid ")
				value = strings.TrimSpace(value)
				value = strDeQuote(value)
				if currentEntry != nil && (msgidValue.Len() > 0 || msgstrValue.Len() > 0) {
					currentEntry.MsgID = poParsedToPoFormat(msgidValue.String())
					currentEntry.MsgStr = poParsedToPoFormat(msgstrValue.String())
					if msgidPluralValue.Len() > 0 {
						currentEntry.MsgIDPlural = poParsedToPoFormat(msgidPluralValue.String())
						currentEntry.MsgStrPlural = make([]string, len(msgstrPluralValues))
						for i, b := range msgstrPluralValues {
							currentEntry.MsgStrPlural[i] = poParsedToPoFormat(b.String())
						}
					}
					currentEntry.RawLines = entryLines
					currentEntry.Fuzzy = entryHasFuzzyFlag(currentEntry.Comments)
					currentEntry.Obsolete = inObsolete
					entries = append(entries, currentEntry)
				}
				if currentEntry == nil || msgidValue.Len() > 0 || msgstrValue.Len() > 0 {
					currentEntry = &GettextEntry{}
					entryLines = []string{}
					msgidValue.Reset()
					msgstrValue.Reset()
					msgidPluralValue.Reset()
					msgstrPluralValues = []strings.Builder{}
				}
				currentEntry.MsgIDPrevious = poParsedToPoFormat(value)
				currentEntry.Obsolete = true
				inObsolete = true
				entryLines = append(entryLines, line)
				continue
			}
		}

		// Check for header (empty msgid entry)
		if !hasSeenHeaderBlock && strings.HasPrefix(trimmed, "msgid ") {
			value := strings.TrimPrefix(trimmed, "msgid ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			if value == "" {
				// This is the header entry (msgid "" block)
				hasSeenHeaderBlock = true
				headerLines = append(headerLines, line)
				entryLines = append(entryLines, line)
				// Continue to collect header
				continue
			}
		}

		// Collect header lines (including continuation lines after msgstr "")
		if inHeader {
			// Check for header msgstr (empty msgstr after empty msgid)
			if strings.HasPrefix(trimmed, "msgstr ") {
				value := strings.TrimPrefix(trimmed, "msgstr ")
				value = strings.TrimSpace(value)
				value = strDeQuote(value)
				if msgidValue.Len() == 0 && value == "" {
					// This is the header msgstr line
					headerLines = append(headerLines, line)
					// Continue collecting header (including continuation lines starting with ")
					// Header ends when we encounter an empty line or a new msgid entry
					continue
				}
			}

			// Check if this is a continuation line of header msgstr (starts with ")
			// Only collect as header if we're still in header mode and haven't started parsing an entry
			// Also check that we're not in the middle of parsing a msgid or msgstr (which would indicate an entry)
			if strings.HasPrefix(trimmed, `"`) {
				// If we're already parsing an entry (currentEntry exists or inMsgid/inMsgstr is set),
				// this continuation line belongs to the entry, not the header
				if currentEntry != nil || inMsgid || inMsgstr || inMsgidPlural {
					// This is a continuation line of an entry, not header
					// Don't process it here, let it be handled by entry parsing logic below
				} else {
					// For header continuation lines, keep the quotes
					headerLines = append(headerLines, trimmed)
					continue
				}
			}
			// Check if this is an empty line
			if trimmed == "" {
				if !hasSeenHeaderBlock {
					// Blank line in comment block (before msgid "") - keep in header
					headerLines = append(headerLines, line)
					continue
				}
				// Blank line after msgid ""/msgstr "" block - end of header
				inHeader = false
				msgidValue.Reset()
				msgstrValue.Reset()
				continue
			}
			// Check if this is a new msgid entry - end of header
			if strings.HasPrefix(trimmed, "msgid ") {
				value := strings.TrimPrefix(trimmed, "msgid ")
				value = strings.TrimSpace(value)
				value = strDeQuote(value)
				if value != "" {
					// This is a real entry, not header
					inHeader = false
					msgidValue.Reset()
					msgstrValue.Reset()
					// Don't continue, let it be processed as a normal entry
				} else {
					// This is a duplicate empty msgid after header - this should not happen
					// in a valid PO file, but if it does, end the header and start a new entry
					inHeader = false
					msgidValue.Reset()
					msgstrValue.Reset()
					// Don't continue, let it be processed as a normal entry
				}
			} else {
				// Other header lines (comments, etc.)
				headerLines = append(headerLines, line)
				continue
			}
		}

		// Parse entry
		if strings.HasPrefix(trimmed, "#") {
			// Comment line
			if currentEntry == nil {
				currentEntry = &GettextEntry{}
				entryLines = []string{}
			}
			currentEntry.Comments = append(currentEntry.Comments, line)
			entryLines = append(entryLines, line)
		} else if strings.HasPrefix(trimmed, "msgid ") {
			// Start of new entry (or obsolete #~ msgid)
			// Save previous entry if we have one and it has content
			if currentEntry != nil && (msgidValue.Len() > 0 || msgstrValue.Len() > 0) {
				currentEntry.MsgID = poParsedToPoFormat(msgidValue.String())
				currentEntry.MsgStr = poParsedToPoFormat(msgstrValue.String())
				if msgidPluralValue.Len() > 0 {
					currentEntry.MsgIDPlural = poParsedToPoFormat(msgidPluralValue.String())
					currentEntry.MsgStrPlural = make([]string, len(msgstrPluralValues))
					for i, b := range msgstrPluralValues {
						currentEntry.MsgStrPlural[i] = poParsedToPoFormat(b.String())
					}
				}
				currentEntry.RawLines = entryLines
				currentEntry.Fuzzy = entryHasFuzzyFlag(currentEntry.Comments)
				currentEntry.Obsolete = inObsolete
				entries = append(entries, currentEntry)
			}
			// Start new entry (or continue existing entry if it only has comments)
			if currentEntry == nil {
				currentEntry = &GettextEntry{}
				entryLines = []string{}
			} else if msgidValue.Len() > 0 || msgstrValue.Len() > 0 {
				currentEntry = &GettextEntry{}
				entryLines = []string{}
			}
			// If this line came from #~ msgid, mark the new entry as obsolete
			if strings.HasPrefix(strings.TrimSpace(line), "#~ ") {
				inObsolete = true
			}
			// If currentEntry has comments but no msgid/msgstr, keep it and continue
			// entryLines already contains the comments, so we don't reset it
			msgidValue.Reset()
			msgstrValue.Reset()
			msgidPluralValue.Reset()
			msgstrPluralValues = []strings.Builder{}
			inMsgid = true
			inMsgstr = false
			inMsgidPlural = false
			currentPluralIndex = -1

			value := strings.TrimPrefix(trimmed, "msgid ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			msgidValue.WriteString(value)
			entryLines = append(entryLines, line)
		} else if strings.HasPrefix(trimmed, "msgid_plural ") {
			inMsgid = false
			inMsgidPlural = true
			value := strings.TrimPrefix(trimmed, "msgid_plural ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			msgidPluralValue.WriteString(value)
			entryLines = append(entryLines, line)
		} else if strings.HasPrefix(trimmed, "msgstr[") {
			// Plural form
			inMsgid = false
			inMsgidPlural = false
			inMsgstr = true
			// Extract index
			idxStr := strings.TrimPrefix(trimmed, "msgstr[")
			idxStr = strings.Split(idxStr, "]")[0]
			var idx int
			fmt.Sscanf(idxStr, "%d", &idx)
			// Extend slice if needed
			for len(msgstrPluralValues) <= idx {
				msgstrPluralValues = append(msgstrPluralValues, strings.Builder{})
			}
			currentPluralIndex = idx
			value := strings.TrimPrefix(trimmed, fmt.Sprintf("msgstr[%d] ", idx))
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			msgstrPluralValues[idx].WriteString(value)
			entryLines = append(entryLines, line)
		} else if strings.HasPrefix(trimmed, "msgstr ") {
			inMsgid = false
			inMsgidPlural = false
			inMsgstr = true
			value := strings.TrimPrefix(trimmed, "msgstr ")
			value = strings.TrimSpace(value)
			value = strDeQuote(value)
			msgstrValue.WriteString(value)
			entryLines = append(entryLines, line)
		} else if strings.HasPrefix(trimmed, `"`) && (inMsgid || inMsgstr || inMsgidPlural) {
			// Continuation line
			value := strDeQuote(trimmed)
			if inMsgid {
				msgidValue.WriteString(value)
			} else if inMsgidPlural {
				msgidPluralValue.WriteString(value)
			} else if inMsgstr {
				if currentPluralIndex >= 0 {
					msgstrPluralValues[currentPluralIndex].WriteString(value)
				} else {
					msgstrValue.WriteString(value)
				}
			}
			entryLines = append(entryLines, line)
		} else if trimmed == "" {
			// Empty line - end of entry (only if we have a current entry)
			// For entries with msgid starting with empty string, we need to check
			// if we have collected any continuation lines (msgidValue.Len() > 0)
			// or if we have a complete entry with msgstr
			if currentEntry != nil && (msgidValue.Len() > 0 || msgstrValue.Len() > 0) {
				currentEntry.MsgID = poParsedToPoFormat(msgidValue.String())
				currentEntry.MsgStr = poParsedToPoFormat(msgstrValue.String())
				if msgidPluralValue.Len() > 0 {
					currentEntry.MsgIDPlural = poParsedToPoFormat(msgidPluralValue.String())
					currentEntry.MsgStrPlural = make([]string, len(msgstrPluralValues))
					for i, b := range msgstrPluralValues {
						currentEntry.MsgStrPlural[i] = poParsedToPoFormat(b.String())
					}
				}
				currentEntry.RawLines = entryLines
				currentEntry.Fuzzy = entryHasFuzzyFlag(currentEntry.Comments)
				currentEntry.Obsolete = inObsolete
				entries = append(entries, currentEntry)
			}
			currentEntry = nil
			entryLines = []string{}
			msgidValue.Reset()
			msgstrValue.Reset()
			msgidPluralValue.Reset()
			msgstrPluralValues = []strings.Builder{}
			inMsgid = false
			inMsgstr = false
			inMsgidPlural = false
			currentPluralIndex = -1
			inObsolete = false
		} else {
			// Other lines (continuation, etc.)
			if currentEntry != nil {
				entryLines = append(entryLines, line)
			} else if !inHeader {
				// If we're not in header and not in an entry, this might be a continuation
				// of a previous entry or a new entry starting
				entryLines = append(entryLines, line)
			}
		}
	}

	// Handle last entry
	if currentEntry != nil && (msgidValue.Len() > 0 || msgstrValue.Len() > 0) {
		currentEntry.MsgID = poParsedToPoFormat(msgidValue.String())
		currentEntry.MsgStr = poParsedToPoFormat(msgstrValue.String())
		if msgidPluralValue.Len() > 0 {
			currentEntry.MsgIDPlural = poParsedToPoFormat(msgidPluralValue.String())
			currentEntry.MsgStrPlural = make([]string, len(msgstrPluralValues))
			for i, b := range msgstrPluralValues {
				currentEntry.MsgStrPlural[i] = poParsedToPoFormat(b.String())
			}
		}
		currentEntry.RawLines = entryLines
		currentEntry.Fuzzy = entryHasFuzzyFlag(currentEntry.Comments)
		currentEntry.Obsolete = inObsolete
		entries = append(entries, currentEntry)
	}

	return entries, headerLines, nil
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
			writeGettextEntryToPO(&b, *entry)
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
