package util

import (
	"bytes"
	"hash"
	"hash/fnv"

	"github.com/pkg/errors"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

func CombineTraces(objA []byte, objB []byte) ([]byte, error) {
	// if the byte arrays are the same, we can return quickly
	if bytes.Equal(objA, objB) {
		return objA, nil
	}

	// hashes differ.  unmarshal and combine traces
	traceA := &tempopb.Trace{}
	traceB := &tempopb.Trace{}

	errA := proto.Unmarshal(objA, traceA)
	errB := proto.Unmarshal(objB, traceB)

	// if we had problems unmarshaling one or the other, return the one that marshalled successfully
	if errA != nil && errB == nil {
		return objB, errors.Wrap(errA, "error unsmarshaling objA")
	} else if errB != nil && errA == nil {
		return objA, errors.Wrap(errB, "error unsmarshaling objB")
	} else if errA != nil && errB != nil {
		// if both failed let's send back an empty trace
		level.Error(log.Logger).Log("msg", "both A and B failed to unmarshal.  returning an empty trace")
		bytes, _ := proto.Marshal(&tempopb.Trace{})
		return bytes, errors.Wrap(errA, "both A and B failed to unmarshal.  returning an empty trace")
	}

	traceComplete, _, _, _ := CombineTraceProtos(traceA, traceB)

	bytes, err := proto.Marshal(traceComplete)
	if err != nil {
		return objA, errors.Wrap(err, "marshalling the combine trace threw an error")
	}
	return bytes, nil
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

	return traceA, spanCountA, spanCountB, spanCountTotal
}

func tokenForID(h hash.Hash32, b []byte) uint32 {
	h.Reset()
	_, _ = h.Write(b)
	return h.Sum32()
}
