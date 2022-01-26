package model

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches(t *testing.T) {
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
								StartTimeUnixNano: uint64(10 * time.Second),
								EndTimeUnixNano:   uint64(20 * time.Second),
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
		for _, e := range allEncodings {
			t.Run(tc.name+":"+e, func(t *testing.T) {
				d := MustNewDecoder(e)

				obj, err := d.(encoderDecoder).Marshal(tc.trace)
				require.NoError(t, err)

				actual, err := d.Matches([]byte{0x01}, obj, tc.req)
				require.NoError(t, err)

				assert.Equal(t, tc.expected, actual)
			})
		}
	}
}

func TestMatchesFails(t *testing.T) {
	for _, e := range allEncodings {
		_, err := MustNewDecoder(e).Matches([]byte{0x01}, []byte{0x02, 0x03}, nil)
		assert.Error(t, err)
	}
}
