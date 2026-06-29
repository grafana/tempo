package tracediff

import "math"

// Fixed tolerances for trace-patch-v0 numeric comparison. Two values are
// considered unchanged when |a-b| <= absTol + relTol*max(|a|, |b|).
const (
	// durationRelTol is intentionally loose: a single pair of traces cannot
	// separate a real regression from run-to-run jitter, so span duration diffs
	// are advisory and only large (roughly first-significant-digit) changes are
	// reported.
	durationRelTol = 0.25
	// durationAbsTolNano is a 1ms floor (in nanoseconds) that suppresses
	// sub-millisecond jitter on short spans.
	durationAbsTolNano = 1_000_000.0

	// attrRelTol is stricter: allow-listed numeric attributes (sizes, token
	// counts, durations) are usually deterministic for the same operation, so a
	// roughly second-significant-digit change is reported.
	attrRelTol = 0.05
	// attrAbsTol is zero: allow-listed values are magnitudes where the relative
	// term scales correctly. As a result any change to or from exactly zero is
	// reported.
	attrAbsTol = 0.0
)

// numericFuzzyAttributes are OpenTelemetry semantic-convention numeric
// attributes that represent magnitudes (byte sizes, token counts, durations)
// where small relative differences are noise. Values for these keys are compared
// with a relative tolerance; every other numeric attribute is compared exactly.
//
// Curated from open-telemetry/semantic-conventions (commit b51e2a6, 2026-06-25).
// Deprecated attribute names are included alongside their current counterparts
// because a trace carries only one. Identifiers, codes, ports, indices, epoch
// timestamps, configuration parameters, small discrete counts, and vendor
// product-specific attributes are intentionally excluded.
var numericFuzzyAttributes = map[string]struct{}{
	// Byte / payload sizes.
	"http.request.body.size":                    {},
	"http.request.size":                         {},
	"http.response.body.size":                   {},
	"http.response.size":                        {},
	"http.request_content_length":               {}, // deprecated
	"http.request_content_length_uncompressed":  {}, // deprecated
	"http.response_content_length":              {}, // deprecated
	"http.response_content_length_uncompressed": {}, // deprecated
	"messaging.message.body.size":               {},
	"messaging.message.envelope.size":           {},
	"message.compressed_size":                   {}, // deprecated
	"message.uncompressed_size":                 {}, // deprecated
	"rpc.message.compressed_size":               {}, // deprecated
	"rpc.message.uncompressed_size":             {}, // deprecated
	"file.size":                                 {},

	// Token counts.
	"gen_ai.usage.input_tokens":                {},
	"gen_ai.usage.output_tokens":               {},
	"gen_ai.usage.prompt_tokens":               {}, // deprecated -> input_tokens
	"gen_ai.usage.completion_tokens":           {}, // deprecated -> output_tokens
	"gen_ai.usage.cache_creation.input_tokens": {},
	"gen_ai.usage.cache_read.input_tokens":     {},
	"gen_ai.usage.reasoning.output_tokens":     {},

	// Durations.
	"gen_ai.response.time_to_first_chunk": {}, // deprecated
	"app.jank.period":                     {},
}

// numericClose reports whether a and b are equal within the given relative and
// absolute tolerances: |a-b| <= absTol + relTol*max(|a|, |b|).
func numericClose(a, b, relTol, absTol float64) bool {
	return math.Abs(a-b) <= absTol+relTol*math.Max(math.Abs(a), math.Abs(b))
}

// isNumericFuzzyAttribute reports whether the attribute key is compared with a
// relative tolerance instead of exact equality.
func isNumericFuzzyAttribute(key string) bool {
	_, ok := numericFuzzyAttributes[key]
	return ok
}

// numericValue returns v as a float64 when it holds an int64 or float64.
func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
