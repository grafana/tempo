package deployments

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/integration/util"
	tempoUtil "github.com/grafana/tempo/v3/pkg/util"
	"github.com/stretchr/testify/require"
)

// under the e2e library's hardcoded `docker stop --time=30`, so a slow exit still fails here
const gracefulStopBudget = 25 * time.Second

// read path first, memberlist gossip seeds last, so nothing stops after its gossip peers are gone
var shutdownOrder = []string{
	util.ServiceQueryFrontend,
	util.ServiceQuerier,
	util.ServiceMetricsGenerator,
	util.ServiceDistributor,
	util.ServiceBackendWorker,
	util.ServiceBackendScheduler,
	util.ServiceBlockBuilder,
	util.ServiceLiveStoreZoneB,
	util.ServiceLiveStoreZoneA,
}

func TestShutdownReadPath(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// serving a query leaves a tenant queue behind on the frontend, which is the state
		// that made it ignore SIGTERM until docker killed it
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		// querier first, so the frontend then has to drain with no querier available
		for _, name := range []string{util.ServiceQuerier, util.ServiceQueryFrontend} {
			start := time.Now()
			err := h.Services[name].Stop()
			elapsed := time.Since(start)

			// Stop returns the container's wait status, so exit status 137 here means SIGKILL
			require.NoError(t, err, "%s did not exit cleanly on SIGTERM", name)
			require.Less(t, elapsed, gracefulStopBudget, "%s took %s to stop", name, elapsed)
		}
	})
}

// single binary is the one mode where neither frontend nor querier enables blocklist polling
func TestShutdownSingleBinary(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		// traces_created_total can increment before the trace answers a query
		h.WaitTracesQueryable(t, 1)
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		// every service name maps to the same container here
		tempo := h.Services[util.ServiceQueryFrontend]
		start := time.Now()
		err := tempo.Stop()
		elapsed := time.Since(start)

		require.NoError(t, err, "single binary did not exit cleanly on SIGTERM")
		require.Less(t, elapsed, gracefulStopBudget, "single binary took %s to stop", elapsed)
	})
}

// accepted queries must finish, and anything arriving later gets a 503, never a 500
func TestShutdownInFlightQueries(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		var (
			mtx         sync.Mutex
			lost        []int
			ok          int
			unavailable int
			total       int
		)

		baseURL := h.BaseURL()
		stop := make(chan struct{}) // tells the loop below to finish
		done := make(chan struct{}) // indicates that we stopped.

		// query without pause, so a query is always in flight whenever SIGTERM lands
		go func() {
			defer close(done)
			for {
				// non-blocking, so the loop keeps querying instead of parking here
				select {
				case <-stop:
					return
				default:
				}

				code, err := searchStatus(baseURL)
				if err != nil {
					// no response at all, which a real client retries elsewhere
					continue
				}

				mtx.Lock()
				total++

				switch {
				case code == http.StatusServiceUnavailable:
					// request arrived after the queue stopped accepting and got 503
					unavailable++
				case code/100 == 5:
					// catch if we got 500
					lost = append(lost, code)
				default:
					// happy path
					ok++
				}
				mtx.Unlock()
			}
		}()

		// give 2 seconds for some queries to be in flight
		time.Sleep(2 * time.Second)

		// snapshot first, so a query still in flight can only land in the post-SIGTERM buckets
		mtx.Lock()
		servedBefore, refusedBefore := ok, unavailable
		mtx.Unlock()

		start := time.Now()
		err := h.Services[util.ServiceQueryFrontend].Stop()
		elapsed := time.Since(start)

		close(stop)
		<-done

		require.NoError(t, err, "query-frontend did not exit cleanly")
		require.Less(t, elapsed, gracefulStopBudget, "query-frontend took %s to stop", elapsed)

		mtx.Lock()
		defer mtx.Unlock()
		t.Logf("total=%d served=%d unavailable=%d lost=%d (before SIGTERM: served=%d refused=%d)",
			total, ok, unavailable, len(lost), servedBefore, refusedBefore)

		require.Equal(t, total, ok+unavailable+len(lost), "every response must land in exactly one bucket")
		require.Empty(t, lost, "frontend returned %v while shutting down", lost)

		// the queue only refuses once SIGTERM lands, so a healthy frontend refuses nothing
		require.Positive(t, servedBefore, "frontend served no queries before SIGTERM")
		require.Zero(t, refusedBefore, "frontend refused %d queries before SIGTERM", refusedBefore)

		// 5% of total is a ~100ms refusal window at this request rate, against ~3ms measured
		require.Less(t, unavailable*20, total, "refused %d of %d queries", unavailable, total)
	})
}

// with no querier to take it, queued work must still be answered rather than left hanging
func TestShutdownQueuedWork(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		require.NoError(t, h.Services[util.ServiceQuerier].Stop())

		const queued = 5

		type result struct {
			code int
			err  error
		}

		var wg sync.WaitGroup
		baseURL := h.BaseURL()
		results := make(chan result, queued)
		for range queued {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, err := searchStatus(baseURL)
				results <- result{code: code, err: err}
			}()
		}

		// let the queries reach the queue before the frontend drains
		time.Sleep(2 * time.Second)

		start := time.Now()
		err := h.Services[util.ServiceQueryFrontend].Stop()
		elapsed := time.Since(start)

		require.NoError(t, err, "query-frontend did not exit cleanly")
		require.Less(t, elapsed, gracefulStopBudget, "query-frontend took %s to stop", elapsed)

		// the frontend is gone by now, so any request still open fails fast rather than hanging
		wg.Wait()
		close(results)

		// none of them can be served with no querier, but every one must still be answered,
		// and with something the client will retry rather than a 500
		require.Len(t, results, queued, "not every queued query was answered")
		for res := range results {
			require.NoError(t, res.err, "queued query never got a response")
			require.Equal(t, http.StatusServiceUnavailable, res.code)
		}
	})
}

func searchStatus(baseURL string) (int, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/search?q=%7B%7D", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Scope-OrgID", tempoUtil.FakeTenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// TestShutdownAllComponents guards the bug class, not just the two deadlocks that prompted it.
func TestShutdownAllComponents(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying |
			util.ComponentsMetricsGeneration | util.ComponentsBackendWork,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		h.WaitTracesQueryable(t, 1)

		// shut down components that have live state, since an idle process can stop cleanly
		// while a working one deadlocks
		util.QueryAndAssertTrace(t, h.APIClientHTTP(""), info)

		type stopResult struct {
			name    string
			elapsed time.Duration
			err     error
		}

		// stop everything before asserting, so the first bad component does not hide the rest
		results := make([]stopResult, 0, len(shutdownOrder))
		for _, name := range shutdownOrder {
			svc := h.Services[name]
			require.NotNil(t, svc, "%s was not started, so this test covers less than it claims", name)

			start := time.Now()
			err := svc.Stop()
			results = append(results, stopResult{name: name, elapsed: time.Since(start), err: err})
		}

		for _, r := range results {
			t.Logf("shutdown %-22s %6.2fs err=%v", r.name, r.elapsed.Seconds(), r.err)
		}

		for _, r := range results {
			// Stop returns the container's wait status, so exit status 137 here means SIGKILL
			require.NoError(t, r.err, "%s did not exit cleanly on SIGTERM", r.name)
			require.Less(t, r.elapsed, gracefulStopBudget, "%s took %s to stop", r.name, r.elapsed)
		}
	})
}
