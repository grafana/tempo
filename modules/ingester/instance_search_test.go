package ingester

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/atomic"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/search"
)

func TestInstanceSearch(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir := t.TempDir()

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	var tagKey = "foo"
	var tagValue = "bar"
	ids, _ := writeTracesWithSearchData(t, i, tagKey, tagValue, false)

	var req = &tempopb.SearchRequest{
		Tags: map[string]string{},
	}
	req.Tags[tagKey] = tagValue

	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	// todo: test that returned results are in sorted time order, create order of id's beforehand
	checkEqual(t, ids, sr)

	// Test after appending to WAL
	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	assert.Equal(t, int(i.traceCount.Load()), len(i.traces))

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)

	// Test after cutting new headblock
	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)

	// Test after completing a block
	err = i.CompleteBlock(blockID)
	require.NoError(t, err)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)

	err = ingester.stopping(nil)
	require.NoError(t, err)

	// create new ingester.  this should replay wal!
	ingester, _, _ = defaultIngester(t, tempDir)

	i, ok := ingester.getInstanceByID("fake")
	assert.True(t, ok)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)
}

func checkEqual(t *testing.T, ids [][]byte, sr *tempopb.SearchResponse) {
	for _, meta := range sr.Traces {
		parsedTraceID, err := util.HexStringToTraceID(meta.TraceID)
		assert.NoError(t, err)

		present := false
		for _, id := range ids {
			if bytes.Equal(parsedTraceID, id) {
				present = true
			}
		}
		assert.True(t, present)
	}
}

func TestInstanceSearchTags(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir := t.TempDir()

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	// add dummy search data
	var tagKey = "foo"
	var tagValue = "bar"

	_, expectedTagValues := writeTracesWithSearchData(t, i, tagKey, tagValue, true)

	userCtx := user.InjectOrgID(context.Background(), "fake")
	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)

	// Test after appending to WAL
	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	assert.Equal(t, int(i.traceCount.Load()), len(i.traces))

	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)

	// Test after cutting new headblock
	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)

	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)

	// Test after completing a block
	err = i.CompleteBlock(blockID)
	require.NoError(t, err)

	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)
}

// nolint:revive,unparam
func testSearchTagsAndValues(t *testing.T, ctx context.Context, i *instance, tagName string, expectedTagValues []string) {
	sr, err := i.SearchTags(ctx)
	require.NoError(t, err)
	srv, err := i.SearchTagValues(ctx, tagName)
	require.NoError(t, err)

	sort.Strings(srv.TagValues)
	assert.Len(t, sr.TagNames, 1)
	assert.Equal(t, tagName, sr.TagNames[0])
	assert.Equal(t, expectedTagValues, srv.TagValues)
}

// TestInstanceSearchMaxBytesPerTagValuesQueryReturnsPartial confirms that SearchTagValues returns
//  partial results if the bytes of the found tag value exceeds the MaxBytesPerTagValuesQuery limit
func TestInstanceSearchMaxBytesPerTagValuesQueryReturnsPartial(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{
		MaxBytesPerTagValuesQuery: 10,
	})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir := t.TempDir()

	ingester, _, _ := defaultIngester(t, tempDir)
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	var tagKey = "foo"
	var tagValue = "bar"

	_, _ = writeTracesWithSearchData(t, i, tagKey, tagValue, true)

	userCtx := user.InjectOrgID(context.Background(), "fake")
	resp, err := i.SearchTagValues(userCtx, tagKey)
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.TagValues)) // Only two values of the form "bar123" fit in the 10 byte limit above.
}

// writes traces to the given instance along with search data. returns
//  ids expected to be returned from a tag search and strings expected to
//  be returned from a tag value search
func writeTracesWithSearchData(t *testing.T, i *instance, tagKey string, tagValue string, postFixValue bool) ([][]byte, []string) {
	// This matches the encoding for live traces, since
	// we are pushing to the instance directly it must match.
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	numTraces := 100
	searchAnnotatedFractionDenominator := 10
	ids := [][]byte{}
	expectedTagValues := []string{}

	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		rand.Read(id)

		testTrace := test.MakeTrace(10, id)
		trace.SortTrace(testTrace)
		traceBytes, err := dec.PrepareForWrite(testTrace, 0, 0)
		require.NoError(t, err)

		// annotate just a fraction of traces with search data
		var searchData []byte
		if j%searchAnnotatedFractionDenominator == 0 {
			tv := tagValue
			if postFixValue {
				tv = tv + strconv.Itoa(j)
			}

			data := &tempofb.SearchEntryMutable{}
			data.TraceID = id
			data.AddTag(tagKey, tv)
			searchData = data.ToBytes()

			expectedTagValues = append(expectedTagValues, tv)
			ids = append(ids, data.TraceID)
		}

		// searchData will be nil if not
		err = i.PushBytes(context.Background(), id, traceBytes, searchData)
		require.NoError(t, err)

		assert.Equal(t, int(i.traceCount.Load()), len(i.traces))
	}

	return ids, expectedTagValues
}

