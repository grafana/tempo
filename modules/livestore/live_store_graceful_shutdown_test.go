package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

// TestConsume_GracefulShutdown_ReturnsNilError verifies CRIT-1 fix:
// consume() should return (nil, nil) on context cancellation, not (nil, ctx.Err()).
// Graceful shutdown is not an error condition and should not be logged as ERROR.
func TestConsume_GracefulShutdown_ReturnsNilError(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup LiveStore
	store, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		store.StopAsync()
		store.AwaitTerminated(context.Background())
	}()

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create test records batch
	id := test.ValidTraceID(nil)
	expectedTrace := test.MakeTrace(5, id)
	traceBytes, err := proto.Marshal(expectedTrace)
	require.NoError(t, err)

	request := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
		Ids:    [][]byte{id},
	}
	records, err := ingest.Encode(0, testTenantID, request, 1_000_000)
	require.NoError(t, err)

	// Set timestamp so records are accepted
	now := time.Now()
	for _, kgoRec := range records {
		kgoRec.Timestamp = now
	}

	// Cancel context immediately to simulate shutdown during consume
	cancel()

	// Call consume with cancelled context
	offset, err := store.consume(ctx, createRecordIter(records), now)

	// ASSERT: Should return nil error on graceful shutdown (CRIT-1 fix)
	// Before fix: returns ctx.Err() which causes ERROR logs
	// After fix: returns nil (graceful shutdown is not an error)
	require.NoError(t, err, "graceful shutdown should not return error")

	// ASSERT: Should return nil offset (no records processed)
	require.Nil(t, offset, "graceful shutdown should not commit partial work")
}

// TestConsume_GracefulShutdown_MidBatch verifies that context cancellation
// during batch processing also returns nil error, not ctx.Err()
func TestConsume_GracefulShutdown_MidBatch(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup LiveStore
	store, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		store.StopAsync()
		store.AwaitTerminated(context.Background())
	}()

	// Create context that will be cancelled mid-processing
	ctx, cancel := context.WithCancel(context.Background())

	// Create multiple test records to increase processing time
	var allRecords []*kgo.Record
	for i := 0; i < 100; i++ {
		id := test.ValidTraceID(nil)
		expectedTrace := test.MakeTrace(5, id)
		traceBytes, err := proto.Marshal(expectedTrace)
		require.NoError(t, err)

		request := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
			Ids:    [][]byte{id},
		}
		records, err := ingest.Encode(0, testTenantID, request, 1_000_000)
		require.NoError(t, err)

		allRecords = append(allRecords, records...)
	}

	// Set timestamp so records are accepted
	now := time.Now()
	for _, kgoRec := range allRecords {
		kgoRec.Timestamp = now
	}

	// Cancel after short delay to simulate mid-batch cancellation
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Call consume - may process some records before cancellation
	offset, err := store.consume(ctx, createRecordIter(allRecords), now)

	// ASSERT: Should return nil error on graceful shutdown
	require.NoError(t, err, "graceful shutdown should not return error even mid-batch")

	// ASSERT: May return offset if records were processed before cancellation
	if offset != nil {
		t.Logf("Partial progress committed: offset=%d", offset.At)
	} else {
		t.Logf("No records processed before shutdown")
	}
}
