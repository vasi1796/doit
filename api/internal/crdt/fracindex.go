package crdt

// Fractional indexing generates position strings that sort lexicographically.
// Used for task and subtask ordering without reindexing.
//
// Positions use lowercase ASCII letters (a-z) as a base-26 system.
// Between "a" and "c" → "b". Between "a" and "b" → "an" (midpoint).

const minChar = 'a'
const maxChar = 'z'

// First returns a position before all existing items.
func First() string {
	return "a"
}

// Last returns a position after all existing items.
func Last() string {
	return "z"
}

// Between generates a position string that sorts between before and after.
// If before is empty, generates a position before after.
// If after is empty, generates a position after before.
func Between(before, after string) string {
	if before == "" {
		before = string(minChar)
	}
	if after == "" {
		after = string(maxChar)
	}

	// Pad shorter string with minChar to equal length
	maxLen := len(before)
	if len(after) > maxLen {
		maxLen = len(after)
	}

	bPadded := padRight(before, maxLen)
	aPadded := padRight(after, maxLen)

	// Find midpoint character by character
	for i := 0; i < maxLen; i++ {
		bChar := bPadded[i]
		aChar := aPadded[i]

		if bChar < aChar {
			mid := bChar + (aChar-bChar)/2
			if mid > bChar {
				return before[:min(i, len(before))] + string(rune(mid))
			}
		}
	}

	// No room between — append a middle character after `before`
	return before + string(rune(minChar+(maxChar-minChar)/2))
}

func padRight(s string, length int) []byte {
	padded := make([]byte, length)
	copy(padded, s)
	for i := len(s); i < length; i++ {
		padded[i] = minChar
	}
	return padded
}

