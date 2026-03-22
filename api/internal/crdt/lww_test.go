package crdt

import (
	"testing"
	"time"

	"github.com/vasi1796/doit/internal/hlc"
)

var base = time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

func TestMergeLWW(t *testing.T) {
	tests := []struct {
		name      string
		localVal  string
		localHLC  hlc.Timestamp
		remoteVal string
		remoteHLC hlc.Timestamp
		wantVal   string
	}{
		{
			name:      "remote later by time",
			localVal:  "local",
			localHLC:  hlc.Timestamp{Time: base, Counter: 0},
			remoteVal: "remote",
			remoteHLC: hlc.Timestamp{Time: base.Add(time.Second), Counter: 0},
			wantVal:   "remote",
		},
		{
			name:      "local later by time",
			localVal:  "local",
			localHLC:  hlc.Timestamp{Time: base.Add(time.Second), Counter: 0},
			remoteVal: "remote",
			remoteHLC: hlc.Timestamp{Time: base, Counter: 0},
			wantVal:   "local",
		},
		{
			name:      "same time, remote later by counter",
			localVal:  "local",
			localHLC:  hlc.Timestamp{Time: base, Counter: 1},
			remoteVal: "remote",
			remoteHLC: hlc.Timestamp{Time: base, Counter: 5},
			wantVal:   "remote",
		},
		{
			name:      "same time, local later by counter",
			localVal:  "local",
			localHLC:  hlc.Timestamp{Time: base, Counter: 5},
			remoteVal: "remote",
			remoteHLC: hlc.Timestamp{Time: base, Counter: 1},
			wantVal:   "local",
		},
		{
			name:      "identical timestamps — remote wins (deterministic tiebreak)",
			localVal:  "local",
			localHLC:  hlc.Timestamp{Time: base, Counter: 3},
			remoteVal: "remote",
			remoteHLC: hlc.Timestamp{Time: base, Counter: 3},
			wantVal:   "remote",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := MergeLWW(tc.localVal, tc.localHLC, tc.remoteVal, tc.remoteHLC)
			if got != tc.wantVal {
				t.Errorf("MergeLWW() = %q, want %q", got, tc.wantVal)
			}
		})
	}
}
