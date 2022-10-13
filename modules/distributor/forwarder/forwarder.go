package forwarder

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"go.uber.org/multierr"

	"github.com/grafana/tempo/modules/distributor/forwarder/otlpgrpc"
	"github.com/grafana/tempo/pkg/tempopb"
)

type Forwarder interface {
	ForwardBatches(ctx context.Context, trace tempopb.Trace) error
	Shutdown(ctx context.Context) error
}

type List []Forwarder

func (l List) ForwardBatches(ctx context.Context, trace tempopb.Trace) error {
	var errs []error

	for _, forwarder := range l {
		if err := forwarder.ForwardBatches(ctx, trace); err != nil {
			errs = append(errs, err)
		}
	}

	return multierr.Combine(errs...)
}

func New(cfg Config, logger log.Logger) (Forwarder, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

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

		return f, nil
	default:
		return nil, fmt.Errorf("%s backend is not supported", cfg.Backend)
	}
}