func TestInstanceSearchNoData(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	assert.NoError(t, err, "unexpected error creating new instance")

	var req = &tempopb.SearchRequest{
		Tags: map[string]string{},
	}

	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	require.Len(t, sr.Traces, 0)
}

func TestInstanceSearchDoesNotRace(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err)
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	require.NoError(t, err)

	// This matches the encoding for live traces, since
	// we are pushing to the instance directly it must match.
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// add dummy search data
	var tagKey = "foo"
	var tagValue = "bar"

	var req = &tempopb.SearchRequest{
		Tags: map[string]string{tagKey: tagValue},
	}

	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	go concurrent(func() {
		id := make([]byte, 16)
		rand.Read(id)

		trace := test.MakeTrace(10, id)
		traceBytes, err := dec.PrepareForWrite(trace, 0, 0)
		require.NoError(t, err)

		searchData := &tempofb.SearchEntryMutable{}
		searchData.TraceID = id
		searchData.AddTag(tagKey, tagValue)
		searchBytes := searchData.ToBytes()

		// searchData will be nil if not
		err = i.PushBytes(context.Background(), id, traceBytes, searchBytes)
		require.NoError(t, err)
	})

	go concurrent(func() {
		err := i.CutCompleteTraces(0, true)
		require.NoError(t, err, "error cutting complete traces")
	})

	go concurrent(func() {
		_, err := i.FindTraceByID(context.Background(), []byte{0x01})
		assert.NoError(t, err, "error finding trace by id")
	})

	go concurrent(func() {
		// Cut wal, complete, delete wal, then flush
		blockID, _ := i.CutBlockIfReady(0, 0, true)
		if blockID != uuid.Nil {
			err := i.CompleteBlock(blockID)
			require.NoError(t, err)
			err = i.ClearCompletingBlock(blockID)
			require.NoError(t, err)
			block := i.GetBlockToBeFlushed(blockID)
			require.NotNil(t, block)
			err = ingester.store.WriteBlock(context.Background(), block)
			require.NoError(t, err)
		}
	})

	go concurrent(func() {
		err = i.ClearFlushedBlocks(0)
		require.NoError(t, err)
	})

	go concurrent(func() {
		_, err := i.Search(context.Background(), req)
		require.NoError(t, err, "error finding trace by id")
	})

	go concurrent(func() {
		// SearchTags queries now require userID in ctx
		ctx := user.InjectOrgID(context.Background(), "test")
		_, err := i.SearchTags(ctx)
		require.NoError(t, err, "error getting search tags")
	})

	go concurrent(func() {
		// SearchTagValues queries now require userID in ctx
		ctx := user.InjectOrgID(context.Background(), "test")
		_, err := i.SearchTagValues(ctx, tagKey)
		require.NoError(t, err, "error getting search tag values")
	})

	time.Sleep(2000 * time.Millisecond)
	close(end)
	// Wait for go funcs to quit before
	// exiting and cleaning up
	time.Sleep(2 * time.Second)
}

func TestWALBlockDeletedDuringSearch(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err)
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	ingester, _, _ := defaultIngester(t, t.TempDir())
	i, err := newInstance("fake", limiter, ingester.store, ingester.local)
	require.NoError(t, err)

	// This matches the encoding for live traces, since
	// we are pushing to the instance directly it must match.
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	for j := 0; j < 500; j++ {
		id := make([]byte, 16)
		rand.Read(id)

		trace := test.MakeTrace(10, id)
		traceBytes, err := dec.PrepareForWrite(trace, 0, 0)
		require.NoError(t, err)

		entry := &tempofb.SearchEntryMutable{}
		entry.TraceID = id
		entry.AddTag("foo", "bar")
		searchBytes := entry.ToBytes()

		err = i.PushBytes(context.Background(), id, traceBytes, searchBytes)
		require.NoError(t, err)
	}

	err = i.CutCompleteTraces(0, true)
	require.NoError(t, err)

	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)

	go concurrent(func() {
		_, err := i.Search(context.Background(), &tempopb.SearchRequest{
			Tags: map[string]string{
				// Not present in the data, so it will be an exhaustive
				// search
				"wuv": "xyz",
			},
		})
		require.NoError(t, err)
	})

	// Let search get going
	time.Sleep(100 * time.Millisecond)

	err = i.ClearCompletingBlock(blockID)
	require.NoError(t, err)

	// Wait for go funcs to quit before
	// exiting and cleaning up
	close(end)
	time.Sleep(2 * time.Second)
}

