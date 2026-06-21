package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileSpansetFilter(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{name: "single attribute filter", query: `{ .http.status_code = 500 }`},
		{name: "intrinsic filter", query: `{ status = error }`},
		{name: "empty filter matches all", query: `{}`},
		{name: "boolean expression", query: `{ .a = 1 && .b = 2 }`},
		{name: "structural operator rejected", query: `{ .a } >> { .b }`, wantErr: true},
		{name: "pipeline with aggregate rejected", query: `{ .a } | count() > 1`, wantErr: true},
		{name: "metrics query rejected", query: `{ .a } | rate()`, wantErr: true},
		{name: "invalid syntax rejected", query: `{ .a = }`, wantErr: true},
		{name: "span:id intrinsic supported", query: `{ span:id = "0102" }`},
		{name: "trace:rootName intrinsic supported", query: `{ trace:rootName = "foo" }`},
		{name: "event intrinsic supported", query: `{ event:name = "exception" }`},
		{name: "childCount intrinsic rejected", query: `{ span:childCount > 0 }`, wantErr: true},
		{name: "nestedSetLeft intrinsic rejected", query: `{ nestedSetLeft > 0 }`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := CompileSpansetFilter(tt.query)
			if tt.wantErr {
				require.Error(t, err)
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
	assert.Equal(t, []byte{0x01}, matched[0].ID())
}

func TestSpansetFilterMatchSpansEmptyInput(t *testing.T) {
	f, err := CompileSpansetFilter(`{ .a = 1 }`)
	require.NoError(t, err)

	matched, err := f.MatchSpans(nil)
	require.NoError(t, err)
	require.Empty(t, matched)
}

func TestSpansetFilterMatchAll(t *testing.T) {
	f, err := CompileSpansetFilter(`{}`)
	require.NoError(t, err)

	spans := []Span{newMockSpan([]byte{0x01}), newMockSpan([]byte{0x02})}
	matched, err := f.MatchSpans(spans)
	require.NoError(t, err)
	require.Len(t, matched, 2)
}
