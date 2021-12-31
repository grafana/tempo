package pushreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	typeStr = "push"
)

type PushChan <-chan pdata.Traces

type Config struct {
	config.ReceiverSettings `mapstructure:",squash"`
}

func NewFactory(pushCh PushChan) component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTracesReceiver(pushCh)),
	)
}

func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewComponentIDWithName(typeStr, typeStr)),
	}
}

func createTracesReceiver(pushCh PushChan) func(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Traces,
) (component.TracesReceiver, error) {
	return func(
		_ context.Context,
		_ component.ReceiverCreateSettings,
		_ config.Receiver,
		nextConsumer consumer.Traces,
	) (component.TracesReceiver, error) {
		r, err := newPushReceiver(pushCh, nextConsumer)

		return r, err
	}
}
