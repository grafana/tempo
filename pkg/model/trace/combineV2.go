package trace

import (
	"fmt"

	"github.com/grafana/tempo/pkg/tempopb"
)

type CombinerV2 struct {
	result       *tempopb.TraceV2
	spans        map[token]struct{}
	combined     bool
	maxSizeBytes int
}

func NewCombinerV2(maxSizeBytes int) *CombinerV2 {
	return &CombinerV2{
		maxSizeBytes: maxSizeBytes,
	}
}

// Consume the given trace and destructively combines its contents.
func (c *CombinerV2) Consume(tr *tempopb.TraceV2) (int, error) {
	return c.ConsumeWithFinalV2(tr, false)
}

// ConsumeWithFinal consumes the trace, but allows for performance savings when
// it is known that this is the last expected input trace.
func (c *CombinerV2) ConsumeWithFinalV2(tr *tempopb.TraceV2, final bool) (int, error) {
	var spanCount int
	if tr == nil {
		return spanCount, c.sizeError()
	}

	h := newHash()
	buffer := make([]byte, 4)

	// First call?
	if c.result == nil {
		c.result = tr

		// Pre-alloc map with input size. This saves having to grow the
		// map from the small starting size.
		n := 0
		for _, b := range c.result.TraceData.ResourceSpans {
			for _, ils := range b.ScopeSpans {
				n += len(ils.Spans)
			}
		}
		c.spans = make(map[token]struct{}, n)

		for _, b := range c.result.TraceData.ResourceSpans {
			for _, ils := range b.ScopeSpans {
				for _, s := range ils.Spans {
					c.spans[tokenForID(h, buffer, int32(s.Kind), s.SpanId)] = struct{}{}
				}
			}
		}
		return spanCount, c.sizeError()
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, b := range tr.TraceData.ResourceSpans {
		notFoundILS := b.ScopeSpans[:0]

		for _, ils := range b.ScopeSpans {
			notFoundSpans := ils.Spans[:0]
			for _, s := range ils.Spans {
				// if not already encountered, then keep
				token := tokenForID(h, buffer, int32(s.Kind), s.SpanId)
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
			c.result.TraceData.ResourceSpans = append(c.result.TraceData.ResourceSpans, b)
		}
	}

	c.combined = true
	return spanCount, c.sizeError()
}

func (c *CombinerV2) sizeError() error {
	if c.result == nil || c.maxSizeBytes <= 0 {
		return nil
	}

	if c.result.Size() > c.maxSizeBytes {
		return fmt.Errorf("%w (max bytes: %d)", ErrTraceTooLarge, c.maxSizeBytes)
	}

	return nil
}

// Result returns the final trace and span count.
func (c *CombinerV2) Result() (*tempopb.TraceV2, int) {
	spanCount := -1

	if c.result != nil && c.combined {
		// Only if anything combined
		SortTraceV2(c.result)
		spanCount = len(c.spans)
	}

	return c.result, spanCount
}
