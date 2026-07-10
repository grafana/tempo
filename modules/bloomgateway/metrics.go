package bloomgateway

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricsNamespace is every series' Namespace below; combined with each
// Name's "bloom_gateway_" prefix, the registered name is always
// tempo_bloom_gateway_<...>, matching DESIGN.md's stated prefix.
const metricsNamespace = "tempo"

// metrics is every tempo_bloom_gateway_* series from DESIGN.md's §
// Metrics, plus unsupportedEncodingBlocks (§0 D7's mandatory subtlety —
// not in DESIGN.md's own table, added because a block that can never be
// column-projected must be observable, not silently invisible).
//
// Deliberately a struct built by newMetrics(reg), NOT package-level
// promauto vars: multiple *BloomGateway instances can exist in one process
// (every WP6/WP20 multi-instance test does exactly this), and package-level
// vars registered against the implicit default registerer would panic on
// the second instance's construction. Each *BloomGateway gets its own
// prometheus.Registerer (module-wiring report convention: constructors take
// a Registerer explicitly).
//
// This doc-comment block is the single source of truth mapping DESIGN.md's
// § Metrics names to the Go field that implements each one — kept
// exhaustive and in the same order as DESIGN.md's own two lists, so a
// side-by-side diff against DESIGN.md catches drift.
//
// Gateway gauges (DESIGN.md name -> field):
//
//	owned_leaves{state}           -> ownedLeaves
//	blocks_live                   -> blocksLive
//	entries_total                 -> entriesTotal
//	garbage_entries_estimate      -> garbageEntriesEstimate
//	memory_bytes{structure}       -> memoryBytes
//	steady_state_memory_bytes     -> steadyStateMemoryBytes
//	tenant_blocks{tenant}         -> tenantBlocks
//	topic_lag_messages            -> topicLagMessages
//	topic_lag_bytes               -> topicLagBytes
//	reconstruction_queue_ranges   -> reconstructionQueueRanges
//	snapshot_age_seconds          -> snapshotAgeSeconds
//	snapshot_bytes                -> snapshotBytes
//	miss_fp_rate_estimate         -> missFPRateEstimate
//	(not in DESIGN.md; §0 D7)      unsupported_encoding_blocks{tenant} -> unsupportedEncodingBlocks
//
// Gateway histograms/counters (DESIGN.md name -> field):
//
//	query_duration_seconds             -> queryDurationSeconds
//	query_candidates                   -> queryCandidates
//	response_bytes                     -> responseBytes
//	queries_total{result}              -> queriesTotal
//	add_apply_duration_seconds         -> addApplyDurationSeconds
//	adds_total{status}                 -> addsTotal
//	add_chunks_total                   -> addChunksTotal
//	deletes_total                      -> deletesTotal
//	sweep_pass_duration_seconds        -> sweepPassDurationSeconds
//	sweep_entries_removed_total        -> sweepEntriesRemovedTotal
//	reconstruction_duration_seconds    -> reconstructionDurationSeconds
//	reconstruction_blocks_total        -> reconstructionBlocksTotal
//	reconciliation_repairs_total{kind} -> reconciliationRepairsTotal
//	snapshot_duration_seconds          -> snapshotDurationSeconds
//
// Query-frontend-side and producer-side metrics from DESIGN.md's § Metrics
// (bloom_gateway_client_*, blocks_filtered_total, bloom_gateway_publishes_
// total, ...) are out of scope for this plan (§6: no QF client, no producer
// hooks) and have no field here.
type metrics struct {
	// gauges
	ownedLeaves               *prometheus.GaugeVec
	blocksLive                prometheus.Gauge
	entriesTotal              prometheus.Gauge
	garbageEntriesEstimate    prometheus.Gauge
	memoryBytes               *prometheus.GaugeVec
	steadyStateMemoryBytes    prometheus.Gauge
	tenantBlocks              *prometheus.GaugeVec
	topicLagMessages          prometheus.Gauge
	topicLagBytes             prometheus.Gauge
	reconstructionQueueRanges prometheus.Gauge
	snapshotAgeSeconds        prometheus.Gauge
	snapshotBytes             prometheus.Gauge
	missFPRateEstimate        prometheus.Gauge
	unsupportedEncodingBlocks *prometheus.GaugeVec

	// histograms / counters
	queryDurationSeconds          prometheus.Histogram
	queryCandidates               prometheus.Histogram
	responseBytes                 prometheus.Histogram
	queriesTotal                  *prometheus.CounterVec
	addApplyDurationSeconds       prometheus.Histogram
	addsTotal                     *prometheus.CounterVec
	addChunksTotal                prometheus.Counter
	deletesTotal                  prometheus.Counter
	sweepPassDurationSeconds      prometheus.Histogram
	sweepEntriesRemovedTotal      prometheus.Counter
	reconstructionDurationSeconds prometheus.Histogram
	reconstructionBlocksTotal     prometheus.Counter
	reconciliationRepairsTotal    *prometheus.CounterVec
	snapshotDurationSeconds       prometheus.Histogram
}

