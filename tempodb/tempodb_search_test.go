package tempodb

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func TestSearchCompleteBlock(t *testing.T) {
	for _, v := range encoding.AllEncodings() {
		vers := v.Version()
		t.Run(vers, func(t *testing.T) {
			testSearchCompleteBlock(t, vers)
		})
	}
}

func testSearchCompleteBlock(t *testing.T, blockVersion string) {
	runCompleteBlockSearchTest(t, blockVersion, func(_ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, searchesThatMatch, searchesThatDontMatch []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
		ctx := context.Background()

		for _, req := range searchesThatMatch {
			res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
			if err == common.ErrUnsupported {
				return
			}
			require.NoError(t, err, "search request: %+v", req)
			require.Equal(t, wantMeta, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
		}

		for _, req := range searchesThatDontMatch {
			res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
			require.NoError(t, err, "search request: %+v", req)
			require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
		}
	})
}

// TestTraceQLCompleteBlock tests basic traceql tag matching conditions and
// aligns with the feature set and testing of the tags search
func TestTraceQLCompleteBlock(t *testing.T) {
	for _, v := range encoding.AllEncodings() {
		vers := v.Version()
		t.Run(vers, func(t *testing.T) {
			testTraceQLCompleteBlock(t, vers)
		})
	}
}

func testTraceQLCompleteBlock(t *testing.T, blockVersion string) {
	e := traceql.NewEngine()

	runCompleteBlockSearchTest(t, blockVersion, func(_ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, searchesThatMatch, searchesThatDontMatch []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
		ctx := context.Background()

		for _, req := range searchesThatMatch {
			fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
			})

			res, err := e.Execute(ctx, req, fetcher)
			require.NoError(t, err, "search request: %+v", req)
			actual := actualForExpectedMeta(wantMeta, res)
			require.NotNil(t, actual, "search request: %v", req)
			actual.SpanSet = nil // todo: add the matching spansets to wantmeta
			require.Equal(t, wantMeta, actual, "search request: %v", req)
		}

		for _, req := range searchesThatDontMatch {
			fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
			})

			res, err := e.Execute(ctx, req, fetcher)
			require.NoError(t, err, "search request: %+v", req)
			require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
		}
	})
}

// TestAdvancedTraceQLCompleteBlock uses the actual trace data to construct complex traceql queries
// it is supposed to cover all major traceql features. if you see one missing add it!
func TestAdvancedTraceQLCompleteBlock(t *testing.T) {
	for _, v := range encoding.AllEncodings() {
		vers := v.Version()
		t.Run(vers, func(t *testing.T) {
			testAdvancedTraceQLCompleteBlock(t, vers)
		})
	}
}

