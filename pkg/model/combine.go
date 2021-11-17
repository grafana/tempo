package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"

	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/pkg/errors"
)

type objectCombiner struct{}

var ObjectCombiner = objectCombiner{}

var _ common.ObjectCombiner = (*objectCombiner)(nil)

// Combine implements tempodb/encoding/common.ObjectCombiner
func (o objectCombiner) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	if len(objs) <= 0 {
		return nil, false, errors.New("no objects provided")
	}

	// check to see if we need to combine
	needCombine := false
	for i := 1; i < len(objs); i++ {
		if !bytes.Equal(objs[0], objs[i]) {
			needCombine = true
			break
		}
	}

	if !needCombine {
		return objs[0], false, nil
	}

	var combinedTrace *tempopb.Trace
	for _, obj := range objs {
		trace, err := Unmarshal(obj, dataEncoding)
		if err != nil {
			return nil, false, fmt.Errorf("error unmarshaling trace: %w", err)
		}

		combinedTrace, _, _, _ = CombineTraceProtos(combinedTrace, trace)
	}

	combinedBytes, err := marshal(combinedTrace, dataEncoding)
	if err != nil {
		return nil, false, fmt.Errorf("error marshaling combinedBytes: %w", err)
	}

	return combinedBytes, true, nil
}

// CombineTraceBytes combines objA and objB encoded using dataEncodingA and dataEncodingB and returns a trace encoded with dataEncodingA
func CombineTraceBytes(objA []byte, objB []byte, dataEncodingA string, dataEncodingB string) (_ []byte, wasCombined bool, _ error) {
	// if the byte arrays are the same, we can return quickly
	if bytes.Equal(objA, objB) {
		return objA, false, nil
	}
	if objB == nil {
		return objA, false, nil
	}

	// bytes differ.  unmarshal and combine traces
	traceA, errA := Unmarshal(objA, dataEncodingA)
	traceB, errB := Unmarshal(objB, dataEncodingB)

	// if we had problems unmarshaling one or the other, return the one that marshalled successfully
	if errA != nil && errB == nil {
		if dataEncodingA != dataEncodingB {
			// have to convert objB to dataEncodingA
			bytes, _ := marshal(traceB, dataEncodingA)
			return bytes, false, fmt.Errorf("error unsmarshaling objA (%s): %w", dataEncodingA, errA)
		}
		return objB, false, fmt.Errorf("error unsmarshaling objA (%s): %w", dataEncodingA, errA)
	} else if errB != nil && errA == nil {
		return objA, false, fmt.Errorf("error unsmarshaling objB (%s): %w", dataEncodingB, errB)
	} else if errA != nil && errB != nil {
		// if both failed let's send back an empty trace
		bytes, _ := marshal(&tempopb.Trace{}, dataEncodingA)
		return bytes, false, fmt.Errorf("both A (%s) and B (%s) failed to unmarshal. returning an empty trace", dataEncodingA, dataEncodingB)
	}

	traceComplete, _, _, _ := CombineTraceProtos(traceA, traceB)

	bytes, err := marshal(traceComplete, dataEncodingA)
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
	buffer := make([]byte, 4)

	spansInA := make(map[uint32]struct{})
	for _, batchA := range traceA.Batches {
		for _, ilsA := range batchA.InstrumentationLibrarySpans {
			for _, spanA := range ilsA.Spans {
				spansInA[tokenForID(h, buffer, int32(spanA.Kind), spanA.SpanId)] = struct{}{}
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
				_, ok := spansInA[tokenForID(h, buffer, int32(spanB.Kind), spanB.SpanId)]
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

// tokenForID returns a uint32 token for use in a hash map given a span id and span kind
//  buffer must be a 4 byte slice and is reused for writing the span kind to the hashing function
//  kind is used along with the actual id b/c in zipkin traces span id is not guaranteed to be unique
//  as it is shared between client and server spans.
func tokenForID(h hash.Hash32, buffer []byte, kind int32, b []byte) uint32 {
	binary.LittleEndian.PutUint32(buffer, uint32(kind))

	h.Reset()
	_, _ = h.Write(b)
	_, _ = h.Write(buffer)
	return h.Sum32()
}
