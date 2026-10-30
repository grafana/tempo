package main

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/util"
)

func TestTraceWriteTrackerLifecycle(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	info := util.NewTraceInfo(seed, "tenant")

	tests := []struct {
		name       string
		operations []string
		canCheck   []bool
		entries    int
	}{
		{
			name:       "successful write is pending until completion",
			operations: []string{"start", "finish"},
			canCheck:   []bool{false, true},
			entries:    0,
		},
		{
			name:       "failed write remains excluded after completion",
			operations: []string{"start", "fail", "finish"},
			canCheck:   []bool{false, false, false},
			entries:    1,
		},
		{
			name:       "success after failure does not clear failed outcome",
			operations: []string{"start", "fail", "finish", "start", "finish"},
			canCheck:   []bool{false, false, false, false, false},
			entries:    1,
		},
		{
			name:       "duplicate writers must both complete",
			operations: []string{"start", "start", "finish", "finish"},
			canCheck:   []bool{false, false, false, true},
			entries:    0,
		},
		{
			name:       "one failed duplicate writer excludes the trace",
			operations: []string{"start", "start", "fail", "finish", "finish"},
			canCheck:   []bool{false, false, false, false, false},
			entries:    1,
		},
		{
			name:       "finishing unknown trace does not create state",
			operations: []string{"finish"},
			canCheck:   []bool{true},
			entries:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := newTraceWriteTracker(time.Hour)
			require.True(t, tracker.canCheck("tenant", info), "unknown eligible traces must still be checked")
			for i, operation := range tt.operations {
				switch operation {
				case "start":
					tracker.startWrite("tenant", info)
				case "fail":
					tracker.failWrite("tenant", info)
				case "finish":
					tracker.finishWrite("tenant", info)
				}
				assert.Equal(t, tt.canCheck[i], tracker.canCheck("tenant", info), "after operation %d (%s)", i, operation)
			}
			assert.Len(t, tracker.writes, tt.entries)
		})
	}
}

func TestTraceWriteTrackerIdentity(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	info := util.NewTraceInfo(seed, "tenant-a")
	nextInfo := util.NewTraceInfo(seed.Add(time.Nanosecond), "tenant-a")
	require.NotEqual(t, info.HexID(), nextInfo.HexID())
	tracker := newTraceWriteTracker(time.Hour)

	tracker.startWrite("tenant-a", info)
	tracker.failWrite("tenant-a", info)
	tracker.finishWrite("tenant-a", info)

	assert.False(t, tracker.canCheck("tenant-a", info))
	assert.True(t, tracker.canCheck("tenant-b", info), "a different tenant can use the same trace ID")
	assert.True(t, tracker.canCheck("tenant-a", nextInfo), "different seeds in the same second are independent")
}

func TestTraceWriteTrackerPrune(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	age := time.Hour
	tracker := newTraceWriteTracker(age)
	expired := util.NewTraceInfo(seed.Add(-time.Nanosecond), "tenant")
	boundary := util.NewTraceInfo(seed, "tenant")
	newer := util.NewTraceInfo(seed.Add(time.Nanosecond), "tenant")
	for _, info := range []*util.TraceInfo{expired, boundary, newer} {
		tracker.startWrite("tenant", info)
		tracker.failWrite("tenant", info)
		tracker.finishWrite("tenant", info)
	}

	tracker.prune(seed.Add(age))
	assert.Len(t, tracker.writes, 2, "the exact boundary is retained")
	assert.False(t, tracker.canCheck("tenant", expired), "a stale ticker event must not validate an expired failure")
	assert.False(t, tracker.canCheck("tenant", boundary))
	assert.False(t, tracker.canCheck("tenant", newer))
	assert.True(t, tracker.canCheck("other-tenant", boundary), "an unknown trace at the exact boundary can be checked")

	// Earlier checker timestamps must never move the expiry floor backwards.
	tracker.prune(seed.Add(age - time.Second))
	assert.Equal(t, seed, tracker.expiresBefore)
	assert.False(t, tracker.canCheck("other-tenant", expired))

	tracker.prune(seed.Add(age + 2*time.Nanosecond))
	assert.Empty(t, tracker.writes)
	assert.False(t, tracker.canCheck("tenant", newer))
}

func TestTraceWriteTrackerIgnoresExpiredCallbacks(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	info := util.NewTraceInfo(seed, "tenant")
	tracker := newTraceWriteTracker(time.Hour)
	tracker.startWrite("tenant", info)
	tracker.prune(seed.Add(time.Hour + time.Nanosecond))
	require.Empty(t, tracker.writes)

	tracker.failWrite("tenant", info)
	tracker.finishWrite("tenant", info)
	tracker.startWrite("tenant", info)
	assert.Empty(t, tracker.writes, "a delayed callback must not recreate expired state")
	assert.False(t, tracker.canCheck("tenant", info))
}

func TestTraceWriteTrackerConcurrentAccess(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	info := util.NewTraceInfo(seed, "tenant")
	tracker := newTraceWriteTracker(time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(fail bool) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tracker.startWrite("tenant", info)
				if fail {
					tracker.failWrite("tenant", info)
				}
				tracker.canCheck("tenant", info)
				tracker.prune(seed.Add(time.Duration(j) * time.Second))
				tracker.finishWrite("tenant", info)
			}
		}(i%2 == 0)
	}
	wg.Wait()

	assert.False(t, tracker.canCheck("tenant", info))
	require.Len(t, tracker.writes, 1)
	assert.Zero(t, tracker.writes[traceWriteKey{tenant: "tenant", traceID: info.HexID()}].pending)
	tracker.prune(seed.Add(2 * time.Hour))
	assert.Empty(t, tracker.writes)
}
