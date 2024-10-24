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

func testConfig() PerTrackerConfig {
	return PerTrackerConfig{
		Enabled:        true,
		MaxCardinality: defaultMaxCardinality,
		StaleDuration:  defaultStaleDuration,
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
								{Key: "attr2", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "attr2Value"}}},
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
	nonSpanSize, _ := nonSpanDataLength(data[0])

	// Helper functions for dividing up data sizes
	nonSpanRatio := func(r float64) uint64 {
		return uint64(math.RoundToEven(float64(nonSpanSize) * r))
	}

	spanSize := func(i int) uint64 {
		sz := data[0].ScopeSpans[0].Spans[i].Size()
		sz += protoLengthMath(sz)
		return uint64(sz)
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
	expected[hash([]string{"service_name"}, []string{"svc"})] = &bucket{
		labels: []string{"svc"},
		bytes:  uint64(data[0].Size()), // The entire batch is included, with the exact number of bytes
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
	expected[hash([]string{"attr"}, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  nonSpanRatio(0.75) + spanSize(0) + spanSize(1) + spanSize(3),
	}
	expected[hash([]string{"attr"}, []string{"2"})] = &bucket{
		labels: []string{"2"},
		bytes:  nonSpanRatio(0.25) + spanSize(2),
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 3 - Missing labels are set to __missing__
	// -------------------------------------------------------------
	name = "missing"
	dimensions = map[string]string{"foo": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"foo"}, []string{missingLabel})] = &bucket{
		labels: []string{missingLabel}, // No spans have "foo" so it is assigned to the missingvalue
		bytes:  uint64(data[0].Size()),
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
	expected[hash([]string{"attr"}, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  nonSpanRatio(0.75) + spanSize(0) + spanSize(1) + spanSize(3), // attr=1 is encountered first and recorded, with 75% of spans
	}
	expected[hash([]string{"attr"}, []string{overflowLabel})] = &bucket{
		labels: []string{overflowLabel},
		bytes:  nonSpanRatio(0.25) + spanSize(2), // attr=2 doesn't fit within cardinality and those 25% of spans go into the overflow series.
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
	name = "rename"
	dimensions = map[string]string{
		"service.name": "foo",
		"attr":         "foo",
	}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"foo"}, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  nonSpanRatio(0.75) + spanSize(0) + spanSize(1) + spanSize(3),
	}
	expected[hash([]string{"foo"}, []string{"2"})] = &bucket{
		labels: []string{"2"},
		bytes:  nonSpanRatio(0.25) + spanSize(2),
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 6 - Some spans missing value
	// Some spans within the same batch are missing values and
	// should continue to inherit the batch value
	// -------------------------------------------------------------
	name = "partially_missing"
	dimensions = map[string]string{
		"attr2": "",
	}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"attr2"}, []string{"attr2Value"})] = &bucket{
		labels: []string{"attr2Value"},
		bytes:  nonSpanRatio(0.25) + spanSize(0),
	}
	expected[hash([]string{"attr2"}, []string{missingLabel})] = &bucket{
		labels: []string{missingLabel},
		bytes:  nonSpanRatio(0.75) + spanSize(1) + spanSize(2) + spanSize(3),
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

			// Ensure total bytes recorded exactly matches the batch
			total := 0
			for _, b := range actual {
				total += int(b.bytes)
			}
			require.Equal(t, data[0].Size(), total, "total")

			for expectedHash, expectedBucket := range tc.expected {
				require.Equal(t, expectedBucket.labels, actual[expectedHash].labels)
				// To make testing less brittle from rounding, just ensure that each series
				// is within 1 byte of expected. We already ensured the total is 100% accurate above.
				require.InDelta(t, expectedBucket.bytes, actual[expectedHash].bytes, 1.0)
			}
		})
	}
}

func BenchmarkUsageTrackerObserve(b *testing.B) {
	var (
		tr   = test.MakeTrace(10, nil)
		dims = map[string]string{"service.name": "service_name"}
		// dims     = map[string]string{"key": ""} // To benchmark span-level attribute
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
