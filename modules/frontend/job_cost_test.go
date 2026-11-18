package frontend

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// newTestEstimator returns an estimator with its initial factors and a live store cost of 1000.
func newTestEstimator() *jobcost.Estimator {
	return jobcost.NewEstimator(jobcost.Config{LiveStoreCost: 1000})
}

func TestBlockJobCost(t *testing.T) {
	m := &backend.BlockMeta{Size_: 1000, TotalRecords: 100}

	tcs := []struct {
		name      string
		estimator *jobcost.Estimator
		startPage int
		pages     int
		expected  jobcost.Cost
	}{
		{
			name:      "whole block",
			estimator: newTestEstimator(),
			pages:     100,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 1000, Estimate: 10},
		},
		{
			name:      "part of the block",
			estimator: newTestEstimator(),
			startPage: 10,
			pages:     20,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 200, Estimate: 2},
		},
		{
			name:      "last job is cut off at the end of the block",
			estimator: newTestEstimator(),
			startPage: 90,
			pages:     40,
			expected:  jobcost.Cost{Kind: jobcost.KindBlock, Base: 100, Estimate: 1},
		},
		{
			name:      "no estimator",
			estimator: nil,
			pages:     100,
			expected:  jobcost.Cost{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, blockJobCost(tc.estimator, jobcost.OpSearch, m, tc.startPage, tc.pages))
		})
	}
}

// costedResponse is a job response that carries the cost of its job, like the responses of the pipeline.
type costedResponse struct {
	slosTestResponse
	cost jobcost.Cost
}

func (r costedResponse) Cost() jobcost.Cost { return r.cost }

func cacheHitResponse(cost jobcost.Cost) costedResponse {
	return costedResponse{
		slosTestResponse: slosTestResponse{header: http.Header{combiner.TempoCacheHeader: {combiner.TempoCacheHit}}},
		cost:             cost,
	}
}

func TestJobCostOpLabels(t *testing.T) {
	require.Equal(t, traceByIDOp, jobcost.OpTraceByID.String())
	require.Equal(t, searchOp, jobcost.OpSearch.String())
	require.Equal(t, metadataOp, jobcost.OpMetadata.String())
	require.Equal(t, metricsOp, jobcost.OpMetrics.String())
}

func TestJobReporter(t *testing.T) {
	reg := prometheus.NewRegistry()
	estimator := newTestEstimator()
	metrics := newTestJobCostMetrics(estimator, reg)
	reporter := newJobReporter(jobcost.OpSearch, estimator, metrics)

	block := jobcost.Cost{Kind: jobcost.KindBlock, Base: 1000, Estimate: 10}
	liveStore := jobcost.Cost{Kind: jobcost.KindLiveStore, Estimate: 20}

	c := combiner.NewSearchMetricsCombiner(reporter.option("tenant-a"))
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 100}, costedResponse{cost: block})                               // 10x the estimate
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 5}, costedResponse{cost: block})                                 // half the estimate
	c.Combine(&tempopb.SearchMetrics{}, costedResponse{cost: block})                                                  // read nothing
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 40}, costedResponse{cost: liveStore})                            // twice the estimate
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 7}, slosTestResponse{})                                          // no cost
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 1000}, cacheHitResponse(block))                                  // cached
	c.Combine(&tempopb.SearchMetrics{InspectedBytes: 3}, costedResponse{cost: jobcost.Cost{Kind: jobcost.KindBlock}}) // estimate of zero

	// the counter has the bytes of every job, the histogram has every job
	require.InDelta(t, 155, testutil.ToFloat64(metrics.inspected.WithLabelValues("tenant-a", "search")), 0)
	require.Equal(t, uint64(6), histogramOf(t, metrics.jobInspected.WithLabelValues("search")).GetSampleCount())

	// only the jobs with an estimate are estimated, cached jobs are not. a block job with an estimate of zero adds nothing
	require.InDelta(t, 30, testutil.ToFloat64(metrics.estimated.WithLabelValues("search", "block")), 0)
	require.InDelta(t, 20, testutil.ToFloat64(metrics.estimated.WithLabelValues("search", "live_store")), 0)

	// the ratio is observed for jobs with an estimate greater than zero, including jobs that read nothing
	blockRatio := histogramOf(t, metrics.ratio.WithLabelValues("search", "block").(prometheus.Histogram))
	require.Equal(t, uint64(3), blockRatio.GetSampleCount())
	require.InDelta(t, 10+0.5+0, blockRatio.GetSampleSum(), 1e-9)

	liveRatio := histogramOf(t, metrics.ratio.WithLabelValues("search", "live_store").(prometheus.Histogram))
	require.Equal(t, uint64(1), liveRatio.GetSampleCount())
	require.InDelta(t, 2, liveRatio.GetSampleSum(), 1e-9)
}

