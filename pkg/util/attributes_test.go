package util

import (
	"testing"

	v1common "github.com/grafana/tempo/v2/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/assert"
)

func TestStringifyAnyValue(t *testing.T) {
	testCases := []struct {
		name     string
		v        *v1common.AnyValue
		expected string
	}{
		{
			name: "string value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_StringValue{
					StringValue: "test",
				},
			},
			expected: "test",
		},
		{
			name: "int value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_IntValue{
					IntValue: 1,
				},
			},
			expected: "1",
		},
		{
			name: "bool value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_BoolValue{
					BoolValue: true,
				},
			},
			expected: "true",
		},
		{
			name: "float value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_DoubleValue{
					DoubleValue: 1.1,
				},
			},
			expected: "1.1",
		},
		{
			name: "array value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_ArrayValue{
					ArrayValue: &v1common.ArrayValue{
						Values: []*v1common.AnyValue{
							{
								Value: &v1common.AnyValue_StringValue{
									StringValue: "test",
								},
							},
						},
					},
				},
			},
			expected: "[test]",
		},
		{
			name: "map value",
			v: &v1common.AnyValue{
				Value: &v1common.AnyValue_KvlistValue{
					KvlistValue: &v1common.KeyValueList{
						Values: []*v1common.KeyValue{
							{
								Key:   "key",
								Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "value"}},
							},
						},
					},
				},
			},
			expected: "{key:value}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			str := StringifyAnyValue(tc.v)
			assert.Equal(t, tc.expected, str)
		})
	}
}
