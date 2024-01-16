package traceql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
)

func TestEngine_Execute(t *testing.T) {
	now := time.Now()
	e := NewEngine()

	req := &tempopb.SearchRequest{
		Query:           `{ .foo = .bar }`,
		SpansPerSpanSet: 2,
	}
	spanSetFetcher := MockSpanSetFetcher{
		iterator: &MockSpanSetIterator{
			results: []*Spanset{
				{
					TraceID:         []byte{1},
					RootSpanName:    "HTTP GET",
					RootServiceName: "my-service",
					Spans: []Span{
						&mockSpan{
							id: []byte{1},
							attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
							},
						},
						&mockSpan{
							id:                 []byte{2},
							startTimeUnixNanos: uint64(now.UnixNano()),
							durationNanos:      uint64((100 * time.Millisecond).Nanoseconds()),
							attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
								NewAttribute("bar"): NewStaticString("value"),
							},
						},
						&mockSpan{
							id:                 []byte{3},
							startTimeUnixNanos: uint64(now.UnixNano()),
							durationNanos:      uint64((200 * time.Millisecond).Nanoseconds()),
							attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
								NewAttribute("bar"): NewStaticString("value"),
							},
						},
						&mockSpan{
							id:                 []byte{4},
							startTimeUnixNanos: uint64(now.UnixNano()),
							durationNanos:      uint64((100 * time.Millisecond).Nanoseconds()),
							attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
								NewAttribute("bar"): NewStaticString("diff"),
							},
						},
						&mockSpan{
							id:                 []byte{5},
							startTimeUnixNanos: uint64(now.UnixNano()),
							durationNanos:      uint64((100 * time.Millisecond).Nanoseconds()),
							attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
								NewAttribute("bar"): NewStaticString("value"),
							},
						},
					},
				},
				{
					TraceID:         []byte{2},
					RootSpanName:    "HTTP POST",
					RootServiceName: "my-service",
					Spans: []Span{
						&mockSpan{
							id: []byte{3},
							attributes: map[Attribute]Static{
								NewAttribute("bar"): NewStaticString("value"),
							},
						},
					},
				},
			},
		},
	}
	response, err := e.ExecuteSearch(context.Background(), req, &spanSetFetcher)

	require.NoError(t, err)

	expectedFetchSpansRequest := FetchSpansRequest{
		Conditions: []Condition{
			newCondition(NewAttribute("foo"), OpNone),
			newCondition(NewAttribute("bar"), OpNone),
		},
		AllConditions:        true,
		SecondPassConditions: SearchMetaConditions(),
	}
	spanSetFetcher.capturedRequest.SecondPass = nil // have to set this to nil b/c assert.Equal does not handle function pointers
	assert.Equal(t, expectedFetchSpansRequest, spanSetFetcher.capturedRequest)

	expectedSpanset := &tempopb.SpanSet{
		Spans: []*tempopb.Span{
			{
				SpanID:            "0000000000000002",
				StartTimeUnixNano: uint64(now.UnixNano()),
				DurationNanos:     100_000_000,
				Attributes: []*v1.KeyValue{
					{
						Key: "foo",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: "value",
							},
						},
					},
					{
						Key: "bar",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: "value",
							},
						},
					},
				},
			},
			{
				SpanID:            "0000000000000003",
				StartTimeUnixNano: uint64(now.UnixNano()),
				DurationNanos:     200_000_000,
				Attributes: []*v1.KeyValue{
					{
						Key: "foo",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: "value",
							},
						},
					},
					{
						Key: "bar",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: "value",
							},
						},
					},
				},
			},
		},
		Matched: 0,
	}
	expectedTraceSearchMetadata := []*tempopb.TraceSearchMetadata{
		{
			TraceID:         "1",
			RootServiceName: "my-service",
			RootTraceName:   "HTTP GET",
			SpanSet:         expectedSpanset,
			SpanSets:        []*tempopb.SpanSet{expectedSpanset},
		},
	}

	// Sort attributes for consistent equality checks
	// This is needed because they are internally stored in maps
	sort := func(mm []*tempopb.TraceSearchMetadata) {
		for _, m := range mm {
			for _, s := range m.SpanSet.Spans {
				sort.Slice(s.Attributes, func(i, j int) bool {
					return s.Attributes[i].Key < s.Attributes[j].Key
				})
			}
		}
	}
	sort(expectedTraceSearchMetadata)
	sort(response.Traces)

	assert.Equal(t, expectedTraceSearchMetadata, response.Traces)

	assert.Equal(t, uint64(100_00), response.Metrics.InspectedBytes)
}

