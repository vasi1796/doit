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

func TestConflictORSetConcurrentAddRemove(t *testing.T) {
	tests := []struct {
		name    string
		deviceA []ORSetOp
		deviceB []ORSetOp
		want    []string // expected materialized set
	}{
		{
			name: "A adds, B removes with different tag — add survives",
			deviceA: []ORSetOp{
				{Value: "label-1", Tag: "add-tag-1", Op: "add"},
			},
			deviceB: []ORSetOp{
				{Value: "label-1", Tag: "remove-tag-2", Op: "remove"},
			},
			want: []string{"label-1"},
		},
		{
			name: "A adds, then A removes same tag — label gone",
			deviceA: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
				{Value: "label-1", Tag: "t1", Op: "remove"},
			},
			deviceB: []ORSetOp{},
			want:    []string{},
		},
		{
			name: "remove then re-add with new tag — label present",
			deviceA: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
				{Value: "label-1", Tag: "t1", Op: "remove"},
			},
			deviceB: []ORSetOp{
				{Value: "label-1", Tag: "t2", Op: "add"},
			},
			want: []string{"label-1"},
		},
		{
			name: "concurrent adds of different labels — both present",
			deviceA: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
			},
			deviceB: []ORSetOp{
				{Value: "label-2", Tag: "t2", Op: "add"},
			},
			want: []string{"label-1", "label-2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Merge A then B
			merged1 := MergeORSet(tc.deviceA, tc.deviceB)
			result1 := Materialize(merged1)
			if result1 == nil {
				result1 = []string{}
			}
			sort.Strings(result1)

			// Merge B then A (commutativity)
			merged2 := MergeORSet(tc.deviceB, tc.deviceA)
			result2 := Materialize(merged2)
			if result2 == nil {
				result2 = []string{}
			}
			sort.Strings(result2)

			want := tc.want
			sort.Strings(want)

			// Check correctness
			assertStringSlice(t, "A-then-B", result1, want)
			// Check commutativity
			assertStringSlice(t, "B-then-A", result2, want)
		})
	}
}

func TestConflictORSetIdempotency(t *testing.T) {
	ops := []ORSetOp{
		{Value: "label-1", Tag: "t1", Op: "add"},
		{Value: "label-2", Tag: "t2", Op: "add"},
	}

	merged := MergeORSet(ops, ops)
	reMerged := MergeORSet(merged, ops)

	result1 := Materialize(merged)
	result2 := Materialize(reMerged)
	sort.Strings(result1)
	sort.Strings(result2)

	assertStringSlice(t, "idempotent", result1, result2)
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

func assertStringSlice(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v (len %d), want %v (len %d)", label, got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", label, got, want)
		}
	}
}
