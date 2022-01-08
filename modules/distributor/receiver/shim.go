package receiver

import (
	"context"
	"fmt"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver"
	"github.com/opentracing/opentracing-go"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/weaveworks/common/logging"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/external/obsreportconfig"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
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
)

type BatchPusher interface {
	PushBatches(ctx context.Context, batches []*v1.ResourceSpans) (*tempopb.PushResponse, error)
}

type receiversShim struct {
	services.Service

	receivers   []component.Receiver
	pusher      BatchPusher
	logger      *tempo_util.RateLimitedLogger
	metricViews []*view.View
}

func (r *receiversShim) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func New(receiverCfg map[string]interface{}, pusher BatchPusher, middleware Middleware, logLevel logging.Level) (services.Service, error) {
	shim := &receiversShim{
		pusher: pusher,
		logger: tempo_util.NewRateLimitedLogger(logsPerSecond, level.Error(log.Logger)),
	}

	// shim otel observability
	zapLogger := newLogger(logLevel)
	views, err := newMetricViews()
	if err != nil {
		return nil, fmt.Errorf("failed to create metric views: %w", err)
	}
	shim.metricViews = views

	// load config
	receiverFactories, err := component.MakeReceiverFactoryMap(
		jaegerreceiver.NewFactory(),
		zipkinreceiver.NewFactory(),
		opencensusreceiver.NewFactory(),
		otlpreceiver.NewFactory(),
		kafkareceiver.NewFactory(),
	)
	if err != nil {
		return nil, err
	}

	p := config.NewMapFromStringMap(map[string]interface{}{
		"receivers": receiverCfg,
	})
	cfgs, err := configunmarshaler.NewDefault().Unmarshal(p, component.Factories{
		Receivers: receiverFactories,
	})
	if err != nil {
		return nil, err
	}

	// todo: propagate a real context?  translate our log configuration into zap?
	ctx := context.Background()
	params := component.ReceiverCreateSettings{TelemetrySettings: component.TelemetrySettings{
		Logger:         zapLogger,
		TracerProvider: trace.NewNoopTracerProvider(),
		MeterProvider:  metric.NewNoopMeterProvider(),
	}}

	for componentID, cfg := range cfgs.Receivers {
		factoryBase := receiverFactories[componentID.Type()]
		if factoryBase == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", componentID.Type())
		}

		receiver, err := factoryBase.CreateTracesReceiver(ctx, params, cfg, middleware.Wrap(shim))
		if err != nil {
			return nil, err
		}

		shim.receivers = append(shim.receivers, receiver)
	}

	shim.Service = services.NewIdleService(shim.starting, shim.stopping)

	return shim, nil
}
func (r *receiversShim) starting(ctx context.Context) error {
	for _, receiver := range r.receivers {
		err := receiver.Start(ctx, r)
		if err != nil {
			return fmt.Errorf("error starting receiver %w", err)
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
func (r *receiversShim) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "distributor.ConsumeTraces")
	defer span.Finish()

	var err error

	// Convert to bytes and back. This is unfortunate for efficiency but it works
	// around the otel-collector internalization of otel-proto which Tempo also uses.
	convert, err := otlp.NewProtobufTracesMarshaler().MarshalTraces(td)
	if err != nil {
		return err
	}

	// tempopb.Trace is wire-compatible with ExportTraceServiceRequest
	// used by ToOtlpProtoBytes
	trace := tempopb.Trace{}
	err = trace.Unmarshal(convert)
	if err != nil {
		return err
	}

	start := time.Now()
	_, err = r.pusher.PushBatches(ctx, trace.Batches)
	metricPushDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		r.logger.Log("msg", "pusher failed to consume trace data", "err", err)
	}

	return err
}

// implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	level.Error(log.Logger).Log("msg", "fatal error reported", "err", err)
}

// implements component.Host
func (r *receiversShim) GetFactory(kind component.Kind, componentType config.Type) component.Factory {
	return nil
}

// implements component.Host
func (r *receiversShim) GetExtensions() map[config.ComponentID]component.Extension {
	return nil
}

// implements component.Host
func (r *receiversShim) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return nil
}

// observability shims
func newLogger(level logging.Level) *zap.Logger {
	zapLevel := zapcore.InfoLevel

	switch level.Logrus {
	case logrus.PanicLevel:
		zapLevel = zapcore.PanicLevel
	case logrus.FatalLevel:
		zapLevel = zapcore.FatalLevel
	case logrus.ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case logrus.WarnLevel:
		zapLevel = zapcore.WarnLevel
	case logrus.InfoLevel:
		zapLevel = zapcore.InfoLevel
	case logrus.DebugLevel:
	case logrus.TraceLevel:
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

func newMetricViews() ([]*view.View, error) {
	views := obsreportconfig.Configure(configtelemetry.LevelNormal)
	err := view.Register(views.Views...)
	if err != nil {
		return nil, fmt.Errorf("failed to register views: %w", err)
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace:  "tempo",
		Registerer: prom_client.DefaultRegisterer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	view.RegisterExporter(pe)

	return views.Views, nil
}
