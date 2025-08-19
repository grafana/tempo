/*
livestore instance_search_test is mostly based on the tests in ingest.
*/
package livestore

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	foo          = "foo"
	bar          = "bar"
	qux          = "qux"
	testTenantID = "fake"
)

func TestInstanceSearch(t *testing.T) {
	i, _ := defaultInstanceAndTmpDir(t)

	tagKey := foo
	tagValue := bar
	ids, _, _, _ := writeTracesForSearch(t, i, "", tagKey, tagValue, false, false)

	req := &tempopb.SearchRequest{
		Query: fmt.Sprintf(`{ span.%s = "%s" }`, tagKey, tagValue),
	}
	req.Limit = uint32(len(ids)) + 1

	// Test after appending to WAL. writeTracesforSearch() makes sure all traces are in the wal
	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)

	// Test after cutting new headblock
	blockID, err := i.cutBlocks(true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)

	// Test after completing a block
	err = i.completeBlock(context.Background(), blockID)
	require.NoError(t, err)

	sr, err = i.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, sr.Traces, len(ids))
	checkEqual(t, ids, sr)
}

// TestInstanceSearchTraceQL is duplicate of TestInstanceSearch for now
func TestInstanceSearchTraceQL(t *testing.T) {
	queries := []string{
		`{ .service.name = "test-service" }`,
		`{ duration >= 1s }`,
		`{ duration >= 1s && .service.name = "test-service" }`,
	}

	for _, query := range queries {
		t.Run(fmt.Sprintf("Query:%s", query), func(t *testing.T) {
			i, _ := defaultInstanceAndTmpDir(t)

			_, ids := pushTracesToInstance(t, i, 10)

			req := &tempopb.SearchRequest{Query: query, Limit: 20, SpansPerSpanSet: 10}

			// Test live traces, these are cut roughly every 5 seconds so these should
			// not exist yet.
			sr, err := i.Search(context.Background(), req)
			assert.NoError(t, err)
			assert.Len(t, sr.Traces, 0)

			// Test after appending to WAL
			require.NoError(t, i.cutIdleTraces(true))

			sr, err = i.Search(context.Background(), req)
			assert.NoError(t, err)
			assert.Len(t, sr.Traces, len(ids))
			checkEqual(t, ids, sr)

			// Test after cutting new headBlock
			blockID, err := i.cutBlocks(true)
			require.NoError(t, err)
			assert.NotEqual(t, blockID, uuid.Nil)

			sr, err = i.Search(context.Background(), req)
			assert.NoError(t, err)
			assert.Len(t, sr.Traces, len(ids))
			checkEqual(t, ids, sr)

			// Test after completing a block
			err = i.completeBlock(context.Background(), blockID)
			require.NoError(t, err)

			sr, err = i.Search(context.Background(), req)
			assert.NoError(t, err)
			assert.Len(t, sr.Traces, len(ids))
			checkEqual(t, ids, sr)
		})
	}
}

