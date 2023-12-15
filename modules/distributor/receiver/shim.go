package receiver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log/level"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/services"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver"
	"github.com/opentracing/opentracing-go"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util/log"
)

const (
	logsPerSecond = 10
)

var (
	metricPushDuration = promauto.NewHistogram(prom_client.HistogramOpts{
		Namespace: "tempo",
		Name:      "distributor_push_duration_seconds",
		Help:      "Records the amount of time to push a batch to the ingester.",
		Buckets:   prom_client.DefBuckets,
	})

	statReceiverOtlp       = usagestats.NewInt("receiver_enabled_otlp")
	statReceiverJaeger     = usagestats.NewInt("receiver_enabled_jaeger")
	statReceiverZipkin     = usagestats.NewInt("receiver_enabled_zipkin")
	statReceiverOpencensus = usagestats.NewInt("receiver_enabled_opencensus")
	statReceiverKafka      = usagestats.NewInt("receiver_enabled_kafka")
)

type RetryableError struct {
	err error
	st  *status.Status
}

func (r *RetryableError) GRPCStatus() *status.Status {
	return r.st
}

func (r *RetryableError) Error() string {
	return r.err.Error()
}

// wrapErrorIfRetryable wraps the passed in error to meet expectations of the otel collector exporter code:
// https://github.com/open-telemetry/opentelemetry-collector/blob/d7b49df5d9e922df6ce56ad4b64ee1c79f9dbdbe/exporter/otlpexporter/otlp.go#L172
// The otel collector considers some errors retryable and other not. "ResourceExhausted" is special in that it requires a
// RetryInfo detail along with the error code
func wrapErrorIfRetryable(err error, dur *durationpb.Duration) error {
	if dur == nil {
		return err
	}

	s, ok := status.FromError(err)
	if !ok {
		return err
	}

	if s.Code() != codes.ResourceExhausted {
		return err
	}

	// ignore error. code only errors if Code() == ok
	s, _ = s.WithDetails(&errdetails.RetryInfo{
		RetryDelay: dur,
	})

	return &RetryableError{
		err: err,
		st:  s,
	}
}

type TracesPusher interface {
	PushTraces(ctx context.Context, traces ptrace.Traces) (*tempopb.PushResponse, error)
}

var _ services.Service = (*receiversShim)(nil)

type receiversShim struct {
	services.Service

	retryDelay  *durationpb.Duration
	receivers   []receiver.Traces
	pusher      TracesPusher
	logger      *log.RateLimitedLogger
	metricViews []*view.View
	fatal       chan error
}

func (r *receiversShim) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

var _ confmap.Provider = (*mapProvider)(nil)

// mapProvider is a confmap.Provider that returns a single confmap.Retrieved instance with a fixed map.
type mapProvider struct {
	raw map[string]interface{}
}

func (m *mapProvider) Retrieve(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return confmap.NewRetrieved(m.raw, []confmap.RetrievedOption{}...)
}

func (m *mapProvider) Scheme() string { return "mock" }

func (m *mapProvider) Shutdown(context.Context) error { return nil }

