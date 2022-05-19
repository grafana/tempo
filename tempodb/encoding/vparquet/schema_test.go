package vparquet

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestEventRoundtrip(t *testing.T) {

	// Test proto -> parquet -> proto

	e := &v1_trace.Span_Event{
		// Basic fields
		TimeUnixNano:           123,
		Name:                   "456",
		DroppedAttributesCount: 789,

		// An attribute of every type
		Attributes: []*v1.KeyValue{
			// String
			{Key: "s", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s2"}}},

			// Int
			{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},

			// Double
			{Key: "d", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 123.456}}},

			// Bool
			{Key: "b", Value: &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},

			// KVList
			{Key: "kv", Value: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{Values: []*v1.KeyValue{
				{Key: "s2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s3"}}},
				{Key: "i2", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 789}}},
			}}}}},

			// Array
			{Key: "a", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{
				{Value: &v1.AnyValue_StringValue{StringValue: "s4"}},
				{Value: &v1.AnyValue_IntValue{IntValue: 101112}},
			}}}}},
		},
	}

	pq := eventToParquet(e)
	e2 := parquetToProtoEvents([]Event{pq})[0]
	require.Equal(t, e, e2)
}
