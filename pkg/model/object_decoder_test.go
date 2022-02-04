package model

import (
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectDecoderMarshalUnmarshal(t *testing.T) {
	empty := &tempopb.Trace{}

	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			encoding, err := NewObjectDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)
			bytes := mustMarshalToObject(trace, e)

			actual, err := encoding.PrepareForRead(bytes)
			require.NoError(t, err)
			assert.True(t, proto.Equal(trace, actual))

			// nil trace
			actual, err = encoding.PrepareForRead(nil)
			assert.NoError(t, err)
			assert.True(t, proto.Equal(empty, actual))

			// empty byte slice
			actual, err = encoding.PrepareForRead([]byte{})
			assert.NoError(t, err)
			assert.True(t, proto.Equal(empty, actual))
		})
	}
}

func TestMatches(t *testing.T) {
	startSeconds := 10
	endSeconds := 20

	testTrace := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				Resource: &v1resource.Resource{
					Attributes: []*v1common.KeyValue{
						{
							Key:   "service.name",
							Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "svc"}},
						},
						{
							Key:   "cluster",
							Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "prod"}},
						},
					},
				},
				InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
					{
						Spans: []*v1.Span{
							{
								Name:              "test",
								StartTimeUnixNano: uint64(time.Duration(startSeconds) * time.Second),
								EndTimeUnixNano:   uint64(time.Duration(endSeconds) * time.Second),
								Attributes: []*v1common.KeyValue{
									{
										Key:   "foo",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "barricus"}},
									},
									{
										Key:   "intfoo",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_IntValue{IntValue: 42}},
									},
									{
										Key:   "floatfoo",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_DoubleValue{DoubleValue: 42.42}},
									},
									{
										Key:   "boolfoo",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_BoolValue{BoolValue: true}},
									},
								},
								Status: &v1.Status{
									Code: v1.Status_STATUS_CODE_OK,
								},
							},
						},
					},
				},
			},
			{
				Resource: &v1resource.Resource{
					Attributes: []*v1common.KeyValue{
						{
							Key:   "service.name",
							Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "svc2"}},
						},
					},
				},
				InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
					{
						Spans: []*v1.Span{
							{
								Name:              "test2",
								StartTimeUnixNano: uint64(time.Duration(startSeconds) * time.Second),
								EndTimeUnixNano:   uint64(time.Duration(endSeconds) * time.Second),
								Attributes: []*v1common.KeyValue{
									{
										Key:   "foo2",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "barricus2"}},
									},
								},
								Status: &v1.Status{
									Code: v1.Status_STATUS_CODE_OK,
								},
							},
						},
					},
				},
			},
		},
	}
	testMetadata := &tempopb.TraceSearchMetadata{
		TraceID:           "1",
		RootServiceName:   "svc",
		RootTraceName:     "test",
		StartTimeUnixNano: uint64(10 * time.Second),
		DurationMs:        10000,
	}

	tests := []struct {
		name     string
		trace    *tempopb.Trace
		req      *tempopb.SearchRequest
		expected *tempopb.TraceSearchMetadata
	}{
		{
			name:  "range before doesn't match",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 0,
				End:   5,
			},
			expected: nil,
		},
		{
			name:  "range after doesn't match",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 25,
				End:   30,
			},
			expected: nil,
		},
		{
			name:  "encompassing range matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 5,
				End:   30,
			},
			expected: testMetadata,
		},
		{
			name:  "encompassed range matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
			},
			expected: testMetadata,
		},
		{
			name:  "overlap start matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 8,
				End:   15,
			},
			expected: testMetadata,
		},
		{
			name:  "overlap end matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   25,
			},
			expected: testMetadata,
		},
		{
			name:  "max duration excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 1,
			},
			expected: nil,
		},
		{
			name:  "max duration includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 10000,
			},
			expected: testMetadata,
		},
		{
			name:  "min duration excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 1,
				MinDurationMs: 10000,
			},
			expected: nil,
		},
		{
			name:  "min duration includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 10000,
				MinDurationMs: 5000,
			},
			expected: testMetadata,
		},
		{
			name:  "string tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "baz"},
			},
			expected: nil,
		},
		{
			name:  "string tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar"},
			},
			expected: testMetadata,
		},
		{
			name:  "resource tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"service.name": "svc"},
			},
			expected: testMetadata,
		},
		{
			name:  "int tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"intfoo": "blerg"},
			},
			expected: nil,
		},
		{
			name:  "int tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"intfoo": "42"},
			},
			expected: testMetadata,
		},
		{
			name:  "float tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"floatfoo": "42.4323"},
			},
			expected: nil,
		},
		{
			name:  "float tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"floatfoo": "42.42"},
			},
			expected: testMetadata,
		},
		{
			name:  "bool tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"boolfoo": "False"},
			},
			expected: nil,
		},
		{
			name:  "bool tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"boolfoo": "true"},
			},
			expected: testMetadata,
		},
		{
			name:  "one includes/one excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar", "boolfoo": "False"},
			},
			expected: nil,
		},
		{
			name:  "one includes/resource tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar", "service.name": "blerg"},
			},
			expected: nil,
		},
		{
			name:  "both include. one resource tag",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar", "service.name": "svc"},
			},
			expected: testMetadata,
		},
		{
			name:  "both include",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar", "boolfoo": "true"},
			},
			expected: testMetadata,
		},
		{
			name:  "both include across batches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"foo": "bar", "service.name": "svc2"},
			},
			expected: testMetadata,
		},
		{
			name:  "two resource tags. one excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"cluster": "prod", "service.name": "not"},
			},
			expected: nil,
		},
		{
			name:  "name includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"name": "test"},
			},
			expected: testMetadata,
		},
		{
			name:  "name excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"name": "no"},
			},
			expected: nil,
		},
		{
			name:  "name excludes with resource tag",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"name": "no", "cluster": "prod"},
			},
			expected: nil,
		},
		{
			name:  "name excludes with span tag",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"name": "no", "foo": "barricus"},
			},
			expected: nil,
		},
		{
			name:  "error excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"error": "true"},
			},
			expected: nil,
		},
		{
			name:  "status.code excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"status.code": "error"},
			},
			expected: nil,
		},
		{
			name:  "status.code includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
				Tags:  map[string]string{"status.code": "ok"},
			},
			expected: testMetadata,
		},
	}

	for _, tc := range tests {
		for _, e := range AllEncodings {
			t.Run(tc.name+":"+e, func(t *testing.T) {
				d := MustNewObjectDecoder(e)
				obj := mustMarshalToObjectWithRange(tc.trace, e, uint32(startSeconds), uint32(endSeconds))

				actual, err := d.Matches([]byte{0x01}, obj, tc.req)
				require.NoError(t, err)

				assert.Equal(t, tc.expected, actual)
			})
		}
	}
}

