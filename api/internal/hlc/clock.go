// Package hlc implements a Hybrid Logical Clock for causal ordering.
//
// An HLC timestamp combines a physical wall-clock time with a logical counter.
// This provides a total ordering that respects causality (if A happens-before B
// then HLC(A) < HLC(B)) while staying close to wall-clock time.
//
// Reference: Kulkarni et al., "Logical Physical Clocks and Consistent Snapshots
// in Globally Distributed Databases" (2014).
package hlc

import (
	"sync"
	"time"
)

// Timestamp is an HLC timestamp: a physical time paired with a logical counter.
// Comparison is lexicographic: later physical time wins; if equal, higher counter wins.
type Timestamp struct {
	Time    time.Time `json:"time"`
	Counter int       `json:"counter"`
}

// Zero returns the zero-value HLC timestamp.
func Zero() Timestamp {
	return Timestamp{}
}

// Compare returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b Timestamp) int {
	switch {
	case a.Time.Before(b.Time):
		return -1
	case a.Time.After(b.Time):
		return 1
	case a.Counter < b.Counter:
		return -1
	case a.Counter > b.Counter:
		return 1
	default:
		return 0
	}
}

// Clock is a thread-safe Hybrid Logical Clock.
type Clock struct {
	mu     sync.Mutex
	now    func() time.Time
	latest Timestamp
}

// New creates a new HLC clock using wall-clock time.
func New() *Clock {
	return &Clock{now: func() time.Time { return time.Now().UTC() }}
}

// NewWithSource creates a clock with an injectable time source (for testing).
func NewWithSource(now func() time.Time) *Clock {
	return &Clock{now: now}
}

// Now generates a new HLC timestamp for a local event.
func (c *Clock) Now() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()

	pt := c.now()

	if pt.After(c.latest.Time) {
		c.latest = Timestamp{Time: pt, Counter: 0}
	} else {
		c.latest.Counter++
	}

	return c.latest
}

// Update merges a remote HLC timestamp with the local clock state.
// Used when receiving events from another device/server.
func (c *Clock) Update(remote Timestamp) Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()

	pt := c.now()
	prev := c.latest

	switch {
	case pt.After(prev.Time) && pt.After(remote.Time):
		// Wall clock is ahead of both — reset counter
		c.latest = Timestamp{Time: pt, Counter: 0}
	case prev.Time.After(remote.Time):
		// Local is ahead of remote — increment local counter
		c.latest = Timestamp{Time: prev.Time, Counter: prev.Counter + 1}
	case remote.Time.After(prev.Time):
		// Remote is ahead of local — adopt remote, increment counter
		c.latest = Timestamp{Time: remote.Time, Counter: remote.Counter + 1}
	default:
		// Same physical time on all three — take max counter + 1
		maxC := prev.Counter
		if remote.Counter > maxC {
			maxC = remote.Counter
		}
		c.latest = Timestamp{Time: prev.Time, Counter: maxC + 1}
	}

	return c.latest
}
