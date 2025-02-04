// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

// This file implements factory for Jaeger receiver.

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver/internal/metadata"
)

const (
	// Protocol values.
	protoGRPC          = "grpc"
	protoThriftHTTP    = "thrift_http"
	protoThriftBinary  = "thrift_binary"
	protoThriftCompact = "thrift_compact"

	// Default endpoints to bind to.
	defaultGRPCEndpoint          = "localhost:14250"
	defaultHTTPEndpoint          = "localhost:14268"
	defaultThriftCompactEndpoint = "localhost:6831"
	defaultThriftBinaryEndpoint  = "localhost:6832"
)

var disableJaegerReceiverRemoteSampling = featuregate.GlobalRegistry().MustRegister(
	"receiver.jaeger.DisableRemoteSampling",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("When enabled, the Jaeger Receiver will fail to start when it is configured with remote_sampling config. When disabled, the receiver will start and the remote_sampling config will be no-op."),
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
			GRPC: &configgrpc.ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  defaultGRPCEndpoint,
					Transport: confignet.TransportTypeTCP,
				},
			},
			ThriftHTTP: &confighttp.ServerConfig{
				Endpoint: defaultHTTPEndpoint,
			},
			ThriftBinaryUDP: &ProtocolUDP{
				Endpoint:        defaultThriftBinaryEndpoint,
				ServerConfigUDP: defaultServerConfigUDP(),
			},
			ThriftCompactUDP: &ProtocolUDP{
				Endpoint:        defaultThriftCompactEndpoint,
				ServerConfigUDP: defaultServerConfigUDP(),
			},
		},
	}
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	// Convert settings in the source config to configuration struct
	// that Jaeger receiver understands.
	// Error handling for the conversion is done in the Validate function from the Config object itself.

	rCfg := cfg.(*Config)

	if rCfg.RemoteSampling != nil {
		set.Logger.Warn("You are using a deprecated no-op `remote_sampling` option which will be removed soon; use a `jaegerremotesampling` extension instead")
	}

	// Create the receiver.
	return newJaegerReceiver(set.ID, rCfg.Protocols, nextConsumer, set)
}
