package distributor

import (
	"sync"

	"github.com/grafana/tempo/pkg/tempopb"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

var pushRequestPool = sync.Pool{
	New: func() interface{} {
		return &tempopb.PushRequest{}
	},
}

var resourceSpansPool = sync.Pool{
	New: func() interface{} {
		return &opentelemetry_proto_trace_v1.ResourceSpans{}
	},
}

func resetPushRequests(reqsPerIngester map[string][]*tempopb.PushRequest) {
	for _, reqs := range reqsPerIngester {
		for _, req := range reqs {
			req.Batch.Spans = nil
			req.Batch.Resource = nil

			resourceSpansPool.Put(req.Batch)
			pushRequestPool.Put(req)
		}
	}
}
