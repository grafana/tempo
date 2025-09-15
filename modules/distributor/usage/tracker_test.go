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

	// -------------------------------------------------------------
	// Test case 7 - Resource scoped attribute only
	// -------------------------------------------------------------
	name = "resource_scoped"
	dimensions = map[string]string{"resource.service.name": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"service_name"}, []string{"svc"})] = &bucket{
		labels: []string{"svc"},
		bytes:  uint64(data[0].Size()), // Uses resource-level service.name value
	}
	testCases = append(testCases, testcase{
		name:       name,
		dimensions: dimensions,
		expected:   expected,
	})

	// -------------------------------------------------------------
	// Test case 8 - Span scoped attribute (exists in span data)
	// -------------------------------------------------------------
	name = "span_scoped"
	dimensions = map[string]string{"span.attr": ""}
	expected = make(map[uint64]*bucket)
	expected[hash([]string{"attr"}, []string{"1"})] = &bucket{
		labels: []string{"1"},
		bytes:  nonSpanRatio(0.75) + spanSize(0) + spanSize(1) + spanSize(3), // attr=1 in 75% of spans
	}
	expected[hash([]string{"attr"}, []string{"2"})] = &bucket{
		labels: []string{"2"},
		bytes:  nonSpanRatio(0.25) + spanSize(2), // attr=2 in 25% of spans
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

func TestScopeAwareAttributeMatching(t *testing.T) {
	// Test data with both resource and span attributes having the same key
	data := []*v1.ResourceSpans{
		{
			Resource: &v1resource.Resource{
				Attributes: []*v1common.KeyValue{
					{
						Key:   "service.name",
						Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "resource-service"}},
					},
					{
						Key:   "k8s.namespace.name",
						Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "resource-namespace"}},
					},
					{
						Key:   "team.name",
						Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "resource-team"}},
					},
				},
			},
			ScopeSpans: []*v1.ScopeSpans{
				{
					Spans: []*v1.Span{
						{
							Attributes: []*v1common.KeyValue{
								{Key: "service.name", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "span-service"}}},
								{Key: "k8s.namespace.name", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "span-namespace"}}},
								{Key: "db.system", Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "postgresql"}}},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		dimensions    map[string]string
		expectSeries  func(t *testing.T, series map[uint64]*bucket)
		expectMapping func(t *testing.T, mapping []mapping)
	}{
		{
			// should only get value from resource scope and not from the spans
			name:       "resource scoped attribute",
			dimensions: map[string]string{"resource.service.name": "service.name"},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"service_name"}, []string{"resource-service"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"resource-service"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, mapping []mapping) {
				require.Equal(t, "service.name", mapping[0].from)
				require.Equal(t, ScopeResource, mapping[0].scope)
			},
		},
		{
			// should only use span-level value
			name:       "span scoped attribute",
			dimensions: map[string]string{"span.db.system": "database"},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"database"}, []string{"postgresql"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"postgresql"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, mapping []mapping) {
				require.Equal(t, "db.system", mapping[0].from)
				require.Equal(t, ScopeSpan, mapping[0].scope)
			},
		},
		{
			// team.name is only at resource level so this should be a missingLabel
			name:       "span scope config doesn't match resource attribute",
			dimensions: map[string]string{"span.team.name": "team_name"},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"team_name"}, []string{missingLabel})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{missingLabel}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, mapping []mapping) {
				require.Equal(t, "team.name", mapping[0].from)
				require.Equal(t, ScopeSpan, mapping[0].scope)
			},
		},
		{
			// should use span value (overwrites resource value)
			name:       "unscoped attribute matches both with span overwriting resource values",
			dimensions: map[string]string{"service.name": "service"},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"service"}, []string{"span-service"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"span-service"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, mapping []mapping) {
				require.Equal(t, "service.name", mapping[0].from)
				require.Equal(t, ScopeAll, mapping[0].scope)
			},
		},
		{
			// should use resource namespace and span database
			name: "multiple scoped attributes",
			dimensions: map[string]string{
				"resource.k8s.namespace.name": "namespace",
				"span.db.system":              "database",
			},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"database", "namespace"}, []string{"postgresql", "resource-namespace"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"postgresql", "resource-namespace"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, actual []mapping) {
				expected := []mapping{
					{from: "k8s.namespace.name", scope: ScopeResource, to: 1}, // "namespace" is at index 1 after sorting ["database", "namespace"]
					{from: "db.system", scope: ScopeSpan, to: 0},              // "database" is at index 0 after sorting
				}
				// to remove flake from the order in the actual mapping
				require.ElementsMatch(t, expected, actual)
			},
		},
		{
			// scoped attributes should not be overwritten
			name: "scoped attribute prevents overwrite",
			dimensions: map[string]string{
				"resource.k8s.namespace.name": "resource_namespace",
				"k8s.namespace.name":          "unscoped_namespace",
			},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"resource_namespace", "unscoped_namespace"}, []string{"resource-namespace", "span-namespace"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"resource-namespace", "span-namespace"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, actual []mapping) {
				expected := []mapping{
					{from: "k8s.namespace.name", scope: ScopeResource, to: 0}, // "resource_namespace" is at index 0 after sorting
					{from: "k8s.namespace.name", scope: ScopeAll, to: 1},      // "unscoped_namespace" is at index 1 after sorting
				}
				// to remove flake from the order in the actual mapping
				require.ElementsMatch(t, expected, actual)
			},
		},
		{
			// scoped attributes should not be overwritten if remapped
			name: "scoped attribute prevents overwrite with remapping",
			dimensions: map[string]string{
				"resource.service.name": "service",
				"span.service.name":     "span_service",
			},
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"service", "span_service"}, []string{"resource-service", "span-service"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"resource-service", "span-service"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, actual []mapping) {
				expected := []mapping{
					{from: "service.name", scope: ScopeResource, to: 0}, // "service" is at index 0 after sorting
					{from: "service.name", scope: ScopeSpan, to: 1},     // "span_service" is at index 1 after sorting
				}
				// to remove flake from the order in the actual mapping
				require.ElementsMatch(t, expected, actual)
			},
		},
		{
			// same attribute at both scopes with no remapping
			name: "same attribute in with scopes behaves as unscoped",
			dimensions: map[string]string{
				"resource.service.name": "",
				"span.service.name":     "",
			},
			// behaves same as unscoped where span level value overwrites resource level values if it exists on both
			// levels, you can avoid it by remapping it
			expectSeries: func(t *testing.T, series map[uint64]*bucket) {
				expectedHash := hash([]string{"service_name"}, []string{"span-service"})
				require.Contains(t, series, expectedHash)
				require.Equal(t, []string{"span-service"}, series[expectedHash].labels)
			},
			expectMapping: func(t *testing.T, actual []mapping) {
				expected := []mapping{
					{from: "service.name", scope: ScopeResource, to: 0}, // only one key so both are in same
					{from: "service.name", scope: ScopeSpan, to: 0},
				}
				// to remove flake from the order in the actual mapping
				require.ElementsMatch(t, expected, actual)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewTracker(testConfig(), "test", func(_ string) map[string]string { return tt.dimensions }, func(_ string) uint64 { return 0 })
			require.NoError(t, err)

			u.Observe("test", data)
			series := u.tenants["test"].series
			mapping := u.tenants["test"].mapping

			tt.expectSeries(t, series)
			tt.expectMapping(t, mapping)
		})
	}
}

