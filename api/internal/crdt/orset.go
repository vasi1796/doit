package crdt

// OpType represents the type of operation in an OR-Set.
type OpType string

const (
	OpAdd    OpType = "add"
	OpRemove OpType = "remove"
)

// ORSetOp represents a single add or remove operation in an Observed-Remove Set.
// Each operation carries a unique tag so that concurrent add/remove of the same
// value can be resolved correctly.
type ORSetOp struct {
	Value string `json:"value"`
	Tag   string `json:"tag"` // unique per operation (UUID)
	Op    OpType `json:"op"`  // OpAdd or OpRemove
}

// MergeORSet merges two OR-Set operation logs by taking their union.
// Duplicate operations (same tag) are deduplicated.
func MergeORSet(local, remote []ORSetOp) []ORSetOp {
	// Deduplicate by (tag, op) pair — the same tag can appear as both add and remove
	type opKey struct {
		tag string
		op  OpType
	}
	seen := make(map[opKey]bool, len(local)+len(remote))
	var merged []ORSetOp

	for _, op := range local {
		k := opKey{op.Tag, op.Op}
		if !seen[k] {
			seen[k] = true
			merged = append(merged, op)
		}
	}
	for _, op := range remote {
		k := opKey{op.Tag, op.Op}
		if !seen[k] {
			seen[k] = true
			merged = append(merged, op)
		}
	}

	return merged
}

// Materialize computes the current set members from an OR-Set operation log.
// A value is present if it has at least one "add" tag that is not cancelled
// by a corresponding "remove" with the same tag.
func Materialize(ops []ORSetOp) []string {
	// Collect all add tags and remove tags per value
	type tagState struct {
		addTags    map[string]bool
		removeTags map[string]bool
	}
	byValue := make(map[string]*tagState)

	for _, op := range ops {
		ts, ok := byValue[op.Value]
		if !ok {
			ts = &tagState{addTags: make(map[string]bool), removeTags: make(map[string]bool)}
			byValue[op.Value] = ts
		}
		if op.Op == OpAdd {
			ts.addTags[op.Tag] = true
		} else {
			ts.removeTags[op.Tag] = true
		}
	}

	// A value is in the set if any add tag is not removed
	var result []string
	for value, ts := range byValue {
		for addTag := range ts.addTags {
			if !ts.removeTags[addTag] {
				result = append(result, value)
				break
			}
		}
	}

	return result
}
