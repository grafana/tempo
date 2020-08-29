package test

import "github.com/grafana/tempo/pkg/tempopb"

func MustTraceID(req *tempopb.PushRequest) []byte {
	if req == nil || req.Batch == nil {
		panic("req nil")
	}

	if len(req.Batch.InstrumentationLibrarySpans) == 0 {
		panic("ils len 0")
	}

	if len(req.Batch.InstrumentationLibrarySpans[0].Spans) == 0 {
		panic("span len 0")
	}

	return req.Batch.InstrumentationLibrarySpans[0].Spans[0].TraceId
}
