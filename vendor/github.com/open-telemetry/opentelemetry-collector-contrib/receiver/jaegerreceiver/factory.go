// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

// This file implements factory for Jaeger receiver.

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver/internal/metadata"
)

const (
	// Protocol values.
	protoGRPC          = "grpc"
	protoThriftHTTP    = "thrift_http"
	protoThriftBinary  = "thrift_binary"
	protoThriftCompact = "thrift_compact"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint          = "0.0.0.0:14250"
	defaultHTTPBindEndpoint          = "0.0.0.0:14268"
	defaultThriftCompactBindEndpoint = "0.0.0.0:6831"
	defaultThriftBinaryBindEndpoint  = "0.0.0.0:6832"
)

var disableJaegerReceiverRemoteSampling = featuregate.GlobalRegistry().MustRegister(
	"receiver.jaeger.DisableRemoteSampling",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("When enabled, the Jaeger Receiver will fail to start when it is configured with remote_sampling config. When disabled, the receiver will start and the remote_sampling config will be no-op."),
)

var once sync.Once

func logDeprecation(logger *zap.Logger) {
	once.Do(func() {
		logger.Warn("jaeger receiver will deprecate Thrift-gen and replace it with Proto-gen to be compatbible to jaeger 1.42.0 and higher. See https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/18485 for more details.")

	})
}

// nolint
var protoGate = featuregate.GlobalRegistry().MustRegister(
	"receiver.jaegerreceiver.replaceThriftWithProto",
	featuregate.StageStable,
	featuregate.WithRegisterDescription(
		"When enabled, the jaegerreceiver will use Proto-gen over Thrift-gen.",
	),
	featuregate.WithRegisterToVersion("0.92.0"),
)

// NewFactory creates a new Jaeger receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability))
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  defaultGRPCBindEndpoint,
					Transport: "tcp",
				},
			},
			ThriftHTTP: &confighttp.HTTPServerSettings{
				Endpoint: defaultHTTPBindEndpoint,
			},
			ThriftBinary: &ProtocolUDP{
				Endpoint:        defaultThriftBinaryBindEndpoint,
				ServerConfigUDP: defaultServerConfigUDP(),
			},
			ThriftCompact: &ProtocolUDP{
				Endpoint:        defaultThriftCompactBindEndpoint,
				ServerConfigUDP: defaultServerConfigUDP(),
			},
		},
	}
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	logDeprecation(set.Logger)

	// Convert settings in the source config to configuration struct
	// that Jaeger receiver understands.
	// Error handling for the conversion is done in the Validate function from the Config object itself.

	rCfg := cfg.(*Config)

	var config configuration
	// Set ports
	if rCfg.Protocols.GRPC != nil {
		config.CollectorGRPCServerSettings = *rCfg.Protocols.GRPC
	}

	if rCfg.Protocols.ThriftHTTP != nil {
		config.CollectorHTTPSettings = *rCfg.ThriftHTTP
	}

	if rCfg.Protocols.ThriftBinary != nil {
		config.AgentBinaryThrift = *rCfg.ThriftBinary
	}

	if rCfg.Protocols.ThriftCompact != nil {
		config.AgentCompactThrift = *rCfg.ThriftCompact
	}

	if rCfg.RemoteSampling != nil {
		set.Logger.Warn("You are using a deprecated no-op `remote_sampling` option which will be removed soon; use a `jaegerremotesampling` extension instead")
	}

	// Create the receiver.
	return newJaegerReceiver(set.ID, &config, nextConsumer, set)
}