func TestInstanceSearchWithStartAndEnd(t *testing.T) {
	i, _ := defaultInstanceAndTmpDir(t)

	tagKey := foo
	tagValue := bar
	ids, _, _, _ := writeTracesForSearch(t, i, "", tagKey, tagValue, false, false)

	search := func(req *tempopb.SearchRequest, start, end uint32) *tempopb.SearchResponse {
		req.Start = start
		req.End = end
		sr, err := i.Search(context.Background(), req)
		assert.NoError(t, err)
		return sr
	}

	searchAndAssert := func(req *tempopb.SearchRequest, _ uint32) {
		sr := search(req, 0, 0)
		assert.Len(t, sr.Traces, len(ids))
		checkEqual(t, ids, sr)

		// writeTracesForSearch will build spans that end 1 second from now
		// query 2 min range to have extra slack and always be within range
		sr = search(req, uint32(time.Now().Add(-5*time.Minute).Unix()), uint32(time.Now().Add(5*time.Minute).Unix()))
		assert.Len(t, sr.Traces, len(ids))
		checkEqual(t, ids, sr)

		// search with start=5m from now, end=10m from now
		sr = search(req, uint32(time.Now().Add(5*time.Minute).Unix()), uint32(time.Now().Add(10*time.Minute).Unix()))
		// no results and should inspect 100 traces in wal
		assert.Len(t, sr.Traces, 0)
	}

	req := &tempopb.SearchRequest{
		Query: fmt.Sprintf(`{ span.%s = "%s" }`, tagKey, tagValue),
	}
	req.Limit = uint32(len(ids)) + 1

	// Test after appending to WAL.
	// writeTracesforSearch() makes sure all traces are in the wal
	searchAndAssert(req, uint32(100))

	// Test after cutting new headblock
	blockID, err := i.cutBlocks(true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)
	searchAndAssert(req, uint32(100))

	// Test after completing a block
	err = i.completeBlock(context.Background(), blockID)
	require.NoError(t, err)
	searchAndAssert(req, uint32(200))
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
	i, _ := defaultInstance(t)

	// add dummy search data
	tagKey := "foo"
	tagValue := bar

	_, expectedTagValues, _, _ := writeTracesForSearch(t, i, "", tagKey, tagValue, true, false)

	userCtx := user.InjectOrgID(context.Background(), "fake")

	// Test after appending to WAL
	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)

	// Test after cutting new headblock
	blockID, err := i.cutBlocks(true)
	require.NoError(t, err)
	assert.NotEqual(t, blockID, uuid.Nil)

	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)

	// Test after completing a block
	err = i.completeBlock(context.Background(), blockID)
	require.NoError(t, err)

	testSearchTagsAndValues(t, userCtx, i, tagKey, expectedTagValues)
}

// nolint:revive,unparam
func testSearchTagsAndValues(t *testing.T, ctx context.Context, i *instance, tagName string, expectedTagValues []string) {
	checkSearchTags := func(scope string, contains bool) {
		sr, err := i.SearchTags(ctx, scope)
		require.NoError(t, err)
		require.Greater(t, sr.Metrics.InspectedBytes, uint64(100)) // at least 100 bytes are inspected
		if contains {
			require.Contains(t, sr.TagNames, tagName)
		} else {
			require.NotContains(t, sr.TagNames, tagName)
		}
	}

	checkSearchTags("", true)
	checkSearchTags("span", true)
	// tags are added to the spans and not resources so they should not be present on resource
	checkSearchTags("resource", false)
	checkSearchTags("event", true)
	checkSearchTags("link", true)

	srv, err := i.SearchTagValues(ctx, tagName, 0, 0)
	require.NoError(t, err)
	require.Greater(t, srv.Metrics.InspectedBytes, uint64(100)) // we scanned at-least 100 bytes

	sort.Strings(expectedTagValues)
	sort.Strings(srv.TagValues)
	require.Equal(t, expectedTagValues, srv.TagValues)
}

func TestInstanceSearchNoData(t *testing.T) {
	i, _ := defaultInstance(t)

	req := &tempopb.SearchRequest{
		Query: "{}",
	}

	sr, err := i.Search(context.Background(), req)
	assert.NoError(t, err)
	require.Len(t, sr.Traces, 0)
}

// Helper functions adapted from ingester module
func defaultInstance(t testing.TB) (*instance, *LiveStore) {
	instance, liveStore := defaultInstanceAndTmpDir(t)
	return instance, liveStore
}

func defaultInstanceAndTmpDir(t testing.TB) (*instance, *LiveStore) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	liveStore.cfg.QueryBlockConcurrency = 1

	// Start the LiveStore service to initialize WAL
	err = liveStore.StartAsync(context.Background())
	require.NoError(t, err)

	err = liveStore.AwaitRunning(context.Background())
	require.NoError(t, err)

	// Create a fake instance for testing
	instance, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err, "unexpected error creating new instance")

	return instance, liveStore
}

