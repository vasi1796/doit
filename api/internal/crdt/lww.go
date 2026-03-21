// Package crdt provides merge functions for conflict-free replicated data types.
package crdt

import "github.com/vasi1796/doit/internal/hlc"

// MergeLWW returns the value with the later HLC timestamp (Last-Writer-Wins).
// On equal timestamps, remote wins for deterministic tiebreaking across devices.
func MergeLWW[T any](localVal T, localHLC hlc.Timestamp, remoteVal T, remoteHLC hlc.Timestamp) (T, hlc.Timestamp) {
	if hlc.Compare(remoteHLC, localHLC) >= 0 {
		return remoteVal, remoteHLC
	}
	return localVal, localHLC
}
