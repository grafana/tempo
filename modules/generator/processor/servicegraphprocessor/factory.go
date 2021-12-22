package servicegraphprocessor

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// TypeStr is the unique identifier for the Prometheus service graph exporter.
	TypeStr = "service_graphs"

	// DefaultWait is the default value to wait for an edge to be completed
	DefaultWait = time.Second * 10
	// DefaultMaxItems is the default amount of edges that will be stored in the storeMap
	DefaultMaxItems = 10_000
	// DefaultWorkers is the default amount of workers that will be used to process the edges
	DefaultWorkers = 10
)

// Config holds the configuration for the Prometheus service graph processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`

	Wait     time.Duration `mapstructure:"wait"`
	MaxItems int           `mapstructure:"max_items"`

	Workers int `mapstructure:"workers"`

	SuccessCodes *successCodes `mapstructure:"success_codes"`
}

type successCodes struct {
	http []int64 `mapstructure:"http"`
	grpc []int64 `mapstructure:"grpc"`
}

// NewFactory returns a new factory for the Prometheus service graph processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		TypeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTracesProcessor),
	)
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewComponentIDWithName(TypeStr, TypeStr)),
	}
}

func createTracesProcessor(
	_ context.Context,
	_ component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces,
) (component.TracesProcessor, error) {
	eCfg := cfg.(*Config)

	return NewProcessor(nextConsumer, eCfg, nil), nil
}
