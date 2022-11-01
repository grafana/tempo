// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

// This file implements factory for Jaeger receiver.

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

const (
	typeStr   = "jaeger"
	stability = component.StabilityLevelBeta

	// Protocol values.
	protoGRPC          = "grpc"
	protoThriftHTTP    = "thrift_http"
	protoThriftBinary  = "thrift_binary"
	protoThriftCompact = "thrift_compact"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint                = "0.0.0.0:14250"
	defaultHTTPBindEndpoint                = "0.0.0.0:14268"
	defaultThriftCompactBindEndpoint       = "0.0.0.0:6831"
	defaultThriftBinaryBindEndpoint        = "0.0.0.0:6832"
	defaultAgentRemoteSamplingHTTPEndpoint = "0.0.0.0:5778"
)

// NewFactory creates a new Jaeger receiver factory.
func NewFactory() component.ReceiverFactory {
	return component.NewReceiverFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesReceiver(createTracesReceiver, stability))
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
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
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {

	// Convert settings in the source config to configuration struct
	// that Jaeger receiver understands.
	// Error handling for the conversion is done in the Validate function from the Config object itself.

	rCfg := cfg.(*Config)
	remoteSamplingConfig := rCfg.RemoteSampling

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

	if remoteSamplingConfig != nil {
		logSamplingDeprecation(set.Logger)

		config.RemoteSamplingClientSettings = remoteSamplingConfig.GRPCClientSettings
		if config.RemoteSamplingClientSettings.Endpoint == "" {
			config.RemoteSamplingClientSettings.Endpoint = defaultGRPCBindEndpoint
		}

		config.AgentHTTPEndpoint = remoteSamplingConfig.HostEndpoint
		if config.AgentHTTPEndpoint == "" {
			config.AgentHTTPEndpoint = defaultAgentRemoteSamplingHTTPEndpoint
		}

		// strategies are served over grpc so if grpc is not enabled and strategies are present return an error
		if len(remoteSamplingConfig.StrategyFile) != 0 {
			config.RemoteSamplingStrategyFile = remoteSamplingConfig.StrategyFile
			config.RemoteSamplingStrategyFileReloadInterval = remoteSamplingConfig.StrategyFileReloadInterval
		}
	}

	// Create the receiver.
	return newJaegerReceiver(rCfg.ID(), &config, nextConsumer, set), nil
}

var once sync.Once

func logSamplingDeprecation(logger *zap.Logger) {
	once.Do(func() {
		logger.Warn(
			"Jaeger remote sampling support is deprecated and will be removed in release v0.61.0. Use the jaegerremotesampling extension instead.",
		)
	})
}
