package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	jaeger "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/grafana/tempo/v3/pkg/util"
)

type emitBatchFunc func(context.Context, *jaeger.Batch) error

var _ util.JaegerClient = emitBatchFunc(nil)

func (f emitBatchFunc) EmitBatch(ctx context.Context, batch *jaeger.Batch) error {
	return f(ctx, batch)
}

func TestRunCheckerSkipsFailedWrite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		config := vultureConfiguration{
			tempoOrgID:                    "test",
			tempoWriteBackoffDuration:     time.Second,
			tempoLongWriteBackoffDuration: time.Second,
			tempoRetentionDuration:        time.Hour,
		}
		seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for util.NewTraceInfo(seed, config.tempoOrgID).LongWritesRemaining() != 0 {
			seed = seed.Add(time.Second)
		}
		var attempts int
		client := emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
			attempts++
			return errors.New("write failed")
		})
		writes := newTraceWriteTracker(time.Hour)
		writeTicks := make(chan time.Time, 1)
		doWrite(client, &time.Ticker{C: writeTicks}, time.Second, config, writes, zap.NewNop())
		writeTicks <- seed
		close(writeTicks)
		synctest.Wait()
		require.Equal(t, 1, attempts)

		var checked atomic.Int32
		checkTicks := make(chan time.Time, 1)
		runChecker(&time.Ticker{C: checkTicks}, config, func(_ time.Time) (time.Time, time.Time, bool) {
			return seed.Add(-time.Hour), seed, false
		}, func(_ *util.TraceInfo, _ *zap.Logger) {
			checked.Add(1)
		}, writes, zap.NewNop())
		checkTicks <- seed.Add(time.Minute)
		close(checkTicks)
		synctest.Wait()

		require.Zero(t, checked.Load(), "failed writes must not trigger trace validation")
	})
}

func TestWriteTraceValidation(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config := vultureConfiguration{
		tempoOrgID:                    "tenant",
		tempoWriteBackoffDuration:     time.Second,
		tempoLongWriteBackoffDuration: 0,
		tempoRetentionDuration:        time.Hour,
	}
	var initialBatches int
	probe := util.NewTraceInfoWithMaxLongWrites(seed, 0, config.tempoOrgID)
	require.NoError(t, probe.EmitBatches(context.Background(), emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
		initialBatches++
		return nil
	})))
	require.Greater(t, initialBatches, 1, "the partial-write test needs multiple batches")

	tests := []struct {
		name         string
		longWrites   int64
		failAt       int
		wantChecked  bool
		wantAttempts int
	}{
		{name: "successful initial write", wantChecked: true, wantAttempts: initialBatches},
		{name: "initial write failure", failAt: 1, wantAttempts: 1},
		{name: "partially written initial trace", failAt: 2, wantAttempts: 2},
		{name: "successful delayed writes", longWrites: 2, wantChecked: true},
		{name: "delayed write failure", longWrites: 1, failAt: initialBatches + 1},
		{name: "later success does not undo delayed failure", longWrites: 2, failAt: initialBatches + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				writes := newTraceWriteTracker(time.Hour)
				info := util.NewTraceInfoWithMaxLongWrites(seed, tt.longWrites, config.tempoOrgID)
				var attempts atomic.Int32
				client := emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
					if int(attempts.Add(1)) == tt.failAt {
						return errors.New("batch write failed")
					}
					return nil
				})
				writeTrace(context.Background(), client, info, config, writes, zap.NewNop())
				synctest.Wait()
				if tt.wantAttempts != 0 {
					require.Equal(t, tt.wantAttempts, int(attempts.Load()))
				}
				if tt.longWrites == 2 && tt.failAt != 0 {
					require.Greater(t, int(attempts.Load()), tt.failAt, "a successful later send must not clear the failure")
				}
				var checked int
				checkTrace(seed.Add(time.Minute), config, func(_ time.Time) (time.Time, time.Time, bool) {
					return seed.Add(-time.Hour), seed, false
				}, func(_ *util.TraceInfo, _ *zap.Logger) {
					checked++
				}, writes, zap.NewNop())
				if tt.wantChecked {
					require.Equal(t, 1, checked)
				} else {
					require.Zero(t, checked)
				}
			})
		})
	}
}

