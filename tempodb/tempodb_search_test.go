package tempodb

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/math"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/wal"
)

type runnerFn func(*testing.T, *tempopb.Trace, *tempopb.TraceSearchMetadata, []*tempopb.SearchRequest, []*tempopb.SearchRequest, *backend.BlockMeta, Reader)

const attributeWithTerminalChars = `{ } ( ) = ~ ! < > & | ^`

func TestSearchCompleteBlock(t *testing.T) {
	for _, v := range encoding.AllEncodings() {
		vers := v.Version()
		t.Run(vers, func(t *testing.T) {
			runCompleteBlockSearchTest(t, vers,
				searchRunner,
				traceQLRunner,
				advancedTraceQLRunner,
				groupTraceQLRunner,
				traceQLStructural,
				traceQLExistence,
			)
		})
	}
}

func searchRunner(t *testing.T, _ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, searchesThatMatch, searchesThatDontMatch []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()

	for _, req := range searchesThatMatch {
		res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
		if errors.Is(err, common.ErrUnsupported) {
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
}

func traceQLRunner(t *testing.T, _ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, searchesThatMatch, searchesThatDontMatch []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()
	e := traceql.NewEngine()

	quotedAttributesThatMatch := []*tempopb.SearchRequest{
		{Query: fmt.Sprintf("{ .%q = %q }", attributeWithTerminalChars, "foobaz")},
		{Query: fmt.Sprintf("{ .%q = %q }", attributeWithTerminalChars, "foobar")},
		{Query: `{ ."res-dedicated.02" = "res-2a" }`},
		{Query: `{ resource."k8s.namespace.name" = "k8sNamespace" }`},
	}

	searchesThatMatch = append(searchesThatMatch, quotedAttributesThatMatch...)
	for _, req := range searchesThatMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, req, fetcher)
		if errors.Is(err, common.ErrUnsupported) {
			continue
		}

		require.NoError(t, err, "search request: %+v", req)
		actual := actualForExpectedMeta(wantMeta, res)
		require.NotNil(t, actual, "search request: %v", req)
		actual.SpanSet = nil // todo: add the matching spansets to wantmeta
		actual.SpanSets = nil
		require.Equal(t, wantMeta, actual, "search request: %v", req)
	}

	quotedAttributesThaDonttMatch := []*tempopb.SearchRequest{
		{Query: fmt.Sprintf("{ .%q = %q }", attributeWithTerminalChars, "value mismatch")},
		{Query: `{ ."unknow".attribute = "res-2a" }`},
		{Query: `{ resource."resource attribute" = "unknown" }`},
	}

	searchesThatDontMatch = append(searchesThatDontMatch, quotedAttributesThaDonttMatch...)
	for _, req := range searchesThatDontMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, req, fetcher)
		require.NoError(t, err, "search request: %+v", req)
		require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
	}
}

