package querier

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

const (
	// cacheKeyPrefixMetricsQueryRowGroup is the prefix and format-version tag for
	// per-row-group metric query cache keys. Bump the version (v1 -> v2) when the
	// cache value format changes incompatibly.
	cacheKeyPrefixMetricsQueryRowGroup = "qrgv1:"

	cacheSkipReasonNoProvider    = "no_provider"
	cacheSkipReasonNoCache       = "no_cache"
	cacheSkipReasonWrongVersion  = "wrong_version"
	cacheSkipReasonSamplerHint   = "sampler_hint"
	cacheSkipReasonZeroQueryHash = "zero_query_hash"
)

var (
	metricRowGroupCacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "querier",
		Name:      "metrics_rowgroup_cache_hits_total",
		Help:      "Per-row-group metric query cache hits.",
	}, []string{"tenant"})

	metricRowGroupCacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "querier",
		Name:      "metrics_rowgroup_cache_misses_total",
		Help:      "Per-row-group metric query cache misses.",
	}, []string{"tenant"})

	metricRowGroupCacheSkips = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "querier",
		Name:      "metrics_rowgroup_cache_skip_total",
		Help:      "Per-row-group metric query cache invocations that bypassed the cache.",
	}, []string{"tenant", "reason"})

	metricRowGroupCacheValueBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Subsystem: "querier",
		Name:      "metrics_rowgroup_cache_value_size_bytes",
		Help:      "Size in bytes of per-row-group metric query cache values written.",
		Buckets:   prometheus.ExponentialBuckets(256, 4, 10),
	})
)

// metricsRowGroupCacheKey builds the cache key for one (tenant, block, row group, query) tuple.
// Returns the empty string when hash == 0 (the sentinel meaning "not cacheable").
func metricsRowGroupCacheKey(tenant string, blockID backend.UUID, rgIdx int, hash uint64) string {
	if hash == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(cacheKeyPrefixMetricsQueryRowGroup) + len(tenant) + 1 + 36 + 1 + 6 + 1 + 20)
	sb.WriteString(cacheKeyPrefixMetricsQueryRowGroup)
	sb.WriteString(tenant)
	sb.WriteByte(':')
	sb.WriteString(blockID.String())
	sb.WriteByte(':')
	sb.WriteString(strconv.Itoa(rgIdx))
	sb.WriteByte(':')
	sb.WriteString(strconv.FormatUint(hash, 10))
	return sb.String()
}

// rowGroupQueryRangeHash returns a stable hash covering every input that affects
// the per-row-group first-stage result. It mirrors frontend.hashForQueryRangeRequest
// (modules/frontend/metrics_query_range_sharder.go) with three differences:
//   - MaxSeries is excluded — per-row-group evaluation runs uncapped; truncation
//     is applied only on the merged result. Including it would poison the cache.
//   - timeOverlapCutoff is included — it affects storageReq.StartTimeUnixNanos
//     pushdown at engine_metrics.go:1298–1301.
//   - spanOnlyFetch is included — it selects between DoSpansOnly and Do paths.
//   - dedicatedColumns signature is included — it affects attribute resolution.
//
// Returns 0 when the query fails to parse (sentinel — caller must treat as
// uncacheable).
func rowGroupQueryRangeHash(req *tempopb.QueryRangeRequest, dedicatedCols backend.DedicatedColumns, timeOverlapCutoff float64, spanOnlyFetch bool) uint64 {
	if req.Query == "" {
		return 0
	}
	ast, err := traceql.ParseNoOptimizations(req.Query)
	if err != nil {
		return 0
	}
	query := ast.String()

	hash := fnv1a.HashString64(query)
	hash = fnv1a.AddUint64(hash, req.Step)
	hash = fnv1a.AddUint64(hash, uint64(req.Exemplars))
	for _, name := range req.SkipASTTransformations {
		hash = fnv1a.AddString64(hash, name)
	}
	hash = fnv1a.AddUint64(hash, math.Float64bits(timeOverlapCutoff))
	if spanOnlyFetch {
		hash = fnv1a.AddUint64(hash, 1)
	} else {
		hash = fnv1a.AddUint64(hash, 0)
	}
	hash = addDedicatedColumnsToHash(hash, dedicatedCols)
	return hash
}

