package attributes

import (
	"math"
	"slices"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/cespare/xxhash/v2"

	"github.com/grafana/tempo/v3/pkg/traceql"
)

const (
	// topValues is how many values an attribute profile keeps. At most 100
	// values can each be carried by 1% of spans, so every value a benchmark
	// targets survives the cut.
	topValues = 100
	// valueCounters bounds the counters kept per non-numeric attribute. Past
	// it a count can be over-stated by at most 1/valueCounters of the spans
	// carrying the attribute, far below the selectivities benchmarked.
	valueCounters = 4096
)

// profileQuantiles give `> value` predicates from 0.99 down to 0.01
// selectivity.
var profileQuantiles = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99}

type attrKey struct {
	scope, name, typ string
}

// attrAccumulator gathers one attribute's statistics over a scan, in memory
// bounded however many spans or distinct values it sees.
type attrAccumulator struct {
	key        attrKey
	totalBytes uint64
	spans      uint64
	sketch     *hyperloglog.Sketch

	// counter is nil once a numeric attribute has too many values to list.
	counter *spaceSaving
	// hist is set for numeric types.
	hist *logHistogram
}

func newAttrAccumulator(k attrKey) *attrAccumulator {
	a := &attrAccumulator{key: k, sketch: hyperloglog.New14()}
	if !isNumericType(k.typ) {
		a.counter = newSpaceSaving(valueCounters)
		return a
	}
	// A numeric attribute only lists its values while they all fit, so its
	// counter holds exactly that many and is exact until it drops.
	a.counter = newSpaceSaving(topValues)
	a.hist = newLogHistogram()
	return a
}

// observe records one stored occurrence of the attribute, given every value it
// holds (one for a scalar, each element for an array) and the spans it covers:
// one for a span attribute, every span beneath it for a resource attribute.
// Bytes count once per occurrence, as they are stored; spans and values count
// once per span, as a query matches them.
func (a *attrAccumulator) observe(vals []traceql.Static, spans uint64) {
	a.spans += spans
	var buf [8]string
	distinct := buf[:0]
	largest := math.Inf(-1)

	for _, v := range vals {
		s := v.EncodeToString(false)
		a.totalBytes += valueSize(v, s)
		if a.hist != nil {
			largest = max(largest, v.Float())
		}

		// A span matches `= value` once however often an array repeats it.
		if slices.Contains(distinct, s) {
			continue
		}
		distinct = append(distinct, s)
		a.sketch.InsertHash(xxhash.Sum64String(s))
		a.addValue(s, spans)
	}

	// `> x` matches a span if any element exceeds x, which is when its largest
	// does.
	if a.hist != nil {
		a.hist.add(largest, spans)
	}
}

func (a *attrAccumulator) addValue(s string, spans uint64) {
	if a.counter == nil {
		return
	}
	a.counter.add(s, spans)
	// A numeric list truncated at topValues would hide the rest of the
	// distribution, so from its first eviction quantiles describe it instead.
	if a.hist != nil && a.counter.evicted {
		a.counter = nil
	}
}

// valueSize is what a value costs in the ranking: a string's length, a
// number's width, and a byte for anything enumerated.
func valueSize(v traceql.Static, s string) uint64 {
	switch v.Type {
	case traceql.TypeString:
		return uint64(len(s))
	case traceql.TypeInt, traceql.TypeFloat, traceql.TypeDuration:
		return 8
	default:
		return 1
	}
}

func (a *attrAccumulator) finish(totalSpans uint64) Profile {
	p := Profile{
		Scope:      a.key.scope,
		Name:       a.key.name,
		Type:       a.key.typ,
		TotalBytes: a.totalBytes,
		Density:    float64(a.spans) / float64(totalSpans),
	}

	if a.counter == nil {
		p.Cardinality = max(a.sketch.Estimate(), topValues+1)
		for _, q := range a.hist.quantiles(profileQuantiles, totalSpans) {
			p.Quantiles = append(p.Quantiles, Quantile{
				Q:           q.q,
				Value:       formatBound(q.upper, a.key.typ),
				Selectivity: q.selectivity,
			})
		}
		return p
	}

	p.Cardinality = uint64(a.counter.distinct())
	if a.counter.evicted {
		p.Cardinality = max(a.sketch.Estimate(), valueCounters+1)
	}
	for _, c := range a.counter.top(topValues) {
		p.Values = append(p.Values, Value{
			Value:       c.value,
			Selectivity: float64(c.count) / float64(totalSpans),
		})
	}
	return p
}

// formatBound renders a bucket bound as a value of the attribute's type, the
// way values are rendered. An integer is at most the bound exactly when it is
// at most the bound's floor, so the floor keeps the selectivity exact.
func formatBound(b float64, typ string) string {
	switch typ {
	case TypeInt:
		return traceql.NewStaticInt(int(math.Floor(b))).EncodeToString(false)
	case TypeDuration:
		return traceql.NewStaticDuration(time.Duration(math.Floor(b))).EncodeToString(false)
	default:
		return traceql.NewStaticFloat(b).EncodeToString(false)
	}
}
