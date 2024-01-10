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

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/metadata"
)

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
		GRPCServerSettings: configgrpc.GRPCServerSettings{
			NetAddr: confignet.NetAddr{
				Endpoint:  "0.0.0.0:55678",
				Transport: "tcp",
			},
			// We almost write 0 bytes, so no need to tune WriteBufferSize.
			ReadBufferSize: 512 * 1024,
		},
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	var err error
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		var recv *ocReceiver
		recv, err = newOpenCensusReceiver(rCfg.NetAddr.Transport, rCfg.NetAddr.Endpoint, nil, nil, set, rCfg.buildOptions()...)
		return recv
	})
	if err != nil {
		return nil, err
	}
	r.Unwrap().(*ocReceiver).traceConsumer = nextConsumer

	return r, nil
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	var err error
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		var recv *ocReceiver
		recv, err = newOpenCensusReceiver(rCfg.NetAddr.Transport, rCfg.NetAddr.Endpoint, nil, nil, set, rCfg.buildOptions()...)
		return recv
	})
	if err != nil {
		return nil, err
	}
	r.Unwrap().(*ocReceiver).metricsConsumer = nextConsumer

	return r, nil
}

// This is the map of already created OpenCensus receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one ocReceiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
