package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1_common "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/common/v1"
)

func TestFindServiceName(t *testing.T) {
	testCases := []struct {
		name                string
		attributes          []*v1_common.KeyValue
		expectedServiceName string
		expectedOk          bool
	}{
		{
			"empty attributes",
			nil,
			"",
			false,
		},
		{
			"service name",
			[]*v1_common.KeyValue{
				{
					Key: "cluster",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test",
						},
					},
				},
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "my-service",
						},
					},
				},
			},
			"my-service",
			true,
		},
		{
			"service name",
			[]*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "",
						},
					},
				},
			},
			"",
			true,
		},
		{
			"no service name",
			[]*v1_common.KeyValue{
				{
					Key: "cluster",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test",
						},
					},
				},
			},
			"",
			false,
		},
		{
			"service name is other type",
			[]*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_BoolValue{
							BoolValue: false,
						},
					},
				},
			},
			"false",
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svcName, ok := FindServiceName(tc.attributes)

			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedServiceName, svcName)
		})
	}
}
