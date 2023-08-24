package receiver

import (
	"context"

	"github.com/grafana/dskit/user"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
)

type ConsumeTracesFunc func(context.Context, ptrace.Traces) error

func (f ConsumeTracesFunc) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (f ConsumeTracesFunc) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return f(ctx, td)
}

type Middleware interface {
	Wrap(consumer.Traces) consumer.Traces
}

type MiddlewareFunc func(consumer.Traces) consumer.Traces

// Wrap implements Interface
func (tc MiddlewareFunc) Wrap(next consumer.Traces) consumer.Traces {
	return tc(next)
}

type fakeTenantMiddleware struct{}

func FakeTenantMiddleware() Middleware {
	return &fakeTenantMiddleware{}
}

func (m *fakeTenantMiddleware) Wrap(next consumer.Traces) consumer.Traces {
	return ConsumeTracesFunc(func(ctx context.Context, td ptrace.Traces) error {
		ctx = user.InjectOrgID(ctx, util.FakeTenantID)
		return next.ConsumeTraces(ctx, td)
	})
}

type multiTenancyMiddleware struct{}

func MultiTenancyMiddleware() Middleware {
	return &multiTenancyMiddleware{}
}

func (m *multiTenancyMiddleware) Wrap(next consumer.Traces) consumer.Traces {
	return ConsumeTracesFunc(func(ctx context.Context, td ptrace.Traces) error {
		var err error
		_, ctx, err = user.ExtractFromGRPCRequest(ctx)
		if err != nil {
			// Maybe its a HTTP request.
			info := client.FromContext(ctx)
			orgIDs := info.Metadata.Get(user.OrgIDHeaderName)
			clientAddr := "unknown"
			if info.Addr != nil {
				clientAddr = info.Addr.String()
			}
			if len(orgIDs) == 0 {
				log.Logger.Log("msg", "failed to extract org id from both grpc and HTTP",
					"err", err, "client", clientAddr)
				return err
			}

			if len(orgIDs) > 1 {
				log.Logger.Log("msg", "more than one orgID found", "orgIDs", orgIDs, "client", clientAddr)
				return err
			}

			ctx = user.InjectOrgID(ctx, orgIDs[0])
		}
		return next.ConsumeTraces(ctx, td)
	})
}