func defaultLiveStore(t testing.TB, tmpDir string) (*LiveStore, error) {
	cfg := Config{}
	cfg.WAL.Filepath = tmpDir
	cfg.WAL.Version = encoding.LatestEncoding().Version()

	// Set up test Kafka configuration
	const testTopic = "traces"
	_, kafkaAddr := testkafka.CreateCluster(t, 1, testTopic)

	cfg.IngestConfig.Kafka.Address = kafkaAddr
	cfg.IngestConfig.Kafka.Topic = testTopic
	cfg.IngestConfig.Kafka.ConsumerGroup = "test-consumer-group"

	cfg.Ring.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
	//	flagext.DefaultValues(&cfg.Ring)
	mockParititionStore, _ := consul.NewInMemoryClient(
		ring.GetPartitionRingCodec(),
		log.NewNopLogger(),
		nil,
	)
	mockStore, _ := consul.NewInMemoryClient(
		ring.GetCodec(),
		log.NewNopLogger(),
		nil,
	)

	cfg.Ring.KVStore.Mock = mockStore
	cfg.Ring.ListenPort = 0
	cfg.Ring.InstanceAddr = "localhost"
	cfg.Ring.InstanceID = "test-1"
	cfg.PartitionRing.KVStore.Mock = mockParititionStore

	// Create overrides
	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}

	// Create metrics
	reg := prometheus.NewRegistry()

	logger := log.NewNopLogger()

	// Use fake Kafka cluster for testing
	liveStore, err := New(cfg, limits, logger, reg, true) // singlePartition = true for testing
	return liveStore, err
}

func pushTracesToInstance(t *testing.T, i *instance, numTraces int) ([]*tempopb.Trace, [][]byte) {
	var ids [][]byte
	var traces []*tempopb.Trace

	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)

		testTrace := test.MakeTrace(10, id)
		trace.SortTrace(testTrace)
		traceBytes, err := testTrace.Marshal()
		require.NoError(t, err)

		// Create a push request for livestore
		req := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
			Ids:    [][]byte{id},
		}
		i.pushBytes(time.Now(), req)

		ids = append(ids, id)
		traces = append(traces, testTrace)
	}
	return traces, ids
}

// writes traces to the given instance along with search data. returns
// ids expected to be returned from a tag search and strings expected to
// be returned from a tag value search
// nolint:revive,unparam
func writeTracesForSearch(t *testing.T, i *instance, spanName, tagKey, tagValue string, postFixValue bool, includeEventLink bool) ([][]byte, []string, []string, []string) {
	numTraces := 5
	ids := make([][]byte, 0, numTraces)
	expectedTagValues := make([]string, 0, numTraces)
	expectedEventTagValues := make([]string, 0, numTraces)
	expectedLinkTagValues := make([]string, 0, numTraces)

	now := time.Now()
	for j := 0; j < numTraces; j++ {
		id := make([]byte, 16)
		_, err := crand.Read(id)
		require.NoError(t, err)

		tv := tagValue
		if postFixValue {
			tv += strconv.Itoa(j)
		}
		kv := &v1.KeyValue{Key: tagKey, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: tv}}}
		eTv := "event-" + tv
		lTv := "link-" + tv
		eventKv := &v1.KeyValue{Key: tagKey, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: eTv}}}
		linkKv := &v1.KeyValue{Key: tagKey, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: lTv}}}
		expectedTagValues = append(expectedTagValues, tv)
		if includeEventLink {
			expectedEventTagValues = append(expectedEventTagValues, eTv)
			expectedLinkTagValues = append(expectedLinkTagValues, lTv)
		}
		ids = append(ids, id)

		testTrace := test.MakeTrace(10, id)
		// add the time
		for _, batch := range testTrace.ResourceSpans {
			for _, ils := range batch.ScopeSpans {
				ils.Scope = &v1.InstrumentationScope{
					Name:       "scope-name",
					Version:    "scope-version",
					Attributes: []*v1.KeyValue{kv},
				}
				for _, span := range ils.Spans {
					span.Name = spanName
					span.StartTimeUnixNano = uint64(now.UnixNano())
					span.EndTimeUnixNano = uint64(now.UnixNano())
				}
			}
		}
		testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes = append(testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes, kv)
		// add link and event
		event := &trace_v1.Span_Event{Name: "event-name", Attributes: []*v1.KeyValue{eventKv}}
		link := &trace_v1.Span_Link{TraceId: id, SpanId: id, Attributes: []*v1.KeyValue{linkKv}}
		testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Events = append(testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Events, event)
		testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Links = append(testTrace.ResourceSpans[0].ScopeSpans[0].Spans[0].Links, link)

		trace.SortTrace(testTrace)

		traceBytes, err := testTrace.Marshal()
		require.NoError(t, err)

		// Create a push request for livestore
		req := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
			Ids:    [][]byte{id},
		}
		i.pushBytes(now, req)
	}

	// traces have to be cut to show up in searches
	err := i.cutIdleTraces(true)
	require.NoError(t, err)

	return ids, expectedTagValues, expectedEventTagValues, expectedLinkTagValues
}

