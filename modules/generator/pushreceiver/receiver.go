package pushreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

type receiver struct {
	recvCh  <-chan pdata.Traces
	closeCh chan struct{}

	nextConsumer consumer.Traces
}

func (r *receiver) Start(ctx context.Context, _ component.Host) error {
	go func() {
		for {
			select {
			case t := <-r.recvCh:
				// TODO: handle error
				_ = r.ConsumeTraces(ctx, t)
			case <-r.closeCh:
				return
			}
		}
	}()
	return nil
}

func (r *receiver) Shutdown(_ context.Context) error {
	close(r.closeCh)
	return nil
}

func (r *receiver) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return r.nextConsumer.ConsumeTraces(ctx, td)
}

func newPushReceiver(recvCh <-chan pdata.Traces, nextConsumer consumer.Traces) (component.TracesReceiver, error) {
	return &receiver{
		recvCh:       recvCh,
		closeCh:      make(chan struct{}),
		nextConsumer: nextConsumer,
	}, nil
}
