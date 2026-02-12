package registry

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func newTestDrainSanitizer(mode string) *DrainSanitizer {
	return NewDrainSanitizer("test-tenant", func(string) string { return mode }, 15*time.Minute)
}

func TestDrainSanitizer_PatternDetection(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	// Train with similar span names that should form a pattern
	lbls1 := labels.FromStrings("span_name", "GET /api/users/123", "service", "api")
	lbls2 := labels.FromStrings("span_name", "GET /api/users/456", "service", "api")
	lbls3 := labels.FromStrings("span_name", "GET /api/users/789", "service", "api")

	// First call should return original (no pattern yet)
	result1 := sanitizer.Sanitize(lbls1)
	require.Equal(t, "GET /api/users/123", result1.Get("span_name"))

	// After training, subsequent similar spans should be sanitized
	result2 := sanitizer.Sanitize(lbls2)
	result3 := sanitizer.Sanitize(lbls3)

	// All should have the same sanitized span_name pattern
	require.Equal(t, result2.Get("span_name"), result3.Get("span_name"))
	// Pattern should contain the parameter marker
	require.Contains(t, result2.Get("span_name"), "<_>")
	// Original labels should be preserved
	require.Equal(t, "api", result2.Get("service"))
	require.Equal(t, "api", result3.Get("service"))
}

func TestDrainSanitizer_DryRunMode(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationDryRun)

	lbls1 := labels.FromStrings("span_name", "GET /api/users/123", "service", "api")
	lbls2 := labels.FromStrings("span_name", "GET /api/users/456", "service", "api")

	// Train with first span
	sanitizer.Sanitize(lbls1)

	// In dry-run mode, even if pattern is detected, return original labels
	result := sanitizer.Sanitize(lbls2)
	require.Equal(t, "GET /api/users/456", result.Get("span_name"))
	require.Equal(t, lbls2, result)
}

func TestDrainSanitizer_RuntimeModeToggle(t *testing.T) {
	t.Parallel()

	mode := SpanNameSanitizationEnabled
	sanitizer := NewDrainSanitizer("test-tenant", func(string) string { return mode }, 15*time.Minute)

	// Train the drain tree with similar span names to establish a pattern
	sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/123"))
	sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/456"))

	// With enabled mode, should sanitize
	result := sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/789"))
	require.Contains(t, result.Get("span_name"), "<_>")

	// Toggle to disabled at runtime - should pass through
	mode = SpanNameSanitizationDisabled
	result = sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/999"))
	require.Equal(t, "GET /api/users/999", result.Get("span_name"))

	// Toggle to dry-run at runtime - should pass through but still train
	mode = SpanNameSanitizationDryRun
	result = sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/111"))
	require.Equal(t, "GET /api/users/111", result.Get("span_name"))

	// Toggle back to enabled - should sanitize again
	mode = SpanNameSanitizationEnabled
	result = sanitizer.Sanitize(labels.FromStrings("span_name", "GET /api/users/222"))
	require.Contains(t, result.Get("span_name"), "<_>")
}

func TestDrainSanitizer_DisabledMode(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationDisabled)

	lbls1 := labels.FromStrings("span_name", "GET /api/users/123", "service", "api")
	lbls2 := labels.FromStrings("span_name", "GET /api/users/456", "service", "api")

	sanitizer.Sanitize(lbls1)

	// When disabled, should always return original labels
	result := sanitizer.Sanitize(lbls2)
	require.Equal(t, "GET /api/users/456", result.Get("span_name"))
	require.Equal(t, lbls2, result)
}

func TestDrainSanitizer_NilClusterHandling(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	// Span name with too few tokens (less than MinTokens=3)
	// Tokenizer will produce tokens like ["a", "<END>"] which is < 3
	lbls := labels.FromStrings("span_name", "a", "service", "api")
	result := sanitizer.Sanitize(lbls)

	// Should return original labels when cluster is nil
	require.Equal(t, "a", result.Get("span_name"))
	require.Equal(t, lbls, result)
}

