package crdt

import (
	"sort"
	"testing"
)

func TestMergeORSet(t *testing.T) {
	tests := []struct {
		name   string
		local  []ORSetOp
		remote []ORSetOp
		want   int // expected number of merged ops
	}{
		{
			name:   "disjoint sets",
			local:  []ORSetOp{{Value: "a", Tag: "t1", Op: "add"}},
			remote: []ORSetOp{{Value: "b", Tag: "t2", Op: "add"}},
			want:   2,
		},
		{
			name:   "duplicate tags deduplicated",
			local:  []ORSetOp{{Value: "a", Tag: "t1", Op: "add"}},
			remote: []ORSetOp{{Value: "a", Tag: "t1", Op: "add"}},
			want:   1,
		},
		{
			name:   "empty local",
			local:  nil,
			remote: []ORSetOp{{Value: "a", Tag: "t1", Op: "add"}},
			want:   1,
		},
		{
			name:   "both empty",
			local:  nil,
			remote: nil,
			want:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			merged := MergeORSet(tc.local, tc.remote)
			if len(merged) != tc.want {
				t.Errorf("MergeORSet() len = %d, want %d", len(merged), tc.want)
			}
		})
	}
}

func TestMaterialize(t *testing.T) {
	tests := []struct {
		name string
		ops  []ORSetOp
		want []string
	}{
		{
			name: "single add",
			ops:  []ORSetOp{{Value: "label-1", Tag: "t1", Op: "add"}},
			want: []string{"label-1"},
		},
		{
			name: "add then remove same tag",
			ops: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
				{Value: "label-1", Tag: "t1", Op: "remove"},
			},
			want: []string{},
		},
		{
			name: "add, remove, re-add with different tag",
			ops: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
				{Value: "label-1", Tag: "t1", Op: "remove"},
				{Value: "label-1", Tag: "t2", Op: "add"},
			},
			want: []string{"label-1"},
		},
		{
			name: "concurrent add and remove — different tags",
			ops: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},   // device A adds
				{Value: "label-1", Tag: "t2", Op: "remove"}, // device B removes (different tag)
			},
			want: []string{"label-1"}, // add tag t1 has no matching remove → label stays
		},
		{
			name: "multiple values",
			ops: []ORSetOp{
				{Value: "label-1", Tag: "t1", Op: "add"},
				{Value: "label-2", Tag: "t2", Op: "add"},
				{Value: "label-3", Tag: "t3", Op: "add"},
				{Value: "label-2", Tag: "t2", Op: "remove"},
			},
			want: []string{"label-1", "label-3"},
		},
		{
			name: "empty ops",
			ops:  nil,
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Materialize(tc.ops)
			if got == nil {
				got = []string{}
			}
			sort.Strings(got)
			sort.Strings(tc.want)

			if len(got) != len(tc.want) {
				t.Fatalf("Materialize() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("Materialize() = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestORSetIdempotentMerge(t *testing.T) {
	ops := []ORSetOp{
		{Value: "label-1", Tag: "t1", Op: "add"},
		{Value: "label-2", Tag: "t2", Op: "add"},
	}

	// Merging with itself should produce same result
	merged1 := MergeORSet(ops, ops)
	merged2 := MergeORSet(merged1, ops)

	result1 := Materialize(merged1)
	result2 := Materialize(merged2)
	sort.Strings(result1)
	sort.Strings(result2)

	if len(result1) != len(result2) {
		t.Fatalf("idempotent merge failed: %v vs %v", result1, result2)
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Fatalf("idempotent merge failed: %v vs %v", result1, result2)
		}
	}
}
