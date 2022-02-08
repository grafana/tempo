package trace

import (
	"encoding/binary"
	"hash"
	"hash/fnv"

	"github.com/grafana/tempo/pkg/tempopb"
)

// CombineTraceProtos combines two trace protos into one.  Note that it is destructive.
//  All spans are combined into traceA.  spanCountA, B, and Total are returned for
//  logging purposes.
func CombineTraceProtos(traceA, traceB *tempopb.Trace) (*tempopb.Trace, int) {
	// if one or the other is nil just return 0 for the one that's nil and -1 for the other.  this will be a clear indication this
	// code path was taken without unnecessarily counting spans
	if traceA == nil {
		return traceB, -1
	}

	if traceB == nil {
		return traceA, -1
	}

	spanCountTotal := 0

	h := newHash()
	buffer := make([]byte, 4)

	spansInA := make(map[token]struct{})
	for _, batchA := range traceA.Batches {
		for _, ilsA := range batchA.InstrumentationLibrarySpans {
			for _, spanA := range ilsA.Spans {
				spansInA[tokenForID(h, buffer, int32(spanA.Kind), spanA.SpanId)] = struct{}{}
			}
			spanCountTotal += len(ilsA.Spans)
		}
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, batchB := range traceB.Batches {
		notFoundILS := batchB.InstrumentationLibrarySpans[:0]

		for _, ilsB := range batchB.InstrumentationLibrarySpans {
			notFoundSpans := ilsB.Spans[:0]
			for _, spanB := range ilsB.Spans {
				// if found in A, remove from the batch
				_, ok := spansInA[tokenForID(h, buffer, int32(spanB.Kind), spanB.SpanId)]
				if !ok {
					notFoundSpans = append(notFoundSpans, spanB)
				}
			}

			if len(notFoundSpans) > 0 {
				spanCountTotal += len(notFoundSpans)
				ilsB.Spans = notFoundSpans
				notFoundILS = append(notFoundILS, ilsB)
			}
		}

		// if there were some spans not found in A, add everything left in the batch
		if len(notFoundILS) > 0 {
			batchB.InstrumentationLibrarySpans = notFoundILS
			traceA.Batches = append(traceA.Batches, batchB)
		}
	}

	SortTrace(traceA)

	return traceA, spanCountTotal
}

type token uint64

func newHash() hash.Hash64 {
	return fnv.New64()
}

// tokenForID returns a uint32 token for use in a hash map given a span id and span kind
//  buffer must be a 4 byte slice and is reused for writing the span kind to the hashing function
//  kind is used along with the actual id b/c in zipkin traces span id is not guaranteed to be unique
//  as it is shared between client and server spans.
func tokenForID(h hash.Hash64, buffer []byte, kind int32, b []byte) token {
	binary.LittleEndian.PutUint32(buffer, uint32(kind))

	h.Reset()
	_, _ = h.Write(b)
	_, _ = h.Write(buffer)
	return token(h.Sum64())
}

type Combiner struct {
	result   *tempopb.Trace
	spans    map[token]struct{}
	combined bool
}

func NewCombiner() *Combiner {
	return &Combiner{
		spans: map[token]struct{}{},
	}
}

func (c *Combiner) ConsumeAll(traces ...*tempopb.Trace) {
	for _, t := range traces {
		c.Consume(t)
	}
}

func (c *Combiner) Consume(tr *tempopb.Trace) {
	if tr == nil {
		return
	}

	h := newHash()
	buffer := make([]byte, 4)

	// First call?
	if c.result == nil {
		c.result = tr
		for _, b := range c.result.Batches {
			for _, ils := range b.InstrumentationLibrarySpans {
				for _, s := range ils.Spans {
					c.spans[tokenForID(h, buffer, int32(s.Kind), s.SpanId)] = struct{}{}
				}
			}
		}
		return
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, b := range tr.Batches {
		notFoundILS := b.InstrumentationLibrarySpans[:0]

		for _, ils := range b.InstrumentationLibrarySpans {
			notFoundSpans := ils.Spans[:0]
			for _, s := range ils.Spans {
				// if not already encountered, then keep
				token := tokenForID(h, buffer, int32(s.Kind), s.SpanId)
				_, ok := c.spans[token]
				if !ok {
					notFoundSpans = append(notFoundSpans, s)
					c.spans[token] = struct{}{}
				}
			}

			if len(notFoundSpans) > 0 {
				ils.Spans = notFoundSpans
				notFoundILS = append(notFoundILS, ils)
			}
		}

		// if there were some spans not found in A, add everything left in the batch
		if len(notFoundILS) > 0 {
			b.InstrumentationLibrarySpans = notFoundILS
			c.result.Batches = append(c.result.Batches, b)
		}
	}

	c.combined = true
}

func (c *Combiner) Result() (*tempopb.Trace, int) {
	spanCount := -1

	if c.result != nil && c.combined {
		// Only if anything combined
		SortTrace(c.result)
		spanCount = len(c.spans)
	}

	return c.result, spanCount
}
