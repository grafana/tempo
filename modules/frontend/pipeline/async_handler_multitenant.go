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
)

var ErrMultiTenantUnsupported = errors.New("multi-tenant query unsupported")

type tenantRoundTripper struct {
	next   AsyncRoundTripper[*http.Response]
	logger log.Logger

	resolver tenant.Resolver
}

// NewMultiTenantMiddleware returns a middleware that takes a request and fans it out to each tenant
// It currently accepts a success and failure counter, to prevent metrics collisions with
func NewMultiTenantMiddleware(logger log.Logger) AsyncMiddleware[*http.Response] {
	return AsyncMiddlewareFunc[*http.Response](func(next AsyncRoundTripper[*http.Response]) AsyncRoundTripper[*http.Response] {
		return &tenantRoundTripper{
			next:     next,
			logger:   logger,
			resolver: tenant.NewMultiResolver(),
		}
	})
}

func (t *tenantRoundTripper) RoundTrip(req *http.Request) (Responses[*http.Response], error) {
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

	return NewAsyncSharder(0, func(tenantIdx int) *http.Request {
		if tenantIdx >= len(tenants) {
			return nil
		}
		return requestForTenant(req.Context(), req, tenants[tenantIdx])
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
	next   AsyncRoundTripper[*http.Response]
	logger log.Logger

	resolver tenant.Resolver
}

func NewMultiTenantUnsupportedMiddleware(logger log.Logger) AsyncMiddleware[*http.Response] {
	return AsyncMiddlewareFunc[*http.Response](func(next AsyncRoundTripper[*http.Response]) AsyncRoundTripper[*http.Response] {
		return &unsupportedRoundTripper{
			next:     next,
			logger:   logger,
			resolver: tenant.NewMultiResolver(),
		}
	})
}

func (t *unsupportedRoundTripper) RoundTrip(req *http.Request) (Responses[*http.Response], error) {
	// extract tenant ids
	tenants, err := t.resolver.TenantIDs(req.Context())
	if err != nil {
		return NewBadRequest(err), nil
	}
	// error if we get more then 1 tenant
	if len(tenants) > 1 {
		return NewBadRequest(ErrMultiTenantUnsupported), nil
	}

	return t.next.RoundTrip(req)
}
