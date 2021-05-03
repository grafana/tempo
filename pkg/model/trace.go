package model

import (
	"bytes"
	"hash"
	"hash/fnv"
	"sort"

	"github.com/pkg/errors"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// todo(jpe):
// - add cross data encoding tests

func CombineTraceBytes(objA []byte, objB []byte, dataEncodingA string, dataEncodingB string) (_ []byte, wasCombined bool, _ error) {
	// if the byte arrays are the same, we can return quickly
	if bytes.Equal(objA, objB) {
		return objA, false, nil
	}

	// bytes differ.  unmarshal and combine traces
	traceA, errA := Unmarshal(objA, dataEncodingA)
	traceB, errB := Unmarshal(objB, dataEncodingB)

	// if we had problems unmarshaling one or the other, return the one that marshalled successfully
	if errA != nil && errB == nil {
		return objB, false, errors.Wrap(errA, "error unsmarshaling objA")
	} else if errB != nil && errA == nil {
		return objA, false, errors.Wrap(errB, "error unsmarshaling objB")
	} else if errA != nil && errB != nil {
		// if both failed let's send back an empty trace
		level.Error(log.Logger).Log("msg", "both A and B failed to unmarshal.  returning an empty trace")
		bytes, _ := proto.Marshal(&tempopb.Trace{})
		return bytes, false, errors.Wrap(errA, "both A and B failed to unmarshal.  returning an empty trace")
	}

	traceComplete, _, _, _ := CombineTraceProtos(traceA, traceB)

	bytes, err := proto.Marshal(traceComplete)
	if err != nil {
		return objA, true, errors.Wrap(err, "marshalling the combine trace threw an error")
	}
	return bytes, true, nil
}

// CombineTraceProtos combines two trace protos into one.  Note that it is destructive.
//  All spans are combined into traceA.  spanCountA, B, and Total are returned for
//  logging purposes.
func CombineTraceProtos(traceA, traceB *tempopb.Trace) (*tempopb.Trace, int, int, int) {
	// if one or the other is nil just return 0 for the one that's nil and -1 for the other.  this will be a clear indication this
	// code path was taken without unnecessarily counting spans
	if traceA == nil {
		return traceB, 0, -1, -1
	}

	if traceB == nil {
		return traceA, -1, 0, -1
	}

	spanCountA := 0
	spanCountB := 0
	spanCountTotal := 0

	h := fnv.New32()

	spansInA := make(map[uint32]struct{})
	for _, batchA := range traceA.Batches {
		for _, ilsA := range batchA.InstrumentationLibrarySpans {
			for _, spanA := range ilsA.Spans {
				spansInA[tokenForID(h, spanA.SpanId)] = struct{}{}
			}
			spanCountA += len(ilsA.Spans)
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
				_, ok := spansInA[tokenForID(h, spanB.SpanId)]
				if !ok {
					notFoundSpans = append(notFoundSpans, spanB)
				}
			}
			spanCountB += len(ilsB.Spans)

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

	return traceA, spanCountA, spanCountB, spanCountTotal
}

func SortTrace(t *tempopb.Trace) {
	// Sort bottom up by span start times
	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			sort.Slice(ils.Spans, func(i, j int) bool {
				return compareSpans(ils.Spans[i], ils.Spans[j])
			})
		}
		sort.Slice(b.InstrumentationLibrarySpans, func(i, j int) bool {
			return compareIls(b.InstrumentationLibrarySpans[i], b.InstrumentationLibrarySpans[j])
		})
	}
	sort.Slice(t.Batches, func(i, j int) bool {
		return compareBatches(t.Batches[i], t.Batches[j])
	})
}

func compareBatches(a *v1.ResourceSpans, b *v1.ResourceSpans) bool {
	if len(a.InstrumentationLibrarySpans) > 0 && len(b.InstrumentationLibrarySpans) > 0 {
		return compareIls(a.InstrumentationLibrarySpans[0], b.InstrumentationLibrarySpans[0])
	}
	return false
}

func compareIls(a *v1.InstrumentationLibrarySpans, b *v1.InstrumentationLibrarySpans) bool {
	if len(a.Spans) > 0 && len(b.Spans) > 0 {
		return compareSpans(a.Spans[0], b.Spans[0])
	}
	return false
}

func compareSpans(a *v1.Span, b *v1.Span) bool {
	// Sort by start time, then id

	if a.StartTimeUnixNano == b.StartTimeUnixNano {
		return bytes.Compare(a.SpanId, b.SpanId) == -1
	}

	return a.StartTimeUnixNano < b.StartTimeUnixNano
}

func tokenForID(h hash.Hash32, b []byte) uint32 {
	h.Reset()
	_, _ = h.Write(b)
	return h.Sum32()
}
