package tracediff

import (
	"testing"

	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Allow-listed keys reused across the tests below.
const (
	testAttrBodySize    = "http.request.body.size"
	testAttrFileSize    = "file.size"
	testAttrInputTokens = "gen_ai.usage.input_tokens"
)

func TestNumericClose(t *testing.T) {
	tests := []struct {
		name           string
		a, b           float64
		relTol, absTol float64
		want           bool
	}{
		{name: "equal", a: 5, b: 5, relTol: 0.05, want: true},
		{name: "within relative", a: 1000, b: 1001, relTol: 0.05, want: true},
		{name: "at relative boundary", a: 1000, b: 1050, relTol: 0.05, want: true},
		{name: "beyond relative", a: 1000, b: 2000, relTol: 0.05, want: false},
		{name: "within absolute floor", a: 0, b: 500_000, relTol: 0.25, absTol: 1_000_000, want: true},
		{name: "beyond floor but within relative", a: 100e6, b: 110e6, relTol: 0.25, absTol: 1_000_000, want: true},
		{name: "beyond both", a: 100e6, b: 250e6, relTol: 0.25, absTol: 1_000_000, want: false},
		{name: "zero transition with no floor", a: 0, b: 1, relTol: 0.05, want: false},
		{name: "both zero", a: 0, b: 0, relTol: 0.05, want: true},
		{name: "negatives within tolerance", a: -100, b: -101, relTol: 0.05, want: true},
		{name: "sign flip", a: -100, b: 100, relTol: 0.05, want: false},
		{name: "zero tolerances reduce to exact equal", a: 5, b: 5, want: true},
		{name: "zero tolerances reduce to exact unequal", a: 5, b: 6, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, numericClose(tt.a, tt.b, tt.relTol, tt.absTol))
		})
	}
}

func TestIsNumericFuzzyAttribute(t *testing.T) {
	allowed := []string{
		testAttrBodySize,
		"http.response_content_length", // deprecated entry
		testAttrInputTokens,
		testAttrFileSize,
		"app.jank.period",
	}
	for _, key := range allowed {
		assert.Truef(t, isNumericFuzzyAttribute(key), "expected %q to be allow-listed", key)
	}

	excluded := []string{
		"http.response.status_code", // status code
		"server.port",               // port
		"db.response.returned_rows", // intentionally excluded
		"aws.dynamodb.count",        // vendor product-specific
		"custom.app.counter",        // unknown -> exact
	}
	for _, key := range excluded {
		assert.Falsef(t, isNumericFuzzyAttribute(key), "expected %q to be excluded", key)
	}
}

func TestNumericValue(t *testing.T) {
	v, ok := numericValue(int64(5))
	assert.True(t, ok)
	assert.Equal(t, 5.0, v)

	v, ok = numericValue(2.5)
	assert.True(t, ok)
	assert.Equal(t, 2.5, v)

	_, ok = numericValue("10")
	assert.False(t, ok)

	_, ok = numericValue(nil)
	assert.False(t, ok)

	_, ok = numericValue(true)
	assert.False(t, ok)
}

func TestAttributeChanged(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		before, after any
		want          bool
	}{
		{name: "allow-listed within tolerance", key: testAttrBodySize, before: int64(1000), after: int64(1001), want: false},
		{name: "allow-listed beyond tolerance", key: testAttrBodySize, before: int64(1000), after: int64(2000), want: true},
		{name: "allow-listed equal", key: testAttrFileSize, before: int64(500), after: int64(500), want: false},
		{name: "allow-listed zero transition", key: testAttrFileSize, before: int64(0), after: int64(1), want: true},
		{name: "allow-listed int vs float equal", key: testAttrInputTokens, before: int64(100), after: 100.0, want: false},
		{name: "allow-listed non-numeric falls back to exact", key: testAttrBodySize, before: "a", after: "b", want: true},
		{name: "non-allow-listed numeric compared exactly", key: "custom.count", before: int64(1000), after: int64(1001), want: true},
		{name: "status code compared exactly", key: "http.response.status_code", before: int64(200), after: int64(201), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, attributeChanged(tt.key, tt.before, tt.after))
		})
	}
}

func TestDiffDurationTolerance(t *testing.T) {
	traceID := []byte("trace-id-0000001")
	tests := []struct {
		name       string
		baseEnd    uint64
		compareEnd uint64
		wantChange bool
	}{
		{name: "below tolerance", baseEnd: 100, compareEnd: 110, wantChange: false},
		{name: "just within relative tolerance", baseEnd: 100, compareEnd: 126, wantChange: false},
		{name: "just beyond relative tolerance", baseEnd: 100, compareEnd: 127, wantChange: true},
		{name: "well beyond tolerance", baseEnd: 100, compareEnd: 250, wantChange: true},
		{name: "within absolute floor on short spans", baseEnd: 0, compareEnd: 1, wantChange: false},
		{name: "beyond absolute floor on short spans", baseEnd: 0, compareEnd: 2, wantChange: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := traceWithNamedSpans(
				spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, tt.baseEnd, tracev1.Status_STATUS_CODE_OK),
			)
			compare := traceWithNamedSpans(
				spanForNormalizeTest(traceID, "root", "", "checkout", "POST /checkout", tracev1.Span_SPAN_KIND_SERVER, 0, tt.compareEnd, tracev1.Status_STATUS_CODE_OK),
			)

			got, err := Diff(base, compare, FormatTracePatchV0)
			require.NoError(t, err)
			require.Equal(t, 1, got.Stats.MatchedSpans)

			if !tt.wantChange {
				assert.Zero(t, got.Stats.FieldChanges)
				assert.Empty(t, got.Modified)
				return
			}

			require.Len(t, got.Modified, 1)
			require.Len(t, got.Modified[0].Changes, 1)
			change := got.Modified[0].Changes[0]
			assert.Equal(t, FieldDurationNanos, change.Target.Name)
			assert.Equal(t, int64(tt.baseEnd*1_000_000), change.Before)
			assert.Equal(t, int64(tt.compareEnd*1_000_000), change.After)
		})
	}
}