func testAdvancedTraceQLCompleteBlock(t *testing.T, blockVersion string) {
	e := traceql.NewEngine()

	runCompleteBlockSearchTest(t, blockVersion, func(wantTr *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
		ctx := context.Background()

		// collect some info about wantTr to use below
		trueConditionsBySpan := [][]string{}
		falseConditions := []string{
			fmt.Sprintf("name=`%v`", test.RandomString()),
			fmt.Sprintf("duration>%dh", rand.Intn(10)),
			// status? can't really construct a status condition that's false for all spans
		}
		totalSpans := 0
		for _, b := range wantTr.Batches {
			trueResourceC, falseResourceC := conditionsForAttributes(b.Resource.Attributes, "resource")
			falseConditions = append(falseConditions, falseResourceC...)

			for _, ss := range b.ScopeSpans {
				totalSpans += len(ss.Spans)
				for _, s := range ss.Spans {
					trueC, falseC := conditionsForAttributes(s.Attributes, "span")

					status := trace.StatusToString(s.Status.Code)
					trueC = append(trueC, fmt.Sprintf("name=`%v`", s.Name))
					trueC = append(trueC, fmt.Sprintf("duration=%dns", s.EndTimeUnixNano-s.StartTimeUnixNano))
					trueC = append(trueC, fmt.Sprintf("status=%s", status))

					trueConditionsBySpan = append(trueConditionsBySpan, trueC)
					trueConditionsBySpan = append(trueConditionsBySpan, trueResourceC)
					falseConditions = append(falseConditions, falseC...)
				}
			}
		}

		rando := func(s []string) string {
			return s[rand.Intn(len(s))]
		}

		searchesThatMatch := []*tempopb.SearchRequest{
			// conditions
			{Query: fmt.Sprintf("{%s && %s && %s && %s && %s}", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
			{Query: fmt.Sprintf("{%s || %s || %s || %s || %s}", rando(falseConditions), rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("{(%s && %s) || %s}", rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]))},
			// spansets
			{Query: fmt.Sprintf("{%s} && {%s}", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[1]))},
			{Query: fmt.Sprintf("{%s} || {%s}", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s && %s} || {%s}", rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]))},
			// pipelines
			{Query: fmt.Sprintf("{%s} | {%s}", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
			{Query: fmt.Sprintf("{%s || %s} | {%s}", rando(falseConditions), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
			// pipeline expressions
			{Query: fmt.Sprintf("({%s} | count() > 0) && ({%s} | count() > 0)", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[1]))},
			{Query: fmt.Sprintf("({%s} | count() > 0) || ({%s} | count() > 0)", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			// counts
			{Query: fmt.Sprintf("{} | count() = %d", totalSpans)},
			{Query: fmt.Sprintf("{} | count() != %d", totalSpans+1)},
			{Query: fmt.Sprintf("{} | count() <= %d", totalSpans)},
			{Query: fmt.Sprintf("{} | count() >= %d", totalSpans)},
			// avgs
			{Query: "{ } | avg(duration) > 0"}, // todo: make this better
		}
		searchesThatDontMatch := []*tempopb.SearchRequest{
			// conditions
			{Query: fmt.Sprintf("{%s && %s}", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s || %s}", rando(falseConditions), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s && (%s || %s)}", rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]))},
			// spansets
			{Query: fmt.Sprintf("{%s} && {%s}", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s} || {%s}", rando(falseConditions), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s && %s} || {%s}", rando(falseConditions), rando(falseConditions), rando(falseConditions))},
			// pipelines
			{Query: fmt.Sprintf("{%s} | {%s}", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("{%s} | {%s}", rando(falseConditions), rando(trueConditionsBySpan[0]))},
			{Query: fmt.Sprintf("{%s || %s} | {%s}", rando(falseConditions), rando(trueConditionsBySpan[0]), rando(falseConditions))},
			// pipeline expressions
			{Query: fmt.Sprintf("({%s} | count() > 0) && ({%s} | count() > 0)", rando(trueConditionsBySpan[0]), rando(falseConditions))},
			{Query: fmt.Sprintf("({%s} | count() > 0) || ({%s} | count() > 0)", rando(falseConditions), rando(falseConditions))},
			// counts
			{Query: fmt.Sprintf("{} | count() = %d", totalSpans+1)},
			{Query: fmt.Sprintf("{} | count() != %d", totalSpans)},
			{Query: fmt.Sprintf("{} | count() < %d", totalSpans)},
			{Query: fmt.Sprintf("{} | count() > %d", totalSpans)},
			// avgs
			{Query: "{ } | avg(.dne) != 0"},
			{Query: "{ } | avg(duration) < 0"},
		}

		for _, req := range searchesThatMatch {
			fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
			})

			res, err := e.Execute(ctx, req, fetcher)
			require.NoError(t, err, "search request: %+v", req)
			actual := actualForExpectedMeta(wantMeta, res)
			require.NotNil(t, actual, "search request: %v", req)
			actual.SpanSet = nil // todo: add the matching spansets to wantmeta
			require.Equal(t, wantMeta, actual, "search request: %v", req)
		}

		for _, req := range searchesThatDontMatch {
			fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
			})

			res, err := e.Execute(ctx, req, fetcher)
			require.NoError(t, err, "search request: %+v", req)
			require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
		}
	})
}

func conditionsForAttributes(atts []*v1_common.KeyValue, scope string) ([]string, []string) {
	trueConditions := []string{}
	falseConditions := []string{}

	for _, a := range atts {
		switch v := a.GetValue().Value.(type) {
		case *v1_common.AnyValue_StringValue:
			trueConditions = append(trueConditions, fmt.Sprintf("%s.%v=`%v`", scope, a.Key, v.StringValue))
			trueConditions = append(trueConditions, fmt.Sprintf(".%v=`%v`", a.Key, v.StringValue))
			falseConditions = append(falseConditions, fmt.Sprintf("%s.%v=`%v`", scope, a.Key, test.RandomString()))
			falseConditions = append(falseConditions, fmt.Sprintf(".%v=`%v`", a.Key, test.RandomString()))
		case *v1_common.AnyValue_BoolValue:
			trueConditions = append(trueConditions, fmt.Sprintf("%s.%v=%t", scope, a.Key, v.BoolValue))
			trueConditions = append(trueConditions, fmt.Sprintf(".%v=%t", a.Key, v.BoolValue))
			// tough to add an always false condition here
		case *v1_common.AnyValue_IntValue:
			trueConditions = append(trueConditions, fmt.Sprintf("%s.%v=%d", scope, a.Key, v.IntValue))
			trueConditions = append(trueConditions, fmt.Sprintf(".%v=%d", a.Key, v.IntValue))
			falseConditions = append(falseConditions, fmt.Sprintf("%s.%v=%d", scope, a.Key, rand.Intn(1000)+20000))
			falseConditions = append(falseConditions, fmt.Sprintf(".%v=%d", a.Key, rand.Intn(1000)+20000))
		case *v1_common.AnyValue_DoubleValue:
			trueConditions = append(trueConditions, fmt.Sprintf("%s.%v=%f", scope, a.Key, v.DoubleValue))
			trueConditions = append(trueConditions, fmt.Sprintf(".%v=%f", a.Key, v.DoubleValue))
			falseConditions = append(falseConditions, fmt.Sprintf("%s.%v=%f", scope, a.Key, rand.Float64()))
			falseConditions = append(falseConditions, fmt.Sprintf(".%v=%f", a.Key, rand.Float64()))
		}
	}

	return trueConditions, falseConditions
}

