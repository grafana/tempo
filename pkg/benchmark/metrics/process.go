package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

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

// observeInto records one execution's worth of registry movement: what each
// counter added, and where each gauge ended up.
//
// Recording every series and filtering at the end, rather than filtering
// against a list of names, means a metric added to Tempo is reported without a
// change here.
func (s Snapshot) observeInto(before Snapshot, c *Collector) {
	for key, after := range s.cumulative {
		c.observe(Counter, PrefixProcess+key, after-before.cumulative[key])
	}
	for key, after := range s.gauges {
		c.observe(Gauge, PrefixProcess+key, after)
	}
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
