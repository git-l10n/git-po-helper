// Package util provides utility functions for counting PO/POT entries.
package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CountPotEntries counts msgid entries in a POT file.
// It excludes the header entry (which has an empty msgid) and counts
// only non-empty msgid entries.
//
// The function:
// - Opens the POT file
// - Scans for lines starting with "msgid " (excluding commented entries)
// - Parses msgid values to identify the header entry (empty msgid)
// - Returns the count of non-empty msgid entries
func CountPotEntries(potFile string) (int, error) {
	f, err := os.Open(potFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open POT file %s: %w", potFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	inMsgid := false
	msgidValue := ""
	headerFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comment lines (obsolete entries, etc.)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// If we were already in a msgid, finish the previous one
			if inMsgid {
				if !headerFound && strings.Trim(msgidValue, `"`) == "" {
					headerFound = true
				} else if strings.Trim(msgidValue, `"`) != "" {
					// Non-empty msgid entry
					count++
				}
			}
			// Start new msgid entry
			inMsgid = true
			// Extract the msgid value (may be on same line or continue on next lines)
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			// Remove quotes if present
			msgidValue = strings.Trim(msgidValue, `"`)
			continue
		}

		// If we're in a msgid entry and this line continues it (starts with quote)
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// Continuation line - append to msgidValue (remove quotes)
			contValue := strings.Trim(trimmed, `"`)
			msgidValue += contValue
			continue
		}

		// If we encounter msgstr, it means we've finished the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// End of msgid entry
			if !headerFound && strings.Trim(msgidValue, `"`) == "" {
				headerFound = true
			} else if strings.Trim(msgidValue, `"`) != "" {
				// Non-empty msgid entry
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}

		// Empty line might indicate end of entry, but we'll rely on msgstr
		// to be more accurate
	}

	// Handle last entry if file doesn't end with newline or msgstr
	if inMsgid {
		if !headerFound && strings.Trim(msgidValue, `"`) == "" {
			headerFound = true
		} else if strings.Trim(msgidValue, `"`) != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read POT file %s: %w", potFile, err)
	}

	return count, nil
}

// CountPoEntries counts msgid entries in a PO file.
// It excludes the header entry (which has an empty msgid) and counts
// only non-empty msgid entries.
//
// The function:
// - Opens the PO file
// - Scans for lines starting with "msgid " (excluding commented entries)
// - Parses msgid values to identify the header entry (empty msgid)
// - Returns the count of non-empty msgid entries
func CountPoEntries(poFile string) (int, error) {
	f, err := os.Open(poFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open PO file %s: %w", poFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	inMsgid := false
	msgidValue := ""
	headerFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comment lines (obsolete entries, etc.)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// If we were already in a msgid, finish the previous one
			if inMsgid {
				if !headerFound && strings.Trim(msgidValue, `"`) == "" {
					headerFound = true
				} else if strings.Trim(msgidValue, `"`) != "" {
					// Non-empty msgid entry
					count++
				}
			}
			// Start new msgid entry
			inMsgid = true
			// Extract the msgid value (may be on same line or continue on next lines)
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			// Remove quotes if present
			msgidValue = strings.Trim(msgidValue, `"`)
			continue
		}

		// If we're in a msgid entry and this line continues it (starts with quote)
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// Continuation line - append to msgidValue (remove quotes)
			contValue := strings.Trim(trimmed, `"`)
			msgidValue += contValue
			continue
		}

		// If we encounter msgstr, it means we've finished the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// End of msgid entry
			if !headerFound && strings.Trim(msgidValue, `"`) == "" {
				headerFound = true
			} else if strings.Trim(msgidValue, `"`) != "" {
				// Non-empty msgid entry
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}

		// Empty line might indicate end of entry, but we'll rely on msgstr
		// to be more accurate
	}

	// Handle last entry if file doesn't end with newline or msgstr
	if inMsgid {
		if !headerFound && strings.Trim(msgidValue, `"`) == "" {
			headerFound = true
		} else if strings.Trim(msgidValue, `"`) != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read PO file %s: %w", poFile, err)
	}

	return count, nil
}

// CountNewEntries counts untranslated entries in a PO file.
// It uses `msgattrib --untranslated` to extract untranslated entries,
// then counts the msgid entries excluding the header entry (empty msgid).
//
// The function:
// - Executes `msgattrib --untranslated poFile`
// - Scans output for lines starting with "msgid "
// - Excludes the header entry (msgid "")
// - Returns the count of untranslated msgid entries
func CountNewEntries(poFile string) (int, error) {
	cmd := exec.Command("msgattrib", "--untranslated", poFile)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("msgattrib failed for %s: %w\nstderr: %s",
				poFile, err, string(exitError.Stderr))
		}
		return 0, fmt.Errorf("failed to execute msgattrib for %s: %w", poFile, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	inMsgid := false
	msgidValue := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// Extract msgid value
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			inMsgid = true
			continue
		}

		// If we're in a msgid and encounter a continuation line
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// This is a multi-line msgid, just mark it as non-empty
			msgidValue += "continuation"
			continue
		}

		// If we encounter msgstr, finish the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// Check if msgid is non-empty (not the header)
			if strings.Trim(msgidValue, `"`) != "" {
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan msgattrib output: %w", err)
	}

	return count, nil
}

// CountFuzzyEntries counts fuzzy entries in a PO file.
// It uses `msgattrib --only-fuzzy` to extract fuzzy entries,
// then counts the msgid entries excluding the header entry (empty msgid).
//
// The function:
// - Executes `msgattrib --only-fuzzy poFile`
// - Scans output for lines starting with "msgid "
// - Excludes the header entry (msgid "")
// - Returns the count of fuzzy msgid entries
func CountFuzzyEntries(poFile string) (int, error) {
	cmd := exec.Command("msgattrib", "--only-fuzzy", poFile)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("msgattrib failed for %s: %w\nstderr: %s",
				poFile, err, string(exitError.Stderr))
		}
		return 0, fmt.Errorf("failed to execute msgattrib for %s: %w", poFile, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	inMsgid := false
	msgidValue := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// Extract msgid value
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			inMsgid = true
			continue
		}

		// If we're in a msgid and encounter a continuation line
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// This is a multi-line msgid, just mark it as non-empty
			msgidValue += "continuation"
			continue
		}

		// If we encounter msgstr, finish the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// Check if msgid is non-empty (not the header)
			if strings.Trim(msgidValue, `"`) != "" {
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan msgattrib output: %w", err)
	}

	return count, nil
}