func actualForExpectedMeta(wantMeta *tempopb.TraceSearchMetadata, res *tempopb.SearchResponse) *tempopb.TraceSearchMetadata {
	// find wantMeta in res
	for _, tr := range res.Traces {
		if tr.TraceID == wantMeta.TraceID {
			return tr
		}
	}

	return nil
}

type runnerFn func(*tempopb.Trace, *tempopb.TraceSearchMetadata, []*tempopb.SearchRequest, []*tempopb.SearchRequest, *backend.BlockMeta, Reader)

func runCompleteBlockSearchTest(t testing.TB, blockVersion string, runner runnerFn) {
	// v2 doesn't support any search. just bail here before doing the work below to save resources
	if blockVersion == v2.VersionString {
		return
	}

	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              blockVersion,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    10000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &SearchConfig{
			ChunkSizeBytes:      1_000_000,
			ReadBufferCount:     8,
			ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})
	rw := r.(*readerWriter)

	wantID, wantTr, start, end, wantMeta, searchesThatMatch, searchesThatDontMatch := searchTestSuite()

	// Write to wal
	wal := w.WAL()
	head, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	totalTraces := 250
	wantTrIdx := rand.Intn(250)
	for i := 0; i < totalTraces; i++ {
		var tr *tempopb.Trace
		var id []byte
		if i == wantTrIdx {
			tr = wantTr
			id = wantID
		} else {
			id = test.ValidTraceID(nil)
			tr = test.MakeTrace(10, id)
		}
		b1, err := dec.PrepareForWrite(tr, start, end)
		require.NoError(t, err)

		b2, err := dec.ToObject([][]byte{b1})
		require.NoError(t, err)
		err = head.Append(id, b2, start, end)
		require.NoError(t, err)
	}

	// Complete block
	block, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)
	meta := block.BlockMeta()

	runner(wantTr, wantMeta, searchesThatMatch, searchesThatDontMatch, meta, rw)

	// todo: do some compaction and then call runner again
}

func stringKV(k, v string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}},
	}
}

func intKV(k string, v int) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: int64(v)}},
	}
}

// Helper function to make a tag search
func makeReq(k, v string) *tempopb.SearchRequest {
	return &tempopb.SearchRequest{
		Tags: map[string]string{
			k: v,
		},
	}
}

func addTraceQL(req *tempopb.SearchRequest) {
	// todo: traceql concepts are different than search concepts. this code maps key/value pairs
	// from search to traceql. we can clean this up after we drop old search and move these tests into
	// the tempodb package.
	traceqlConditions := []string{}
	for k, v := range req.Tags {
		traceqlKey := k
		switch traceqlKey {
		case "root.service.name":
			traceqlKey = ".service.name"
		case "root.name":
			traceqlKey = "name"
		case "name":
		case "status.code":
			traceqlKey = "status"
		default:
			traceqlKey = "." + traceqlKey
		}

		traceqlVal := v
		switch traceqlKey {
		case ".http.status_code":
			break
		case "status":
			break
		default:
			traceqlVal = fmt.Sprintf(`"%s"`, v)
		}
		traceqlConditions = append(traceqlConditions, fmt.Sprintf("%s=%s", traceqlKey, traceqlVal))
	}
	if req.MaxDurationMs != 0 {
		traceqlConditions = append(traceqlConditions, fmt.Sprintf("duration < %dms", req.MaxDurationMs))
	}
	if req.MinDurationMs != 0 {
		traceqlConditions = append(traceqlConditions, fmt.Sprintf("duration > %dms", req.MinDurationMs))
	}

	req.Query = "{" + strings.Join(traceqlConditions, "&&") + "}"
}

