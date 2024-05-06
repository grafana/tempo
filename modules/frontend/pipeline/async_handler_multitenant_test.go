package pipeline

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestMultiTenant(t *testing.T) {
	tests := []struct {
		name    string
		tenants string
	}{
		{
			name:    "single tenant",
			tenants: "single-tenant",
		},
		{
			name:    "multiple tenants",
			tenants: "tenant-1|tenant-2|tenant-3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tenantMiddleware := NewMultiTenantMiddleware(log.NewNopLogger())

			var reqCount atomic.Int32

			tenantsMap := make(map[string]struct{}, len(tc.tenants))
			tenants := strings.Split(tc.tenants, "|")
			for _, tenant := range tenants {
				tenantsMap[tenant] = struct{}{}
			}

			traceID := make([]byte, 16)
			_, err := rand.Read(traceID)
			require.NoError(t, err)
			trace := test.MakeTrace(10, traceID)

			once := sync.Once{}
			next := AsyncRoundTripperFunc[combiner.PipelineResponse](func(req *http.Request) (Responses[combiner.PipelineResponse], error) {
				reqCount.Inc() // Count the number of requests.

				// Check if the tenant is in the list of tenants.
				tenantID, err := user.ExtractOrgID(req.Context())
				require.NoError(t, err)
				_, ok := tenantsMap[tenantID]
				require.True(t, ok)

				// we do this in requestForTenant method, which is skipped for single tenant
				if len(tenants) > 1 {
					// ensure that tenant id in http header is same as org id in context
					orgID, err := user.ExtractOrgID(req.Context())
					require.NoError(t, err)
					require.Equal(t, tenantID, orgID)
				}

				statusCode := http.StatusNotFound
				var body []byte
				once.Do(func() {
					statusCode = http.StatusOK
					buff, err := trace.Marshal()
					require.NoError(t, err)
					body = buff
				})

				return NewHTTPToAsyncResponse(&http.Response{
					StatusCode: statusCode,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}), nil
			})

			rt := tenantMiddleware.Wrap(next)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			require.NoError(t, err)
			ctx := user.InjectOrgID(context.Background(), tc.tenants)
			req = req.WithContext(ctx)

			resps, err := rt.RoundTrip(req)
			require.NoError(t, err)

			for {
				res, done, err := resps.Next(context.Background())
				if done {
					break
				}

				require.NotNil(t, res)
				require.NoError(t, err)
			}

			require.Equal(t, len(tenants), int(reqCount.Load()))
		})
	}
}

func TestMultiTenantNotSupported(t *testing.T) {
	tests := []struct {
		name         string
		tenant       string
		expectedResp *http.Response
		context      bool
	}{
		{
			name:   "multi-tenant queries disabled",
			tenant: "test",
			expectedResp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
		},
		{
			name:   "multi-tenant queries disabled with multiple tenant",
			tenant: "test|test1",
			expectedResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(ErrMultiTenantUnsupported.Error())),
			},
		},
		{
			name: "no org id in request context",
			expectedResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(user.ErrNoOrgID.Error())),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.tenant != "" {
				ctx := user.InjectOrgID(context.Background(), tc.tenant)
				req = req.WithContext(ctx)
			}

			test := NewMultiTenantUnsupportedMiddleware(log.NewNopLogger())
			next := AsyncRoundTripperFunc[combiner.PipelineResponse](func(req *http.Request) (Responses[combiner.PipelineResponse], error) {
				return NewSuccessfulResponse("foo"), nil
			})

			rt := test.Wrap(next)
			resps, err := rt.RoundTrip(req)
			require.NoError(t, err) // no error expected. tenant unsupported should be passed back as a bad request. errors bubble up as 5xx

			r, done, err := resps.Next(context.Background())
			require.True(t, done)
			require.NoError(t, err)

			res := r.HTTPResponse()
			require.Equal(t, tc.expectedResp.StatusCode, res.StatusCode)
			require.Equal(t, tc.expectedResp.Status, res.Status)

			expectedBody, err := io.ReadAll(tc.expectedResp.Body)
			require.NoError(t, err)
			actualBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			require.Equal(t, expectedBody, actualBody)
		})
	}
}
