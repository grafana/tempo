package pipeline

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusCodeLabel = "status_code"
	tenantLabel     = "tenant"
)

var ErrMultiTenantUnsupported = errors.New("multi-tenant query unsupported")

var (
	tenantSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tempo_query_frontend",
			Name:      "multitenant_success_total",
			Help:      "Total number of successful fetches of a trace per tenant.",
		},
		[]string{tenantLabel})

	tenantFailureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tempo_query_frontend",
			Name:      "multitenant_failures_total",
			Help:      "Total number of failing fetches of a trace per tenant.",
		},
		[]string{tenantLabel, statusCodeLabel})
)

type tenantRoundTripper struct {
	next   AsyncRoundTripper
	logger log.Logger

	resolver tenant.Resolver

	tenantSuccessTotal *prometheus.CounterVec
	tenantFailureTotal *prometheus.CounterVec
}

// newMultiTenantMiddleware returns a middleware that takes a request and fans it out to each tenant  - jpe used to take a combiner. save for the end of async
func newMultiTenantMiddleware(logger log.Logger) AsyncMiddleware {
	return AsyncMiddlewareFunc(func(next AsyncRoundTripper) AsyncRoundTripper {
		return &tenantRoundTripper{
			next:               next,
			logger:             logger,
			resolver:           tenant.NewMultiResolver(),
			tenantSuccessTotal: tenantSuccessTotal,
			tenantFailureTotal: tenantFailureTotal,
		}
	})
}

// jpe -  used to accept a config, for "enabled" or not. just don't install if not enabled
func (t *tenantRoundTripper) RoundTrip(req *http.Request) (Responses, error) {
	// extract tenant ids, this will normalize and de-duplicate tenant ids
	tenants, err := t.resolver.TenantIDs(req.Context())
	if err != nil {
		// if we return an err here, downstream handler will turn it into HTTP 500 Internal Server Error.
		// respond with 400 and error as body and return nil error.
		return NewBadRequest(err), nil
	}

	// for single tenant, go to next round tripper
	if len(tenants) <= 1 {
		return t.next.RoundTrip(req)
	}

	// join tenants for logger because list value type is unsupported.
	_ = level.Debug(t.logger).Log("msg", "handling multi-tenant query", "tenants", strings.Join(tenants, ","))

	// jpe a lot of code was lost here. respCombiner.ShouldQuit() cancelly stuff was removed. this needs to exist at a higher lvl. does it exist once per pipeline? or after each sharding layer to control combine and reply logic?
	return NewAsyncSharder(0, func(tenantIdx int) (*http.Request, *http.Response) {
		if tenantIdx >= len(tenants) {
			return nil, nil
		}
		return requestForTenant(req.Context(), req, tenants[tenantIdx]), nil // jpe we used to do a subctx with a cancel, how does the new pattern work for cancellations?
	}, t.next), nil
}

// requestForTenant makes a copy of request and injects the tenant id into context and Header.
// this allows us to keep all multi-tenant logic in query frontend and keep other components single tenant
func requestForTenant(ctx context.Context, r *http.Request, tenant string) *http.Request {
	ctx = user.InjectOrgID(ctx, tenant)
	rCopy := r.Clone(ctx)
	rCopy.Header.Set(user.OrgIDHeaderName, tenant)
	return rCopy
}

type unsupportedRoundTripper struct {
	next   AsyncRoundTripper
	logger log.Logger

	resolver tenant.Resolver
}

// jpe - used to take a config, for "enabled" or not. just don't install if not enabled. or maybe install everywhere if enabled?
func newMultiTenantUnsupportedMiddleware(logger log.Logger) AsyncMiddleware {
	return AsyncMiddlewareFunc(func(next AsyncRoundTripper) AsyncRoundTripper {
		return &unsupportedRoundTripper{
			next:     next,
			logger:   logger,
			resolver: tenant.NewMultiResolver(),
		}
	})
}

func (t *unsupportedRoundTripper) RoundTrip(req *http.Request) (Responses, error) {
	// extract tenant ids
	tenants, err := t.resolver.TenantIDs(req.Context())
	if err != nil {
		return nil, err
	}
	// error if we get more then 1 tenant
	if len(tenants) > 1 {
		return NewBadRequest(ErrMultiTenantUnsupported), nil
	}

	return t.next.RoundTrip(req)
}