func TestEngine_asTraceSearchMetadata(t *testing.T) {
	now := time.Now()

	traceID, err := util.HexStringToTraceID("123456789abcdef")
	require.NoError(t, err)
	spanID1 := traceID[:8]
	spanID2 := traceID[8:]

	spanSet := &Spanset{
		TraceID:            traceID,
		RootServiceName:    "my-service",
		RootSpanName:       "HTTP GET",
		StartTimeUnixNanos: 1000,
		DurationNanos:      uint64(time.Second.Nanoseconds()),
		Spans: []Span{
			&mockSpan{
				id:                 spanID1,
				startTimeUnixNanos: uint64(now.UnixNano()),
				durationNanos:      uint64((10 * time.Second).Nanoseconds()),
				attributes: map[Attribute]Static{
					NewIntrinsic(IntrinsicName):     NewStaticString("HTTP GET"),
					NewIntrinsic(IntrinsicStatus):   NewStaticStatus(StatusOk),
					NewIntrinsic(IntrinsicKind):     NewStaticKind(KindClient),
					NewAttribute("cluster"):         NewStaticString("prod"),
					NewAttribute("count"):           NewStaticInt(5),
					NewAttribute("count_but_float"): NewStaticFloat(5.0),
					NewAttribute("is_ok"):           NewStaticBool(true),
					NewIntrinsic(IntrinsicDuration): NewStaticDuration(10 * time.Second),
				},
			},
			&mockSpan{
				id:                 spanID2,
				startTimeUnixNanos: uint64(now.Add(2 * time.Second).UnixNano()),
				durationNanos:      uint64((20 * time.Second).Nanoseconds()),
				attributes:         map[Attribute]Static{},
			},
		},
		Attributes: []*SpansetAttribute{
			{Name: attributeMatched, Val: NewStaticInt(2)},
			{Name: "avg(duration)", Val: NewStaticFloat(15.0)},
		},
	}

	e := NewEngine()

	traceSearchMetadata := e.asTraceSearchMetadata(spanSet)

	expectedSpanset := &tempopb.SpanSet{
		Matched: 2,
		Spans: []*tempopb.Span{
			{
				SpanID:            util.SpanIDToHexString(spanID1),
				Name:              "HTTP GET",
				StartTimeUnixNano: uint64(now.UnixNano()),
				DurationNanos:     10_000_000_000,
				Attributes: []*v1.KeyValue{
					{
						Key: "cluster",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: "prod",
							},
						},
					},
					{
						Key: "count",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_IntValue{
								IntValue: 5,
							},
						},
					},
					{
						Key: "count_but_float",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_DoubleValue{
								DoubleValue: 5.0,
							},
						},
					},
					{
						Key: "is_ok",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_BoolValue{
								BoolValue: true,
							},
						},
					},
					{
						Key: "kind",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: KindClient.String(),
							},
						},
					},
					{
						Key: "status",
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: StatusOk.String(),
							},
						},
					},
				},
			},
			{
				SpanID:            util.SpanIDToHexString(spanID2),
				StartTimeUnixNano: uint64(now.Add(2 * time.Second).UnixNano()),
				DurationNanos:     20_000_000_000,
				Attributes:        nil,
			},
		},
		Attributes: []*v1.KeyValue{
			{
				Key: "avg(duration)",
				Value: &v1.AnyValue{
					Value: &v1.AnyValue_DoubleValue{
						DoubleValue: 15.0,
					},
				},
			},
		},
	}
	expectedTraceSearchMetadata := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(traceID),
		RootServiceName:   "my-service",
		RootTraceName:     "HTTP GET",
		StartTimeUnixNano: 1000,
		DurationMs:        uint32(time.Second.Milliseconds()),
		SpanSet:           expectedSpanset,
		SpanSets:          []*tempopb.SpanSet{expectedSpanset},
	}

	// Ensure attributes are sorted to avoid a flaky test
	sort.Slice(traceSearchMetadata.SpanSet.Spans[0].Attributes, func(i, j int) bool {
		return strings.Compare(traceSearchMetadata.SpanSet.Spans[0].Attributes[i].Key, traceSearchMetadata.SpanSet.Spans[0].Attributes[j].Key) == -1
	})

	assert.Equal(t, expectedTraceSearchMetadata, traceSearchMetadata)
}

var _ AutocompleteFetcher = (*MockAutocompleteFetcher)(nil)

