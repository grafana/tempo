package receiver

import (
	"context"
	"fmt"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
	"go.uber.org/zap"

	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const (
	logsPerSecond = 10
)

type Receivers interface {
	Start() error
	Shutdown() error
}

type receiversShim struct {
	authEnabled bool
	receivers   []receiver.TraceReceiver
	pusher      tempopb.PusherServer
	logger      *tempo_util.RateLimitedLogger
}

func New(receiverCfg map[string]interface{}, pusher tempopb.PusherServer, authEnabled bool) (Receivers, error) {
	shim := &receiversShim{
		authEnabled: authEnabled,
		pusher:      pusher,
		logger:      tempo_util.NewRateLimitedLogger(logsPerSecond, level.Error(util.Logger)),
	}

	v := viper.New()
	err := v.MergeConfigMap(receiverCfg)
	if err != nil {
		return nil, err
	}

	// get factories somehow?
	factories, err := receiver.Build(
		&jaegerreceiver.Factory{},
		&zipkinreceiver.Factory{},
		&opencensusreceiver.Factory{},
	)
	if err != nil {
		return nil, err
	}

	configs, err := loadReceivers(v, receiverCfg, factories)
	if err != nil {
		return nil, err
	}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		factory := factories[config.Type()]
		if factory == nil {
			return nil, fmt.Errorf("receiver factory not found for type: %s", config.Type())
		}

		// todo: propagate a real context?  translate our log configuration into zap?
		receiver, err := factory.CreateTraceReceiver(context.Background(), zapLogger, config, shim)
		if err != nil {
			return nil, err
		}

		shim.receivers = append(shim.receivers, receiver)
	}

	return shim, nil
}

// implements Receivers
func (r *receiversShim) Start() error {
	for _, rcv := range r.receivers {
		err := rcv.Start(r)
		if err != nil {
			return err
		}
	}

	return nil
}

// implements Receivers
func (r *receiversShim) Shutdown() error {
	for _, rcv := range r.receivers {
		err := rcv.Shutdown()
		if err != nil {
			// log, but keep on shutting down
			level.Error(util.Logger).Log("msg", "failed to stop receiver", "err", err)
		}
	}
	r.logger.Stop()

	return nil
}

// implements consumer.TraceConsumer
func (r *receiversShim) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	if !r.authEnabled {
		ctx = user.InjectOrgID(ctx, tempo_util.FakeTenantID)
	}

	// todo: eventually otel collector intends to start using otel proto internally instead of opencensus
	//  when that happens we need to update our dependency and we can remove all of this translation logic
	// also note: this translation logic is woefully incomplete and is meant as a stopgap while we wait for the otel collector
	batches := ocToOtlp(td)

	var err error
	for _, b := range batches {
		_, err = r.pusher.Push(ctx, &tempopb.PushRequest{
			Batch: b,
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
func (r *receiversShim) Context() context.Context {
	// todo: something better here?
	return context.Background()
}
