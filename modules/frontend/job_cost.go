package frontend

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/blocksharding"
)

// jobCostMetrics are the metrics the reporters of all ops write to.
type jobCostMetrics struct {
	// inspected has the labels tenant and op, jobInspected has the label op.
	inspected    *prometheus.CounterVec
	jobInspected *prometheus.HistogramVec

	// ratio has the labels op and kind.
	ratio *prometheus.HistogramVec
}

// jobReporter reports what the jobs of a single op read and were estimated to read as they complete, and
// calibrates the estimator with it.
type jobReporter struct {
	op        jobcost.Op
	estimator *jobcost.Estimator

	inspected    *prometheus.CounterVec
	jobInspected prometheus.Observer

	// the ratio histograms of the kinds of cost that have an estimate
	blockRatio     prometheus.Observer
	liveStoreRatio prometheus.Observer
}

// newJobReporter returns a reporter for op.
func newJobReporter(op jobcost.Op, estimator *jobcost.Estimator, metrics *jobCostMetrics) *jobReporter {
	return &jobReporter{
		op:             op,
		estimator:      estimator,
		inspected:      metrics.inspected.MustCurryWith(prometheus.Labels{metricLabelOp: op.String()}),
		jobInspected:   metrics.jobInspected.WithLabelValues(op.String()),
		blockRatio:     metrics.ratio.WithLabelValues(op.String(), jobcost.KindBlock.String()),
		liveStoreRatio: metrics.ratio.WithLabelValues(op.String(), jobcost.KindLiveStore.String()),
	}
}

// option returns a combiner option that reports the jobs of a query of tenant as they complete. The
// estimates of the jobs are also added to the cell on ctx, which makes them available when the query is logged.
func (r *jobReporter) option(ctx context.Context, tenant string) combiner.Option {
	addEstimatedBytes := pipeline.EstimatedBytesAdder(ctx)

	return combiner.WithOptions(
		combiner.WithInspectedBytesReporter(func(b uint64) {
			r.inspected.WithLabelValues(tenant).Add(float64(b))
		}),
		combiner.WithJobInspectedBytesObserver(func(inspectedBytes uint64, cost jobcost.Cost) {
			r.observeJob(inspectedBytes, cost)
			addEstimatedBytes(cost.Estimate)
		}),
	)
}

func (r *jobReporter) observeJob(inspectedBytes uint64, cost jobcost.Cost) {
	r.jobInspected.Observe(float64(inspectedBytes))

	var ratio prometheus.Observer
	switch cost.Kind {
	case jobcost.KindBlock:
		ratio = r.blockRatio
	case jobcost.KindLiveStore:
		ratio = r.liveStoreRatio
	default:
		return
	}

	if cost.Estimate > 0 {
		ratio.Observe(float64(inspectedBytes) / float64(cost.Estimate))
	}

	r.estimator.Observe(r.op, cost, inspectedBytes)
}

func newJobCostMetrics(estimator *jobcost.Estimator, registerer prometheus.Registerer) *jobCostMetrics {
	factory := promauto.With(registerer)

	for _, op := range []jobcost.Op{jobcost.OpTraceByID, jobcost.OpSearch, jobcost.OpMetadata, jobcost.OpMetrics} {
		factory.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace:   metricNamespace,
			Name:        "query_frontend_job_cost_factor",
			Help:        "Bytes a block job is estimated to read per unit of its base. The base is the bytes of the block a job covers, or the number of blocks for trace by id jobs.",
			ConstLabels: prometheus.Labels{metricLabelOp: op.String()},
		}, func() float64 { return estimator.Factor(op) })
	}

	return &jobCostMetrics{
		inspected:    queryInspectedBytes,
		jobInspected: queryJobInspectedBytes,
		ratio: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                       metricNamespace,
			Name:                            "query_frontend_job_actual_to_estimated_bytes_ratio",
			Help:                            "Bytes inspected by a querier job divided by the bytes the job was estimated to read, excluding cached job responses and jobs with an estimate of zero.",
			Buckets:                         prometheus.ExponentialBuckets(0.01, 2, 15), // 0.01 to 164
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 1 * time.Hour,
		}, []string{metricLabelOp, metricLabelKind}),
	}
}

// blockJobCost returns the cost of a job that reads pages of the block m, starting at startPage.
func blockJobCost(estimator *jobcost.Estimator, op jobcost.Op, m *backend.BlockMeta, startPage, pages int) jobcost.Cost {
	return estimator.Block(op, blocksharding.JobBytes(m.Size_, uint32(startPage), uint32(pages), m.TotalRecords))
}