type MockAutocompleteFetcher struct {
	query    string
	iterator SpansetIterator
}

func (m *MockAutocompleteFetcher) Fetch(ctx context.Context, req AutocompleteRequest, cb AutocompleteCallback) error {
	rootExpr, err := Parse(m.query)
	if err != nil {
		return err
	}
	if err := rootExpr.validate(); err != nil {
		return err
	}

	for {
		spanset, err := m.iterator.Next(ctx)
		if err != nil && errors.Is(err, io.EOF) {
			return err
		}
		if spanset == nil {
			break
		}
		if len(spanset.Spans) == 0 {
			continue
		}

		evalSS, _ := rootExpr.Pipeline.evaluate([]*Spanset{spanset})

		for _, ss := range evalSS {
			for _, s := range ss.Spans {
				for attr, static := range s.AllAttributes() {
					if attr.Name != req.TagName.Name {
						continue
					}
					if cb(static) {
						return nil
					}
				}
			}
		}
	}

	return nil
}

type MockSpanSetFetcher struct {
	iterator        SpansetIterator
	capturedRequest FetchSpansRequest
}

var _ = (SpansetFetcher)(&MockSpanSetFetcher{})

func (m *MockSpanSetFetcher) Fetch(_ context.Context, request FetchSpansRequest) (FetchSpansResponse, error) {
	m.capturedRequest = request
	m.iterator.(*MockSpanSetIterator).filter = request.SecondPass
	return FetchSpansResponse{
		Results: m.iterator,
		Bytes: func() uint64 {
			return 100_00 // hardcoded in tests
		},
	}, nil
}

type MockSpanSetIterator struct {
	results []*Spanset
	filter  SecondPassFn
}

func (m *MockSpanSetIterator) Next(context.Context) (*Spanset, error) {
	for {
		if len(m.results) == 0 {
			return nil, nil
		}
		r := m.results[0]
		m.results = m.results[1:]

		if m.filter == nil {
			return r, nil
		}

		ss, err := m.filter(r)
		if err != nil {
			return nil, err
		}
		if len(ss) == 0 {
			continue
		}

		r.Spans = ss[0].Spans
		return r, nil
	}
}

func (m *MockSpanSetIterator) Close() {}

func newCondition(attr Attribute, op Operator, operands ...Static) Condition {
	return Condition{
		Attribute: attr,
		Op:        op,
		Operands:  operands,
	}
}

func TestUnixSecToNano(t *testing.T) {
	now := time.Now()
	// tolerate delta's up to 1 second
	assert.InDelta(t, uint64(now.UnixNano()), unixSecToNano(uint32(now.Unix())), float64(time.Second.Nanoseconds()))
}

func TestStatic_AsAnyValue(t *testing.T) {
	tt := []struct {
		s        Static
		expected *v1.AnyValue
	}{
		{NewStaticInt(5), &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 5}}},
		{NewStaticString("foo"), &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "foo"}}},
		{NewStaticFloat(5.0), &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 5.0}}},
		{NewStaticBool(true), &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},
		{NewStaticDuration(5 * time.Second), &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "5s"}}},
		{NewStaticStatus(StatusOk), &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "ok"}}},
		{NewStaticKind(KindInternal), &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "internal"}}},
		{NewStaticNil(), &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "nil"}}},
	}
	for _, tc := range tt {
		t.Run(fmt.Sprintf("%v", tc.s), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.s.AsAnyValue())
		})
	}
}

func TestExamplesInEngine(t *testing.T) {
	b, err := os.ReadFile(testExamplesFile)
	require.NoError(t, err)

	queries := &TestQueries{}
	err = yaml.Unmarshal(b, queries)
	require.NoError(t, err)

	e := NewEngine()

	for _, q := range queries.Valid {
		t.Run("valid - "+q, func(t *testing.T) {
			_, err := e.parseQuery(&tempopb.SearchRequest{
				Query: q,
			})
			require.NoError(t, err)
		})
	}

	for _, q := range queries.ParseFails {
		t.Run("parse fails - "+q, func(t *testing.T) {
			_, err := e.parseQuery(&tempopb.SearchRequest{
				Query: q,
			})
			require.Error(t, err)
		})
	}

	for _, q := range queries.ValidateFails {
		t.Run("validate fails - "+q, func(t *testing.T) {
			_, err := e.parseQuery(&tempopb.SearchRequest{
				Query: q,
			})
			require.Error(t, err)
			var unErr *unsupportedError
			require.False(t, errors.As(err, &unErr))
		})
	}

	for _, q := range queries.Unsupported {
		t.Run("unsupported - "+q, func(t *testing.T) {
			_, err := e.parseQuery(&tempopb.SearchRequest{
				Query: q,
			})
			require.Error(t, err)
			var unErr *unsupportedError
			require.True(t, errors.As(err, &unErr))
		})
	}
}