func TestCheckTraceSkipsPendingWrite(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config := vultureConfiguration{
		tempoOrgID:                    "tenant",
		tempoWriteBackoffDuration:     time.Second,
		tempoLongWriteBackoffDuration: 0,
	}
	tests := []struct {
		name       string
		longWrites int64
		fail       bool
	}{
		{name: "slow initial success"},
		{name: "slow initial failure", fail: true},
		{name: "slow delayed success", longWrites: 1},
		{name: "slow delayed failure", longWrites: 1, fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				info := util.NewTraceInfoWithMaxLongWrites(seed, tt.longWrites, config.tempoOrgID)
				var initialBatches int
				probe := util.NewTraceInfoWithMaxLongWrites(seed, 0, config.tempoOrgID)
				require.NoError(t, probe.EmitBatches(context.Background(), emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
					initialBatches++
					return nil
				})))
				blockAt := 1
				if tt.longWrites != 0 {
					blockAt = initialBatches + 1
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered := make(chan struct{})
				release := make(chan struct{})
				var attempts atomic.Int32
				client := emitBatchFunc(func(ctx context.Context, _ *jaeger.Batch) error {
					if int(attempts.Add(1)) != blockAt {
						return nil
					}
					close(entered)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-release:
					}
					if tt.fail {
						return errors.New("slow write failed")
					}
					return nil
				})
				writes := newTraceWriteTracker(time.Hour)
				go writeTrace(ctx, client, info, config, writes, zap.NewNop())
				select {
				case <-entered:
				case <-time.After(5 * time.Second):
					t.Fatal("write did not reach the blocking batch")
				}
				var checked int
				selector := func(_ time.Time) (time.Time, time.Time, bool) {
					return seed.Add(-time.Hour), seed, false
				}
				checker := func(_ *util.TraceInfo, _ *zap.Logger) { checked++ }
				checkTrace(seed.Add(time.Minute), config, selector, checker, writes, zap.NewNop())
				require.Zero(t, checked, "elapsed readiness time must not make an in-flight write eligible")
				close(release)
				synctest.Wait()
				checkTrace(seed.Add(time.Minute), config, selector, checker, writes, zap.NewNop())
				if tt.fail {
					require.Zero(t, checked)
				} else {
					require.Equal(t, 1, checked)
				}
			})
		})
	}
}

func TestSuccessfulWriteStillReportsReadFailure(t *testing.T) {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config := vultureConfiguration{
		tempoOrgID:                    "tenant",
		tempoWriteBackoffDuration:     time.Second,
		tempoLongWriteBackoffDuration: 0,
	}
	writes := newTraceWriteTracker(time.Hour)
	info := util.NewTraceInfoWithMaxLongWrites(seed, 0, config.tempoOrgID)
	writeTrace(context.Background(), emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
		return nil
	}), info, config, writes, zap.NewNop())
	client := &MockHTTPClient{err: util.ErrTraceNotFound}
	var metrics traceMetrics
	var queryErr error
	checkTrace(seed.Add(time.Minute), config, func(_ time.Time) (time.Time, time.Time, bool) {
		return seed.Add(-time.Hour), seed, false
	}, func(info *util.TraceInfo, l *zap.Logger) {
		metrics, queryErr = queryTrace(client, info, l)
	}, writes, zap.NewNop())
	require.ErrorIs(t, queryErr, util.ErrTraceNotFound)
	require.Equal(t, 1, client.GetRequestsCount())
	require.Equal(t, traceMetrics{requested: 1, notFoundByID: 1}, metrics)
}

func TestCancelledDelayedWriteRemainsExcluded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		info := util.NewTraceInfoWithMaxLongWrites(seed, 1, "tenant")
		writes := newTraceWriteTracker(time.Hour)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		config := vultureConfiguration{tempoOrgID: "tenant", tempoLongWriteBackoffDuration: time.Hour}
		writeTrace(ctx, emitBatchFunc(func(_ context.Context, _ *jaeger.Batch) error {
			return nil
		}), info, config, writes, zap.NewNop())
		cancel()
		synctest.Wait()
		require.False(t, writes.canCheck(config.tempoOrgID, info))
	})
}
