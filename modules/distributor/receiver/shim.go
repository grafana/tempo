package receiver

import (
	"context"
	"fmt"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/converter"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
	"go.uber.org/zap"

	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const (
	logsPerSecond = 10
)

type receiversShim struct {
	services.Service

	authEnabled bool
	receivers   []component.Receiver
	pusher      tempopb.PusherServer
	logger      *tempo_util.RateLimitedLogger
}

func New(receiverCfg map[string]interface{}, pusher tempopb.PusherServer, authEnabled bool) (services.Service, error) {
	shim := &receiversShim{
		authEnabled: authEnabled,
		pusher:      pusher,
		logger:      tempo_util.NewRateLimitedLogger(logsPerSecond, level.Error(util.Logger)),
	}

	v := viper.New()
	err := v.MergeConfigMap(map[string]interface{}{
		"receivers": receiverCfg,
	})
	if err != nil {
		return nil, err
	}

	receiverFactories, err := component.MakeReceiverFactoryMap(
		jaegerreceiver.NewFactory(),
		&zipkinreceiver.Factory{},
		&opencensusreceiver.Factory{},
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		return nil, err
	}

	cfgs, err := config.Load(v, config.Factories{
		Receivers: receiverFactories,
	})
	if err != nil {
		return nil, err
	}

	zapLogger, err := zap.NewProduction()
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

		if factory, ok := factoryBase.(component.ReceiverFactory); ok {
			receiver, err := factory.CreateTraceReceiver(ctx, params, cfg, shim)
			if err != nil {
				return nil, err
			}

			shim.receivers = append(shim.receivers, receiver)
			continue
		}

		factory := factoryBase.(component.ReceiverFactoryOld)
		receiver, err := factory.CreateTraceReceiver(ctx, zapLogger, cfg, converter.NewOCToInternalTraceConverter(shim))
		if err != nil {
			return nil, err
		}
		shim.receivers = append(shim.receivers, receiver)
	}

	shim.Service = services.NewBasicService(shim.starting, nil, shim.stopping)

	return shim, nil
}
func (d *Distributor) starting(ctx context.Context) error {
	for _, receiver := range r.receivers {
		err := receiver.Start(ctx, r)
		if err != nil {
			return err
		}
	}

	return nil
}

// Called after distributor is asked to stop via StopAsync.
func (d *Distributor) stopping(_ error) error {
	ctx := context.WithTimeout(context.Background(), 30*time.Second)
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
	return nil
}

// implements consumer.TraceConsumer
func (r *receiversShim) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if !r.authEnabled {
		ctx = user.InjectOrgID(ctx, tempo_util.FakeTenantID)
	}

	var err error
	for _, resourceSpan := range pdata.TracesToOtlp(td) {
		_, err = r.pusher.Push(ctx, &tempopb.PushRequest{
			Batch: resourceSpan,
		})
		if err != nil {
			r.logger.Log("msg", "pusher failed to consume trace data", "err", err)
			break
		}
	}

	// todo:  confirm/deny if this error propagates back to the caller
	return err
}

// implements component.Host
func (r *receiversShim) ReportFatalError(err error) {
	level.Error(util.Logger).Log("msg", "fatal error reported", "err", err)
	panic(fmt.Sprintf("Fatal error %v", err))
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
