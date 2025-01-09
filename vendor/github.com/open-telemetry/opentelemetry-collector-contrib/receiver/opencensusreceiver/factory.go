// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensusreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/metadata"
)

const grpcPort = 55678

// NewFactory creates a new OpenCensus receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		ServerConfig: configgrpc.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  testutil.EndpointForPort(grpcPort),
				Transport: confignet.TransportTypeTCP,
			},
			// We almost write 0 bytes, so no need to tune WriteBufferSize.
			ReadBufferSize: 512 * 1024,
		},
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		return newOpenCensusReceiver(rCfg, nil, nil, set, rCfg.buildOptions()...)
	})
	r.Unwrap().(*ocReceiver).traceConsumer = nextConsumer

	return r, nil
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		return newOpenCensusReceiver(rCfg, nil, nil, set, rCfg.buildOptions()...)
	})
	r.Unwrap().(*ocReceiver).metricsConsumer = nextConsumer

	return r, nil
}

// This is the map of already created OpenCensus receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one ocReceiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
