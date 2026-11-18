package frontend

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
)

// jobCostMetrics are the metrics about the estimated cost of jobs, shared by the reporters of all ops.
type jobCostMetrics struct {
	estimated *prometheus.CounterVec
	ratio     *prometheus.HistogramVec
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
		estimated: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "query_frontend_job_estimated_bytes_total",
			Help:      "Bytes querier jobs were estimated to read when they were created, excluding cached job responses and jobs without an estimate.",
		}, []string{metricLabelOp, metricLabelKind}),
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

// jobReporter reports what the jobs of a single op read and were estimated to read as they complete, and
// calibrates the estimator with it.
type jobReporter struct {
	op        jobcost.Op
	estimator *jobcost.Estimator

	inspected    *prometheus.CounterVec
	jobInspected prometheus.Observer
	estimated    *prometheus.CounterVec
	ratio        prometheus.ObserverVec
}

// newJobReporter returns a reporter for op. inspected has the tenant as its only label and jobInspected is
// the histogram of the bytes inspected by each job.
func newJobReporter(op jobcost.Op, estimator *jobcost.Estimator, inspected *prometheus.CounterVec, jobInspected prometheus.Observer, metrics *jobCostMetrics) *jobReporter {
	labels := prometheus.Labels{metricLabelOp: op.String()}

	return &jobReporter{
		op:           op,
		estimator:    estimator,
		inspected:    inspected,
		jobInspected: jobInspected,
		estimated:    metrics.estimated.MustCurryWith(labels),
		ratio:        metrics.ratio.MustCurryWith(labels),
	}
}

// option returns a combiner option that reports the jobs of a query of tenant as they complete.
func (r *jobReporter) option(tenant string) combiner.Option {
	return combiner.WithOptions(
		combiner.WithInspectedBytesReporter(func(b uint64) {
			r.inspected.WithLabelValues(tenant).Add(float64(b))
		}),
		combiner.WithJobInspectedBytesObserver(r.observeJob),
	)
}

func (r *jobReporter) observeJob(inspectedBytes uint64, cost jobcost.Cost) {
	r.jobInspected.Observe(float64(inspectedBytes))

	if cost.Kind == jobcost.KindNone {
		return
	}

	kind := cost.Kind.String()
	r.estimated.WithLabelValues(kind).Add(float64(cost.Estimate))
	if cost.Estimate > 0 {
		r.ratio.WithLabelValues(kind).Observe(float64(inspectedBytes) / float64(cost.Estimate))
	}

	r.estimator.Observe(r.op, cost, inspectedBytes)
}
