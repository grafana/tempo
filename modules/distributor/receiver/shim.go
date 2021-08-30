package receiver

import (
	"context"
	"fmt"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/dskit/services"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/kafkareceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const (
	logsPerSecond = 10
)

type receiversShim struct {
	services.Service

	multitenancyEnabled bool
	receivers           []component.Receiver
	pusher              tempopb.PusherServer
	logger              *tempo_util.RateLimitedLogger
	metricViews         []*view.View
}

func New(receiverCfg map[string]interface{}, pusher tempopb.PusherServer, multitenancyEnabled bool, logLevel logging.Level) (services.Service, error) {
	shim := &receiversShim{
		multitenancyEnabled: multitenancyEnabled,
		pusher:              pusher,
		logger:              tempo_util.NewRateLimitedLogger(logsPerSecond, level.Error(log.Logger)),
	}

	v := viper.New()
	err := v.MergeConfigMap(map[string]interface{}{
		"receivers": receiverCfg,
	})
	if err != nil {
		return nil, err
	}

	// shim otel observability
	zapLogger := newLogger(logLevel)
	shim.metricViews, err = newMetricViews()
	if err != nil {
		return nil, fmt.Errorf("failed to create metric views: %w", err)
	}

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

	cfgs, err := config.Load(v, component.Factories{
		Receivers: receiverFactories,
	})
	if err != nil {
		return nil, err
	}

	// todo: propagate a real context?  translate our log configuration into zap?
	ctx := context.Background()
	params := component.ReceiverCreateParams{Logger: zapLogger}

	for _, cfg := range cfgs.Receivers {
		factoryBase := receiverFactories[cfg.Type()]
		if factoryBase == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", cfg.Type())
		}

		receiver, err := factoryBase.CreateTracesReceiver(ctx, params, cfg, shim)
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
			return fmt.Errorf("Error starting receiver %w", err)
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
		return componenterror.CombineErrors(errs)
	}

	view.Unregister(r.metricViews...)
	return nil
}

// implements consumer.TraceConsumer
func (r *receiversShim) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if !r.multitenancyEnabled {
		ctx = user.InjectOrgID(ctx, tempo_util.FakeTenantID)
	} else {
		var err error
		_, ctx, err = user.ExtractFromGRPCRequest(ctx)
		if err != nil {
			r.logger.Log("msg", "failed to extract org id", "err", err)
			return err
		}
	}

	var err error

	// Convert to bytes and back. This is unfortunate for efficiency but it works
	// around the otel-collector internalization of otel-proto which Tempo also uses.
	convert, err := td.ToOtlpProtoBytes()
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

	for _, batch := range trace.Batches {
		_, err = r.pusher.Push(ctx, &tempopb.PushRequest{
			Batch: batch,
		})
		if err != nil {
			r.logger.Log("msg", "pusher failed to consume trace data", "err", err)
			break
		}
	}

	return err
}

// implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	level.Error(log.Logger).Log("msg", "fatal error reported", "err", err)
}

// implements component.Host
func (r *receiversShim) GetFactory(kind component.Kind, componentType configmodels.Type) component.Factory {
	return nil
}

// implements component.Host
func (r *receiversShim) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

// implements component.Host
func (r *receiversShim) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
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
	views := obsreport.Configure(configtelemetry.LevelNormal)
	err := view.Register(views...)
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

	return views, nil
}
