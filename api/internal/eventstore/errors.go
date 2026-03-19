package eventstore

import "errors"

// ErrVersionConflict is returned when an append fails due to an
// optimistic concurrency violation (duplicate aggregate_id + version).
var ErrVersionConflict = errors.New("eventstore: version conflict")

// ErrNoEvents is returned when Append is called with an empty slice.
var ErrNoEvents = errors.New("eventstore: no events to append")