func TestInstanceSearchDoesNotRace(t *testing.T) {
	i, _ := defaultInstanceAndTmpDir(t)

	// add dummy search data
	tagKey := foo
	tagValue := "bar"

	req := &tempopb.SearchRequest{
		Query: fmt.Sprintf(`{ span.%s = "%s" }`, tagKey, tagValue),
	}

	end := make(chan struct{})
	wg := sync.WaitGroup{}

	concurrent := func(f func()) {
		wg.Add(1)
		defer wg.Done()
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
		_, err := crand.Read(id)
		require.NoError(t, err)

		trace := test.MakeTrace(10, id)
		traceBytes, err := trace.Marshal()
		require.NoError(t, err)

		// Create a push request for livestore
		req := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
			Ids:    [][]byte{id},
		}
		i.pushBytes(time.Now(), req)
	})

	go concurrent(func() {
		err := i.cutIdleTraces(true)
		require.NoError(t, err, "error cutting complete traces")
	})

	go concurrent(func() {
		// Cut wal, complete
		blockID, _ := i.cutBlocks(true)
		if blockID != uuid.Nil {
			err := i.completeBlock(context.Background(), blockID)
			require.NoError(t, err)
		}
	})

	go concurrent(func() {
		err := i.deleteOldBlocks() // livestore cleanup
		require.NoError(t, err)
	})

	go concurrent(func() {
		_, err := i.Search(context.Background(), req)
		require.NoError(t, err, "error searching")
	})

	go concurrent(func() {
		// SearchTags queries now require userID in ctx
		ctx := user.InjectOrgID(context.Background(), "test")
		_, err := i.SearchTags(ctx, "")
		require.NoError(t, err, "error getting search tags")
	})

	go concurrent(func() {
		// SearchTagValues queries now require userID in ctx
		ctx := user.InjectOrgID(context.Background(), "test")
		_, err := i.SearchTagValues(ctx, tagKey, 0, 0)
		require.NoError(t, err, "error getting search tag values")
	})

	time.Sleep(2000 * time.Millisecond)
	close(end)
	// Wait for go funcs to quit before
	// exiting and cleaning up
	wg.Wait()
}

func TestInstanceSearchMetrics(t *testing.T) {
	t.Parallel()
	i, _ := defaultInstance(t)

	numTraces := uint32(500)
	numBytes := uint64(0)
	for j := uint32(0); j < numTraces; j++ {
		id := test.ValidTraceID(nil)

		// Trace bytes have to be pushed as raw tempopb.Trace bytes
		trace := test.MakeTrace(10, id)

		traceBytes, err := trace.Marshal()
		require.NoError(t, err)

		// Create a push request for livestore
		req := &tempopb.PushBytesRequest{
			Traces: []tempopb.PreallocBytes{{Slice: traceBytes}},
			Ids:    [][]byte{id},
		}
		i.pushBytes(time.Now(), req)
	}

	search := func() *tempopb.SearchMetrics {
		sr, err := i.Search(context.Background(), &tempopb.SearchRequest{
			Query: fmt.Sprintf(`{ span.%s = "%s" }`, "foo", "bar"),
		})
		require.NoError(t, err)
		return sr.Metrics
	}

	// Live traces
	m := search()
	require.Equal(t, uint32(0), m.InspectedTraces) // we don't search live traces
	require.Equal(t, uint64(0), m.InspectedBytes)  // we don't search live traces

	// Test after appending to WAL
	err := i.cutIdleTraces(true)
	require.NoError(t, err)
	m = search()
	require.Less(t, numBytes, m.InspectedBytes)

	// Test after cutting new headblock
	blockID, err := i.cutBlocks(true)
	require.NoError(t, err)
	m = search()
	require.Less(t, numBytes, m.InspectedBytes)

	// Test after completing a block
	err = i.completeBlock(context.Background(), blockID)
	require.NoError(t, err)
	m = search()
	require.Less(t, numBytes, m.InspectedBytes)
}

