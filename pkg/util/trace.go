package util

import (
	"hash/fnv"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

func CombineTraces(objA []byte, objB []byte) []byte {
	// if the byte arrays are the same, we can return quickly
	hasher := fnv.New32a()

	_, _ = hasher.Write(objA)
	hashA := hasher.Sum32()
	hasher.Reset()
	_, _ = hasher.Write(objB)
	hashB := hasher.Sum32()
	if hashA == hashB {
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

	spansInA := make(map[uint64]struct{})
	for _, batchA := range traceA.Batches {
		for _, spanA := range batchA.Spans {
			spansInA[Fingerprint(spanA.SpanId)] = struct{}{}
		}
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, batchB := range traceB.Batches {
		notFoundSpans := batchB.Spans[:0]
		for _, spanB := range batchB.Spans {
			// if found in A, remove from the batch
			_, ok := spansInA[Fingerprint(spanB.SpanId)]
			if !ok {
				notFoundSpans = append(notFoundSpans, spanB)
			}
		}

		// if there were some spans not found in A, add everything left in the batch
		if len(notFoundSpans) > 0 {
			batchB.Spans = notFoundSpans
			traceA.Batches = append(traceA.Batches, batchB)
		}
	}

	bytes, err := proto.Marshal(traceA)
	if err != nil {
		level.Error(util.Logger).Log("msg", "marshalling the combine trace threw an error.", "err", err)
		return objA
	}
	return bytes
}