func advancedTraceQLRunner(t *testing.T, wantTr *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()
	e := traceql.NewEngine()

	// collect some info about wantTr to use below
	trueConditionsBySpan := [][]string{}
	durationBySpan := []uint64{}
	falseConditions := []string{
		fmt.Sprintf("name=`%v`", test.RandomString()),
		fmt.Sprintf("duration>%dh", rand.Intn(10)+1),
		fmt.Sprintf("rootServiceName=`%v`", test.RandomString()),
		// status? can't really construct a status condition that's false for all spans
	}
	trueTraceC := []string{
		fmt.Sprintf("traceDuration=%dms", wantMeta.DurationMs),
		fmt.Sprintf("rootServiceName=`%s`", wantMeta.RootServiceName),
		fmt.Sprintf("rootName=`%s`", wantMeta.RootTraceName),
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
				kind := trace.KindToString(s.Kind)
				trueC = append(trueC, fmt.Sprintf("name=`%v`", s.Name))
				trueC = append(trueC, fmt.Sprintf("duration=%dns", s.EndTimeUnixNano-s.StartTimeUnixNano))
				trueC = append(trueC, fmt.Sprintf("status=%s", status))
				trueC = append(trueC, fmt.Sprintf("kind=%s", kind))
				trueC = append(trueC, trueResourceC...)
				trueC = append(trueC, trueTraceC...)

				trueConditionsBySpan = append(trueConditionsBySpan, trueC)
				falseConditions = append(falseConditions, falseC...)
				durationBySpan = append(durationBySpan, s.EndTimeUnixNano-s.StartTimeUnixNano)
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
		{Query: fmt.Sprintf("{%s} && {%s} && {%s} && {%s} && {%s}", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
		{Query: fmt.Sprintf("{%s} || {%s} || {%s} || {%s} || {%s}", rando(falseConditions), rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]), rando(falseConditions))},
		{Query: fmt.Sprintf("{%s && %s} || {%s}", rando(falseConditions), rando(falseConditions), rando(trueConditionsBySpan[0]))},
		// pipelines
		{Query: fmt.Sprintf("{%s} | {%s}", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
		{Query: fmt.Sprintf("{%s || %s} | {%s}", rando(falseConditions), rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
		// pipeline expressions
		{Query: fmt.Sprintf("({%s} | count() > 0) && ({%s} | count() > 0)", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[1]))},
		{Query: fmt.Sprintf("({%s} | count() > 0) || ({%s} | count() > 0)", rando(trueConditionsBySpan[0]), rando(falseConditions))},
		// counts
		{Query: "{} | count() > -1"},
		{Query: fmt.Sprintf("{} | count() = %d", totalSpans)},
		{Query: fmt.Sprintf("{} | count() != %d", totalSpans+1)},
		{Query: fmt.Sprintf("{ true } && { true } | count() = %d", totalSpans)},
		{Query: fmt.Sprintf("{ true } || { true } | count() = %d", totalSpans)},
		{Query: fmt.Sprintf("{ %s && %s && name=`MySpan` } | count() = 1", rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]))},
		// avgs/min/max/sum
		// These tests are tricky because asserting avg/sum/min/max only work if exactly the
		// expected spans are selected.  However there are random conditions such as traceDuration > 0
		// that always match multiple spans. The only way to keep these tests from being brittle
		// is to ensure that all spans are selected.  It's ok if a span still shows up in multiple
		// filters (because of traceDuration > 0 for example) because the && operator ensures final uniqueness.
		{Query: fmt.Sprintf("{ %s && %s } && { %s && %s } && { %s && %s } && { %s && %s } | avg(duration) = %dns",
			rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]),
			rando(trueConditionsBySpan[1]), rando(trueConditionsBySpan[1]),
			rando(trueConditionsBySpan[2]), rando(trueConditionsBySpan[2]),
			rando(trueConditionsBySpan[3]), rando(trueConditionsBySpan[3]),
			(durationBySpan[0]+durationBySpan[1]+durationBySpan[2]+durationBySpan[3])/4)},
		{Query: fmt.Sprintf("{ %s && %s } && { %s && %s } && { %s && %s } && { %s && %s } | min(duration) = %dns",
			rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]),
			rando(trueConditionsBySpan[1]), rando(trueConditionsBySpan[1]),
			rando(trueConditionsBySpan[2]), rando(trueConditionsBySpan[2]),
			rando(trueConditionsBySpan[3]), rando(trueConditionsBySpan[3]),
			math.Min64(durationBySpan[0], durationBySpan[1], durationBySpan[2], durationBySpan[3]))},
		{Query: fmt.Sprintf("{ %s && %s } && { %s && %s }  && { %s && %s } && { %s && %s } | max(duration) = %dns",
			rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]),
			rando(trueConditionsBySpan[1]), rando(trueConditionsBySpan[1]),
			rando(trueConditionsBySpan[2]), rando(trueConditionsBySpan[2]),
			rando(trueConditionsBySpan[3]), rando(trueConditionsBySpan[3]),
			math.Max64(durationBySpan[0], durationBySpan[1], durationBySpan[2], durationBySpan[3]))},
		{Query: fmt.Sprintf("{ %s && %s } && { %s && %s } && { %s && %s }  && { %s && %s }| sum(duration) = %dns",
			rando(trueConditionsBySpan[0]), rando(trueConditionsBySpan[0]),
			rando(trueConditionsBySpan[1]), rando(trueConditionsBySpan[1]),
			rando(trueConditionsBySpan[2]), rando(trueConditionsBySpan[2]),
			rando(trueConditionsBySpan[3]), rando(trueConditionsBySpan[3]),
			durationBySpan[0]+durationBySpan[1]+durationBySpan[2]+durationBySpan[3])},
		// groupin' (.foo is a known attribute that is the same on both spans)
		{Query: "{} | by(span.foo) | count() = 2"},
		{Query: "{} | by(resource.service.name) | count() = 1"},
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
		{Query: "{ } | min(duration) < 0"},
		{Query: "{ } | max(duration) < 0"},
		{Query: "{ } | sum(duration) < 0"},
		// groupin' (.foo is a known attribute that is the same on both spans)
		{Query: "{} | by(span.foo) | count() = 1"},
		{Query: "{} | by(resource.service.name) | count() = 3"},
	}

	for _, req := range searchesThatMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, req, fetcher)
		require.NoError(t, err, "search request: %+v", req)
		actual := actualForExpectedMeta(wantMeta, res)
		require.NotNil(t, actual, "search request: %v", req)
		actual.SpanSet = nil // todo: add the matching spansets to wantmeta
		actual.SpanSets = nil
		require.Equal(t, wantMeta, actual, "search request: %v", req)
	}

	for _, req := range searchesThatDontMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, req, fetcher)
		require.NoError(t, err, "search request: %+v", req)
		require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", req)
	}
}

