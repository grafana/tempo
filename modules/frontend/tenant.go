package frontend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusCodeLabel = "status_code"
	tenantLabel     = "tenant"
)

var (
	tenantSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "tenant_federation_success_total",
			Help:      "Total number of successful fetches of a trace per tenant.",
		},
		[]string{tenantLabel})

	tenantFailureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "tenant_federation_failures_total",
			Help:      "Total number of failing fetches of a trace per tenant.",
		},
		[]string{tenantLabel, statusCodeLabel})
)

type tenantRoundTripper struct {
	cfg    Config
	next   http.RoundTripper
	logger log.Logger

	resolver tenant.Resolver

	newCombiner func() combiner.Combiner

	tenantSuccessTotal *prometheus.CounterVec
	tenantFailureTotal *prometheus.CounterVec
}

// newMultiTenantMiddleware returns a middleware that takes a request and fans it out to each tenant
func newMultiTenantMiddleware(cfg Config, combinerFn func() combiner.Combiner, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return &tenantRoundTripper{
			cfg:                cfg,
			next:               next,
			logger:             logger,
			resolver:           tenant.NewMultiResolver(),
			newCombiner:        combinerFn,
			tenantSuccessTotal: tenantSuccessTotal,
			tenantFailureTotal: tenantFailureTotal,
		}
	})
}

func (t *tenantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.cfg.MultiTenantQueriesEnabled {
		// move on to next tripper if multi-tenant queries are not enabled
		return t.next.RoundTrip(req)
	}

	_, ctx, err := user.ExtractOrgIDFromHTTPRequest(req)
	if err == user.ErrNoOrgID {
		// no org id, move to next tripper
		return t.next.RoundTrip(req)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to extract org id from request: %w", err)
	}

	// extract tenant ids
	tenants, err := t.resolver.TenantIDs(ctx)
	if err != nil {
		return nil, err
	}
	// for single tenant, fall through to next round tripper
	if len(tenants) <= 1 {
		return t.next.RoundTrip(req)
	}

	// join tenants for logger because list value type is unsupported.
	_ = level.Debug(t.logger).Log("msg", "handling multi-tenant query", "tenants", strings.Join(tenants, ","))

	var wg sync.WaitGroup
	respCombiner := t.newCombiner()

	// call RoundTrip for each tenant and combine results
	// Send one request per tenant to down-stream tripper
	// Return early if statusCode is already set by a previous response
	for _, tenantID := range tenants {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			// build a sub request context for each tenant because we want to modify and inject a tenant id into the context.
			// this is done so that components downstream of frontend doesn't need to know anything about multi-tenant query
			subCtx, cancel := context.WithCancel(req.Context())
			defer cancel()

			if respCombiner.ShouldQuit() {
				return
			}

			_ = level.Info(t.logger).Log("msg", "sending request for tenant", "path", req.URL.EscapedPath(), "tenant", tenant)

			r := requestForTenant(subCtx, req, tenant)
			resp, err := t.next.RoundTrip(r)

			if respCombiner.ShouldQuit() {
				return
			}

			// Check http error
			if err != nil {
				_ = level.Error(t.logger).Log("msg", "error querying for tenant", "tenant", tenant, "err", err)
				t.tenantFailureTotal.With(prometheus.Labels{tenantLabel: tenant, statusCodeLabel: strconv.Itoa(respCombiner.StatusCode())}).Inc()
				return
			}

			// If we get here, we have a successful response
			if err := respCombiner.AddRequest(resp, tenant); err != nil {
				_ = level.Error(t.logger).Log("msg", "error combining responses", "tenant", tenant, "err", err)
				t.tenantFailureTotal.With(prometheus.Labels{tenantLabel: tenant, statusCodeLabel: strconv.Itoa(resp.StatusCode)}).Inc()
				return
			}

			_ = level.Debug(t.logger).Log("msg", "multi-tenant request success", "tenant", tenant)
			t.tenantSuccessTotal.With(prometheus.Labels{tenantLabel: tenant}).Inc()
		}(tenantID)
	}

	// TODO: will this work for search streaming??, look into it. might need a search steaming combiner
	wg.Wait()

	return respCombiner.Complete()
}

// requestForTenant makes a copy of request and injects the tenant id into context and Header.
// this allows us to keep all multi-tenant logic in query frontend and keep other components single tenant
func requestForTenant(ctx context.Context, r *http.Request, tenant string) *http.Request {
	ctx = user.InjectOrgID(ctx, tenant)
	rCopy := r.Clone(ctx)
	rCopy.Header.Set(user.OrgIDHeaderName, tenant)
	return rCopy
}

// newMultiTenantUnsupportedMiddleware(cfg, handler)
// return error if we have multiple tenants.
// pass through to handler if we get single tenant.

type unsupportedRoundTripper struct {
	cfg    Config
	next   http.RoundTripper
	logger log.Logger

	resolver tenant.Resolver
}

func newMultiTenantUnsupportedMiddleware(cfg Config, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return &unsupportedRoundTripper{
			cfg:      cfg,
			next:     next,
			logger:   logger,
			resolver: tenant.NewMultiResolver(),
		}
	})
}

// TODO: is it easy to have a handler instead of Middleware here? maybe yes??
// FIXME: I think we need handler to wrap newSearchStreamingWSHandler and newSearchStreamingGRPCHandler

func (t *unsupportedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.cfg.MultiTenantQueriesEnabled {
		// move on to next tripper if multi-tenant queries are not enabled
		return t.next.RoundTrip(req)
	}

	if !t.cfg.MultiTenantQueriesEnabled {
		// move on to next tripper if multi-tenant queries are not enabled
		return t.next.RoundTrip(req)
	}

	_, ctx, err := user.ExtractOrgIDFromHTTPRequest(req)
	if err == user.ErrNoOrgID {
		// no org id, move to next tripper
		return t.next.RoundTrip(req)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to extract org id from request: %w", err)
	}

	// extract tenant ids
	tenants, err := t.resolver.TenantIDs(ctx)
	if err != nil {
		return nil, err
	}
	// for single tenant, fall through to next round tripper
	if len(tenants) <= 1 {
		return t.next.RoundTrip(req)
	} else {
		// fail in case we get multiple tenants
		return nil, common.ErrUnsupported
	}
}
