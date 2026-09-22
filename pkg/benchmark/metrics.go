package benchmark

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// responseMetrics flattens a metrics message from a query response into a map
// keyed by the message's own field names.
//
// It round-trips through JSON rather than reading named fields, so a metric
// added to SearchMetrics or TraceByIDMetrics is reported without a change here.
// The protobuf json tags are omitempty, so a field Tempo did not populate is
// absent from the map rather than present as zero: absent means "not reported",
// which is not the same claim as zero.
//
// AdditionalMetrics is merged in at the top level, since its keys are already
// distinct metric names (see tempopb.AdditionalMetric*).
func responseMetrics(msg any) (map[string]int64, error) {
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
		// A scalar counter.
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
			continue
		}
		// Anything else is not a counter, so it is not a metric.
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// metricSpansDeduped is the key SpansDeduped is reported under. The metrics
// evaluator counts it but SearchMetrics has no field for it, so it is named
// here rather than added to Tempo's wire contract for the benchmark's sake.
const metricSpansDeduped = "spansDeduped"

// evaluatorResponseMetrics flattens what a TraceQL metrics query reported.
//
// The metrics path returns traceql.EvaluatorMetrics, a plain struct with no
// wire tags, so it is handled separately from the messages the other APIs
// return. It is mapped onto SearchMetrics rather than to string keys directly,
// so the names still come from the protobuf tags and cannot drift from the ones
// the search and trace-by-ID cases report under.
func evaluatorResponseMetrics(m traceql.EvaluatorMetrics) (map[string]int64, error) {
	additional := make(map[string]int64, len(m.AdditionalMetrics)+1)
	for k, v := range m.AdditionalMetrics {
		additional[k] = v
	}
	if m.SpansDeduped > 0 {
		additional[metricSpansDeduped] += int64(m.SpansDeduped)
	}
	if len(additional) == 0 {
		additional = nil
	}

	return responseMetrics(&tempopb.SearchMetrics{
		InspectedBytes:    m.Bytes,
		InspectedSpans:    m.SpansTotal,
		BackendReads:      m.BackendReads,
		BackendBytes:      m.BackendBytes,
		AdditionalMetrics: additional,
	})
}

// addMetrics sums src into dst, which is allocated if needed.
func addMetrics(dst map[string]int64, src map[string]int64) map[string]int64 {
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

// ProcessMetrics holds the Prometheus metrics Tempo emitted while a case ran.
//
// Deltas and Gauges are reported apart because they are different claims:
// a delta is what this case added to a cumulative counter, a gauge is only the
// value left behind when the case finished.
type ProcessMetrics struct {
	// Deltas are counter, histogram and summary values accumulated by the
	// case. Only entries that moved are kept.
	Deltas map[string]float64 `json:"deltas,omitempty"`
	// Gauges are snapshots taken after the case, kept only when the case
	// changed them. Differencing a gauge is meaningless, so these are values,
	// not deltas.
	Gauges map[string]float64 `json:"gauges,omitempty"`
}

// metricSnapshot is one Gather, flattened.
type metricSnapshot struct {
	cumulative map[string]float64
	gauges     map[string]float64
}

// gatherMetrics flattens everything the gatherer reports.
//
// Every promauto metric in Tempo registers into prometheus.DefaultRegisterer at
// package init, so an in-process run sees them without a scrape endpoint. The
// gatherer is a parameter so the same collector works against a scraped
// registry if a run ever drives a separate process.
func gatherMetrics(g prometheus.Gatherer) (metricSnapshot, error) {
	snap := metricSnapshot{
		cumulative: map[string]float64{},
		gauges:     map[string]float64{},
	}
	if g == nil {
		return snap, nil
	}

	families, err := g.Gather()
	if err != nil {
		return snap, fmt.Errorf("gathering metrics: %w", err)
	}

	for _, mf := range families {
		name := mf.GetName()
		for _, m := range mf.GetMetric() {
			key := metricKey(name, m.GetLabel())

			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				snap.cumulative[key] = m.GetCounter().GetValue()

			case dto.MetricType_HISTOGRAM:
				h := m.GetHistogram()
				snap.cumulative[key+"_sum"] = h.GetSampleSum()
				snap.cumulative[key+"_count"] = float64(h.GetSampleCount())
				for _, b := range h.GetBucket() {
					le := strconv.FormatFloat(b.GetUpperBound(), 'g', -1, 64)
					snap.cumulative[metricKeyWith(name+"_bucket", m.GetLabel(), "le", le)] = float64(b.GetCumulativeCount())
				}

			case dto.MetricType_SUMMARY:
				s := m.GetSummary()
				snap.cumulative[key+"_sum"] = s.GetSampleSum()
				snap.cumulative[key+"_count"] = float64(s.GetSampleCount())

			case dto.MetricType_GAUGE:
				snap.gauges[key] = m.GetGauge().GetValue()

			default:
				// Untyped and native-histogram families carry no reliable
				// accumulation semantics, so treat them as snapshots.
				snap.gauges[key] = m.GetUntyped().GetValue()
			}
		}
	}
	return snap, nil
}

// since reduces two snapshots to what the case did: counters that moved, and
// gauges that ended up somewhere new. Filtering on change rather than on a
// name list means a metric added to Tempo is reported without a change here,
// and a metric the case never touched does not pad the result.
func (s metricSnapshot) since(before metricSnapshot) ProcessMetrics {
	var out ProcessMetrics

	for key, after := range s.cumulative {
		if d := after - before.cumulative[key]; d != 0 {
			if out.Deltas == nil {
				out.Deltas = map[string]float64{}
			}
			out.Deltas[key] = d
		}
	}

	for key, after := range s.gauges {
		if prev, ok := before.gauges[key]; !ok || prev != after {
			if out.Gauges == nil {
				out.Gauges = map[string]float64{}
			}
			out.Gauges[key] = after
		}
	}
	return out
}

// metricKey renders a metric as name{label="value",...}, with labels sorted so
// the key is stable across gathers.
func metricKey(name string, labels []*dto.LabelPair) string {
	return metricKeyWith(name, labels, "", "")
}

// metricKeyWith renders a key with one extra label appended, for histogram
// buckets, which carry their upper bound as a label rather than in the name.
func metricKeyWith(name string, labels []*dto.LabelPair, extraName, extraValue string) string {
	pairs := make([]string, 0, len(labels)+1)
	for _, l := range labels {
		pairs = append(pairs, l.GetName()+`="`+l.GetValue()+`"`)
	}
	if extraName != "" {
		pairs = append(pairs, extraName+`="`+extraValue+`"`)
	}
	if len(pairs) == 0 {
		return name
	}
	sort.Strings(pairs)

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	b.WriteString(strings.Join(pairs, ","))
	b.WriteByte('}')
	return b.String()
}
