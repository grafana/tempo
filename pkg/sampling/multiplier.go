// Package sampling provides helpers for extracting per-span sampling
// extrapolation multipliers from OpenTelemetry-style W3C tracestate.
package sampling

import (
	"math"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
)

// MultiplierFromTraceState returns the OpenTelemetry probability-sampling
// multiplier (= 1 / sampling probability) parsed from a W3C tracestate
// string. When the tracestate also carries an OTEP-235 R-value (`rv:`
// field), the multiplier is stochastically rounded to an integer using
// that R-value as the random source — E[result] equals the underlying
// multiplier, so aggregates over many spans converge to the same
// expected value as the raw float, but each per-span contribution is
// integer-valued (a cleaner UX for count-style aggregates and Prometheus
// counter increments).
//
// Falls back to the raw float multiplier when the tracestate has a
// threshold but no R-value (for example, the SDK relies on the trace
// ID's random flag per W3C trace context v2). Returns 0 when the
// tracestate has no usable probability-sampling data — callers should
// treat 0 as "unknown" and fall back to 1.0 (or to a configured ratio
// attribute) themselves.
func MultiplierFromTraceState(traceState string) float64 {
	ot := extractOpenTelemetryTraceState(traceState)
	if ot == "" {
		return 0
	}
	otts, err := sampling.NewOpenTelemetryTraceState(ot)
	if err != nil {
		return 0
	}
	m := otts.AdjustedCount()
	rnd, hasRV := otts.RValueRandomness()
	if m <= 0 || !hasRV {
		return m
	}
	i := math.Floor(m)
	frac := m - i
	if frac == 0 {
		return i
	}
	// Top 53 bits of the 56-bit R-value as a uniform float64 in [0, 1).
	r := float64(rnd.Unsigned()>>3) / (1 << 53)
	if r < frac {
		return i + 1
	}
	return i
}

// extractOpenTelemetryTraceState parses the tracestate for the ot vendor key
// and returns the value of the key (or empty if it does not exist). It is
// about twice as fast as sampling.NewW3CTraceState and does no allocations.
func extractOpenTelemetryTraceState(traceState string) string {
	// tracestate is formatted like vendor1=value1,vendor2=value2. See
	// https://www.w3.org/TR/trace-context/#list.
	for {
		traceState = strings.TrimSpace(traceState)
		nextComma := strings.IndexByte(traceState, ',')
		if strings.HasPrefix(traceState, "ot=") {
			end := len(traceState)
			if nextComma > 0 {
				end = nextComma
			}
			return traceState[3:end]
		}

		if nextComma == -1 {
			return ""
		}
		traceState = strings.TrimSpace(traceState[nextComma+1:])
	}
}