func TestMatchesFails(t *testing.T) {
	for _, e := range AllEncodings {
		_, err := MustNewObjectDecoder(e).Matches([]byte{0x01}, []byte{0x02, 0x03}, nil)
		assert.Error(t, err)
	}
}

func TestCombines(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	trace.SortTrace(t1)
	trace.SortTrace(t2)

	// split t2 into 3 traces
	t2a := &tempopb.Trace{}
	t2b := &tempopb.Trace{}
	t2c := &tempopb.Trace{}
	for _, b := range t2.Batches {
		switch rand.Int() % 3 {
		case 0:
			t2a.Batches = append(t2a.Batches, b)
		case 1:
			t2b.Batches = append(t2b.Batches, b)
		case 2:
			t2c.Batches = append(t2c.Batches, b)
		}
	}

	for _, e := range AllEncodings {
		tests := []struct {
			name          string
			traces        [][]byte
			expected      *tempopb.Trace
			expectedStart uint32
			expectedEnd   uint32
			expectError   bool
		}{
			{
				name:          "one trace",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "same trace - replace end",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 30, 40)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   40,
			},
			{
				name:          "same trace - replace start",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 5, 15)},
				expected:      t1,
				expectedStart: 5,
				expectedEnd:   20,
			},
			{
				name:          "same trace - replace both",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 5, 30)},
				expected:      t1,
				expectedStart: 5,
				expectedEnd:   30,
			},
			{
				name:          "same trace - replace neither",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 12, 14)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "3 traces",
				traces:        [][]byte{mustMarshalToObjectWithRange(t2a, e, 10, 20), mustMarshalToObjectWithRange(t2b, e, 5, 15), mustMarshalToObjectWithRange(t2c, e, 20, 30)},
				expected:      t2,
				expectedStart: 5,
				expectedEnd:   30,
			},
			{
				name:          "nil trace",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), nil},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "nil trace 2",
				traces:        [][]byte{nil, mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:        "bad trace",
				traces:      [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), {0x01, 0x02}},
				expectError: true,
			},
			{
				name:        "bad trace 2",
				traces:      [][]byte{{0x01, 0x02}, mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name+"-"+e, func(t *testing.T) {
				d := MustNewObjectDecoder(e)
				actualBytes, err := d.Combine(tt.traces...)

				if tt.expectError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}

				if tt.expected != nil {
					actual, err := d.PrepareForRead(actualBytes)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, actual)

					start, end, err := d.FastRange(actualBytes)
					if err == decoder.ErrUnsupported {
						return
					}
					require.NoError(t, err)
					assert.Equal(t, tt.expectedStart, start)
					assert.Equal(t, tt.expectedEnd, end)
				}
			})
		}
	}
}