func addDedicatedColumnsToHash(hash uint64, dcs backend.DedicatedColumns) uint64 {
	hash = fnv1a.AddUint64(hash, uint64(len(dcs)))
	for _, dc := range dcs {
		hash = fnv1a.AddString64(hash, string(dc.Scope))
		hash = fnv1a.AddString64(hash, dc.Name)
		hash = fnv1a.AddString64(hash, string(dc.Type))
		hash = fnv1a.AddUint64(hash, uint64(len(dc.Options)))
		for _, opt := range dc.Options {
			hash = fnv1a.AddString64(hash, string(opt))
		}
	}
	return hash
}

// rowGroupCacheEligibility describes whether the per-row-group cache can be used
// for the current request, and if not, a stable reason label for metrics.
type rowGroupCacheEligibility struct {
	use    bool
	reason string
	cache  cache.Cache
}

// evaluateRowGroupCacheEligibility decides whether the per-row-group cache can serve
// the request. Returns the underlying Cache when usable.
func evaluateRowGroupCacheEligibility(req *tempopb.QueryRangeRequest, expr *traceql.RootExpr, provider cache.Provider, allowUnsafeHints bool, queryHash uint64) rowGroupCacheEligibility {
	if provider == nil {
		return rowGroupCacheEligibility{reason: cacheSkipReasonNoProvider}
	}
	if req.Version != vparquet5.VersionString {
		return rowGroupCacheEligibility{reason: cacheSkipReasonWrongVersion}
	}
	if queryHash == 0 {
		return rowGroupCacheEligibility{reason: cacheSkipReasonZeroQueryHash}
	}
	if expr != nil && hasSamplerHint(expr, allowUnsafeHints) {
		return rowGroupCacheEligibility{reason: cacheSkipReasonSamplerHint}
	}
	c := provider.CacheFor(cache.RoleMetricsQueryRowGroup)
	if c == nil {
		return rowGroupCacheEligibility{reason: cacheSkipReasonNoCache}
	}
	return rowGroupCacheEligibility{use: true, cache: c}
}

func hasSamplerHint(expr *traceql.RootExpr, allowUnsafeHints bool) bool {
	if expr == nil || expr.Hints == nil {
		return false
	}
	if _, ok := expr.Hints.GetFloat(traceql.HintTraceSample, allowUnsafeHints); ok {
		return true
	}
	if _, ok := expr.Hints.GetFloat(traceql.HintSpanSample, allowUnsafeHints); ok {
		return true
	}
	if _, ok := expr.Hints.GetBool(traceql.HintSample, allowUnsafeHints); ok {
		return true
	}
	if _, ok := expr.Hints.GetFloat(traceql.HintSample, allowUnsafeHints); ok {
		return true
	}
	return false
}

// marshalRowGroupCacheValue serializes a per-row-group partial response for caching.
// We reuse tempopb.QueryRangeResponse as the wire format — it already carries the
// series + inspected bytes/spans we need. Status/ShardStatus fields are unset on
// cache values and are not consulted on read.
func marshalRowGroupCacheValue(resp *tempopb.QueryRangeResponse) ([]byte, error) {
	return proto.Marshal(resp)
}

func unmarshalRowGroupCacheValue(b []byte) (*tempopb.QueryRangeResponse, error) {
	out := &tempopb.QueryRangeResponse{}
	if err := proto.Unmarshal(b, out); err != nil {
		return nil, err
	}
	return out, nil
}

// rowGroupCacheStore writes one value to the cache, observing the value-size
// histogram and respecting MaxItemSize.
func rowGroupCacheStore(ctx context.Context, c cache.Cache, key string, value []byte) {
	if c == nil || key == "" || len(value) == 0 {
		return
	}
	if max := c.MaxItemSize(); max > 0 && len(value) > max {
		return
	}
	metricRowGroupCacheValueBytes.Observe(float64(len(value)))
	c.Store(ctx, []string{key}, [][]byte{value})
}
