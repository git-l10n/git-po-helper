// Package version implements versions related functions
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxSemverParts is the number of numeric components used for comparison
// (major.minor.patch). Extra components from git describe (commit distance,
// g<oid>, dirty) are ignored.
const MaxSemverParts = 3

// Version is the version of git-po-helper (injected via -ldflags at build time).
// Fallback must be major.minor.patch (three components), never "1.0" or similar.
var Version = "0.0.0"

// ParseLeadingNumericParts extracts consecutive numeric dotted components from
// the start of s (e.g. "0.8.4.23.gabc.dirty" → [0,8,4,23]). Non-numeric
// segments and anything after them are ignored. Returns an error if s has no
// leading numeric component.
func ParseLeadingNumericParts(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty version")
	}
	// Strip optional leading 'v' (git describe style).
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")

	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			break
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		if n < 0 {
			return nil, fmt.Errorf("invalid version component %q in %q", p, s)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no numeric version in %q", s)
	}
	return out, nil
}

// ParseSemverParts returns the first MaxSemverParts numeric components of s.
// The version must have at least MaxSemverParts leading numeric components
// (tags must be like v1.0.0, not v1.0). Components beyond patch (e.g. git
// describe commit distance) are discarded.
func ParseSemverParts(s string) ([]int, error) {
	parts, err := ParseLeadingNumericParts(s)
	if err != nil {
		return nil, err
	}
	if len(parts) < MaxSemverParts {
		return nil, fmt.Errorf("version %q must have at least %d numeric components (major.minor.patch); got %d (use tags like v1.0.0, not v1.0)",
			s, MaxSemverParts, len(parts))
	}
	return parts[:MaxSemverParts], nil
}

// ParseConstraintParts parses a user-supplied version constraint. Every
// component must be a non-negative integer (e.g. "1", "0.8", "0.8.4").
// At most MaxSemverParts components are kept.
func ParseConstraintParts(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty version")
	}
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")

	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("invalid version %q", s)
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q: %w", s, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("invalid version component %q in %q", p, s)
		}
		out = append(out, n)
		if len(out) == MaxSemverParts {
			break
		}
	}
	return out, nil
}

// CompareAtPrecision compares actual to constraint using only as many leading
// components as constraint has. Actual is always normalized to major.minor.patch
// (max 3 parts). Returns -1 / 0 / +1.
func CompareAtPrecision(actual, constraint string) (int, error) {
	a, c, err := parseActualAndConstraint(actual, constraint)
	if err != nil {
		return 0, err
	}
	n := len(c)
	aCmp := make([]int, n)
	copy(aCmp, a)
	return compareParts(aCmp, c), nil
}

// ComparePadded compares actual to constraint after padding the shorter side
// with zeros. Actual is major.minor.patch only (so 0.8.4.24.g… ≡ 0.8.4).
func ComparePadded(actual, constraint string) (int, error) {
	a, c, err := parseActualAndConstraint(actual, constraint)
	if err != nil {
		return 0, err
	}
	n := len(a)
	if len(c) > n {
		n = len(c)
	}
	aPad := make([]int, n)
	cPad := make([]int, n)
	copy(aPad, a)
	copy(cPad, c)
	return compareParts(aPad, cPad), nil
}

func parseActualAndConstraint(actual, constraint string) (a, c []int, err error) {
	a, err = ParseSemverParts(actual)
	if err != nil {
		return nil, nil, fmt.Errorf("actual version: %w", err)
	}
	c, err = ParseConstraintParts(constraint)
	if err != nil {
		return nil, nil, fmt.Errorf("constraint version: %w", err)
	}
	return a, c, nil
}

func compareParts(a, c []int) int {
	n := len(a)
	if len(c) < n {
		n = len(c)
	}
	for i := 0; i < n; i++ {
		if a[i] < c[i] {
			return -1
		}
		if a[i] > c[i] {
			return 1
		}
	}
	if len(a) < len(c) {
		for i := len(a); i < len(c); i++ {
			if c[i] != 0 {
				return -1
			}
		}
		return 0
	}
	if len(a) > len(c) {
		for i := len(c); i < len(a); i++ {
			if a[i] != 0 {
				return 1
			}
		}
		return 0
	}
	return 0
}

// Op is a version comparison operator.
type Op string

const (
	OpLT Op = "lt"
	OpLE Op = "le"
	OpGT Op = "gt"
	OpGE Op = "ge"
	OpEQ Op = "eq"
)

// Satisfies reports whether actual satisfies op against constraint.
// Actual must be at least major.minor.patch; only those three parts are used
// (git describe distance / g<oid> / dirty are ignored).
// --eq uses precision of the constraint (0.8.4 satisfies --eq 0.8).
// --lt/--le/--gt/--ge pad the shorter side with zeros (0.8.4 satisfies --gt 0.8).
func Satisfies(actual string, op Op, constraint string) (bool, error) {
	var cmp int
	var err error
	if op == OpEQ {
		cmp, err = CompareAtPrecision(actual, constraint)
	} else {
		cmp, err = ComparePadded(actual, constraint)
	}
	if err != nil {
		return false, err
	}
	switch op {
	case OpLT:
		return cmp < 0, nil
	case OpLE:
		return cmp <= 0, nil
	case OpGT:
		return cmp > 0, nil
	case OpGE:
		return cmp >= 0, nil
	case OpEQ:
		return cmp == 0, nil
	default:
		return false, fmt.Errorf("unknown operator %q", op)
	}
}