func groupTraceQLRunner(t *testing.T, _ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()
	e := traceql.NewEngine()

	type test struct {
		req      *tempopb.SearchRequest
		expected []*tempopb.TraceSearchMetadata
	}

	searchesThatMatch := []*test{
		{
			req: &tempopb.SearchRequest{Query: "{} | by(span.foo) | count() = 2"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						// Spanset for value
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "foo", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
									},
								},
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "foo", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
									},
								},
							},
							Matched: 2,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(span.foo)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 2}}},
							},
						},
						// Spanset for nil
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000070809",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes:        nil,
								},
								{
									SpanID:            "0000000000000000",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes:        nil,
								},
							},
							Matched: 2,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(span.foo)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "nil"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 2}}},
							},
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{} | by(resource.service.name) | count() = 1"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "MyService"}}},
									},
								},
							},
							Matched: 1,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(resource.service.name)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "MyService"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 1}}},
							},
						},
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "RootService"}}},
									},
								},
							},
							Matched: 1,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(resource.service.name)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "RootService"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 1}}},
							},
						},
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000070809",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Service3"}}},
									},
								},
							},
							Matched: 1,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(resource.service.name)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Service3"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 1}}},
							},
						},
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000000000",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "BrokenService"}}},
									},
								},
							},
							Matched: 1,
							Attributes: []*v1_common.KeyValue{
								{Key: "by(resource.service.name)", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "BrokenService"}}},
								{Key: "count()", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: 1}}},
							},
						},
					},
				},
			},
		},
	}
	searchesThatDontMatch := []*tempopb.SearchRequest{
		{Query: "{} | by(span.foo) | count() = 1"}, // Both spansets (foo!=nil, and foo=nil) have 2 spans
		{Query: "{} | by(resource.service.name) | count() = 3"},
	}

	for _, tc := range searchesThatMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, tc.req, fetcher)
		require.NoError(t, err, "search request: %+v", tc)

		// copy the root stuff in directly, spansets defined in test cases above.
		for _, ss := range tc.expected {
			ss.DurationMs = wantMeta.DurationMs
			ss.RootServiceName = wantMeta.RootServiceName
			ss.RootTraceName = wantMeta.RootTraceName
			ss.StartTimeUnixNano = wantMeta.StartTimeUnixNano
			ss.TraceID = wantMeta.TraceID
		}

		// the actual spanset is impossible to predict since it's chosen randomly from the Spansets slice
		// so set it to nil here and just test the slice using the testcases above
		for _, tr := range res.Traces {
			tr.SpanSet = nil
		}

		require.NotNil(t, res, "search request: %v", tc)
		require.Equal(t, tc.expected, res.Traces, "search request", tc.req)
	}

	for _, tc := range searchesThatDontMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, tc, fetcher)
		require.NoError(t, err, "search request: %+v", tc)
		require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", tc)
	}
}

