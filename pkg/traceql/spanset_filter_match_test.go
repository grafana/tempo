package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileSpansetFilter(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{name: "compiles a single span-attribute filter", query: `{ .http.status_code = 500 }`},
		{name: "compiles an intrinsic filter", query: `{ status = error }`},
		{name: "empty filter matches all", query: `{}`},
		{name: "compiles a boolean AND", query: `{ .a = 1 && .b = 2 }`},
		{name: "structural operator rejected", query: `{ .a } >> { .b }`, wantErr: errUnsupportedQuery},
		{name: "pipeline with aggregate rejected", query: `{ .a } | count() > 1`, wantErr: errUnsupportedQuery},
		{name: "multi-element spanset pipeline rejected", query: `{ .a } | { .b }`, wantErr: errUnsupportedQuery},
		{name: "metrics query rejected", query: `{ .a } | rate()`, wantErr: errUnsupportedQuery},
		{name: "invalid syntax rejected", query: `{ .a = }`, wantErr: errParseFilter},
		{name: "span:id intrinsic supported", query: `{ span:id = "0102" }`},
		{name: "event intrinsic supported", query: `{ event:name = "exception" }`},
		{name: "event-scoped attribute supported", query: `{ event.exception.message = "boom" }`},
		{name: "link-scoped attribute supported", query: `{ link.foo = "x" }`},
		{name: "link intrinsic supported", query: `{ link:spanID = "0102" }`},
		{name: "instrumentation-scoped attribute supported", query: `{ instrumentation.foo = "x" }`},
		{name: "attribute = nil supported (not-exists)", query: `{ span.foo = nil }`},
		{name: "attribute != nil supported (exists)", query: `{ span.foo != nil }`},
		{name: "invalid regex rejected", query: `{ span.foo =~ "[" }`, wantErr: errParseFilter},
		{name: "intrinsic = nil rejected", query: `{ status = nil }`, wantErr: errParseFilter},
		{name: "childCount intrinsic rejected", query: `{ span:childCount > 0 }`, wantErr: errUnsupportedIntrinsic},
		{name: "nestedSetLeft intrinsic rejected", query: `{ nestedSetLeft > 0 }`, wantErr: errUnsupportedIntrinsic},
		{name: "trace:rootName intrinsic rejected", query: `{ trace:rootName = "foo" }`, wantErr: errUnsupportedIntrinsic},
		{name: "trace:duration intrinsic rejected", query: `{ trace:duration > 1s }`, wantErr: errUnsupportedIntrinsic},
		{name: "trace:id intrinsic rejected", query: `{ trace:id = "abcd" }`, wantErr: errUnsupportedIntrinsic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := CompileSpansetFilter(tt.query)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f)
		})
	}
}

func TestSpansetFilterMatchSpans(t *testing.T) {
	matching := newMockSpan([]byte{0x01}).WithSpanInt("http.status_code", 500)
	nonMatching := newMockSpan([]byte{0x02}).WithSpanInt("http.status_code", 200)

	f, err := CompileSpansetFilter(`{ .http.status_code = 500 }`)
	require.NoError(t, err)

	matched, err := f.MatchSpans([]Span{matching, nonMatching})
	require.NoError(t, err)
	require.Len(t, matched, 1)
	require.Equal(t, []byte{0x01}, matched[0].ID())
}

func TestSpansetFilterMatchSpansEmptyInput(t *testing.T) {
	f, err := CompileSpansetFilter(`{ .a = 1 }`)
	require.NoError(t, err)

	matched, err := f.MatchSpans(nil)
	require.NoError(t, err)
	require.Empty(t, matched)
}

func TestSpansetFilterMatchAll(t *testing.T) {
	// `{}` is a constant-true filter and should return every span.
	f, err := CompileSpansetFilter(`{}`)
	require.NoError(t, err)

	spans := []Span{newMockSpan([]byte{0x01}), newMockSpan([]byte{0x02})}
	matched, err := f.MatchSpans(spans)
	require.NoError(t, err)
	require.Len(t, matched, 2)
}
