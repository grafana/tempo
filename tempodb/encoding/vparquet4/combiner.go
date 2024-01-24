package vparquet4

import (
	"bytes"
	"sort"

	"github.com/grafana/tempo/pkg/util"
)

func combineTraces(traces ...*Trace) *Trace {
	if len(traces) == 1 {
		return traces[0]
	}

	c := NewCombiner()
	for i := 0; i < len(traces); i++ {
		c.ConsumeWithFinal(traces[i], i == len(traces)-1)
	}
	res, _, _ := c.Result() // for now ignore the connected return. see comment in walBlock.Iterator()
	return res
}

// Combiner combines multiple partial traces into one, deduping spans based on
// ID and kind.  Note that it is destructive. There are design decisions for
// efficiency:
// * Only scan/hash the spans for each input once, which is reused across calls.
// * Only sort the final result once and if needed.
// * Don't scan/hash the spans for the last input (final=true).
type Combiner struct {
	result   *Trace
	spans    map[uint64]struct{}
	combined bool
}

func NewCombiner() *Combiner {
	return &Combiner{}
}

// Consume the given trace and destructively combines its contents.
func (c *Combiner) Consume(tr *Trace) (spanCount int) {
	return c.ConsumeWithFinal(tr, false)
}

// ConsumeWithFinal consumes the trace, but allows for performance savings when
// it is known that this is the last expected input trace.
func (c *Combiner) ConsumeWithFinal(tr *Trace, final bool) (spanCount int) {
	if tr == nil {
		return
	}

	// First call?
	if c.result == nil {
		c.result = tr

		// Pre-alloc map with input size. This saves having to grow the
		// map from the small starting size.
		n := 0
		for _, b := range c.result.ResourceSpans {
			for _, ils := range b.ScopeSpans {
				n += len(ils.Spans)
			}
		}
		c.spans = make(map[uint64]struct{}, n)

		for _, b := range c.result.ResourceSpans {
			for _, ils := range b.ScopeSpans {
				for _, s := range ils.Spans {
					c.spans[util.SpanIDAndKindToToken(s.SpanID, s.Kind)] = struct{}{}
				}
			}
		}
		return
	}

	// coalesce root level information
	if tr.EndTimeUnixNano > c.result.EndTimeUnixNano {
		c.result.EndTimeUnixNano = tr.EndTimeUnixNano
	}
	if tr.StartTimeUnixNano < c.result.StartTimeUnixNano || c.result.StartTimeUnixNano == 0 {
		c.result.StartTimeUnixNano = tr.StartTimeUnixNano
	}
	if c.result.RootServiceName == "" {
		c.result.RootServiceName = tr.RootServiceName
	}
	if c.result.RootSpanName == "" {
		c.result.RootSpanName = tr.RootSpanName
	}
	c.result.DurationNano = c.result.EndTimeUnixNano - c.result.StartTimeUnixNano

	// Merge service stats
	for service, incomingStats := range tr.ServiceStats {
		combinedStats := c.result.ServiceStats[service]
		combinedStats.SpanCount += incomingStats.SpanCount
		combinedStats.ErrorCount += incomingStats.ErrorCount
		c.result.ServiceStats[service] = combinedStats
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, b := range tr.ResourceSpans {
		notFoundILS := b.ScopeSpans[:0]

		for _, ils := range b.ScopeSpans {
			notFoundSpans := ils.Spans[:0]
			for _, s := range ils.Spans {
				// if not already encountered, then keep
				token := util.SpanIDAndKindToToken(s.SpanID, s.Kind)
				_, ok := c.spans[token]
				if !ok {
					notFoundSpans = append(notFoundSpans, s)

					// If last expected input, then we don't need to record
					// the visited spans. Optimization has significant savings.
					if !final {
						c.spans[token] = struct{}{}
					}
				}
			}

			if len(notFoundSpans) > 0 {
				ils.Spans = notFoundSpans
				spanCount += len(notFoundSpans)
				notFoundILS = append(notFoundILS, ils)
			}
		}

		// if there were some spans not found in A, add everything left in the batch
		if len(notFoundILS) > 0 {
			b.ScopeSpans = notFoundILS
			c.result.ResourceSpans = append(c.result.ResourceSpans, b)
		}
	}

	c.combined = true
	return
}

// Result returns the final trace, its span count, and a bool indicating whether the trace is a connected graph.
func (c *Combiner) Result() (*Trace, int, bool) {
	spanCount := -1

	connected := true
	if c.result != nil && c.combined {
		// Only if anything combined
		SortTrace(c.result)
		connected = assignNestedSetModelBounds(c.result)
		spanCount = len(c.spans)
	}

	return c.result, spanCount, connected
}

// SortTrace sorts a parquet *Trace
func SortTrace(t *Trace) {
	// Sort bottom up by span start times
	for _, b := range t.ResourceSpans {
		for _, ils := range b.ScopeSpans {
			sort.Slice(ils.Spans, func(i, j int) bool {
				return compareSpans(&ils.Spans[i], &ils.Spans[j])
			})
		}
		sort.Slice(b.ScopeSpans, func(i, j int) bool {
			return compareScopeSpans(&b.ScopeSpans[i], &b.ScopeSpans[j])
		})
	}
	sort.Slice(t.ResourceSpans, func(i, j int) bool {
		return compareBatches(&t.ResourceSpans[i], &t.ResourceSpans[j])
	})
}

func compareBatches(a, b *ResourceSpans) bool {
	if len(a.ScopeSpans) > 0 && len(b.ScopeSpans) > 0 {
		return compareScopeSpans(&a.ScopeSpans[0], &b.ScopeSpans[0])
	}
	return false
}

func compareScopeSpans(a, b *ScopeSpans) bool {
	if len(a.Spans) > 0 && len(b.Spans) > 0 {
		return compareSpans(&a.Spans[0], &b.Spans[0])
	}
	return false
}

func compareSpans(a, b *Span) bool {
	// Sort by start time, then id
	if a.StartTimeUnixNano == b.StartTimeUnixNano {
		return bytes.Compare(a.SpanID, b.SpanID) == -1
	}

	return a.StartTimeUnixNano < b.StartTimeUnixNano
}
