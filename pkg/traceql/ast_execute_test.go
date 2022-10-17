package traceql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpansetFilter_matches(t *testing.T) {
	tests := []struct {
		query   string
		span    Span
		matches bool
		// TODO do we actually care about the error mesasge?
		err bool
	}{
		{
			query: `{ ("foo" != "bar") && !("foo" = "bar") }`,
			span: Span{
				Attributes: nil,
			},
			matches: true,
		},
		{
			query: `{ .foo = .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("bzz"),
					NewAttribute("bar"): NewStaticString("bzz"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo = "bar" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("fzz"): NewStaticString("bar"),
				},
			},
			err: true,
		},
		{
			query: `{ .foo = .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticString("str"),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			err: true,
		},
		{
			query: `{ .foo =~ .bar }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"): NewStaticInt(3),
					NewAttribute("bar"): NewStaticInt(5),
				},
			},
			err: true,
		},
		{
			query: `{ .field1 =~ "hello w.*" && .field2 !~ "bye b.*" }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("field1"): NewStaticString("hello world"),
					NewAttribute("field2"): NewStaticString("bye world"),
				},
			},
			matches: true,
		},
		{
			query: `{ .foo > 2 && .foo >= 3.5 && .foo < 5 && .foo <= 3.5 && .duration > 1800ms }`,
			span: Span{
				Attributes: map[Attribute]Static{
					NewAttribute("foo"):      NewStaticFloat(3.5),
					NewAttribute("duration"): NewStaticDuration(2 * time.Second),
				},
			},
			matches: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			spansetFilter := expr.Pipeline.Elements[0].(SpansetFilter)

			matches, err := spansetFilter.matches(tt.span)

			if tt.err {
				fmt.Println(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.matches, matches)
			}
		})
	}

}