func TestInstanceSearchMetrics(t *testing.T) {

	i := defaultInstance(t, t.TempDir())

	// This matches the encoding for live traces, since
	// we are pushing to the instance directly it must match.
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	numTraces := uint32(500)
	numBytes := uint64(0)
	for j := uint32(0); j < numTraces; j++ {
		id := test.ValidTraceID(nil)

		// Trace bytes have to be pushed in the expected data encoding
		trace := test.MakeTrace(10, id)

		traceBytes, err := dec.PrepareForWrite(trace, 0, 0)
		require.NoError(t, err)

		data := &tempofb.SearchEntryMutable{}
		data.TraceID = id
		data.AddTag("foo", "bar")
		searchData := data.ToBytes()

		numBytes += uint64(len(searchData))

		err = i.PushBytes(context.Background(), id, traceBytes, searchData)
		require.NoError(t, err)

		assert.Equal(t, int(i.traceCount.Load()), len(i.traces))
	}

	search := func() *tempopb.SearchMetrics {
		sr, err := i.Search(context.Background(), &tempopb.SearchRequest{
			// Exhaustive search
			Tags: map[string]string{search.SecretExhaustiveSearchTag: "!"},
		})
		require.NoError(t, err)
		return sr.Metrics
	}

	// Live traces
	m := search()
	require.Equal(t, numTraces, m.InspectedTraces)
	require.Equal(t, numBytes, m.InspectedBytes)
	require.Equal(t, uint32(1), m.InspectedBlocks) // 1 head block

	// Test after appending to WAL
	err := i.CutCompleteTraces(0, true)
	require.NoError(t, err)
	m = search()
	require.Equal(t, numTraces, m.InspectedTraces)
	require.Equal(t, numBytes, m.InspectedBytes)
	require.Equal(t, uint32(1), m.InspectedBlocks) // 1 head block

	// Test after cutting new headblock
	blockID, err := i.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)
	m = search()
	require.Equal(t, numTraces, m.InspectedTraces)
	require.Equal(t, numBytes, m.InspectedBytes)
	require.Equal(t, uint32(2), m.InspectedBlocks) // 1 head block, 1 completing block

	// Test after completing a block
	err = i.CompleteBlock(blockID)
	require.NoError(t, err)
	err = i.ClearCompletingBlock(blockID)
	require.NoError(t, err)
	// Complete blocks are paged and search data is normalized, therefore smaller than individual wal entries.
	m = search()
	require.Equal(t, numTraces, m.InspectedTraces)
	require.Less(t, m.InspectedBytes, numBytes)
	require.Equal(t, uint32(2), m.InspectedBlocks) // 1 head block, 1 complete block
}

func BenchmarkInstanceSearchUnderLoad(b *testing.B) {
	ctx := context.TODO()

	i := defaultInstance(b, b.TempDir())

	// This matches the encoding for live traces, since
	// we are pushing to the instance directly it must match.
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	// Push data
	var tracesPushed atomic.Int32
	for j := 0; j < 2; j++ {
		go concurrent(func() {
			id := test.ValidTraceID(nil)

			trace := test.MakeTrace(10, id)
			traceBytes, err := dec.PrepareForWrite(trace, 0, 0)
			require.NoError(b, err)

			searchData := &tempofb.SearchEntryMutable{}
			searchData.TraceID = id
			searchData.AddTag("foo", "bar")
			searchData.AddTag("foo", "baz")
			searchData.AddTag("bar", "bar")
			searchData.AddTag("bar", "baz")
			searchBytes := searchData.ToBytes()

			// searchData will be nil if not
			err = i.PushBytes(context.Background(), id, traceBytes, searchBytes)
			require.NoError(b, err)

			tracesPushed.Inc()
		})
	}

	cuts := 0
	go concurrent(func() {
		time.Sleep(250 * time.Millisecond)
		err := i.CutCompleteTraces(0, true)
		require.NoError(b, err, "error cutting complete traces")
		cuts++
	})

	go concurrent(func() {
		// Slow this down to prevent "too many open files" error
		time.Sleep(100 * time.Millisecond)
		_, err := i.CutBlockIfReady(0, 0, true)
		require.NoError(b, err)
	})

	var searches atomic.Int32
	var bytesInspected atomic.Uint64
	var tracesInspected atomic.Uint32

	for j := 0; j < 2; j++ {
		go concurrent(func() {
			//time.Sleep(1 * time.Millisecond)
			var req = &tempopb.SearchRequest{
				Tags: map[string]string{search.SecretExhaustiveSearchTag: "!"},
			}
			resp, err := i.Search(ctx, req)
			require.NoError(b, err)
			searches.Inc()
			bytesInspected.Add(resp.Metrics.InspectedBytes)
			tracesInspected.Add(resp.Metrics.InspectedTraces)
		})
	}

	b.ResetTimer()
	start := time.Now()
	time.Sleep(time.Duration(b.N) * time.Millisecond)
	elapsed := time.Since(start)

	fmt.Printf("Instance search throughput under load: %v elapsed %.2f MB = %.2f MiB/s throughput inspected %.2f traces/s pushed %.2f traces/s %.2f searches/s %.2f cuts/s\n",
		elapsed,
		float64(bytesInspected.Load())/(1024*1024),
		float64(bytesInspected.Load())/(elapsed.Seconds())/(1024*1024),
		float64(tracesInspected.Load())/(elapsed.Seconds()),
		float64(tracesPushed.Load())/(elapsed.Seconds()),
		float64(searches.Load())/(elapsed.Seconds()),
		float64(cuts)/(elapsed.Seconds()),
	)

	b.StopTimer()
	close(end)
	// Wait for go funcs to quit before
	// exiting and cleaning up
	time.Sleep(1 * time.Second)
}
