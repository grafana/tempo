package tracediff

import "math"

// Fixed tolerances for numeric comparisons in trace diffs. Two values are
// considered unchanged when |a-b| <= absTol + relTol*max(|a|, |b|).
const (
	// durationRelTolerance is intentionally loose in order to separate a real regression from run-to-run
	// jitter, so span duration diffs only report large and significant changes.
	durationRelTolerance = 0.20
	// durationAbsTolerance is a 1ms floor (in nanoseconds) that suppresses
	// sub-millisecond jitter on very short spans.
	durationAbsTolerance = 1_000_000.0

	// attrRelTolerance is the default relative tolerance for allow-listed numeric attributes
	// like sizes, token counts, durations.
	attrRelTolerance = 0.05
	// attrAbsTolerance is zero: allow-listed values are magnitudes where the relative
	// tolerance scales correctly.
	attrAbsTolerance = 0.0
)

// numericFuzzyAttributes are OpenTelemetry semantic-convention numeric
// attributes that represent magnitudes (byte sizes, token counts, durations)
// where small relative differences are noise. Values for these keys are compared
// with a relative tolerance; every other numeric attribute is compared exactly.
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
