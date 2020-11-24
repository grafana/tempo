package util

import (
	"bytes"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

func CombineTraces(objA []byte, objB []byte) []byte {
	// if the byte arrays are the same, we can return quickly
	if bytes.Compare(objA, objB) == 0 {
		return objA
	}

	// hashes differ.  unmarshal and combine traces
	traceA := &tempopb.Trace{}
	traceB := &tempopb.Trace{}

	errA := proto.Unmarshal(objA, traceA)
	if errA != nil {
		level.Error(util.Logger).Log("msg", "error unsmarshaling objA", "err", errA)
	}

	errB := proto.Unmarshal(objB, traceB)
	if errB != nil {
		level.Error(util.Logger).Log("msg", "error unsmarshaling objB", "err", errB)
	}

	// if we had problems unmarshaling one or the other, return the one that marshalled successfully
	if errA != nil && errB == nil {
		return objB
	} else if errB != nil && errA == nil {
		return objA
	} else if errA != nil && errB != nil {
		// if both failed let's send back an empty trace
		level.Error(util.Logger).Log("msg", "both A and B failed to unmarshal.  returning an empty trace")
		bytes, err := proto.Marshal(&tempopb.Trace{})
		if err != nil {
			level.Error(util.Logger).Log("msg", "somehow marshalling an empty trace threw an error.", "err", err)
		}
		return bytes
	}

	traceComplete, _, _, _ := CombineTraceProtos(traceA, traceB)

	bytes, err := proto.Marshal(traceComplete)
	if err != nil {
		level.Error(util.Logger).Log("msg", "marshalling the combine trace threw an error.", "err", err)
		return objA
	}
	return bytes
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

	spansInA := make(map[uint32]struct{})
	for _, batchA := range traceA.Batches {
		for _, ilsA := range batchA.InstrumentationLibrarySpans {
			for _, spanA := range ilsA.Spans {
				spanCountA++
				spanCountTotal++
				spansInA[TokenForTraceID(spanA.SpanId)] = struct{}{}
			}
		}
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, batchB := range traceB.Batches {
		notFoundILS := batchB.InstrumentationLibrarySpans[:0]

		for _, ilsB := range batchB.InstrumentationLibrarySpans {
			notFoundSpans := ilsB.Spans[:0]
			for _, spanB := range ilsB.Spans {
				spanCountB++
				// if found in A, remove from the batch
				_, ok := spansInA[TokenForTraceID(spanB.SpanId)]
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

	return traceA, spanCountA, spanCountB, spanCountTotal
}
