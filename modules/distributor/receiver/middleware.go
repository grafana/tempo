package receiver

import (
	"context"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type ConsumeTracesFunc func(context.Context, ptrace.Traces) error

func (f ConsumeTracesFunc) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

var _ propagation.TextMapCarrier = (*clientMetadataCarrier)(nil)

// clientMetadataCarrier is a propagation.TextMapCarrier for client.Metadata
type clientMetadataCarrier struct {
	metadata client.Metadata
}

func (c *clientMetadataCarrier) Get(key string) string {
	values := c.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *clientMetadataCarrier) Set(string, string) {} // Not implemented as we only need extraction

func (c *clientMetadataCarrier) Keys() []string {
	var keys []string
	for key := range c.metadata.Keys() {
		keys = append(keys, key)
	}
	return keys
}

type onlySampledTraces struct {
	propagation.TextMapPropagator
}

func (o onlySampledTraces) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsSampled() {
		return
	}
	o.TextMapPropagator.Inject(ctx, carrier)
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
			// Maybe it's an HTTP request.
			info := client.FromContext(ctx)

			// Extract trace context from HTTP headers
			carrier := &clientMetadataCarrier{metadata: info.Metadata}
			propagator := &onlySampledTraces{otel.GetTextMapPropagator()}
			ctx = propagator.Extract(ctx, carrier)

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
