// Package metrics collects what Tempo reported while a benchmark case ran.
//
// Query responses are flattened into maps keyed by Tempo's own metric names,
// and the process registry is read by differencing two gathers. Neither one
// keeps a list of metrics it knows about, so a metric added to Tempo shows up
// here without a change.
package metrics

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// spansDeduped is the key SpansDeduped is reported under. The metrics evaluator
// counts it but SearchMetrics has no field for it, so it is named here rather
// than added to Tempo's wire contract.
const spansDeduped = "spansDeduped"

// FromResponse flattens a metrics message from a query response.
//
// It marshals to JSON instead of reading named fields, so a field added to
// SearchMetrics or TraceByIDMetrics is picked up automatically. The protobuf
// tags are omitempty, so a field Tempo did not populate is absent from the map
// rather than zero: "not reported" and "zero" are different claims.
//
// AdditionalMetrics is merged in at the top level, since its keys are already
// metric names.
func FromResponse(msg any) (map[string]int64, error) {
	if msg == nil {
		return nil, nil
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshalling response metrics: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("unmarshalling response metrics: %w", err)
	}

	out := map[string]int64{}
	for name, val := range fields {
		var n int64
		if err := json.Unmarshal(val, &n); err == nil {
			out[name] = n
			continue
		}
		// A nested map of counters, i.e. additionalMetrics.
		var nested map[string]int64
		if err := json.Unmarshal(val, &nested); err == nil {
			for k, v := range nested {
				out[k] += v
			}
		}
		// Anything else is not a counter, so it is not a metric.
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// FromEvaluator flattens what a TraceQL metrics query reported.
//
// The metrics path returns a plain struct with no wire tags, so it is mapped
// onto SearchMetrics first. That way the keys still come from the protobuf tags
// and cannot drift from the ones the other APIs report under.
func FromEvaluator(m traceql.EvaluatorMetrics) (map[string]int64, error) {
	additional := make(map[string]int64, len(m.AdditionalMetrics)+1)
	for k, v := range m.AdditionalMetrics {
		additional[k] = v
	}
	if m.SpansDeduped > 0 {
		additional[spansDeduped] += int64(m.SpansDeduped)
	}
	if len(additional) == 0 {
		additional = nil
	}

	return FromResponse(&tempopb.SearchMetrics{
		InspectedBytes:    m.Bytes,
		InspectedSpans:    m.SpansTotal,
		BackendReads:      m.BackendReads,
		BackendBytes:      m.BackendBytes,
		AdditionalMetrics: additional,
	})
}

// BytesRead reports a bytes-read count from an API that reports nothing else.
// The tag APIs take a callback for it instead of returning a metrics message.
func BytesRead(n int64) (map[string]int64, error) {
	if n <= 0 {
		return nil, nil
	}
	return FromResponse(&tempopb.SearchMetrics{InspectedBytes: uint64(n)})
}

// Add sums src into dst, allocating dst if needed.
func Add(dst, src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]int64, len(src))
	}
	for k, v := range src {
		dst[k] += v
	}
	return dst
}
