package frontend

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/tempo/modules/frontend/pipeline"
)

var ErrMultiTenantUnsupported = errors.New("multi-tenant query unsupported")

type unsupportedRoundTripper struct {
	cfg    Config
	next   http.RoundTripper
	logger log.Logger

	resolver tenant.Resolver
}

func newMultiTenantUnsupportedMiddleware(cfg Config, logger log.Logger) pipeline.Middleware {
	return pipeline.MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
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