func TestExecuteTagValues(t *testing.T) {
	// TODO: This test is stupid, it's using the traceql engine to execute the query
	//  and doesn't actually test the ExecuteTagValues function
	now := time.Now()
	e := Engine{}

	mockSpansetFetcher := func(query string) AutocompleteFetcher {
		return &MockAutocompleteFetcher{
			query: query,
			iterator: &MockSpanSetIterator{
				results: []*Spanset{
					{
						TraceID:         []byte{1},
						RootSpanName:    "HTTP GET",
						RootServiceName: "my-service",
						Spans: []Span{
							&mockSpan{
								id: []byte{1},
								attributes: map[Attribute]Static{
									NewIntrinsic(IntrinsicName):                                       NewStaticString("HTTP POST /api/v1/users"),
									NewScopedAttribute(AttributeScopeSpan, false, "http.method"):      NewStaticString("POST"),
									NewScopedAttribute(AttributeScopeSpan, false, "http.target"):      NewStaticString("/api/v1/users"),
									NewScopedAttribute(AttributeScopeResource, false, "service.name"): NewStaticString("my-service"),
								},
							},
							&mockSpan{
								id:                 []byte{2},
								startTimeUnixNanos: uint64(now.UnixNano()),
								durationNanos:      uint64((100 * time.Millisecond).Nanoseconds()),
								attributes: map[Attribute]Static{
									NewIntrinsic(IntrinsicName):                                       NewStaticString("redis call"),
									NewScopedAttribute(AttributeScopeResource, false, "service.name"): NewStaticString("my-service"),
								},
							},
						},
					},
					{
						TraceID:         []byte{2},
						RootSpanName:    "HTTP POST",
						RootServiceName: "my-service",
						Spans: []Span{
							&mockSpan{
								id: []byte{3},
								attributes: map[Attribute]Static{
									NewIntrinsic(IntrinsicName):                                       NewStaticString("HTTP GET /status"),
									NewScopedAttribute(AttributeScopeSpan, false, "http.method"):      NewStaticString("GET"),
									NewScopedAttribute(AttributeScopeSpan, false, "http.target"):      NewStaticString("/status"),
									NewScopedAttribute(AttributeScopeResource, false, "service.name"): NewStaticString("my-service"),
								},
							},
						},
					},
				},
			},
		}
	}

	type testCase struct {
		name             string
		attribute, query string
		expectedValues   []tempopb.TagValue
	}

	for _, tc := range []testCase{
		{
			name:           "scoped param, no query",
			attribute:      "resource.service.name",
			query:          "{}",
			expectedValues: []tempopb.TagValue{{Type: "string", Value: "my-service"}},
		},
		{
			name:      "intrinsic param, no query",
			attribute: "name",
			query:     "{}",
			expectedValues: []tempopb.TagValue{
				{Type: "string", Value: "HTTP GET /status"},
				{Type: "string", Value: "HTTP POST /api/v1/users"},
				{Type: "string", Value: "redis call"},
			},
		},
		{
			name:           "scoped param, with query",
			attribute:      "span.http.method",
			query:          `{ span.http.target = "/api/v1/users" }`,
			expectedValues: []tempopb.TagValue{{Type: "string", Value: "POST"}},
		},
		{
			name:           "intrinsic param, with query",
			attribute:      "name",
			query:          `{ span.http.target = "/api/v1/users" }`,
			expectedValues: []tempopb.TagValue{{Type: "string", Value: "HTTP POST /api/v1/users"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			distinctValues := util.NewDistinctValueCollector[tempopb.TagValue](100_000, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

			tag, err := ParseIdentifier(tc.attribute)
			assert.NoError(t, err)
			assert.NoError(t, e.ExecuteTagValues(context.Background(), tag, tc.query, MakeCollectTagValueFunc(distinctValues.Collect), mockSpansetFetcher(tc.query)))
			values := distinctValues.Values()
			sort.Slice(values, func(i, j int) bool {
				return values[i].Value < values[j].Value
			})
			assert.Equal(t, tc.expectedValues, values)
		})
	}
}
