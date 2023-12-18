package forwarder

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	dslog "github.com/grafana/dskit/log"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grafana/tempo/modules/distributor/forwarder/otlpgrpc"
)

type Forwarder interface {
	ForwardTraces(ctx context.Context, traces ptrace.Traces) error
	Shutdown(ctx context.Context) error
}

type List []Forwarder

func (l List) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	var errs []error

	for _, forwarder := range l {
		if err := forwarder.ForwardTraces(ctx, traces); err != nil {
			errs = append(errs, err)
		}
	}

	return multierr.Combine(errs...)
}

func New(cfg Config, logger log.Logger, logLevel dslog.Level) (Forwarder, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	var forwarder Forwarder
	switch cfg.Backend {
	case OTLPGRPCBackend:
		f, err := otlpgrpc.NewForwarder(cfg.OTLPGRPC, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create new otlpgrpc forwarder: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := f.Dial(ctx); err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}

		forwarder = f
	default:
		return nil, fmt.Errorf("%s backend is not supported", cfg.Backend)
	}

	if len(cfg.Filter.Traces.SpanConditions) > 0 || len(cfg.Filter.Traces.SpanEventConditions) > 0 {
		return NewFilterForwarder(cfg.Filter, forwarder, logLevel)
	}

	return forwarder, nil
}

type FilterForwarder struct {
	filterProcessor processor.Traces
	next            Forwarder
	fatalError      error
	fatalErrorMu    sync.RWMutex
}

func NewFilterForwarder(cfg FilterConfig, next Forwarder, logLevel dslog.Level) (*FilterForwarder, error) {
	factory := filterprocessor.NewFactory()

	set := processor.CreateSettings{
		ID: component.ID{},
		TelemetrySettings: component.TelemetrySettings{
			Logger:         newLogger(logLevel),
			TracerProvider: tracenoop.NewTracerProvider(),
			MeterProvider:  metricnoop.NewMeterProvider(),
		},
		BuildInfo: component.BuildInfo{},
	}
	fpCfg := &filterprocessor.Config{
		ErrorMode: ottl.IgnoreError,
		Traces: filterprocessor.TraceFilters{
			SpanConditions:      cfg.Traces.SpanConditions,
			SpanEventConditions: cfg.Traces.SpanEventConditions,
		},
	}
	fp, err := factory.CreateTracesProcessor(context.Background(), set, fpCfg, consumerToForwarderAdapter{forwarder: next})
	if err != nil {
		return nil, fmt.Errorf("failed to create filter processor: %w", err)
	}

	f := &FilterForwarder{
		filterProcessor: fp,
		next:            next,
		fatalError:      nil,
		fatalErrorMu:    sync.RWMutex{},
	}

	if err := f.filterProcessor.Start(context.TODO(), f); err != nil {
		return nil, fmt.Errorf("failed to start filter processor: %w", err)
	}

	return f, nil
}

func (f *FilterForwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	f.fatalErrorMu.RLock()
	fatalErr := f.fatalError
	f.fatalErrorMu.RUnlock()

	if fatalErr != nil {
		return fmt.Errorf("fatal error occurred in filter forwarder: %w", fatalErr)
	}

	// Copying the traces to avoid mutating the original.
	tracesCopy := ptrace.NewTraces()
	traces.CopyTo(tracesCopy)

	err := f.filterProcessor.ConsumeTraces(ctx, tracesCopy)
	if err != nil {
		return fmt.Errorf("failed to filter traces: %w", err)
	}

	return nil
}

func (f *FilterForwarder) Shutdown(ctx context.Context) error {
	var errs []error

	if err := f.filterProcessor.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown filter processor: %w", err))
	}

	if err := f.next.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown next forwarder: %w", err))
	}

	return multierr.Combine(errs...)
}

// ReportFatalError implements component.Host
func (f *FilterForwarder) ReportFatalError(err error) {
	f.fatalErrorMu.Lock()
	f.fatalError = err
	f.fatalErrorMu.Unlock()
}

// GetFactory implements component.Host
func (f *FilterForwarder) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

// GetExtensions implements component.Host
func (f *FilterForwarder) GetExtensions() map[component.ID]extension.Extension {
	return nil
}

// GetExporters implements component.Host
func (f *FilterForwarder) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return nil
}

type consumerToForwarderAdapter struct {
	forwarder Forwarder
}

func (c consumerToForwarderAdapter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return c.forwarder.ForwardTraces(ctx, td)
}

func (c consumerToForwarderAdapter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newLogger(level dslog.Level) *zap.Logger {
	zapLevel := zapcore.InfoLevel

	switch level.String() {
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "debug":
		zapLevel = zapcore.DebugLevel
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339))
	}
	logger := zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapLevel,
	))
	logger = logger.With(zap.String("component", "tempo"))
	logger.Info("filter forwarder logger initialized")

	return logger
}
