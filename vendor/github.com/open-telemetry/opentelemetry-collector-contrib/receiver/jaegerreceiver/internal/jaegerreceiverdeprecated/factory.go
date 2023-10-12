// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiverdeprecated // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver/internal/jaegerreceiverdeprecated"

// This file implements factory for Jaeger receiver.

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
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
	defaultGRPCBindEndpoint          = "0.0.0.0:14250"
	defaultHTTPBindEndpoint          = "0.0.0.0:14268"
	defaultThriftCompactBindEndpoint = "0.0.0.0:6831"
	defaultThriftBinaryBindEndpoint  = "0.0.0.0:6832"
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
				ServerConfigUDP: DefaultServerConfigUDP(),
			},
			ThriftCompact: &ProtocolUDP{
				Endpoint:        defaultThriftCompactBindEndpoint,
				ServerConfigUDP: DefaultServerConfigUDP(),
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

	// Create the receiver.
	return newJaegerReceiver(set.ID, &config, nextConsumer, set)
}
