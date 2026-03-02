// Package util provides JSON repair for LLM-generated output.
package util

import (
	"bytes"

	log "github.com/sirupsen/logrus"
)

// PrepareJSONForParse preprocesses LLM-generated JSON for parsing.
// Handles: UTF-8 BOM, markdown code blocks (```json ... ```), leading/trailing text.
// Returns cleaned JSON bytes or original if no preprocessing needed.
// Used by both review JSON and gettext JSON parsers.
func PrepareJSONForParse(data []byte, parseErr error) []byte {
	log.Warnf("fall back to prepare (remove BOM and markdown) to fix json: %v", parseErr)
	data = bytes.TrimSpace(data)
	// Strip UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	// Extract from markdown code block: ```json ... ``` or ``` ... ```
	if idx := bytes.Index(data, []byte("```")); idx >= 0 {
		data = data[idx+3:]
		if bytes.HasPrefix(data, []byte("json")) {
			data = bytes.TrimSpace(data[4:])
		}
		if end := bytes.Index(data, []byte("```")); end >= 0 {
			data = bytes.TrimSpace(data[:end])
		}
	}
	// Extract JSON object by brace matching (handles leading/trailing text)
	if extracted, err := ExtractJSONFromOutput(data); err == nil {
		return extracted
	}
	return data
}
