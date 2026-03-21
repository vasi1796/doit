package crdt

import "testing"

func TestBetween(t *testing.T) {
	tests := []struct {
		name   string
		before string
		after  string
	}{
		{name: "between a and c", before: "a", after: "c"},
		{name: "between a and z", before: "a", after: "z"},
		{name: "between a and b", before: "a", after: "b"},
		{name: "before first", before: "", after: "m"},
		{name: "after last", before: "m", after: ""},
		{name: "between close values", before: "abc", after: "abd"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Between(tc.before, tc.after)

			effectiveBefore := tc.before
			if effectiveBefore == "" {
				effectiveBefore = "a"
			}
			effectiveAfter := tc.after
			if effectiveAfter == "" {
				effectiveAfter = "z"
			}

			if got <= effectiveBefore {
				t.Errorf("Between(%q, %q) = %q, not after %q", tc.before, tc.after, got, effectiveBefore)
			}
			if got >= effectiveAfter {
				t.Errorf("Between(%q, %q) = %q, not before %q", tc.before, tc.after, got, effectiveAfter)
			}
		})
	}
}

func TestBetweenOrdering(t *testing.T) {
	// Insert 20 items sequentially and verify all positions sort correctly
	positions := []string{First()}
	for i := 0; i < 20; i++ {
		prev := positions[len(positions)-1]
		pos := Between(prev, "")
		if pos <= prev {
			t.Fatalf("position %d: %q not after %q", i, pos, prev)
		}
		positions = append(positions, pos)
	}

	// Verify sorted order
	for i := 1; i < len(positions); i++ {
		if positions[i] <= positions[i-1] {
			t.Fatalf("position %d (%q) <= position %d (%q)", i, positions[i], i-1, positions[i-1])
		}
	}
}

func TestBetweenKeyLength(t *testing.T) {
	// Insert 50 items and check keys don't grow unreasonably
	pos := First()
	for i := 0; i < 50; i++ {
		pos = Between(pos, "")
	}
	// Position should not exceed ~50 chars for 50 sequential inserts
	if len(pos) > 60 {
		t.Errorf("position after 50 inserts is %d chars, expected < 60", len(pos))
	}
}

func TestFirstLast(t *testing.T) {
	if First() >= Last() {
		t.Errorf("First() %q >= Last() %q", First(), Last())
	}
}
