package frontend

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/tempopb"
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
			cfg := Config{
				TraceByID: TraceByIDConfig{
					QueryShards: minQueryShards,
					SLO:         testSLOcfg,
				},
				Search: SearchConfig{
					Sharder: SearchSharderConfig{
						ConcurrentRequests:    defaultConcurrentRequests,
						TargetBytesPerRequest: defaultTargetBytesPerRequest,
					},
					SLO: testSLOcfg,
				},
				MultiTenantQueriesEnabled: true,
			}
			tenantMiddleware := newMultiTenantMiddleware(cfg, combiner.NewTraceByID, log.NewNopLogger())

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
			next := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				reqCount.Inc() // Count the number of requests.

				// Check if the tenant is in the list of tenants.
				tenantID, _, err := user.ExtractOrgIDFromHTTPRequest(req)
				require.NoError(t, err)
				_, ok := tenantsMap[tenantID]
				require.True(t, ok)

				// we do this in requestForTenant method, which is skipped for single tenant
				if len(tenants) > 1 {
					// ensure that tenant id in http header is same as org id in context
					// some places are using http headers and some are using context to
					// extract tenant id form the request so need both to be set and be correct.
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

				return &http.Response{
					StatusCode: statusCode,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			})

			rt := NewRoundTripper(next, tenantMiddleware)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			require.NoError(t, err)
			req.Header.Set(user.OrgIDHeaderName, tc.tenants)

			res, err := rt.RoundTrip(req)
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
