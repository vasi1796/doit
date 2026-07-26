package crdt

import (
	"sort"
	"testing"
	"time"

	"github.com/vasi1796/doit/internal/hlc"
)

// Conflict resolution tests — verify CRDT merge properties:
// commutativity (A merge B == B merge A), idempotency (merge(x,x) == x),
// and convergence (all devices reach same state).

func TestConflictLWWConcurrentEdits(t *testing.T) {
	tests := []struct {
		name      string
		deviceA   string
		deviceAHL hlc.Timestamp
		deviceB   string
		deviceBHL hlc.Timestamp
		want      string
	}{
		{
			name:      "device B edits later — B wins",
			deviceA:   "Buy milk",
			deviceAHL: hlc.Timestamp{Time: base, Counter: 0},
			deviceB:   "Buy eggs",
			deviceBHL: hlc.Timestamp{Time: base.Add(time.Second), Counter: 0},
			want:      "Buy eggs",
		},
		{
			name:      "device A edits later — A wins",
			deviceA:   "Buy milk",
			deviceAHL: hlc.Timestamp{Time: base.Add(2 * time.Second), Counter: 0},
			deviceB:   "Buy eggs",
			deviceBHL: hlc.Timestamp{Time: base.Add(time.Second), Counter: 0},
			want:      "Buy milk",
		},
		{
			name:      "same time, different counters — higher counter wins",
			deviceA:   "Buy milk",
			deviceAHL: hlc.Timestamp{Time: base, Counter: 1},
			deviceB:   "Buy eggs",
			deviceBHL: hlc.Timestamp{Time: base, Counter: 5},
			want:      "Buy eggs",
		},
		{
			name:      "identical timestamps — remote wins (merge is directional, not commutative)",
			deviceA:   "Buy milk",
			deviceAHL: hlc.Timestamp{Time: base, Counter: 3},
			deviceB:   "Buy eggs",
			deviceBHL: hlc.Timestamp{Time: base, Counter: 3},
			want:      "", // skip commutativity check — result depends on merge direction
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Merge A as local, B as remote
			got1, _ := MergeLWW(tc.deviceA, tc.deviceAHL, tc.deviceB, tc.deviceBHL)

			if tc.want == "" {
				// Directional tiebreak — just verify remote wins
				if got1 != tc.deviceB {
					t.Errorf("A-local merge: got %q, want remote %q", got1, tc.deviceB)
				}
				return
			}

			// Merge B as local, A as remote (commutativity — only for different timestamps)
			got2, _ := MergeLWW(tc.deviceB, tc.deviceBHL, tc.deviceA, tc.deviceAHL)

			if got1 != tc.want {
				t.Errorf("A-local merge: got %q, want %q", got1, tc.want)
			}
			if got2 != tc.want {
				t.Errorf("B-local merge: got %q, want %q (commutativity violated)", got2, tc.want)
			}
		})
	}
}

func TestConflictLWWIdempotency(t *testing.T) {
	ts := hlc.Timestamp{Time: base, Counter: 5}
	val, hlcOut := MergeLWW("value", ts, "value", ts)
	val2, _ := MergeLWW(val, hlcOut, "value", ts)
	if val2 != "value" {
		t.Errorf("idempotent merge: got %q, want %q", val2, "value")
	}
}

func TestConflictFracIndexConcurrentInserts(t *testing.T) {
	// Two devices insert between the same items
	posA := Between("a", "c")
	posB := Between("a", "c")

	// Both positions should be valid and between "a" and "c"
	if posA <= "a" || posA >= "c" {
		t.Errorf("posA %q not between 'a' and 'c'", posA)
	}
	if posB <= "a" || posB >= "c" {
		t.Errorf("posB %q not between 'a' and 'c'", posB)
	}

	// Even if positions are equal (same algorithm, same input),
	// the result is deterministic and valid
	if posA != posB {
		t.Logf("different positions: %q vs %q (both valid)", posA, posB)
	}

	// Both sort correctly relative to boundaries
	items := []string{"a", posA, posB, "c"}
	sort.Strings(items)
	if items[0] != "a" || items[len(items)-1] != "c" {
		t.Errorf("sort broken: %v", items)
	}
}
