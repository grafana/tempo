package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Process holds the Prometheus metrics Tempo emitted while a case ran.
//
// Deltas and Gauges are kept apart because they are different claims: a delta
// is what the case added to a counter, a gauge is only the value left behind
// when it finished.
type Process struct {
	// Deltas are counter, histogram and summary values the case accumulated.
	// Only entries that moved are kept.
	Deltas map[string]float64 `json:"deltas,omitempty"`
	// Gauges are values read after the case, kept only when the case changed
	// them. Differencing a gauge is meaningless, so these are not deltas.
	Gauges map[string]float64 `json:"gauges,omitempty"`
}

// Snapshot is one Gather, flattened.
type Snapshot struct {
	cumulative map[string]float64
	gauges     map[string]float64
}

// Gather flattens everything the gatherer reports.
//
// Every promauto metric in Tempo registers into prometheus.DefaultRegisterer at
// package init, so an in-process run sees them without a scrape endpoint. The
// gatherer is a parameter so the same code works against a scraped registry if
// a run ever drives a separate process.
func Gather(g prometheus.Gatherer) (Snapshot, error) {
	snap := Snapshot{
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
				// Untyped and native-histogram families have no reliable
				// accumulation semantics, so treat them as snapshots.
				snap.gauges[key] = m.GetUntyped().GetValue()
			}
		}
	}
	return snap, nil
}

// Since reduces two snapshots to what the case did: counters that moved, and
// gauges that ended up somewhere new.
//
// Filtering on change rather than on a list of names means a metric added to
// Tempo is reported without a change here, and a metric the case never touched
// does not pad the result.
func (s Snapshot) Since(before Snapshot) Process {
	var out Process

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

// metricKeyWith appends one extra label, for histogram buckets, which carry
// their upper bound as a label rather than in the name.
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
