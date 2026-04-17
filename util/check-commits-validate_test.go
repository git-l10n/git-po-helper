package util

import (
	"strings"
	"testing"
)

func TestCommitMsgDisplayWidth_utf8Punctuation(t *testing.T) {
	// U+201C LEFT DOUBLE QUOTATION MARK: 3 UTF-8 bytes, display width 1.
	const ldqm = "\u201c"
	line := strings.Repeat("a", 70) + ldqm + "b"
	if got, want := len(line), 74; got != want {
		t.Fatalf("len(line) = %d, want %d (byte length includes multibyte punct)", got, want)
	}
	if got, want := commitMsgDisplayWidth(line), 72; got != want {
		t.Fatalf("commitMsgDisplayWidth(line) = %d, want %d", got, want)
	}
}
