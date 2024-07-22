package trace

import (
	pb "github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// ConvertToTraceV2 converts a Trace message to a TraceV2 message
func ConvertToTraceV2(trace *pb.Trace) *pb.TraceV2 {
	return &pb.TraceV2{
		TraceData: &v1.TracesData{
			// This is not copying the content, may be an issue
			ResourceSpans: trace.Batches,
		},
	}
}

func ConvertToTraceV1(trace *pb.TraceV2) *pb.Trace {
	return &pb.Trace{
		// This is not copying the content, may be an issue
		Batches: trace.TraceData.ResourceSpans,
	}
}
