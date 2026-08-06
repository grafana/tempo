package frontend

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	metricNamespace   = "tempo"
	metricLabelTenant = "tenant"
	metricLabelOp     = "op"
	metricLabelResult = "result"
)

var queryEngineBytes = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: metricNamespace,
	Name:      "query_frontend_engine_bytes_total",
	Help:      "Bytes processed by the query engine by query type per tenant",
}, []string{metricLabelOp, metricLabelTenant})

func recordQueryMetrics(tenant string, op string, metrics *tempopb.SearchMetrics) {
	if metrics == nil {
		return
	}
	// TODO - Organize other metrics here, inspected bytes would be a good one.
	if metrics.AdditionalMetrics == nil {
		return
	}
	recordQueryAdditionalMetrics(op, tenant, metrics.AdditionalMetrics)
}

func recordTraceByIDMetrics(tenant string, metrics *tempopb.TraceByIDMetrics) {
	if metrics == nil {
		return
	}
	if metrics.AdditionalMetrics == nil {
		return
	}
	recordQueryAdditionalMetrics(traceByIDOp, tenant, metrics.AdditionalMetrics)
}

func recordQueryMetaDataMetrics(tenant string, metrics *tempopb.MetadataMetrics) {
	if metrics == nil {
		return
	}
	if metrics.AdditionalMetrics == nil {
		return
	}
	recordQueryAdditionalMetrics(metadataOp, tenant, metrics.AdditionalMetrics)
}

func recordQueryAdditionalMetrics(op string, tenant string, additionalMetrics map[string]int64) {
	if engineBytes, ok := additionalMetrics[tempopb.AdditionalMetricEngineBytes]; ok {
		queryEngineBytes.WithLabelValues(op, tenant).Add(float64(engineBytes))
	}
}