func TestInstanceFindByTraceID(t *testing.T) {
	i, _ := defaultInstanceAndTmpDir(t)

	tagKey := foo
	tagValue := bar
	ids, _, _, _ := writeTracesForSearch(t, i, "", tagKey, tagValue, false, false)
	require.Greater(t, len(ids), 0, "writeTracesForSearch should create traces")

	// Test 1: Find traces after being cut to WAL
	resp, err := i.FindByTraceID(context.Background(), ids[0])
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Trace)
	require.Equal(t, ids[0], resp.Trace.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId)

	// Test 2: Move traces through different sections
	blockID, err := i.cutBlocks(true)
	require.NoError(t, err)
	require.NotEqual(t, blockID, uuid.Nil)

	// Verify we can still find traces from walBlocks
	resp, err = i.FindByTraceID(context.Background(), ids[0])
	require.NoError(t, err)
	require.NotNil(t, resp.Trace)

	// Test 3: Complete block (moves to completeBlocks)
	err = i.completeBlock(context.Background(), blockID)
	require.NoError(t, err)

	// Verify we can find traces from completed blocks
	resp, err = i.FindByTraceID(context.Background(), ids[0])
	require.NoError(t, err)
	require.NotNil(t, resp.Trace)

	// Test 4: Add more traces to new head block
	moreIDs, _, _, _ := writeTracesForSearch(t, i, "", tagKey, "baz", false, false)
	require.Greater(t, len(moreIDs), 0, "should create more traces")

	// Verify we can find both old and new traces
	resp1, err := i.FindByTraceID(context.Background(), ids[0])
	require.NoError(t, err)
	require.NotNil(t, resp1.Trace, "Should find trace from completed blocks")

	resp2, err := i.FindByTraceID(context.Background(), moreIDs[0])
	require.NoError(t, err)
	require.NotNil(t, resp2.Trace, "Should find trace from head block")
}

func TestIncludeBlock(t *testing.T) {
	tests := []struct {
		blocKStart int64
		blockEnd   int64
		reqStart   uint32
		reqEnd     uint32
		expected   bool
	}{
		// if request is 0s, block start/end don't matter
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   0,
			reqEnd:     0,
			expected:   true,
		},
		// req before
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   50,
			reqEnd:     99,
			expected:   false,
		},
		// overlap front
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   50,
			reqEnd:     150,
			expected:   true,
		},
		// inside block
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   110,
			reqEnd:     150,
			expected:   true,
		},
		// overlap end
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   150,
			reqEnd:     250,
			expected:   true,
		},
		// after block
		{
			blocKStart: 100,
			blockEnd:   200,
			reqStart:   201,
			reqEnd:     250,
			expected:   false,
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d-%d-%d-%d", tc.blocKStart, tc.blockEnd, tc.reqStart, tc.reqEnd), func(t *testing.T) {
			actual := includeBlock(&backend.BlockMeta{
				StartTime: time.Unix(tc.blocKStart, 0),
				EndTime:   time.Unix(tc.blockEnd, 0),
			}, &tempopb.SearchRequest{
				Start: tc.reqStart,
				End:   tc.reqEnd,
			})

			require.Equal(t, tc.expected, actual)
		})
	}
}
