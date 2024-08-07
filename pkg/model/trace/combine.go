package trace

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"

	"github.com/grafana/tempo/pkg/tempopb"
)

// token is uint64 to reduce hash collision rates.  Experimentally, it was observed
// that fnv32 could approach a collision rate of 1 in 10,000. fnv64 avoids collisions
// when tested against traces with up to 1M spans (see matching test). A collision
// results in a dropped span during combine.
type token uint64

func newHash() hash.Hash64 {
	return fnv.New64()
}

// tokenForID returns a token for use in a hash map given a span id and span kind
// buffer must be a 4 byte slice and is reused for writing the span kind to the hashing function
// kind is used along with the actual id b/c in zipkin traces span id is not guaranteed to be unique
// as it is shared between client and server spans.
func tokenForID(h hash.Hash64, buffer []byte, kind int32, b []byte) token {
	binary.LittleEndian.PutUint32(buffer, uint32(kind))

	h.Reset()
	_, _ = h.Write(b)
	_, _ = h.Write(buffer)
	return token(h.Sum64())
}

var ErrTraceTooLarge = fmt.Errorf("trace exceeds max size")

// Combiner combines multiple partial traces into one, deduping spans based on
// ID and kind.  Note that it is destructive. There are design decisions for
// efficiency:
// * Only scan/hash the spans for each input once, which is reused across calls.
// * Only sort the final result once and if needed.
// * Don't scan/hash the spans for the last input (final=true).
type Combiner struct {
	result              *tempopb.Trace
	spans               map[token]struct{}
	combined            bool
	maxSizeBytes        int
	allowPartialTrace   bool
	maxTraceSizeReached bool
}

func NewCombiner(maxSizeBytes int, allowPartialTrace bool) *Combiner {
	return &Combiner{
		maxSizeBytes:      maxSizeBytes,
		allowPartialTrace: allowPartialTrace,
	}
}

// Consume the given trace and destructively combines its contents.
func (c *Combiner) Consume(tr *tempopb.Trace) (int, error) {
	return c.ConsumeWithFinal(tr, false)
}

// ConsumeWithFinal consumes the trace, but allows for performance savings when
// it is known that this is the last expected input trace.
func (c *Combiner) ConsumeWithFinal(tr *tempopb.Trace, final bool) (int, error) {
	var spanCount int
	if tr == nil {
		return spanCount, nil
	}

	h := newHash()
	buffer := make([]byte, 4)

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
		c.spans = make(map[token]struct{}, n)

		for _, b := range c.result.ResourceSpans {
			for _, ils := range b.ScopeSpans {
				for _, s := range ils.Spans {
					c.spans[tokenForID(h, buffer, int32(s.Kind), s.SpanId)] = struct{}{}
				}
			}
		}
		maxSizeErr := c.sizeError()
		if maxSizeErr != nil && c.allowPartialTrace {
			return spanCount, nil
		}
		return spanCount, maxSizeErr
	}

	// Do not combine more spans for now
	if c.maxTraceSizeReached && c.allowPartialTrace {
		return spanCount, nil
	}
	// loop through every span and copy spans in B that don't exist to A
	for _, b := range tr.ResourceSpans {
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
			c.result.ResourceSpans = append(c.result.ResourceSpans, b)
		}
	}

	c.combined = true
	maxSizeErr := c.sizeError()
	if maxSizeErr != nil && c.allowPartialTrace {
		return spanCount, nil
	}

	return spanCount, maxSizeErr
}

func (c *Combiner) sizeError() error {
	// Should we allow a maxSizeBytes <= 0?
	if c.result == nil || c.maxSizeBytes <= 0 {
		return nil
	}

	if c.result.Size() > c.maxSizeBytes {
		// To avoid recalculing the size
		c.maxTraceSizeReached = true
		return fmt.Errorf("%w (max bytes: %d)", ErrTraceTooLarge, c.maxSizeBytes)
	}

	return nil
}

// Result returns the final trace and span count.
func (c *Combiner) Result() (*tempopb.Trace, int) {
	spanCount := -1

	if c.result != nil && c.combined {
		// Only if anything combined
		SortTrace(c.result)
		spanCount = len(c.spans)
	}

	return c.result, spanCount
}
