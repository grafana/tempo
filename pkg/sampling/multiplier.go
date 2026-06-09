// Package sampling provides helpers for extracting per-span sampling
// extrapolation multipliers from OpenTelemetry-style W3C tracestate.
package sampling

import (
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
)

// MultiplierFromTraceState extracts the OpenTelemetry probability-sampling
// multiplier (= 1 / sampling probability) from a W3C tracestate string.
//
// It returns 0 when the tracestate is empty, has no ot= segment, or fails to
// parse — callers should treat 0 as "unknown" and fall back to 1.0 (or to a
// configured attribute key) themselves.
func MultiplierFromTraceState(traceState string) float64 {
	ot := extractOpenTelemetryTraceState(traceState)
	if ot == "" {
		return 0
	}

	otts, err := sampling.NewOpenTelemetryTraceState(ot)
	if err != nil {
		return 0
	}

	return otts.AdjustedCount()
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
