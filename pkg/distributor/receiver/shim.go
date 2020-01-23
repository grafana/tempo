package receiver

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"	

	"github.com/joe-elliott/frigg/pkg/friggpb"

	opentelemetry_proto_collector_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/traces/v1"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	opentelemetry_proto_resource_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	opentelemetry_proto_common_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver"	
)

type Receivers interface {
	Start() error
	Shutdown() error
}

type receiversShim struct {
	receivers []receiver.TraceReceiver
	pusher friggpb.PusherServer
}

func New(receiverCfg map[string]interface{}, pusher friggpb.PusherServer) (Receivers, error) {
	shim := &receiversShim{}

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

	configs, err := loadReceivers(v, factories)
	if err != nil {
		return nil, err
	}

	for _, config := configs {
		factory := factories[config.Type()]
		if factory == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", config.Type())
		}

		// todo: propagate a real context?  translate our log configuration into zap?
		receiver, err := factory.CreateTraceReceiver(context.Background(), shim, zap.NewProduction())
		if err != nil {
			return err
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
func (r *receiversShim) Shutdown() {
	for _, rcv := range r.receivers {
		err := rcv.Stop()
		if err != nil {
			// log, but keep on shutting down
			level.Error().Log("msg", "failed to stop receiver", "err", err)
		}
	}

	return nil
}

// implements consumer.TraceConsumer
func (r *receiversShim) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	// todo: eventually otel collector intends to start using otel proto internally instead of opencensus
	//  when that happens we need to update our depedency and we can remove all of this translation logic
	// also note: this translation logic is woefully incomplete and is meant as a stopgap while we wait for the otel collector

	batch := &opentelemetry_proto_collector_trace_v1.ResourceSpans{
		Spans : make([]*opentelemetry_proto_trace_v1.Span, 0, len(td.Spans)),
		Resource: &opentelemetry_proto_resource_v1.Resource{
			Attributes: make([]&opentelemetry_proto_common_v1.AttributeKeyValue, 0, len(td.Node.Attributes) + 2)
		},
	}

	for _, fromSpan := range td.Spans {
		toSpan := &opentelemetry_proto_trace_v1.Span{
			TraceId: fromSpan.TraceId,
			SpanId: fromSpan.SpanId,
			TraceState: fromSpan.TraceState,
			ParentSpanId: fromSpan.ParentSpanId,
			Name: fromSpan.Name,
			// Kind: fromSpan.Kind,
			StartTimeUnixnano: fromSpan.StartTime.GetNanos(),
			EndTimeUnixnano: fromSpan.EndTime.GetNanos(),
			Attributes: make([]&opentelemetry_proto_common_v1.AttributeKeyValue, 0, len(fromSpan.Attributes)),
		}

		for _, att := range fromSpan.Attributes {
			toSpan.Attributes = append(toSpan.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
				Key: att.Key,
				Type: att.Type,
				StringValue: att.StringValue,
				IntValue: att.IntValue,
				DoubleValue: att.DoubleValue,
				BoolValue: att.BoolValue,
			})
		}

		batch.Spans = append(batch.Spans, toSpan)
	}

	batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
		Key: "process_name",
		Type: opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
		StringValue: td.Node.ServiceInfo.Name,
	})
	batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
		Key: "host_name",
		Type: opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
		StringValue: td.Node.Identifier.HostName,
	})
	for _, att := range td.Node.Attributes {
		batch.Resource.Attributes = append(batch.Resource.Attributes, &opentelemetry_proto_common_v1.AttributeKeyValue{
			Key: att.Key,
			Type: att.Type,
			StringValue: att.StringValue,
			IntValue: att.IntValue,
			DoubleValue: att.DoubleValue,
			BoolValue: att.BoolValue,
		})
	}

	r.pusher.Push(ctx, &friggpb.PushRequest{
		Batch: batch,
	})
}

// implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	level.Error().Log("msg", "fatal error reported", "err", err)
	panic(fmt.Sprintf("Fatal error %v", err))
}

// implements component.Host
func (r *receiversShim) Context() context.Context {
	// todo: something better here?
	return context.Background()
}

// extracted from otel collector : https://github.com/open-telemetry/opentelemetry-collector/blob/master/config/config.go
func loadReceivers(v *viper.Viper, factories map[string]receiver.Factory) (configmodels.Receivers, error) {
	// Get the list of all "receivers" sub vipers from config source.
	subViper := v.Sub(receiversKeyName)

	// Get the map of "receivers" sub-keys.
	keyMap := v.GetStringMap(receiversKeyName)

	// Currently there is no default receiver enabled. The configuration must specify at least one receiver to enable
	// functionality.
	if len(keyMap) == 0 {
		return nil, &configError{
			code: errMissingReceivers,
			msg:  "no receivers specified in config",
		}
	}

	// Prepare resulting map
	receivers := make(configmodels.Receivers)

	// Iterate over input map and create a config for each.
	for key := range keyMap {
		// Decode the key into type and fullName components.
		typeStr, fullName, err := decodeTypeAndName(key)
		if err != nil || typeStr == "" {
			return nil, &configError{
				code: errInvalidTypeAndNameKey,
				msg:  fmt.Sprintf("invalid key %q: %s", key, err.Error()),
			}
		}

		// Find receiver factory based on "type" that we read from config source
		factory := factories[typeStr]
		if factory == nil {
			return nil, &configError{
				code: errUnknownReceiverType,
				msg:  fmt.Sprintf("unknown receiver type %q", typeStr),
			}
		}

		// Create the default config for this receiver.
		receiverCfg := factory.CreateDefaultConfig()
		receiverCfg.SetType(typeStr)
		receiverCfg.SetName(fullName)

		// Unmarshal only the subconfig for this exporter.
		sv := getConfigSection(subViper, key)

		// Now that the default config struct is created we can Unmarshal into it
		// and it will apply user-defined config on top of the default.
		customUnmarshaler := factory.CustomUnmarshaler()
		if customUnmarshaler != nil {
			// This configuration requires a custom unmarshaler, use it.
			err = customUnmarshaler(subViper, key, sv, receiverCfg)
		} else {
			err = sv.UnmarshalExact(receiverCfg)
		}

		if err != nil {
			return nil, &configError{
				code: errUnmarshalErrorOnReceiver,
				msg:  fmt.Sprintf("error reading settings for receiver type %q: %v", typeStr, err),
			}
		}

		if receivers[fullName] != nil {
			return nil, &configError{
				code: errDuplicateReceiverName,
				msg:  fmt.Sprintf("duplicate receiver name %q", fullName),
			}
		}
		receivers[fullName] = receiverCfg
	}

	return receivers, nil
}
