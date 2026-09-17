package version

import "testing"

func TestParseLeadingNumericParts(t *testing.T) {
	tests := []struct {
		in      string
		want    []int
		wantErr bool
	}{
		{"0.8.4", []int{0, 8, 4}, false},
		{"0.8.4.23.gabc.dirty", []int{0, 8, 4, 23}, false},
		{"v1.2.3", []int{1, 2, 3}, false},
		{"10", []int{10}, false},
		{"1.0", []int{1, 0}, false}, // too short for semver; still parses
		{"", nil, true},
		{"undefined", nil, true},
		{"gabc", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseLeadingNumericParts(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseLeadingNumericParts(%q) err = nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLeadingNumericParts(%q): %v", tt.in, err)
			continue
		}
		if !intSliceEqual(got, tt.want) {
			t.Errorf("ParseLeadingNumericParts(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseSemverParts(t *testing.T) {
	tests := []struct {
		in      string
		want    []int
		wantErr bool
	}{
		{"0.8.4", []int{0, 8, 4}, false},
		{"0.8.4.23.gabc.dirty", []int{0, 8, 4}, false}, // truncate distance
		{"v1.0.0", []int{1, 0, 0}, false},
		{"1.0.0.dirty", []int{1, 0, 0}, false}, // VERSION-GEN style dirty suffix
		{"1.0", nil, true},                     // bad tag: only major.minor
		{"1", nil, true},
		{"v1.0", nil, true},
		{"0.8.4.23", []int{0, 8, 4}, false},
		{"", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseSemverParts(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSemverParts(%q) err = nil, want error (reject short tags)", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSemverParts(%q): %v", tt.in, err)
			continue
		}
		if !intSliceEqual(got, tt.want) {
			t.Errorf("ParseSemverParts(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseSemverParts_RejectsTwoComponentTags(t *testing.T) {
	// Guards against mistaken release tags like v1.0 instead of v1.0.0.
	for _, bad := range []string{"1.0", "v1.0", "0.8", "v0.8", "2.1"} {
		if _, err := ParseSemverParts(bad); err == nil {
			t.Errorf("ParseSemverParts(%q) succeeded; want error for non-X.Y.Z tag", bad)
		}
	}
	for _, good := range []string{"1.0.0", "v1.0.0", "0.8.4", "0.8.4.12.gabcdef"} {
		if _, err := ParseSemverParts(good); err != nil {
			t.Errorf("ParseSemverParts(%q): %v", good, err)
		}
	}
}

func TestParseConstraintParts(t *testing.T) {
	tests := []struct {
		in      string
		want    []int
		wantErr bool
	}{
		{"1", []int{1}, false},
		{"0.8", []int{0, 8}, false},
		{"0.8.4", []int{0, 8, 4}, false},
		{"0.8.4.99", []int{0, 8, 4}, false}, // constraint also capped at 3
		{"v0.8", []int{0, 8}, false},
		{"0.8.x", nil, true},
		{"", nil, true},
		{"0.8.g1", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseConstraintParts(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseConstraintParts(%q) err = nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseConstraintParts(%q): %v", tt.in, err)
			continue
		}
		if !intSliceEqual(got, tt.want) {
			t.Errorf("ParseConstraintParts(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCompareAtPrecision(t *testing.T) {
	tests := []struct {
		actual, constraint string
		want               int
	}{
		{"0.8.4", "0.8", 0},
		{"0.8.4", "0.8.4", 0},
		{"0.8.4", "0.9", -1},
		{"0.8.4", "0.7", 1},
		{"0.8.4", "1", -1},
		{"1.0.0", "1", 0},
		{"1.2.3", "1.2", 0},
		{"1.2.3", "1.3", -1},
		{"0.8.4.23.gabc", "0.8.4", 0}, // distance ignored
		{"0.8.4.23.gabc", "0.8", 0},
	}
	for _, tt := range tests {
		got, err := CompareAtPrecision(tt.actual, tt.constraint)
		if err != nil {
			t.Errorf("CompareAtPrecision(%q, %q): %v", tt.actual, tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("CompareAtPrecision(%q, %q) = %d, want %d", tt.actual, tt.constraint, got, tt.want)
		}
	}
}

func TestCompareAtPrecision_RejectsShortActual(t *testing.T) {
	if _, err := CompareAtPrecision("1.0", "1.0.0"); err == nil {
		t.Fatal("CompareAtPrecision with actual 1.0: want error")
	}
}

func TestComparePadded(t *testing.T) {
	tests := []struct {
		actual, constraint string
		want               int
	}{
		{"0.8.4", "0.8", 1}, // 0.8.4 > 0.8.0
		{"0.8.0", "0.8", 0},
		{"0.8.4", "0.8.4", 0},
		{"1.0.0", "1", 0},
		{"0.8.4.99.gabc", "0.8.4", 0}, // ≡ 0.8.4 after truncate
		{"0.8.4.99.gabc", "0.8.5", -1},
		{"0.8.5.1.gabc", "0.8.4", 1},
	}
	for _, tt := range tests {
		got, err := ComparePadded(tt.actual, tt.constraint)
		if err != nil {
			t.Errorf("ComparePadded(%q, %q): %v", tt.actual, tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ComparePadded(%q, %q) = %d, want %d", tt.actual, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	actual := "0.8.4.23.gabc" // compare as 0.8.4
	tests := []struct {
		op         Op
		constraint string
		want       bool
	}{
		{OpEQ, "0.8", true},
		{OpEQ, "0.8.4", true},
		{OpEQ, "0.8.5", false},
		{OpEQ, "1", false},
		{OpGE, "0.8", true},
		{OpGE, "0.8.4", true},
		{OpGE, "0.8.5", false},
		{OpGE, "1", false},
		{OpGT, "0.8", true},
		{OpGT, "0.7", true},
		{OpGT, "0.8.3", true},
		{OpGT, "0.8.4", false}, // equal after truncate to 3 parts
		{OpLT, "1", true},
		{OpLT, "0.8", false},
		{OpLE, "0.8", false},
		{OpLE, "0.8.4", true},
		{OpLE, "0.8.4.23", true}, // constraint capped at 0.8.4
		{OpLE, "0.9", true},
		{OpGE, "0.8.4.23", true}, // constraint capped at 0.8.4
		{OpGE, "0.8.4.24", true}, // still 0.8.4 after cap
	}
	for _, tt := range tests {
		got, err := Satisfies(actual, tt.op, tt.constraint)
		if err != nil {
			t.Errorf("Satisfies(%q, %s, %q): %v", actual, tt.op, tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Satisfies(%q, %s, %q) = %v, want %v", actual, tt.op, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfies_RejectsTwoComponentActual(t *testing.T) {
	_, err := Satisfies("1.0", OpGE, "1.0.0")
	if err == nil {
		t.Fatal("Satisfies(1.0, ...) succeeded; want error for bad tag shape")
	}
}

func TestDefaultVersionHasThreeParts(t *testing.T) {
	parts, err := ParseSemverParts(Version)
	if err != nil {
		t.Fatalf("default Version %q: %v", Version, err)
	}
	if len(parts) != MaxSemverParts {
		t.Fatalf("default Version %q: got %d parts, want %d", Version, len(parts), MaxSemverParts)
	}
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
