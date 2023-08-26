package frontend

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/opentracing/opentracing-go"
)

type Combiner interface {
	Consume(*http.Response)
	Result() (*http.Response, error)
}

// TODO: which config exactly as different usecases need diff configs, and we dont have inheritance in configs
func newCrossTenantHandler(cfg *Config, combiner Combiner, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return parseQuery{
			next:     next,
			combiner: combiner,
			logger:   logger,
		}
	})
}

type parseQuery struct {
	next     http.RoundTripper
	combiner Combiner
	logger   log.Logger
}

func (p parseQuery) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ParseQueryForMultiTenants")
	defer span.Finish()

	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	tenantSpecificRequests, err := buildTenantSpecificRequests(r, orgID)
	if err != nil {
		return nil, err
	}

	wg := boundedwaitgroup.New(uint(len(tenantSpecificRequests)))
	mtx := sync.Mutex{}

	var overallError error
	for _, request := range tenantSpecificRequests {
		wg.Add(1)

		go func(innerR *http.Request) {
			defer wg.Done()

			resp, err := p.next.RoundTrip(innerR)

			mtx.Lock()
			defer mtx.Unlock()

			if err != nil {
				_ = level.Error(p.logger).Log("msg", "error querying proxy target", "url", innerR.RequestURI, "err", err)
				// TODO: as soon as there is an error, we stop porcessing all requests.
				// Currently we keep processing, which seems unnecessary as on line 84 we are returning nil if overallError is not nil
				overallError = err
				return
			}

			p.combiner.Consume(resp)
		}(request)
	}
	wg.Wait()

	if overallError != nil {
		return nil, overallError
	}

	return p.combiner.Result()
}

func buildTenantSpecificRequests(parent *http.Request, orgID string) ([]*http.Request, error) {
	tenants, err := tenant.TenantIDsFromOrgID(orgID)
	if err != nil {
		return nil, err
	}
	reqs := make([]*http.Request, len(tenants))
	for i, tenant := range tenants {
		reqs[i] = buildRequestForTenant(parent, tenant)
	}

	return reqs, nil
}

func buildRequestForTenant(parent *http.Request, tenant string) *http.Request {
	ctx := parent.Context()
	req := parent.Clone(ctx)
	q := req.URL.Query()
	req.Header.Set(user.OrgIDHeaderName, tenant)
	uri := buildUpstreamRequestURI(req.URL.Path, q)
	req.RequestURI = uri

	return req
}
