// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// # TraceState representation
//
// A [W3CTraceState] object parses and stores the OpenTelemetry
// tracestate field and any other fields that are present in the
// W3C tracestate header, part of the [W3C tracecontext specification].
//
// An [OpenTelemetryTraceState] object parses and stores fields of
// the OpenTelemetry-specific tracestate field, including those recognized
// for probability sampling and any other fields that are present.  The
// syntax of the OpenTelemetry field is specified in [Tracestate handling].
//
// The probability sampling-specific fields used here are specified in
// [OTEP 235].  The principal named fields are:
//
//   - T-value: The sampling rejection threshold, expresses a 56-bit
//     hexadecimal number of traces that will be rejected by sampling.
//   - R-value: The sampling randomness value can be implicit in a TraceID,
//     otherwise it is explicitly encoded as an R-value.
//
// # Low-level types
//
// The three key data types implemented in this package represent sampling
// decisions.
//
//   - [Threshold]: Represents an exact sampling probability.
//   - [Randomness]: Randomness used for sampling decisions.
//   - [Threshold.Probability]: a float64 in the range [MinSamplingProbability, 1.0].
//
// # Example use-case
//
// To configure a consistent tail sampler in an OpenTelemetry
// Collector using a fixed probability for all traces in an
// "equalizing" arrangement, where the effect of sampling is
// conditioned on how much sampling has already taken place, use the
// following pseudocode.
//
// func Setup() {
// 	// Get a fixed probability value from the configuration, in
// 	// the range (0, 1].
// 	probability := *FLAG_probability
//
// 	// Calculate the sampling threshold from probability using 3
// 	// hex digits of precision.
// 	fixedThreshold, err = ProbabilityToThresholdWithPrecision(probability, 3)
// 	if err != nil {
// 		// error case: Probability is not valid.
// 	}
// }
//
// func MakeDecision(tracestate string, tid TraceID) bool {
// 	// Parse the incoming tracestate
// 	ts, err := NewW3CTraceState(tracestate)
// 	if err != nil {
// 		// error case: Tracestate is ill-formed.
// 	}
// 	// For an absolute probability sample, we check the incoming
// 	// tracestate to see whether it was already sampled enough.
// 	if threshold, hasThreshold := ts.OTelValue().TValueThreshold(); hasThreshold {
// 		// If the incoming tracestate was already sampled at
// 		// least as much as our threshold implies, then its
// 		// (rejection) threshold is higher.  If so, then no
// 		// further sampling is called for.
// 		if ThresholdGreater(threshold, fixedThreshold) {
// 			// Do not update.
// 			return true
// 		}
//		// The error below is ignored because we've tested
//              // the equivalent condition above.  This lowers the sampling
//              // probability expressed in the tracestate T-value.
// 		_ = ts.OTelValue().UpdateThresholdWithSampling(fixedThreshold)
// 	}
// 	var rnd Randomness
// 	// If the R-value is present, use it.  If not, rely on TraceID
// 	// randomness.  Note that OTLP v1.1.0 introduces a new Span flag
// 	// to convey trace randomness correctly, and if the context has
// 	// neither the randomness bit set or the R-value set, we need a
// 	// fallback, which can be to synthesize an R-value or to assume
// 	// the TraceID has sufficient randomness.  This detail is left
// 	// out of scope.
// 	if rv, hasRand := ts.OTelValue().RValueRandomness(); hasRand {
// 		rnd = rv
// 	} else {
// 		rnd = TraceIDToRandomness(tid)
// 	}
// 	return fixedThreshold.ShouldSample(rnd)
// }
//
// [W3C tracecontext specification]: https://www.w3.org/TR/trace-context/#tracestate-header
// [Tracestate handling]: https://opentelemetry.io/docs/specs/otel/trace/tracestate-handling/

package sampling // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
