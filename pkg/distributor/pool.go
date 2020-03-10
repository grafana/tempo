package distributor

import (
	"sync"

	"github.com/grafana/frigg/pkg/friggpb"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

var pushRequestPool = sync.Pool{
	New: func() interface{} {
		return &friggpb.PushRequest{}
	},
}

var resourceSpansPool = sync.Pool{
	New: func() interface{} {
		return &opentelemetry_proto_trace_v1.ResourceSpans{}
	},
}

func resetPushRequests(reqsPerIngester map[string][]*friggpb.PushRequest) {
	for _, reqs := range reqsPerIngester {
		for _, req := range reqs {
			req.Batch.Spans = nil
			req.Batch.Resource = nil
		}
	}
}