func TestParseDimensionKey(t *testing.T) {
	tests := []struct {
		input         string
		expectedAttr  string
		expectedScope Scope
	}{
		{input: "resource.service.name", expectedAttr: "service.name", expectedScope: ScopeResource},
		{input: "resource.k8s.cluster.name", expectedAttr: "k8s.cluster.name", expectedScope: ScopeResource},
		{input: "resource.k8s.namespace.name", expectedAttr: "k8s.namespace.name", expectedScope: ScopeResource},
		{input: "resource.telemetry.sdk.name", expectedAttr: "telemetry.sdk.name", expectedScope: ScopeResource},
		{input: "span.db.system", expectedAttr: "db.system", expectedScope: ScopeSpan},
		{input: "span.http.method", expectedAttr: "http.method", expectedScope: ScopeSpan},
		{input: "span.rpc.method", expectedAttr: "rpc.method", expectedScope: ScopeSpan},
		{input: "span.k8s.namespace.name", expectedAttr: "k8s.namespace.name", expectedScope: ScopeSpan},
		{input: "http.status_code", expectedAttr: "http.status_code", expectedScope: ScopeAll},
		{input: "service.name", expectedAttr: "service.name", expectedScope: ScopeAll},
		{input: "k8s.namespace.name", expectedAttr: "k8s.namespace.name", expectedScope: ScopeAll},
		{input: "k8s.cluster.name", expectedAttr: "k8s.cluster.name", expectedScope: ScopeAll},
		{input: "user.team", expectedAttr: "user.team", expectedScope: ScopeAll},
		{input: "teamName", expectedAttr: "teamName", expectedScope: ScopeAll},
		{input: "team_name", expectedAttr: "team_name", expectedScope: ScopeAll},
		{input: "MyCustomAttribute", expectedAttr: "MyCustomAttribute", expectedScope: ScopeAll},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			attr, scope := parseDimensionKey(tt.input)
			require.Equal(t, tt.expectedAttr, attr)
			require.Equal(t, tt.expectedScope, scope)
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