func TestJobReporterKinds(t *testing.T) {
	reg := prometheus.NewRegistry()
	estimator := newTestEstimator()
	metrics := newTestJobCostMetrics(estimator, reg)
	reporter := newJobReporter(jobcost.OpMetadata, estimator, metrics)

	// every kind but none has metrics
	require.Equal(t, kindMetrics{}, reporter.kinds[jobcost.KindNone])
	for kind := jobcost.KindBlock; int(kind) < len(reporter.kinds); kind++ {
		require.NotNil(t, reporter.kinds[kind].estimated, kind.String())
		require.NotNil(t, reporter.kinds[kind].ratio, kind.String())
	}

	// the series of every kind exist before the first job is reported
	for _, kind := range []string{"block", "live_store"} {
		labels := map[string]string{"op": "metadata", "kind": kind}
		require.Zero(t, metricValue(t, reg, "tempo_query_frontend_job_estimated_bytes_total", labels).GetCounter().GetValue())
		require.Zero(t, metricValue(t, reg, "tempo_query_frontend_job_actual_to_estimated_bytes_ratio", labels).GetHistogram().GetSampleCount())
	}

	// a kind without metrics only observes the job
	reporter.observeJob(10, jobcost.Cost{Kind: jobcost.Kind(99), Estimate: 5})
	reporter.observeJob(10, jobcost.Cost{Kind: jobcost.KindNone, Estimate: 5})

	require.Equal(t, uint64(2), histogramOf(t, metrics.jobInspected.WithLabelValues("metadata")).GetSampleCount())
	for _, kind := range []string{"block", "live_store"} {
		labels := map[string]string{"op": "metadata", "kind": kind}
		require.Zero(t, metricValue(t, reg, "tempo_query_frontend_job_estimated_bytes_total", labels).GetCounter().GetValue())
		require.Zero(t, metricValue(t, reg, "tempo_query_frontend_job_actual_to_estimated_bytes_ratio", labels).GetHistogram().GetSampleCount())
	}
}

func TestJobReporterCalibratesTheEstimator(t *testing.T) {
	reg := prometheus.NewRegistry()
	estimator := jobcost.NewEstimator(jobcost.Config{
		CalibrationEnabled: true,
		LiveStoreCost:      20,
	})
	metrics := newTestJobCostMetrics(estimator, reg)
	reporter := newJobReporter(jobcost.OpSearch, estimator, metrics)

	c := combiner.NewSearchMetricsCombiner(reporter.option("tenant-a"))
	initial := estimator.Factor(jobcost.OpSearch)
	require.InDelta(t, 0.01, initial, 1e-12)
	require.InDelta(t, initial, gaugeValue(t, reg, "tempo_query_frontend_job_cost_factor", "search"), 1e-12)

	// jobs that do not say how much a block job reads do not calibrate the estimator
	for range 2 * 100 {
		c.Combine(&tempopb.SearchMetrics{InspectedBytes: 1_000_000}, costedResponse{cost: estimator.LiveStore()}) // live store jobs
		c.Combine(nil, costedResponse{cost: estimator.Block(jobcost.OpSearch, 1000)})                             // jobs without metrics
		c.Combine(&tempopb.SearchMetrics{}, cacheHitResponse(estimator.Block(jobcost.OpSearch, 1000)))            // cached jobs
		c.Combine(&tempopb.SearchMetrics{InspectedBytes: 5}, slosTestResponse{})                                  // jobs without a cost
	}
	require.Equal(t, initial, estimator.Factor(jobcost.OpSearch))

	// block jobs read 5% of the bytes they cover, the factor follows after enough of them
	for range 100 {
		c.Combine(&tempopb.SearchMetrics{InspectedBytes: 50}, costedResponse{cost: estimator.Block(jobcost.OpSearch, 1000)})
	}
	require.InDelta(t, 0.05, estimator.Factor(jobcost.OpSearch), 1e-9)
	require.InDelta(t, 0.05, gaugeValue(t, reg, "tempo_query_frontend_job_cost_factor", "search"), 1e-9)

	// the other ops are not affected
	require.InDelta(t, 0.01, estimator.Factor(jobcost.OpMetrics), 1e-12)
	require.InDelta(t, 0.01, gaugeValue(t, reg, "tempo_query_frontend_job_cost_factor", "metrics"), 1e-12)
}

