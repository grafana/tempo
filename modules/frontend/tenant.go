package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
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

	// extract tenant ids, this will normalize and de-duplicate tenant ids
	tenants, err := t.resolver.TenantIDs(req.Context())
	if err != nil {
		// if we return an err here, downstream handler will turn it into HTTP 500 Internal Server Error.
		// respond with 400 and error as body and return nil error.
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	// for single tenant, go to next round tripper
	if len(tenants) <= 1 {
		return t.next.RoundTrip(req)
	}

	// join tenants for logger because list value type is unsupported.
	_ = level.Debug(t.logger).Log("msg", "handling multi-tenant query", "tenants", strings.Join(tenants, ","))

	var wg sync.WaitGroup
	respCombiner := t.newCombiner()

	// call RoundTrip for each tenant and combine results
	// Send one request per tenant to downstream tripper
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

			_ = level.Info(t.logger).Log("msg", "sending request for tenant", "tenant", tenant, "path", req.URL.EscapedPath())

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

func (t *unsupportedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	err := MultiTenantNotSupported(t.cfg, t.resolver, req)
	if err != nil {
		_ = level.Debug(t.logger).Log("msg", "multi-tenant query unsupported", "error", err, "path", req.URL.EscapedPath())

		// if we return an err here, downstream handler will turn it into HTTP 500 Internal Server Error.
		// respond with 400 and error as body and return nil error.
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	return t.next.RoundTrip(req)
}

func MultiTenantNotSupported(cfg Config, resolver tenant.Resolver, req *http.Request) error {
	if !cfg.MultiTenantQueriesEnabled {
		return nil
	}

	// extract tenant ids
	tenants, err := resolver.TenantIDs(req.Context())
	if err != nil {
		return err
	}
	// error if we get more then 1 tenant
	if len(tenants) > 1 {
		return ErrMultiTenantUnsupported
	}
	return nil
}
