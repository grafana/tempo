package usage

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func testConfig() Config {
	return Config{
		Enabled:        true,
		MaxCardinality: defaultMaxCardinality,
		StaleDuration:  defaultStaleDuration,
		PurgePeriod:    defaultPurgePeriod,
	}
}

func TestUsageTracker(t *testing.T) {
	type testcase struct {
		name       string
		max        int
		dimensions map[string]string
		expected   map[uint64]*bucket
	}

	// Reused for all test cases
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
		dimensions map[string]string
		expected   map[uint64]*bucket
	)

	// -------------------------------------------------------------
	// Test case 1 - Group by service.name, entire batch is 1 series
	// -------------------------------------------------------------
	name = "standard"
	dimensions = map[string]string{"service.name": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"service_name"}, map[string]string{"service_name": "svc"})] = &bucket{
		labels: []string{"svc"},
		bytes:  uint64(data[0].Size()),
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 2 - Group by attr, batch is split 75%/25%
	// -------------------------------------------------------------
	name = "splitbatch"
	dimensions = map[string]string{"attr": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"attr"}, map[string]string{"attr": "1"})] = &bucket{
		labels: []string{"1"},
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.75)),
	}
	expected[hash([]string{"attr"}, map[string]string{"attr": "2"})] = &bucket{
		labels: []string{"2"},
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.25)),
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 3 - Missing values go into series without that label
	// -------------------------------------------------------------
	name = "missing"
	dimensions = map[string]string{"foo": ""}
	expected = make(map[uint64]*bucket)
	expected[emptyHash] = &bucket{
		labels: nil, // No spans have "foo" so they all go into an unlabeled series
		bytes:  uint64(float64(data[0].Size())),
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 4 - Max cardinality
	// -------------------------------------------------------------
	name = "maxcardinality"
	dimensions = map[string]string{"attr": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"attr"}, map[string]string{"attr": "1"})] = &bucket{
		labels: []string{"1"},
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.75)), // attr=1 is encountered first and recorded, with 75% of spans
	}
	expected[emptyHash] = &bucket{
		labels: nil,
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.25)), // attr=2 doesn't fit within cardinality and those 25% of spans go into the unlabled series.
	}
	testCases = append(testCases, testcase{
		name:       name,
		max:        1,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 5 - Multiple labels with rename
	// Multiple dimensions are renamed into the same output label
	// -------------------------------------------------------------
	name = "maxcardinality"
	dimensions = map[string]string{
		"service.name": "foo",
		"attr":         "foo",
	}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"foo"}, map[string]string{"foo": "1"})] = &bucket{
		labels: []string{"1"},
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.75)), // attr=1 is encountered first and recorded, with 75% of spans
	}
	expected[hash([]string{"foo"}, map[string]string{"foo": "2"})] = &bucket{
		labels: []string{"2"},
		bytes:  uint64(math.RoundToEven(float64(data[0].Size()) * 0.25)), // attr=2 doesn't fit within cardinality and those 25% of spans go into the unlabled series.
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			if tc.max > 0 {
				cfg.MaxCardinality = uint64(tc.max)
			}

			u, err := NewTracker(cfg, "test", func(_ string) map[string]string { return tc.dimensions }, func(_ string) uint64 { return 0 })
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

func BenchmarkUsageTrackerObserve(b *testing.B) {
	var (
		tr       = test.MakeTrace(10, nil)
		dims     = map[string]string{"service.name": "service_name"}
		labelsFn = func(_ string) map[string]string { return dims } // Allocation outside the function to not influence benchmark
		maxFn    = func(_ string) uint64 { return 0 }
	)

	u, err := NewTracker(testConfig(), "test", labelsFn, maxFn)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		u.Observe("test", tr.ResourceSpans)
	}
}

func BenchmarkUsageTrackerCollect(b *testing.B) {
	var (
		tr       = test.MakeTrace(10, nil)
		dims     = map[string]string{"service.name": ""}
		labelsFn = func(_ string) map[string]string { return dims } // Allocation outside the function to not influence benchmark
		maxFn    = func(_ string) uint64 { return 0 }
		req      = httptest.NewRequest("", "/", nil)
		resp     = &NoopHTTPResponseWriter{}
	)

	u, err := NewTracker(testConfig(), "test", labelsFn, maxFn)
	require.NoError(b, err)

	u.Observe("test", tr.ResourceSpans)

	handler := u.Handler()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(resp, req)
	}
}

type NoopHTTPResponseWriter struct {
	headers map[string][]string
}

var _ http.ResponseWriter = (*NoopHTTPResponseWriter)(nil)

func (n *NoopHTTPResponseWriter) Header() http.Header {
	if n.headers == nil {
		n.headers = make(map[string][]string)
	}
	return n.headers
}
func (NoopHTTPResponseWriter) Write(buf []byte) (int, error) { return len(buf), nil }
func (NoopHTTPResponseWriter) WriteHeader(_ int)             {}
