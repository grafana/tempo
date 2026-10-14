package main

import (
	"sync"
	"time"

	"github.com/grafana/tempo/v3/pkg/util"
)

type traceWriteKey struct {
	tenant  string
	traceID string
}

type traceWriteState struct {
	timestamp time.Time
	pending   int
	failed    bool
}

// traceWriteTracker remembers incomplete or failed writes for the checker loops.
// Successful writes need no entry, so unknown traces within the sampling window
// remain eligible for validation.
type traceWriteTracker struct {
	mu            sync.Mutex
	writes        map[traceWriteKey]traceWriteState
	maxAge        time.Duration
	expiresBefore time.Time
}

func newTraceWriteTracker(maxAge time.Duration) *traceWriteTracker {
	return &traceWriteTracker{
		writes: make(map[traceWriteKey]traceWriteState),
		maxAge: maxAge,
	}
}

func (t *traceWriteTracker) startWrite(tenant string, info *util.TraceInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info.Timestamp().Before(t.expiresBefore) {
		return
	}

	key := traceWriteKey{tenant: tenant, traceID: info.HexID()}
	state := t.writes[key]
	state.timestamp = info.Timestamp()
	state.pending++
	t.writes[key] = state
}

func (t *traceWriteTracker) failWrite(tenant string, info *util.TraceInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info.Timestamp().Before(t.expiresBefore) {
		return
	}

	key := traceWriteKey{tenant: tenant, traceID: info.HexID()}
	state := t.writes[key]
	state.timestamp = info.Timestamp()
	state.failed = true
	t.writes[key] = state
}

func (t *traceWriteTracker) finishWrite(tenant string, info *util.TraceInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info.Timestamp().Before(t.expiresBefore) {
		return
	}

	key := traceWriteKey{tenant: tenant, traceID: info.HexID()}
	state, ok := t.writes[key]
	if !ok {
		return
	}
	if state.pending > 0 {
		state.pending--
	}
	if state.pending == 0 && !state.failed {
		delete(t.writes, key)
		return
	}
	t.writes[key] = state
}

func (t *traceWriteTracker) canCheck(tenant string, info *util.TraceInfo) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// A checker can hold a stale ticker event while an earlier request completes.
	// Skip seeds whose write outcome has already been forgotten by pruning.
	if info.Timestamp().Before(t.expiresBefore) {
		return false
	}

	state := t.writes[traceWriteKey{tenant: tenant, traceID: info.HexID()}]
	return state.pending == 0 && !state.failed
}

func (t *traceWriteTracker) prune(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	expiresBefore := now.Add(-t.maxAge)
	if !expiresBefore.After(t.expiresBefore) {
		return
	}
	t.expiresBefore = expiresBefore
	for key, state := range t.writes {
		if state.timestamp.Before(expiresBefore) {
			delete(t.writes, key)
		}
	}
}
