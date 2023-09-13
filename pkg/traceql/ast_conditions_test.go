package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpansetFilter_extractConditions(t *testing.T) {
	tests := []struct {
		query         string
		conditions    []Condition
		allConditions bool
	}{
		{
			query: `{ .foo = "bar" && "bzz" = .fzz }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("fzz"), OpEqual, NewStaticString("bzz")),
			},
			allConditions: true,
		},
		{
			query: `{ .foo = "bar" || "bzz" = .fzz }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("fzz"), OpEqual, NewStaticString("bzz")),
			},
			allConditions: false,
		},
		{
			query:         `{ "foo" = "bar" }`,
			conditions:    []Condition{},
			allConditions: true,
		},
		{
			query: `{ .foo = .bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpNone),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ (.foo = "bar") = true }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		{
			query: `{ true = (.foo = "bar") }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		{
			query: `{ (.foo = "bar") = .bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
		{
			query: `{ .bar = (.foo = "bar") }`,
			conditions: []Condition{
				newCondition(NewAttribute("bar"), OpNone),
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
			},
			allConditions: true,
		},
		// TODO we need a smarter engine to handle this - we can either negate OpEqual or just fetch .fzz
		// {
		// 	query: `{ (.foo = "bar") = !(.fzz = "bzz") }`,
		// 	conditions: []Condition{
		// 		newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
		// 		newCondition(NewAttribute("fzz"), OpNotEqual, NewStaticString("bzz")),
		// 	},
		// 	allConditions: true,
		// },
		{
			query: `{ (.foo = "bar") = !.bar }`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				newCondition(NewAttribute("bar"), OpNone),
			},
			allConditions: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			spansetFilter := expr.Pipeline.Elements[0].(*SpansetFilter)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			spansetFilter.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Nil(t, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestScalarFilter_extractConditions(t *testing.T) {
	tests := []struct {
		query         string
		conditions    []Condition
		allConditions bool
	}{
		{
			query: `{ .foo = "a" } | count() > 10`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
			},
			allConditions: false,
		},
		{
			query: `{ .foo = "a" } | avg(duration) > 10ms`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
				newCondition(NewIntrinsic(IntrinsicDuration), OpNone),
			},
			allConditions: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			expr.Pipeline.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Nil(t, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}

func TestSelect_extractConditions(t *testing.T) {
	tests := []struct {
		query                string
		conditions           []Condition
		secondPassConditions []Condition
		allConditions        bool
	}{
		{
			query: `{ .foo = "a" } | select(resource.service.name)`,
			conditions: []Condition{
				newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
			},
			secondPassConditions: []Condition{
				newCondition(NewScopedAttribute(AttributeScopeResource, false, "service.name"), OpNone),
			},
			allConditions: false,
		},
		{
			query: `{ } | select(.name,name)`,
			conditions: []Condition{
				newCondition(NewIntrinsic(IntrinsicDuration), OpNone),
			},
			secondPassConditions: []Condition{
				newCondition(NewAttribute("name"), OpNone),
				newCondition(NewIntrinsic(IntrinsicName), OpNone),
			},
			allConditions: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)

			req := &FetchSpansRequest{
				Conditions:    []Condition{},
				AllConditions: true,
			}
			expr.Pipeline.extractConditions(req)

			assert.Equal(t, tt.conditions, req.Conditions)
			assert.Equal(t, tt.secondPassConditions, req.SecondPassConditions)
			assert.Equal(t, tt.allConditions, req.AllConditions, "FetchSpansRequest.AllConditions")
		})
	}
}
