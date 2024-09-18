package usage

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestUsageTracker(t *testing.T) {
	type testcase struct {
		name       string
		cfg        Config
		dimensions []string
		expected   map[uint64]*bucket
	}

	// Reused for all test cases
	cfg := Config{
		MaxCardinality: defaultMaxCardinality,
		StaleDuration:  defaultStaleDuration,
		PurgePeriod:    defaultPurgePeriod,
	}
	data := []*v1.ResourceSpans{
		{
			Resource: &v1resource.Resource{
				Attributes: []*v1common.KeyValue{
					{
						Key:   "service.name",
						Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "svc"}},
					},
				},
			},
			ScopeSpans: []*v1.ScopeSpans{
				{
					Spans: []*v1.Span{
						{
							Attributes: []*v1common.KeyValue{
								{Key: "attr", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "1"}}},
							},
						},
						{
							Attributes: []*v1common.KeyValue{
								{Key: "attr", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "1"}}},
							},
						},
						{
							Attributes: []*v1common.KeyValue{
								{Key: "attr", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "2"}}},
							},
						},
						{
							Attributes: []*v1common.KeyValue{
								{Key: "attr", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "1"}}},
							},
						},
					},
					SchemaUrl: "test",
				},
			},
			SchemaUrl: "test",
		},
	}

	var (
		testCases  []testcase
		name       string
		dimensions []string
		labels     []string
		expected   map[uint64]*bucket
	)

	// -------------------------------------------------------------
	// Test case 1 - Group by service.name, entire batch is 1 series
	// -------------------------------------------------------------
	name = "standard"
	dimensions = []string{"service.name"}
	labels = []string{"service_name"}
	expected = make(map[uint64]*bucket)
	expected[hash(labels, []string{"svc"})] = &bucket{
		labels: []string{"svc"},
		bytes:  uint64(data[0].Size()),
	}
	testCases = append(testCases, testcase{
		name:       name,
		cfg:        cfg,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 2 - Group by foo Batch is split 75%/25%
	// -------------------------------------------------------------
	name = "splitbatch"
	dimensions = []string{"attr"}
	labels = []string{"attr"}
	expected = make(map[uint64]*bucket)
	expected[hash(labels, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  uint64(float64(data[0].Size()) * 0.75),
	}
	expected[hash(labels, []string{"2"})] = &bucket{
		labels: []string{"2"},
		bytes:  uint64(float64(data[0].Size()) * 0.25),
	}
	testCases = append(testCases, testcase{
		name:       name,
		cfg:        cfg,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 3 - Missing values go into series without that label
	// -------------------------------------------------------------
	name = "missing"
	dimensions = []string{"foo"}
	labels = []string{"foo"}
	expected = make(map[uint64]*bucket)
	expected[hash(labels, []string{})] = &bucket{
		labels: nil, // No spans have "foo" so they all go into an unlabeled series
		bytes:  uint64(float64(data[0].Size())),
	}
	testCases = append(testCases, testcase{
		name:       name,
		cfg:        cfg,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 4 - Max cardinality
	// -------------------------------------------------------------
	name = "maxcardinality"
	dimensions = []string{"attr"}
	labels = []string{"attr"}
	cfg.MaxCardinality = 1
	expected = make(map[uint64]*bucket)
	expected[hash(labels, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  uint64(float64(data[0].Size()) * 0.75), // attr=1 is encountered first and record, with 75% of spans
	}
	expected[hash(labels, nil)] = &bucket{
		labels: nil,
		bytes:  uint64(float64(data[0].Size()) * 0.25), // attr=2 doesn't fit within cardinality and those 25% of spans go into the unlabled series.
	}
	testCases = append(testCases, testcase{
		name:       name,
		cfg:        cfg,
		dimensions: dimensions,
		expected:   expected,
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := NewTracker(tc.cfg, "test", func(s string) []string { return tc.dimensions })
			require.NoError(t, err)

			u.Observe("test", data)
			actual := u.tenants["test"].series

			require.Equal(t, len(tc.expected), len(actual))
			for expectedHash, expectedBucket := range tc.expected {
				require.Equal(t, expectedBucket.labels, actual[expectedHash].labels)
				require.Equal(t, expectedBucket.bytes, actual[expectedHash].bytes)
			}
		})
	}
}

func BenchmarkUsageTracker(b *testing.B) {
	tr := test.MakeTrace(10, nil)
	dims := []string{"service.name"}

	u, err := NewTracker(Config{}, "test", func(s string) []string { return dims })
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		u.Observe("test", tr.ResourceSpans)
	}
}