// newTestJobCostMetrics returns metrics that do not share the inspected bytes with the metrics of the frontend.
func newTestJobCostMetrics(estimator *jobcost.Estimator, reg prometheus.Registerer) *jobCostMetrics {
	metrics := newJobCostMetrics(estimator, reg)
	metrics.inspected = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "inspected_bytes_test"}, []string{metricLabelTenant, metricLabelOp})
	metrics.jobInspected = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "job_inspected_bytes_test"}, []string{metricLabelOp})
	return metrics
}

func histogramOf(t *testing.T, h prometheus.Observer) *dto.Histogram {
	t.Helper()

	var m dto.Metric
	require.NoError(t, h.(prometheus.Metric).Write(&m))
	return m.GetHistogram()
}

func gaugeValue(t *testing.T, reg *prometheus.Registry, name, op string) float64 {
	t.Helper()

	return metricValue(t, reg, name, map[string]string{"op": op}).GetGauge().GetValue()
}

// metricValue returns the metric with the name and exactly the labels from the registry.
func metricValue(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) *dto.Metric {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)

	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.GetMetric() {
			actual := map[string]string{}
			for _, l := range m.GetLabel() {
				actual[l.GetName()] = l.GetValue()
			}
			if maps.Equal(actual, labels) {
				return m
			}
		}
	}

	require.FailNow(t, "metric not found", "%s%v", name, labels)
	return nil
}

func TestFrontendTagsReportsJobCostsAsMetadata(t *testing.T) {
	reg := prometheus.NewRegistry()
	rdr := &mockReader{
		metas: []*backend.BlockMeta{{
			StartTime:    time.Unix(15, 0),
			EndTime:      time.Unix(16, 0),
			Size_:        defaultTargetBytesPerRequest,
			TotalRecords: 1,
			BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000123"),
		}},
	}
	f := frontendWithSettingsAndRegisterer(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.SearchTagsV2Response{Metrics: &tempopb.MetadataMetrics{InspectedBytes: 1}}
		},
	}, rdr, nil, nil, nil, reg, func(cfg *Config, _ *overrides.Config) {
		cfg.JobCost = jobcost.DefaultConfig()
	})

	req := httptest.NewRequest("GET", "/api/v2/search/tags?start=10&end=20&scope=resource", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "tenant"))
	resp := httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	// one block job that covers the whole block and is estimated to read 1% of it
	block := map[string]string{"op": "metadata", "kind": "block"}
	estimated := metricValue(t, reg, "tempo_query_frontend_job_estimated_bytes_total", block).GetCounter().GetValue()
	require.InDelta(t, defaultTargetBytesPerRequest/100, estimated, 1)
	ratio := metricValue(t, reg, "tempo_query_frontend_job_actual_to_estimated_bytes_ratio", block).GetHistogram()
	require.Equal(t, uint64(1), ratio.GetSampleCount())

	// the other ops did not estimate anything
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, fam := range families {
		if fam.GetName() != "tempo_query_frontend_job_estimated_bytes_total" {
			continue
		}
		for _, m := range fam.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "op" && l.GetValue() != "metadata" {
					require.Zero(t, m.GetCounter().GetValue(), "op %s", l.GetValue())
				}
			}
		}
	}
}
