package hlc

import (
	"sync"
	"testing"
	"time"
)

var baseTime = time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b Timestamp
		want int
	}{
		{
			name: "a before b by time",
			a:    Timestamp{Time: baseTime, Counter: 0},
			b:    Timestamp{Time: baseTime.Add(time.Second), Counter: 0},
			want: -1,
		},
		{
			name: "a after b by time",
			a:    Timestamp{Time: baseTime.Add(time.Second), Counter: 0},
			b:    Timestamp{Time: baseTime, Counter: 0},
			want: 1,
		},
		{
			name: "same time, a before b by counter",
			a:    Timestamp{Time: baseTime, Counter: 1},
			b:    Timestamp{Time: baseTime, Counter: 3},
			want: -1,
		},
		{
			name: "same time, a after b by counter",
			a:    Timestamp{Time: baseTime, Counter: 5},
			b:    Timestamp{Time: baseTime, Counter: 2},
			want: 1,
		},
		{
			name: "equal",
			a:    Timestamp{Time: baseTime, Counter: 7},
			b:    Timestamp{Time: baseTime, Counter: 7},
			want: 0,
		},
		{
			name: "time wins over counter",
			a:    Timestamp{Time: baseTime.Add(time.Second), Counter: 0},
			b:    Timestamp{Time: baseTime, Counter: 999},
			want: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Compare(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("Compare(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestClockNow(t *testing.T) {
	tests := []struct {
		name        string
		wallTimes   []time.Time
		wantTimes   []time.Time
		wantCounters []int
	}{
		{
			name:         "advancing wall clock resets counter",
			wallTimes:    []time.Time{baseTime, baseTime.Add(time.Second), baseTime.Add(2 * time.Second)},
			wantTimes:    []time.Time{baseTime, baseTime.Add(time.Second), baseTime.Add(2 * time.Second)},
			wantCounters: []int{0, 0, 0},
		},
		{
			name:         "same wall clock increments counter",
			wallTimes:    []time.Time{baseTime, baseTime, baseTime},
			wantTimes:    []time.Time{baseTime, baseTime, baseTime},
			wantCounters: []int{0, 1, 2},
		},
		{
			name:         "clock never goes backward",
			wallTimes:    []time.Time{baseTime.Add(time.Second), baseTime, baseTime},
			wantTimes:    []time.Time{baseTime.Add(time.Second), baseTime.Add(time.Second), baseTime.Add(time.Second)},
			wantCounters: []int{0, 1, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idx := 0
			c := NewWithSource(func() time.Time {
				t := tc.wallTimes[idx]
				idx++
				return t
			})

			for i := range tc.wallTimes {
				ts := c.Now()
				if !ts.Time.Equal(tc.wantTimes[i]) {
					t.Errorf("call %d: time = %v, want %v", i, ts.Time, tc.wantTimes[i])
				}
				if ts.Counter != tc.wantCounters[i] {
					t.Errorf("call %d: counter = %d, want %d", i, ts.Counter, tc.wantCounters[i])
				}
			}
		})
	}
}

func TestClockUpdate(t *testing.T) {
	tests := []struct {
		name        string
		wallTime    time.Time
		localState  Timestamp
		remote      Timestamp
		wantTime    time.Time
		wantCounter int
	}{
		{
			name:        "wall clock ahead of both",
			wallTime:    baseTime.Add(10 * time.Second),
			localState:  Timestamp{Time: baseTime, Counter: 5},
			remote:      Timestamp{Time: baseTime.Add(5 * time.Second), Counter: 3},
			wantTime:    baseTime.Add(10 * time.Second),
			wantCounter: 0,
		},
		{
			name:        "local ahead of remote and wall",
			wallTime:    baseTime,
			localState:  Timestamp{Time: baseTime.Add(5 * time.Second), Counter: 3},
			remote:      Timestamp{Time: baseTime.Add(2 * time.Second), Counter: 7},
			wantTime:    baseTime.Add(5 * time.Second),
			wantCounter: 4,
		},
		{
			name:        "remote ahead of local and wall",
			wallTime:    baseTime,
			localState:  Timestamp{Time: baseTime.Add(2 * time.Second), Counter: 7},
			remote:      Timestamp{Time: baseTime.Add(5 * time.Second), Counter: 3},
			wantTime:    baseTime.Add(5 * time.Second),
			wantCounter: 4,
		},
		{
			name:        "all three same time — max counter plus one",
			wallTime:    baseTime,
			localState:  Timestamp{Time: baseTime, Counter: 5},
			remote:      Timestamp{Time: baseTime, Counter: 8},
			wantTime:    baseTime,
			wantCounter: 9,
		},
		{
			name:        "all three same time — local counter higher",
			wallTime:    baseTime,
			localState:  Timestamp{Time: baseTime, Counter: 10},
			remote:      Timestamp{Time: baseTime, Counter: 3},
			wantTime:    baseTime,
			wantCounter: 11,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewWithSource(func() time.Time { return tc.wallTime })
			c.latest = tc.localState

			ts := c.Update(tc.remote)
			if !ts.Time.Equal(tc.wantTime) {
				t.Errorf("time = %v, want %v", ts.Time, tc.wantTime)
			}
			if ts.Counter != tc.wantCounter {
				t.Errorf("counter = %d, want %d", ts.Counter, tc.wantCounter)
			}
		})
	}
}

func TestClockMonotonicity(t *testing.T) {
	c := New()
	prev := c.Now()

	for i := 0; i < 1000; i++ {
		curr := c.Now()
		if Compare(curr, prev) <= 0 {
			t.Fatalf("iteration %d: %v not after %v", i, curr, prev)
		}
		prev = curr
	}
}

func TestClockConcurrency(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	results := make([]Timestamp, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = c.Now()
		}(i)
	}

	wg.Wait()

	// All timestamps must be unique
	seen := make(map[Timestamp]bool)
	for _, ts := range results {
		if seen[ts] {
			t.Fatalf("duplicate timestamp: %v", ts)
		}
		seen[ts] = true
	}
}