func traceQLStructural(t *testing.T, _ *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()
	e := traceql.NewEngine()

	type test struct {
		req      *tempopb.SearchRequest
		expected []*tempopb.TraceSearchMetadata
	}

	searchesThatMatch := []*test{
		{
			req: &tempopb.SearchRequest{Query: "{ .parent } >> { .child }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "child", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } << { .parent }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "parent", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .parent } > { .child }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "child", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } < { .parent }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Name:              "",
									Attributes: []*v1_common.KeyValue{
										{Key: "parent", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } !> { .parent }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "parent", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } !>> { .parent }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "parent", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } !~ { .parent }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "parent", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{  } !~ {  }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000040506",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     2000000000,
									Attributes:        nil,
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .parent } !< { .child }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "child", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .parent } !<< { .child }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "child", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .child } ~ { .child2 }"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000070809",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
									Attributes: []*v1_common.KeyValue{
										{Key: "child2", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}}},
									},
								},
							},
							Matched: 1,
						},
					},
				},
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ .parent } >> {}"},
			expected: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Spans: []*tempopb.Span{
								{
									SpanID:            "0000000000010203",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
								},
								{
									SpanID:            "0000000000070809",
									StartTimeUnixNano: 1000000000000,
									DurationNanos:     1000000000,
								},
							},
							Matched: 2,
						},
					},
				},
			},
		},
	}

	searchesThatDontMatch := []*tempopb.SearchRequest{
		{Query: "{ .child } >> { .parent }"},
		{Query: "{ .child } > { .parent }"},
		{Query: "{ .child } ~ { .parent }"},
		{Query: "{ .child } ~ { .child }"},
		{Query: "{ .broken} >> {}"},
		{Query: "{ .broken} > {}"},
		{Query: "{ .broken} ~ {}"},
		{Query: "{} >> {.broken}"},
		{Query: "{} > {.broken}"},
		{Query: "{} ~ {.broken}"},
		{Query: "{ .child } !< { .parent }"},
		{Query: "{ .parent } !> { .child }"},
		{Query: "{ .child } !~ { .child2 }"},
	}

	for _, tc := range searchesThatMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, tc.req, fetcher)
		if errors.Is(err, common.ErrUnsupported) {
			continue
		}

		require.NoError(t, err, "search request: %+v", tc)

		// copy the root stuff in directly, spansets defined in test cases above.
		for _, ss := range tc.expected {
			ss.DurationMs = wantMeta.DurationMs
			ss.RootServiceName = wantMeta.RootServiceName
			ss.RootTraceName = wantMeta.RootTraceName
			ss.StartTimeUnixNano = wantMeta.StartTimeUnixNano
			ss.TraceID = wantMeta.TraceID
		}

		// the actual spanset is impossible to predict since it's chosen randomly from the Spansets slice
		// so set it to nil here and just test the slice using the testcases above
		for _, tr := range res.Traces {
			tr.SpanSet = nil
		}

		require.NotNil(t, res, "search request: %v", tc)
		require.Equal(t, tc.expected, res.Traces, "search request:", tc.req)
	}

	for _, tc := range searchesThatDontMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, tc, fetcher)
		if errors.Is(err, common.ErrUnsupported) {
			continue
		}
		require.NoError(t, err, "search request: %+v", tc)
		require.Nil(t, actualForExpectedMeta(wantMeta, res), "search request: %v", tc)
	}
}

// existence
func traceQLExistence(t *testing.T, _ *tempopb.Trace, _ *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
	ctx := context.Background()
	e := traceql.NewEngine()
	const intrinsicName = "name"

	type expected struct {
		key string
	}

	type test struct {
		req      *tempopb.SearchRequest
		expected expected
	}

	searchesThatMatch := []*test{
		{
			req: &tempopb.SearchRequest{Query: "{ name != nil }", Limit: 10},
			expected: expected{
				key: intrinsicName,
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ duration != nil }", Limit: 10},
			expected: expected{
				key: "duration",
			},
		},
		{
			req: &tempopb.SearchRequest{Query: "{ resource.service.name != nil }", Limit: 10},
			expected: expected{
				key: "resource.service.name",
			},
		},
	}
	// TODO re-enable commented searches after fixing structural operator bugs in vParquet3
	//      https://github.com/grafana/tempo/issues/2674
	searchesThatDontMatch := []*tempopb.SearchRequest{
		{Query: "{ name = nil }"},
		{Query: "{ duration = nil }"},
		{Query: "{ .not_an_attribute = nil }"},
	}

	for _, tc := range searchesThatMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		res, err := e.ExecuteSearch(ctx, tc.req, fetcher)
		if errors.Is(err, common.ErrUnsupported) {
			continue
		}

		require.NoError(t, err, "search request: %+v", tc)

		// the actual spanset is impossible to predict since it's chosen randomly from the Spansets slice
		// so set it to nil here and just test the slice using the testcases above
		for _, tr := range res.Traces {
			tr.SpanSet = nil
		}

		// make sure every spanset returned has the attribute we searched for
		for _, tr := range res.Traces {
			spanSet := tr.SpanSets[0]
			for _, span := range spanSet.Spans {
				switch tc.expected.key {
				case intrinsicName:
					require.NotNil(t, span.Name)
				case "duration":
					require.NotNil(t, span.DurationNanos)
				default:
					for _, attribute := range span.Attributes {
						if attribute.Key == "service.name" {
							require.NotNil(t, attribute.Value)
						}
					}
				}
			}
		}
	}

	for _, tc := range searchesThatDontMatch {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
		})

		_, err := e.ExecuteSearch(ctx, tc, fetcher)
		if errors.Is(err, common.ErrUnsupported) {
			continue
		}
		require.Error(t, err, "search request: %+v", tc)
	}
}

