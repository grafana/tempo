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
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
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
			tenantMiddleware := newMultiTenantMiddleware(log.NewNopLogger())

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
			var fastestTenant string
			next := AsyncRoundTripperFunc(func(req *http.Request) (Responses, error) {
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
					fastestTenant = tenantID
					statusCode = http.StatusOK
					buff, err := trace.Marshal()
					require.NoError(t, err)
					body = buff
				})

				return NewSyncResponse(&http.Response{
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

			res, err, done := resps.Next(context.Background())
			require.NotNil(t, res)
			require.True(t, done)

			require.NoError(t, err)
			require.Equal(t, len(tenants), int(reqCount.Load()))
			require.NotNil(t, res)
			require.Equal(t, http.StatusOK, res.StatusCode)

			buff, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			// Unmarshal response into a trace.
			responseTrace := &tempopb.Trace{}
			require.NoError(t, responseTrace.Unmarshal(buff))
			// Add tenant to the original trace to compare.
			if len(tenants) > 1 {
				combiner.InjectTenantResource(fastestTenant, trace)
			}
			// Check if the trace is the same as the original.
			require.Equal(t, trace, responseTrace)
		})
	}
}

func TestMultiTenantNotSupported(t *testing.T) {
	tests := []struct {
		name    string
		tenant  string
		err     error
		context bool
	}{
		{
			name:   "multi-tenant queries disabled",
			tenant: "test",
			err:    nil,
		},
		{
			name:   "multi-tenant queries disabled with multiple tenant",
			tenant: "test|test1",
			err:    ErrMultiTenantUnsupported,
		},
		{
			name:   "no org id in request context",
			tenant: "test",
			err:    user.ErrNoOrgID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.tenant != "" {
				ctx := user.InjectOrgID(context.Background(), tc.tenant)
				req = req.WithContext(ctx)
			}

			test := newMultiTenantUnsupportedMiddleware(log.NewNopLogger())
			next := AsyncRoundTripperFunc(func(req *http.Request) (Responses, error) { return nil, nil })

			rt := test.Wrap(next)
			_, err := rt.RoundTrip(req) // jpe assert resps?
			assert.Equal(t, tc.err, err)
		})
	}
}
