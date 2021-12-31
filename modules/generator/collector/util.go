package collector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	noopExporter = "noop"
	noopReceiver = "noop"
)

func newNoopExporterFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		noopExporter,
		func() config.Exporter {
			exporterSettings := config.NewExporterSettings(config.NewComponentIDWithName(noopExporter, noopExporter))
			return &exporterSettings
		},
		exporterhelper.WithTraces(func(
			context.Context,
			component.ExporterCreateSettings,
			config.Exporter) (
			component.TracesExporter,
			error) {
			return &noopExp{}, nil
		}),
	)
}

type noopExp struct{}

func (n noopExp) Start(context.Context, component.Host) error { return nil }

func (n noopExp) Shutdown(context.Context) error { return nil }

func (n noopExp) Capabilities() consumer.Capabilities { return consumer.Capabilities{} }

func (n noopExp) ConsumeTraces(context.Context, pdata.Traces) error { return nil }

func newNoopReceiverFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		noopReceiver,
		func() config.Receiver {
			receiverSettings := config.NewReceiverSettings(config.NewComponentIDWithName(noopReceiver, noopReceiver))
			return &receiverSettings
		},
		receiverhelper.WithMetrics(func(
			context.Context,
			component.ReceiverCreateSettings,
			config.Receiver,
			consumer.Metrics) (
			component.MetricsReceiver,
			error) {
			return &noopRecv{}, nil
		}),
	)
}

type noopRecv struct{}

func (n noopRecv) Start(context.Context, component.Host) error { return nil }

func (n noopRecv) Shutdown(context.Context) error { return nil }