// newMetrics registers every series above against reg and returns them.
// Every Name below is deliberately "bloom_gateway_<...>" with
// Namespace "tempo" (matching e.g. modules/backendscheduler/metrics.go's
// convention) so the final registered name is tempo_bloom_gateway_<...>,
// exactly DESIGN.md's stated prefix.
func newMetrics(reg prometheus.Registerer) *metrics {
	durationBuckets := prometheus.DefBuckets

	return &metrics{
		ownedLeaves: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_owned_leaves",
			Help:      "Number of leaves this instance owns, by lifecycle state (constructing|complete).",
		}, []string{"state"}),
		blocksLive: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_blocks_live",
			Help:      "Number of blocks in this instance's registry that are live (Live or LiveUnsupportedEncoding) and therefore in the read path's view.",
		}),
		entriesTotal: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_entries_total",
			Help:      "Total number of (fingerprint, handle) entries across every leaf this instance owns, including garbage awaiting sweep.",
		}),
		garbageEntriesEstimate: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_garbage_entries_estimate",
			Help:      "Estimated number of leaf entries referencing deleted blocks, not yet removed by the background sweep.",
		}),
		memoryBytes: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_memory_bytes",
			Help:      "Estimated memory usage by structure (entries|directory|registry|tenants|garbage|queue).",
		}, []string{"structure"}),
		steadyStateMemoryBytes: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_steady_state_memory_bytes",
			Help:      "The HPA autoscaling signal: entries + directory + registry + A_T + garbage estimate, deliberately excluding reconstruction transients and queue buffers so rebuilds and replay don't flap the autoscaler (§ Autoscaling).",
		}),
		tenantBlocks: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_tenant_blocks",
			Help:      "Number of live blocks per tenant in this instance's A_T.",
		}, []string{"tenant"}),
		topicLagMessages: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_topic_lag_messages",
			Help:      "Consumer lag, in messages, summed across every partition this instance consumes.",
		}),
		topicLagBytes: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_topic_lag_bytes",
			Help:      "Consumer lag, in bytes, summed across every partition this instance consumes.",
		}),
		reconstructionQueueRanges: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_reconstruction_queue_ranges",
			Help:      "Number of leaf ranges currently pending in the reconstruction queue. Nonzero annotates expected freshness lag (§ Reconstruction).",
		}),
		snapshotAgeSeconds: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_snapshot_age_seconds",
			Help:      "Age of the most recently loaded or saved snapshot, in seconds.",
		}),
		snapshotBytes: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_snapshot_bytes",
			Help:      "Size of the most recently saved snapshot, in bytes.",
		}),
		missFPRateEstimate: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_miss_fp_rate_estimate",
			Help:      "Estimated miss-path false-positive rate, pairs / 2^(d+f) (§ Sizing).",
		}),
		unsupportedEncodingBlocks: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_unsupported_encoding_blocks",
			Help:      "Number of live blocks per tenant whose parquet encoding cannot be column-projected for reconstruction/reconciliation; these are never rejectable and always searched (§0 D7).",
		}, []string{"tenant"}),

		queryDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Name:                            "bloom_gateway_query_duration_seconds",
			Help:                            "Duration of a single Query RPC.",
			Buckets:                         durationBuckets,
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		}),
		queryCandidates: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_query_candidates",
			Help:      "Number of candidate blocks (matched references intersected with the tenant window) per query.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 12), // 1 .. 2048
		}),
		responseBytes: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_response_bytes",
			Help:      "Serialized size of the query response's rejection set.",
			Buckets:   prometheus.ExponentialBuckets(64, 4, 12), // 64 B .. ~16 MiB
		}),
		queriesTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_queries_total",
			Help:      "Total queries, by result (reject_all|candidates|empty).",
		}, []string{"result"}),
		addApplyDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Name:                            "bloom_gateway_add_apply_duration_seconds",
			Help:                            "Duration of applying one AddChunk event.",
			Buckets:                         durationBuckets,
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		}),
		addsTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_adds_total",
			Help:      "Total AddChunk events processed, by status (applied|dropped).",
		}, []string{"status"}),
		addChunksTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_add_chunks_total",
			Help:      "Total AddChunk chunks applied.",
		}),
		deletesTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_deletes_total",
			Help:      "Total Delete events applied.",
		}),
		sweepPassDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_sweep_pass_duration_seconds",
			Help:      "Duration of one full background sweep pass.",
			Buckets:   prometheus.DefBuckets,
		}),
		sweepEntriesRemovedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_sweep_entries_removed_total",
			Help:      "Total leaf entries removed by the background sweep.",
		}),
		reconstructionDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_reconstruction_duration_seconds",
			Help:      "Duration of one coalesced reconstruction column pass.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 14), // 1s .. ~2.3h
		}),
		reconstructionBlocksTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_reconstruction_blocks_total",
			Help:      "Total blocks processed by the reconstruction column pass.",
		}),
		reconciliationRepairsTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_reconciliation_repairs_total",
			Help:      "Total reconciliation repairs, by kind (add|delete). Nonzero steady-state rates indicate a broken producer, not a working safety net (§ Reconciliation).",
		}, []string{"kind"}),
		snapshotDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bloom_gateway_snapshot_duration_seconds",
			Help:      "Duration of one snapshot save.",
			Buckets:   prometheus.DefBuckets,
		}),
	}
}