func TestDrainSanitizer_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	var wg sync.WaitGroup
	numGoroutines := 10
	numCallsPerGoroutine := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				lbls := labels.FromStrings("span_name", "GET /api/users/123", "id", string(rune(id*1000+j)))
				result := sanitizer.Sanitize(lbls)
				// Should always return valid labels
				require.NotNil(t, result)
				require.NotEmpty(t, result.Get("span_name"))
			}
		}(i)
	}

	wg.Wait()
	// No panics or race conditions should occur
}

func TestDrainSanitizer_DemandTracking(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	// Create labels with different span names
	lbls1 := labels.FromStrings("span_name", "GET /api/users/123")
	lbls2 := labels.FromStrings("span_name", "GET /api/posts/456")
	lbls3 := labels.FromStrings("span_name", "POST /api/users/789")

	// Sanitize multiple times
	sanitizer.Sanitize(lbls1)
	sanitizer.Sanitize(lbls2)
	sanitizer.Sanitize(lbls3)

	// Demand should be tracked (we can't easily verify exact count without exposing internals,
	// but we can verify the sanitizer doesn't crash and processes all labels)
	// The demand gauge will be updated periodically via doPeriodicMaintenance
	demandEstimate := sanitizer.demand.Estimate()
	require.GreaterOrEqual(t, demandEstimate, uint64(1))
}

func TestDrainSanitizer_NoSpanNameLabel(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	// Labels without span_name
	lbls := labels.FromStrings("service", "api", "method", "GET")
	result := sanitizer.Sanitize(lbls)

	// Should return original labels (span_name is empty string, which drain will reject)
	require.Equal(t, lbls, result)
}

func TestDrainSanitizer_PatternBeforeSanitization(t *testing.T) {
	t.Parallel()

	sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

	// First span name - no pattern yet, so returns original
	lbls1 := labels.FromStrings("span_name", "GET /api/users/123")
	result1 := sanitizer.Sanitize(lbls1)
	require.Equal(t, "GET /api/users/123", result1.Get("span_name"))

	// Same span name again - still no pattern (only one instance)
	result2 := sanitizer.Sanitize(lbls1)
	require.Equal(t, "GET /api/users/123", result2.Get("span_name"))

	// Different but similar span name - now pattern should emerge
	lbls2 := labels.FromStrings("span_name", "GET /api/users/456")
	result3 := sanitizer.Sanitize(lbls2)
	// After pattern detection, should return sanitized version
	require.NotEqual(t, "GET /api/users/456", result3.Get("span_name"))
	require.Contains(t, result3.Get("span_name"), "<_>")
}

func TestDrainSanitizer_FullSanitizedOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		inputs         []string
		expectedOutput string
	}{
		{
			name:           "REST API path with user ID",
			inputs:         []string{"GET /api/users/123", "GET /api/users/456", "GET /api/users/789"},
			expectedOutput: "GET /api/users/<_>",
		},
		{
			name:           "REST API path with multiple IDs",
			inputs:         []string{"POST /api/orders/100/items/1", "POST /api/orders/200/items/2", "POST /api/orders/300/items/3"},
			expectedOutput: "POST /api/orders/<_>/items/<_>",
		},
		{
			name:           "database query with table name",
			inputs:         []string{"SELECT * FROM users_100", "SELECT * FROM users_200", "SELECT * FROM users_300"},
			expectedOutput: "SELECT * FROM users_<_>",
		},
		{
			name:           "gRPC method call",
			inputs:         []string{"grpc.client/service.Method/123", "grpc.client/service.Method/456", "grpc.client/service.Method/789"},
			expectedOutput: "grpc.client/service.Method/<_>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sanitizer := newTestDrainSanitizer(SpanNameSanitizationEnabled)

			// Train with first inputs
			for _, input := range tc.inputs[:len(tc.inputs)-1] {
				sanitizer.Sanitize(labels.FromStrings("span_name", input))
			}

			// Last input should produce the expected sanitized output
			lastInput := tc.inputs[len(tc.inputs)-1]
			result := sanitizer.Sanitize(labels.FromStrings("span_name", lastInput))
			require.Equal(t, tc.expectedOutput, result.Get("span_name"))
		})
	}
}
