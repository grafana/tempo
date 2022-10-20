package traceql

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
)

func TestEngine_Execute(t *testing.T) {
	now := time.Now()
	e := Engine{}

	req := &tempopb.SearchRequest{
		Query: `{ .foo = .bar }`,
	}
	spanSetFetcher := MockSpanSetFetcher{
		iterator: &MockSpanSetIterator{
			results: []*Spanset{
				{
					TraceID:         []byte{1},
					RootSpanName:    "HTTP GET",
					RootServiceName: "my-service",
					Spans: []Span{
						{
							ID: []byte{1},
							Attributes: map[Attribute]Static{
								NewAttribute("foo"): NewStaticString("value"),
							},
						},
						{
							ID:                 []byte{2},
							StartTimeUnixNanos: uint64(now.UnixNano()),
							EndtimeUnixNanos:   uint64(now.Add(100 * time.Millisecond).UnixNano()),
							Attributes: map[Attribute]Static{
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
						{
							ID: []byte{3},
							Attributes: map[Attribute]Static{
								NewAttribute("bar"): NewStaticString("value"),
							},
						},
					},
				},
			},
		},
	}
	response, err := e.Execute(context.Background(), req, &spanSetFetcher)

	require.NoError(t, err)

	expectedFetchSpansRequest := FetchSpansRequest{
		Conditions: []Condition{
			newCondition(NewAttribute("foo"), OpNone),
			newCondition(NewAttribute("bar"), OpNone),
		},
		AllConditions: true,
	}
	assert.Equal(t, expectedFetchSpansRequest, spanSetFetcher.capturedRequest)

	expectedTraceSearchMetadata := []*tempopb.TraceSearchMetadata{
		{
			TraceID:           "1",
			RootServiceName:   "my-service",
			RootTraceName:     "HTTP GET",
			StartTimeUnixNano: 0,
			DurationMs:        0,
			SpanSet: &tempopb.SpanSet{
				Spans: []*tempopb.Span{
					{
						SpanID:            "2",
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
				},
				Matched: 1,
			},
		},
	}
	assert.Equal(t, expectedTraceSearchMetadata, response.Traces)
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
			{
				ID:                 spanID1,
				StartTimeUnixNanos: uint64(now.UnixNano()),
				EndtimeUnixNanos:   uint64(now.Add(10 * time.Second).UnixNano()),
				Attributes: map[Attribute]Static{
					NewIntrinsic(IntrinsicStatus):   NewStaticStatus(StatusOk),
					NewAttribute("cluster"):         NewStaticString("prod"),
					NewAttribute("count"):           NewStaticInt(5),
					NewAttribute("count_but_float"): NewStaticFloat(5.0),
					NewAttribute("is_ok"):           NewStaticBool(true),
					NewIntrinsic(IntrinsicDuration): NewStaticDuration(10 * time.Second),
				},
			},
			{
				ID:                 spanID2,
				StartTimeUnixNanos: uint64(now.Add(2 * time.Second).UnixNano()),
				EndtimeUnixNanos:   uint64(now.Add(20 * time.Second).UnixNano()),
				Attributes:         map[Attribute]Static{},
			},
		},
	}

	e := NewEngine()

	traceSearchMetadata, err := e.asTraceSearchMetadata(spanSet)
	require.NoError(t, err)

	expectedTraceSearchMetadata := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(traceID),
		RootServiceName:   "my-service",
		RootTraceName:     "HTTP GET",
		StartTimeUnixNano: 1000,
		DurationMs:        uint32(time.Second.Milliseconds()),
		SpanSet: &tempopb.SpanSet{
			Matched: 2,
			Spans: []*tempopb.Span{
				{
					SpanID:            util.TraceIDToHexString(spanID1),
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
							Key: "duration",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{
									StringValue: "10s",
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
					SpanID:            util.TraceIDToHexString(spanID2),
					StartTimeUnixNano: uint64(now.Add(2 * time.Second).UnixNano()),
					DurationNanos:     18_000_000_000,
					Attributes:        nil,
				},
			},
		},
	}

	// Ensure attributes are sorted to avoid a flaky test
	sort.Slice(traceSearchMetadata.SpanSet.Spans[0].Attributes, func(i, j int) bool {
		return strings.Compare(traceSearchMetadata.SpanSet.Spans[0].Attributes[i].Key, traceSearchMetadata.SpanSet.Spans[0].Attributes[j].Key) == -1
	})

	assert.Equal(t, expectedTraceSearchMetadata, traceSearchMetadata)
}

type MockSpanSetFetcher struct {
	iterator        SpansetIterator
	capturedRequest FetchSpansRequest
}

func (m *MockSpanSetFetcher) Fetch(ctx context.Context, request FetchSpansRequest) (FetchSpansResponse, error) {
	m.capturedRequest = request
	return FetchSpansResponse{
		Results: m.iterator,
	}, nil
}

type MockSpanSetIterator struct {
	results []*Spanset
}

func (m *MockSpanSetIterator) Next(ctx context.Context) (*Spanset, error) {
	if len(m.results) == 0 {
		return nil, nil
	}
	r := m.results[0]
	m.results = m.results[1:]
	return r, nil
}

func newCondition(attr Attribute, op Operator, operands ...Static) Condition {
	return Condition{
		Attribute: attr,
		Op:        op,
		Operands:  operands,
	}
}
