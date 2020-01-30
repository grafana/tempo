package frigg

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/frigg/pkg/friggpb"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	opentelemetry_resource_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

type Backend struct {
	friggEndpoint string
}

func New(cfg *Config) *Backend {
	return &Backend{
		friggEndpoint: "http://frigg:3100/api/traces/", //jpe
	}
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)
	resp, err := http.Get(b.friggEndpoint + hexID)
	if err != nil {
		return nil, err
	}

	out := &friggpb.Trace{}
	err = json.NewDecoder(resp.Body).Decode(out)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if len(out.Batches) == 0 {
		return nil, fmt.Errorf("TraceID Not Found: " + hexID)
	}

	jaegerTrace := &model.Trace{
		Spans:      []*model.Span{},
		ProcessMap: []model.Trace_ProcessMapping{},
	}

	// now convert trace to jaeger
	// todo: remove custom code in favor of otelcol once it's complete
	for _, batch := range out.Batches {
		//jaegerTrace.ProcessMap = addResourceIfNotExists(jaegerTrace.ProcessMap, batch.Resource)
		for _, span := range batch.Spans {
			jaegerTrace.Spans = append(jaegerTrace.Spans, protoSpanToJaegerSpan(span, batch.Resource))
		}
	}

	return jaegerTrace, nil
}

func (b *Backend) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (b *Backend) GetOperations(ctx context.Context, query spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	return nil, nil
}
func (b *Backend) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}
func (b *Backend) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}
func (b *Backend) WriteSpan(span *model.Span) error {
	return nil
}

func protoSpanToJaegerSpan(in *opentelemetry_proto_trace_v1.Span, resource *opentelemetry_resource_trace_v1.Resource) *model.Span {

	traceID := model.TraceID{
		High: binary.BigEndian.Uint64(in.TraceId[:8]),
		Low:  binary.BigEndian.Uint64(in.TraceId[8:]),
	}

	s := &model.Span{
		TraceID:       traceID,
		SpanID:        model.SpanID(binary.BigEndian.Uint64(in.SpanId)),
		OperationName: in.Name,
		StartTime:     time.Unix(0, int64(in.StartTimeUnixnano)),
		Duration:      time.Unix(0, int64(in.StartTimeUnixnano)).Sub(time.Unix(0, int64(in.EndTimeUnixnano))),
	}

	for _, link := range in.Links {
		s.References = append(s.References, model.SpanRef{
			TraceID: traceID,
			SpanID:  model.SpanID(binary.BigEndian.Uint64(link.SpanId)),
			RefType: model.SpanRefType_CHILD_OF,
		})
	}

	s.Process = &model.Process{}
	for _, att := range resource.Attributes {
		if att.Key == "process_name" {
			s.Process.ServiceName = att.StringValue
		}
	}

	return s
}

func timestampToTimeProto(ts *timestamp.Timestamp) (t time.Time) {
	if ts == nil {
		return
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
}