func New(receiverCfg map[string]interface{}, pusher TracesPusher, middleware Middleware, retryAfterDuration time.Duration, logLevel dslog.Level) (services.Service, error) {
	shim := &receiversShim{
		pusher: pusher,
		logger: log.NewRateLimitedLogger(logsPerSecond, level.Error(log.Logger)),
		fatal:  make(chan error),
	}

	if retryAfterDuration > 0 {
		shim.retryDelay = durationpb.New(retryAfterDuration)
	}

	// shim otel observability
	zapLogger := newLogger(logLevel)
	views, err := newMetricViews()
	if err != nil {
		return nil, fmt.Errorf("failed to create metric traceReceiverViews: %w", err)
	}
	shim.metricViews = views

	// load config
	receiverFactories, err := receiver.MakeFactoryMap(
		jaegerreceiver.NewFactory(),
		zipkinreceiver.NewFactory(),
		opencensusreceiver.NewFactory(),
		otlpreceiver.NewFactory(),
		kafkareceiver.NewFactory(),
	)
	if err != nil {
		return nil, err
	}

	for recv := range receiverCfg {
		switch recv {
		case "otlp":
			statReceiverOtlp.Set(1)
		case "jaeger":
			statReceiverJaeger.Set(1)
		case "zipkin":
			statReceiverZipkin.Set(1)
		case "opencensus":
			statReceiverOpencensus.Set(1)
		case "kafka":
			statReceiverKafka.Set(1)
		}
	}

	receivers := make([]string, 0, len(receiverCfg))
	for k := range receiverCfg {
		receivers = append(receivers, k)
	}

	// Creates a config provider with the given config map.
	// The provider will be used to retrieve the actual config for the pipeline (although we only need the receivers).
	pro, err := otelcol.NewConfigProvider(otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{"mock:/"},
			Providers: map[string]confmap.Provider{"mock": &mapProvider{raw: map[string]interface{}{
				"receivers": receiverCfg,
				"exporters": map[string]interface{}{
					"nop": map[string]interface{}{},
				},
				"service": map[string]interface{}{
					"pipelines": map[string]interface{}{
						"traces": map[string]interface{}{
							"exporters": []string{"nop"}, // nop exporter to avoid errors
							"receivers": receivers,
						},
					},
				},
			}}},
		},
	})
	if err != nil {
		return nil, err
	}

	// Creates the configuration for the pipeline.
	// We only need the receivers, the rest of the configuration is not used.
	conf, err := pro.Get(context.Background(), otelcol.Factories{
		Receivers: receiverFactories,
		Exporters: map[component.Type]exporter.Factory{"nop": exportertest.NewNopFactory()}, // nop exporter to avoid errors
	})
	if err != nil {
		return nil, err
	}

	// todo: propagate a real context?  translate our log configuration into zap?
	ctx := context.Background()
	params := receiver.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger:         zapLogger,
			TracerProvider: tracenoop.NewTracerProvider(),
			MeterProvider:  metricnoop.NewMeterProvider(),
			ReportComponentStatus: func(*component.StatusEvent) error {
				return nil
			},
		},
	}

	for componentID, cfg := range conf.Receivers {
		factoryBase := receiverFactories[componentID.Type()]
		if factoryBase == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", componentID.Type())
		}

		// Make sure that the headers are added to context. Required for Authentication.
		switch componentID.Type() {
		case "otlp":
			otlpRecvCfg := cfg.(*otlpreceiver.Config)

			if otlpRecvCfg.HTTP != nil {
				otlpRecvCfg.HTTP.IncludeMetadata = true
				cfg = otlpRecvCfg
			}

		case "zipkin":
			zipkinRecvCfg := cfg.(*zipkinreceiver.Config)

			zipkinRecvCfg.HTTPServerSettings.IncludeMetadata = true
			cfg = zipkinRecvCfg

		case "jaeger":
			jaegerRecvCfg := cfg.(*jaegerreceiver.Config)

			if jaegerRecvCfg.ThriftHTTP != nil {
				jaegerRecvCfg.ThriftHTTP.IncludeMetadata = true
			}

			cfg = jaegerRecvCfg
		}

		receiver, err := factoryBase.CreateTracesReceiver(ctx, params, cfg, middleware.Wrap(shim))
		if err != nil {
			return nil, err
		}

		shim.receivers = append(shim.receivers, receiver)
	}

	shim.Service = services.NewBasicService(shim.starting, shim.running, shim.stopping)

	return shim, nil
}

func (r *receiversShim) running(ctx context.Context) error {
	select {
	case err := <-r.fatal:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *receiversShim) starting(ctx context.Context) error {
	for _, receiver := range r.receivers {
		err := receiver.Start(ctx, r)
		if err != nil {
			return fmt.Errorf("error starting receiver: %w", err)
		}
	}

	return nil
}

// Called after distributor is asked to stop via StopAsync.
func (r *receiversShim) stopping(_ error) error {
	// when shutdown is called on the receiver it immediately shuts down its connection
	// which drops requests on the floor. at this point in the shutdown process
	// the readiness handler is already down so we are not receiving any more requests.
	// sleep for 30 seconds to here to all pending requests to finish.
	time.Sleep(30 * time.Second)

	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	errs := make([]error, 0)

	for _, receiver := range r.receivers {
		err := receiver.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}

	return nil
}

// ConsumeTraces implements consumer.Trace
func (r *receiversShim) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "distributor.ConsumeTraces")
	defer span.Finish()

	var err error

	start := time.Now()
	_, err = r.pusher.PushTraces(ctx, td)
	metricPushDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		r.logger.Log("msg", "pusher failed to consume trace data", "err", err)
		err = wrapErrorIfRetryable(err, r.retryDelay)
	}

	return err
}

// ReportFatalError implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	_ = level.Error(log.Logger).Log("msg", "fatal error reported", "err", err)
	r.fatal <- err
}

// GetFactory implements component.Host
func (r *receiversShim) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

// GetExtensions implements component.Host
func (r *receiversShim) GetExtensions() map[component.ID]extension.Extension { return nil }

func (r *receiversShim) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return nil
}

// observability shims
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
	logger.Info("OTel Shim Logger Initialized")

	return logger
}