// oneQueryRunner is a good place to place a single query for debugging
// func oneQueryRunner(t *testing.T, wantTr *tempopb.Trace, wantMeta *tempopb.TraceSearchMetadata, _, _ []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
// 	ctx := context.Background()
// 	e := traceql.NewEngine()

// 	searchesThatMatch := []*tempopb.SearchRequest{
// 		// conditions
// 		{Query: "{rootServiceName=`fotlVYVqts`} || {.k8s.container.name=`k8sContainer`}"},
// 	}

// 	for _, req := range searchesThatMatch {
// 		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
// 			return r.Fetch(ctx, meta, req, common.DefaultSearchOptions())
// 		})

// 		res, err := e.ExecuteSearch(ctx, req, fetcher)
// 		require.NoError(t, err, "search request: %+v", req)
// 		actual := actualForExpectedMeta(wantMeta, res)
// 		require.NotNil(t, actual, "search request: %v", req)
// 		actual.SpanSet = nil // todo: add the matching spansets to wantmeta
// 		actual.SpanSets = nil
// 		require.Equal(t, wantMeta, actual, "search request: %v", req)
// 	}
// }

func conditionsForAttributes(atts []*v1_common.KeyValue, scope string) ([]string, []string) {
	trueConditions := []string{}
	falseConditions := []string{}

	for _, a := range atts {
		// surround attribute with quote if contains terminal char
		if a.Key == attributeWithTerminalChars {
			a.Key = fmt.Sprintf("%q", a.Key)
		}
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

func runCompleteBlockSearchTest(t *testing.T, blockVersion string, runners ...runnerFn) {
	// v2 doesn't support any search. just bail here before doing the work below to save resources
	if blockVersion == v2.VersionString {
		return
	}

	tempDir := t.TempDir()

	dc := backend.DedicatedColumns{
		{Scope: "resource", Name: "res-dedicated.01", Type: "string"},
		{Scope: "resource", Name: "res-dedicated.02", Type: "string"},
		{Scope: "span", Name: "span-dedicated.01", Type: "string"},
		{Scope: "span", Name: "span-dedicated.02", Type: "string"},
	}
	r, w, c, err := New(&Config{
		Backend: backend.Local,
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
			DedicatedColumns:     dc,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &SearchConfig{
			ChunkSizeBytes:  1_000_000,
			ReadBufferCount: 8, ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	err = c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	ctx := context.Background()
	r.EnablePolling(ctx, &mockJobSharder{})
	rw := r.(*readerWriter)

	wantID, wantTr, start, end, wantMeta, searchesThatMatch, searchesThatDontMatch := searchTestSuite()

	// Write to wal
	wal := w.WAL()
	head, err := wal.NewBlockWithDedicatedColumns(uuid.New(), testTenantID, model.CurrentEncoding, dc)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	totalTraces := 50
	wantTrIdx := rand.Intn(totalTraces)
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

	for _, r := range runners {
		r(t, wantTr, wantMeta, searchesThatMatch, searchesThatDontMatch, meta, rw)
	}

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

func boolKV(k string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   k,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}},
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
						stringKV("res-dedicated.01", "res-1a"),
						stringKV("res-dedicated.02", "res-2a"),
						stringKV(attributeWithTerminalChars, "foobar"),
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
									boolKV("child"),
									stringKV("span-dedicated.01", "span-1a"),
									stringKV("span-dedicated.02", "span-2a"),
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
						stringKV("res-dedicated.01", "res-1b"),
						stringKV("res-dedicated.02", "res-2b"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								Name:              "RootSpan",
								SpanId:            []byte{4, 5, 6},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1002 * time.Second),
								Status:            &v1.Status{},
								Kind:              v1.Span_SPAN_KIND_CLIENT,
								Attributes: []*v1_common.KeyValue{
									stringKV("foo", "Bar"),
									boolKV("parent"),
									stringKV("span-dedicated.01", "span-1b"),
									stringKV("span-dedicated.02", "span-2b"),
									stringKV(attributeWithTerminalChars, "foobaz"),
								},
							},
						},
					},
				},
			},
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "Service3"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								TraceId:           id,
								SpanId:            []byte{7, 8, 9},
								ParentSpanId:      []byte{4, 5, 6},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Kind:              v1.Span_SPAN_KIND_PRODUCER,
								Status:            &v1.Status{Code: v1.Status_STATUS_CODE_OK},
								Attributes: []*v1_common.KeyValue{
									boolKV("child2"),
								},
							},
						},
					},
				},
			},
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1_common.KeyValue{
						stringKV("service.name", "BrokenService"),
					},
				},
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{
								Name:              "BrokenSpan",
								TraceId:           id,
								SpanId:            []byte{0, 0, 0},
								ParentSpanId:      []byte{0, 0, 0},
								StartTimeUnixNano: uint64(1000 * time.Second),
								EndTimeUnixNano:   uint64(1001 * time.Second),
								Status:            &v1.Status{Code: v1.Status_STATUS_CODE_OK},
								Attributes: []*v1_common.KeyValue{
									boolKV("broken"),
								},
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
		DurationMs:        2000,
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
			MaxDurationMs: 2001,
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

		// Dedicated span and resource attributes
		makeReq("res-dedicated.01", "res-1a"),
		makeReq("res-dedicated.02", "res-2b"),
		makeReq("span-dedicated.01", "span-1a"),
		makeReq("span-dedicated.02", "span-2b"),

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
			MinDurationMs: 2001,
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
		// makeReq("status.code", "ok"),
		makeReq("root.service.name", "NotRootService"),
		makeReq("root.name", "NotRootSpan"),

		// Dedicated span and resource attributes
		makeReq("res-dedicated.01", "res-2a"),
		makeReq("res-dedicated.02", "does-not-exist"),
		makeReq("span-dedicated.01", "span-2a"),
		makeReq("span-dedicated.02", "does-not-exist"),

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

func TestWALBlockGetMetrics(t *testing.T) {
	var (
		ctx     = context.Background()
		tempDir = t.TempDir()
	)

	r, w, c, err := New(&Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.DefaultEncoding().Version(),
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    10000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &SearchConfig{
			ChunkSizeBytes:  1_000_000,
			ReadBufferCount: 8, ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	err = c.EnableCompaction(context.Background(), &CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})
	require.NoError(t, err)

	r.EnablePolling(ctx, &mockJobSharder{})

	wal := w.WAL()
	head, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err)

	// Write to wal
	err = head.AppendTrace(common.ID{0x01}, &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				ScopeSpans: []*v1.ScopeSpans{
					{
						Spans: []*v1.Span{
							{Name: "1", StartTimeUnixNano: 1, EndTimeUnixNano: 2}, // Included
							{Name: "2", StartTimeUnixNano: 2, EndTimeUnixNano: 4}, // Included
							{Name: "3", StartTimeUnixNano: 100},                   // Excluded, endtime is exclusive
							{Name: "4", StartTimeUnixNano: 101},                   // Excluded
						},
					},
				},
			},
		},
	}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, head.Flush())

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return head.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	res, err := traceqlmetrics.GetMetrics(ctx, "{}", "name", 0, 1, 100, f)
	require.NoError(t, err)

	one := traceqlmetrics.MetricSeries{traceqlmetrics.KeyValue{Key: "name", Value: traceql.NewStaticString("1")}}
	two := traceqlmetrics.MetricSeries{traceqlmetrics.KeyValue{Key: "name", Value: traceql.NewStaticString("2")}}

	require.Equal(t, 2, len(res.Series))
	require.Equal(t, 2, res.SpanCount)
	require.Equal(t, 1, res.Series[one].Count())
	require.Equal(t, 1, res.Series[two].Count())
	require.Equal(t, uint64(1), res.Series[one].Percentile(1.0)) // The only span was 1ns
	require.Equal(t, uint64(2), res.Series[two].Percentile(1.0)) // The only span was 2ns
}
