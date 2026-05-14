package policymatch

import (
	"testing"

	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestIntrinsicPolicyMatch_Matches(t *testing.T) {
	cases := []struct {
		policy *IntrinsicPolicyMatch
		span   *tracev1.Span
		expect bool
		name   string
	}{
		{
			name:   "match on name, kind and status",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched name",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test2",
			},
		},
		{
			name:   "unmatched status",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_CLIENT,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_ERROR,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched kind",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					NewKindIntrinsicFilter(tracev1.Span_SPAN_KIND_SERVER),
					NewStatusIntrinsicFilter(tracev1.Status_STATUS_CODE_OK),
					NewNameIntrinsicFilter("test"),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_CLIENT,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "matched regex kind and status",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_KIND_.*")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_CODE_.*")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex kind",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_CLIENT")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_OK")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "unmatched regex status",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, ".*_SERVER")),
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicStatus, ".*_ERROR")),
				},
			},
			span: &tracev1.Span{
				Kind: tracev1.Span_SPAN_KIND_SERVER,
				Status: &tracev1.Status{
					Code: tracev1.Status_STATUS_CODE_OK,
				},
				Name: "test",
			},
		},
		{
			name:   "optimized regex kind matches SERVER",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_SERVER},
		},
		{
			name:   "optimized regex kind matches CLIENT",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_CLIENT},
		},
		{
			name:   "optimized regex kind matches PRODUCER",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_PRODUCER},
		},
		{
			name:   "optimized regex kind matches CONSUMER",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_CONSUMER},
		},
		{
			name:   "optimized regex kind rejects INTERNAL",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_INTERNAL},
		},
		{
			name:   "optimized regex kind rejects UNSPECIFIED",
			expect: false,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_UNSPECIFIED},
		},
		{
			name:   "regex kind subset matches SERVER",
			expect: true,
			policy: &IntrinsicPolicyMatch{
				filters: []IntrinsicFilter{
					must(NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_(CLIENT|SERVER)")),
				},
			},
			span: &tracev1.Span{Kind: tracev1.Span_SPAN_KIND_SERVER},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.policy.Matches(tc.span)
			require.Equal(t, tc.expect, r)
		})
	}
}

// TestKindRegexMask verifies the constant-time bitmask that
// NewRegexpIntrinsicFilter precomputes for a kind regex. The bitmask is set
// when the regex matches some non-empty subset of {SERVER, CLIENT, PRODUCER,
// CONSUMER} and no other canonical kind string; otherwise the mask is zero.
// The compiled regex is always retained as a fallback for spans whose Kind
// is outside the canonical enum (proto unknown integer values stringify to
// decimal, e.g. "6") — see TestKindRegex_UnknownEnumValuePreservesRegexPath.
//
// Without this test, a regression in `kindRegexMask` that silently widens or
// narrows the fast-path acceptance set is invisible: TestIntrinsicPolicyMatch
// only asserts the boolean match result, which is the same on either path.
func TestKindRegexMask(t *testing.T) {
	type expectedMask uint8
	cases := []struct {
		regex          string
		expectKind     expectedMask // 0 means regex engine fallback expected
		expectMaskZero bool
	}{
		{regex: "SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)", expectKind: expectedMask(spanKindServerBit | spanKindClientBit | spanKindProducerBit | spanKindConsumerBit)},
		{regex: "SPAN_KIND_(CLIENT|SERVER|PRODUCER|CONSUMER)", expectKind: expectedMask(spanKindServerBit | spanKindClientBit | spanKindProducerBit | spanKindConsumerBit)},
		{regex: "^SPAN_KIND_(SERVER|CONSUMER|CLIENT|PRODUCER)$", expectKind: expectedMask(spanKindServerBit | spanKindClientBit | spanKindProducerBit | spanKindConsumerBit)},
		{regex: "SPAN_KIND_SERVER", expectKind: expectedMask(spanKindServerBit)},
		{regex: "SPAN_KIND_(SERVER|CLIENT)", expectKind: expectedMask(spanKindServerBit | spanKindClientBit)},

		// Regexes that ALSO match INTERNAL or UNSPECIFIED must not produce a
		// mask, since the bitmask only has bits for the four "edge" kinds.
		{regex: "SPAN_KIND_.*", expectMaskZero: true},
		{regex: ".*", expectMaskZero: true},
		// Catch-all that matches the empty string must not produce a mask;
		// the mask path can never represent "matches everything".
		{regex: "", expectMaskZero: true},
	}

	for _, tc := range cases {
		t.Run(tc.regex, func(t *testing.T) {
			f, err := NewRegexpIntrinsicFilter(traceql.IntrinsicKind, tc.regex)
			require.NoError(t, err)
			// The compiled regex is ALWAYS retained as a fallback for unknown
			// enum values, regardless of whether the mask was set.
			require.NotNil(t, f.regex, "filter must retain compiled regex as fallback for unknown kinds")
			if tc.expectMaskZero {
				require.Zero(t, f.kindMask, "expected zero mask but got %b for %q", f.kindMask, tc.regex)
				return
			}
			require.Equal(t, uint8(tc.expectKind), f.kindMask, "wrong mask for %q", tc.regex)
		})
	}
}

// TestKindRegex_UnknownEnumValuePreservesRegexPath verifies that a span with
// a proto-unknown Span_SpanKind value (which stringifies to a decimal number)
// is still matched by the regex engine, not silently rejected by the bitmask
// path. The mask only has bits for the four canonical enum values; an unknown
// value's spanKindMask result is zero, and the filter must fall through to
// the compiled regex to preserve main's behavior.
func TestKindRegex_UnknownEnumValuePreservesRegexPath(t *testing.T) {
	// Regex matches SPAN_KIND_SERVER (mask candidate) plus an unknown
	// proto-enum integer string "6". On main the regex engine handled both;
	// the mask path alone can only express SERVER.
	f, err := NewRegexpIntrinsicFilter(traceql.IntrinsicKind, "SPAN_KIND_SERVER|6")
	require.NoError(t, err)
	require.Equal(t, spanKindServerBit, f.kindMask, "SPAN_KIND_SERVER must contribute to mask")
	require.NotNil(t, f.regex, "regex must be retained for unknown-kind fallback")

	// Known SERVER: mask path matches.
	require.True(t, f.Matches(&tracev1.Span{Kind: tracev1.Span_SPAN_KIND_SERVER}))
	// Known CLIENT: neither mask nor regex match.
	require.False(t, f.Matches(&tracev1.Span{Kind: tracev1.Span_SPAN_KIND_CLIENT}))
	// Unknown enum value 6 (stringifies to "6"): mask returns zero, must fall
	// through to regex which matches "6". A regression that drops the regex
	// when the mask is set would fail this assertion.
	require.True(t, f.Matches(&tracev1.Span{Kind: tracev1.Span_SpanKind(6)}),
		"unknown enum value must be matched via regex fallback")
}