// searchTestSuite returns a set of search test cases that ensure
// search behavior is consistent across block types and modules.
// The return parameters are:
//   - trace ID
//   - trace - a fully-populated trace that is searched for every condition. If testing a
//     block format, then write this trace to the block.
//   - start, end - the unix second start/end times for the trace, i.e. slack-adjusted timestamps
//   - expected - The exact search result that should be returned for every matching request
//   - searchesThatMatch - List of search requests that are expected to match the trace
//   - searchesThatDontMatch - List of requests that don't match the trace
func searchTestSuite() (
	id []byte,
	tr *tempopb.Trace,
	start, end uint32,
	expected *tempopb.TraceSearchMetadata,
	searchesThatMatch []*tempopb.SearchRequest,
	searchesThatDontMatch []*tempopb.SearchRequest,
) {

	id = test.ValidTraceID(nil)

	start = 1000
	end = 1001

	tr = &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "MyService"),
						stringKV("cluster", "MyCluster"),
						stringKV("namespace", "MyNamespace"),
						stringKV("pod", "MyPod"),
						stringKV("container", "MyContainer"),
						stringKV("k8s.cluster.name", "k8sCluster"),
						stringKV("k8s.namespace.name", "k8sNamespace"),
						stringKV("k8s.pod.name", "k8sPod"),
						stringKV("k8s.container.name", "k8sContainer"),
						stringKV("bat", "Baz"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "MySpan",
								SpanId:            []byte{1, 2, 3},
								ParentSpanId:      []byte{4, 5, 6},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status: &v1.Status{
									Code: v1.Status_STATUS_CODE_ERROR,
								},
								Attributes: []*v1_common.KeyValue{
									stringKV("http.method", "Get"),
									stringKV("http.url", "url/Hello/World"),
									intKV("http.status_code", 500),
									stringKV("foo", "Bar"),
								},
							},
						},
					},
				},
			},
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "RootService"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "RootSpan",
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status:            &v1.Status{},
							},
						},
					},
				},
			},
		},
	}

	expected = &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(id),
		StartTimeUnixNano: uint64(1000 * time.Second),
		DurationMs:        1000,
		RootServiceName:   "RootService",
		RootTraceName:     "RootSpan",
	}

	// Matches
	searchesThatMatch = []*tempopb.SearchRequest{
		{
			// Empty request
		},
		{
			MinDurationMs: 999,
			MaxDurationMs: 1001,
		},
		{
			Start: 1000,
			End:   2000,
		},
		{
			// Overlaps start
			Start: 999,
			End:   1001,
		},
		{
			// Overlaps end
			Start: 1001,
			End:   1002,
		},

		// Well-known resource attributes
		makeReq("service.name", "MyService"),
		makeReq("cluster", "MyCluster"),
		makeReq("namespace", "MyNamespace"),
		makeReq("pod", "MyPod"),
		makeReq("container", "MyContainer"),
		makeReq("k8s.cluster.name", "k8sCluster"),
		makeReq("k8s.namespace.name", "k8sNamespace"),
		makeReq("k8s.pod.name", "k8sPod"),
		makeReq("k8s.container.name", "k8sContainer"),
		makeReq("root.service.name", "RootService"),
		makeReq("root.name", "RootSpan"),

		// Well-known span attributes
		makeReq("name", "MySpan"),
		makeReq("http.method", "Get"),
		makeReq("http.url", "url/Hello/World"),
		makeReq("http.status_code", "500"),
		makeReq("status.code", "error"),

		// Span attributes
		makeReq("foo", "Bar"),
		// Resource attributes
		makeReq("bat", "Baz"),

		// Multiple
		{
			Tags: map[string]string{
				"service.name": "MyService",
				"http.method":  "Get",
				"foo":          "Bar",
			},
		},
	}

	// Excludes
	searchesThatDontMatch = []*tempopb.SearchRequest{
		{
			MinDurationMs: 1001,
		},
		{
			MaxDurationMs: 999,
		},
		{
			Start: 100,
			End:   200,
		},

		// Well-known resource attributes
		makeReq("service.name", "service"), // wrong case
		makeReq("cluster", "cluster"),      // wrong case
		makeReq("namespace", "namespace"),  // wrong case
		makeReq("pod", "pod"),              // wrong case
		makeReq("container", "container"),  // wrong case

		// Well-known span attributes
		makeReq("http.method", "post"),
		makeReq("http.url", "asdf"),
		makeReq("http.status_code", "200"),
		makeReq("status.code", "ok"),
		makeReq("root.service.name", "NotRootService"),
		makeReq("root.name", "NotRootSpan"),

		// Span attributes
		makeReq("foo", "baz"), // wrong case
	}

	// add traceql to all searches
	for _, req := range searchesThatDontMatch {
		addTraceQL(req)
	}
	for _, req := range searchesThatMatch {
		addTraceQL(req)
	}

	return
}
