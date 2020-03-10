package receiver

import (
	"context"
	"fmt"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
	opentelemetry_proto_common_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	opentelemetry_proto_resource_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/spf13/viper"
	"github.com/weaveworks/common/user"
	"go.uber.org/zap"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

type Receivers interface {
	Start() error
	Shutdown() error
}

type receiversShim struct {
	authEnabled bool
	receivers   []receiver.TraceReceiver
	pusher      tempopb.PusherServer
}

func New(receiverCfg map[string]interface{}, pusher tempopb.PusherServer, authEnabled bool) (Receivers, error) {
	shim := &receiversShim{
		authEnabled: authEnabled,
		pusher:      pusher,
	}

	v := viper.New()
	err := v.MergeConfigMap(receiverCfg)
	if err != nil {
		return nil, err
	}

	// get factories somehow?
	factories, err := receiver.Build(
		&jaegerreceiver.Factory{},
		&zipkinreceiver.Factory{},
		&opencensusreceiver.Factory{},
	)
	if err != nil {
		return nil, err
	}

	configs, err := loadReceivers(v, receiverCfg, factories)
	if err != nil {
		return nil, err
	}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		factory := factories[config.Type()]
		if factory == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", config.Type())
		}

		// todo: propagate a real context?  translate our log configuration into zap?
		receiver, err := factory.CreateTraceReceiver(context.Background(), zapLogger, config, shim)
		if err != nil {
			return nil, err
		}

		shim.receivers = append(shim.receivers, receiver)
	}

	return shim, nil
}

// implements Receivers
func (r *receiversShim) Start() error {
	for _, rcv := range r.receivers {
		err := rcv.Start(r)
		if err != nil {
			return err
		}
	}

	return nil
}

// implements Receivers
func (r *receiversShim) Shutdown() error {
	for _, rcv := range r.receivers {
		err := rcv.Shutdown()
		if err != nil {
			// log, but keep on shutting down
			level.Error(util.Logger).Log("msg", "failed to stop receiver", "err", err)
		}
	}

	return nil
}

// implements consumer.TraceConsumer
func (r *receiversShim) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	if !r.authEnabled {
		ctx = user.InjectOrgID(ctx, tempo_util.FakeTenantID)
	}

	// todo: eventually otel collector intends to start using otel proto internally instead of opencensus
	//  when that happens we need to update our dependency and we can remove all of this translation logic
	// also note: this translation logic is woefully incomplete and is meant as a stopgap while we wait for the otel collector
	_, err := r.pusher.Push(ctx, &tempopb.PushRequest{
		Batch: convertTraceData(td),
	})

	// todo:  confirm/deny if this error propagates back to the caller
	return err
}

// implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	level.Error(util.Logger).Log("msg", "fatal error reported", "err", err)
	panic(fmt.Sprintf("Fatal error %v", err))
}

// implements component.Host
func (r *receiversShim) Context() context.Context {
	// todo: something better here?
	return context.Background()
}

func convertTraceData(td consumerdata.TraceData) *opentelemetry_proto_trace_v1.ResourceSpans {
	batch := &opentelemetry_proto_trace_v1.ResourceSpans{
		Spans: make([]*opentelemetry_proto_trace_v1.Span, 0, len(td.Spans)),
		Resource: &opentelemetry_proto_resource_v1.Resource{
			Attributes: make([]*opentelemetry_proto_common_v1.AttributeKeyValue, 0, len(td.Node.Attributes)+2),
		},
	}

	for _, fromSpan := range td.Spans {
		toSpan := &opentelemetry_proto_trace_v1.Span{
			TraceId:      fromSpan.TraceId,
			SpanId:       fromSpan.SpanId,
			Tracestate:   fromSpan.Tracestate.String(),
			ParentSpanId: fromSpan.ParentSpanId,
			Name:         fromSpan.Name.String(),
			// Kind: fromSpan.Kind,
			StartTimeUnixnano: uint64(fromSpan.StartTime.GetNanos()),
			EndTimeUnixnano:   uint64(fromSpan.EndTime.GetNanos()),
		}

		if fromSpan.Attributes != nil {
			toSpan.Attributes = make([]*opentelemetry_proto_common_v1.AttributeKeyValue, 0, len(fromSpan.Attributes.AttributeMap))

			for key, att := range fromSpan.Attributes.AttributeMap {
				toAtt := &opentelemetry_proto_common_v1.AttributeKeyValue{
					Key: key,
				}

				switch att.Value.(type) {
				case *tracepb.AttributeValue_StringValue:
					toAtt.Type = opentelemetry_proto_common_v1.AttributeKeyValue_STRING
					toAtt.StringValue = att.GetStringValue().String()
				case *tracepb.AttributeValue_IntValue:
					toAtt.Type = opentelemetry_proto_common_v1.AttributeKeyValue_INT
					toAtt.IntValue = att.GetIntValue()
				case *tracepb.AttributeValue_BoolValue:
					toAtt.Type = opentelemetry_proto_common_v1.AttributeKeyValue_BOOL
					toAtt.BoolValue = att.GetBoolValue()
				case *tracepb.AttributeValue_DoubleValue:
					toAtt.Type = opentelemetry_proto_common_v1.AttributeKeyValue_BOOL
					toAtt.DoubleValue = att.GetDoubleValue()
				}

				toSpan.Attributes = append(toSpan.Attributes, toAtt)
			}
		}

		batch.Spans = append(batch.Spans, toSpan)
	}

	batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
		Key:         "process_name",
		Type:        opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
		StringValue: td.Node.ServiceInfo.Name,
	})
	batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
		Key:         "host_name",
		Type:        opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
		StringValue: td.Node.Identifier.HostName,
	})
	for key, att := range td.Node.Attributes {
		batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
			Key:         key,
			Type:        opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
			StringValue: att,
		})
	}

	return batch
}
