package bloomgatewayevents

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	// defaultMaxInflightProduceRequests bounds Produce concurrency per
	// broker (ingest.NewWriterClient's own maxInflightProduceRequests
	// parameter). Steady-state churn is ~1 block/s cell-wide (DESIGN.md §
	// Sizing), so there is no real load this needs to be tuned against --
	// no flag is warranted.
	defaultMaxInflightProduceRequests = 4

	// producerEventVersion is the only BloomGatewayEvent envelope version
	// this producer ever emits. It must match
	// modules/bloomgateway/events.go's supportedEventVersion by
	// construction: a consumer that doesn't recognize a version drops the
	// whole event (DESIGN.md § Write path's versioned envelope).
	producerEventVersion uint32 = 1

	// minTraceIDLen and maxTraceIDLen bound a valid trace ID, matching
	// modules/bloomgateway/events.go's validateTraceIDs exactly. That
	// gateway-side check drops a WHOLE chunk if even one ID falls outside
	// this range, so filtering here -- before chunking -- keeps one
	// malformed ID from poisoning every valid ID that would otherwise land
	// in the same chunk.
	minTraceIDLen = 1
	maxTraceIDLen = 16

	resultOK          = "ok"
	resultDropped     = "dropped"
	resultRateLimited = "rate_limited"
)

// Publisher is the producer-side half of the bloom-gateway Kafka write path
// (DESIGN.md § Write path): it turns a block's trace IDs into versioned
// BloomGatewayEvent records and publishes them, bounded by cfg.Kafka's own
// retry budget. Every producer (block-builder, backend-worker) constructs
// its own Publisher.
type Publisher struct {
	cfg           Config
	client        *kgo.Client // nil when disabled
	numPartitions int32
	metrics       *metrics // nil when disabled
	logger        log.Logger

	// limiter enforces the per-tenant publish rate limit (DESIGN.md §
	// Multi-tenant cells). Always non-nil -- New sets it to an unlimited
	// tenantRateLimiter by default, and WithTenantLimits replaces it -- so
	// PublishAdd/PublishDelete never need a nil check before calling
	// limiter.allow.
	limiter *tenantRateLimiter

	// closeOnce makes Close idempotent -- kgo.Client's own Close is not
	// documented as safe to call twice, so this guards it ourselves rather
	// than relying on that.
	closeOnce sync.Once
}

// Option configures optional Publisher behavior not needed by every
// producer.
type Option func(*Publisher)

// WithTenantLimits enables the per-tenant publish rate limit (DESIGN.md §
// Multi-tenant cells's producer-side guardrail): limits returns a tenant's
// configured publishes/sec (0 = unlimited), typically a method value off
// modules/overrides.Interface. This package can't import modules/overrides
// directly (pkg/* must not depend on modules/*), so the getter is handed in
// instead.
func WithTenantLimits(limits TenantLimits) Option {
	return func(p *Publisher) {
		p.limiter = newTenantRateLimiter(limits)
	}
}

// New always returns a usable *Publisher. When cfg.Enabled is false, the
// returned Publisher's client stays nil and every method becomes a no-op,
// so callers never need to branch on cfg.Enabled themselves.
//
// cfg.Validate() is called unconditionally -- nothing else validates this
// config -- before anything is constructed, so a rejected config leaves no
// partially-registered metrics or client behind.
func New(cfg Config, logger log.Logger, reg prometheus.Registerer, opts ...Option) (*Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Publisher{
		cfg:     cfg,
		logger:  logger,
		limiter: newTenantRateLimiter(nil), // unlimited until/unless WithTenantLimits below replaces it
	}
	for _, opt := range opts {
		opt(p)
	}
	if !cfg.Enabled {
		return p, nil
	}

	// Metrics are registered only on the enabled path: a disabled Publisher
	// must leave zero footprint on reg, since a process (or, as this
	// package's own tests do, a single test binary) can construct more than
	// one disabled Publisher against the same prometheus.DefaultRegisterer --
	// unconditional registration would panic on the second one via
	// promauto's MustRegister. Every method that touches p.metrics already
	// returns before reaching it on the disabled path (PublishAdd/
	// PublishDelete via Enabled(), Close via p.client == nil), so leaving it
	// nil here is safe.
	p.metrics = newMetrics(reg)

	client, err := ingest.NewWriterClient(cfg.Kafka, defaultMaxInflightProduceRequests, logger, reg)
	if err != nil {
		return nil, err
	}

	p.client = client
	// K: both this producer and the gateway's consumer default to the same
	// value by construction (config.go's
	// defaultAutoCreateTopicDefaultPartitions), and resizing is a rare,
	// operator-driven event (DESIGN.md § Sizing) -- so a static read at
	// construction time, not a per-publish lookup, is enough.
	p.numPartitions = int32(cfg.Kafka.AutoCreateTopicDefaultPartitions)
	return p, nil
}

// Enabled reports whether publishing is active. Callers may use it to skip
// work that only matters when publishing is on (e.g. capturing trace IDs in
// the first place) instead of paying that cost just to hand it to a no-op.
func (p *Publisher) Enabled() bool {
	return p.client != nil
}

// PublishAdd publishes traceIDs as one or more AddChunk events for a single,
// now-durable block. It never returns an error: any failure increments
// publishes_total{result="dropped"} once (per block, not per chunk) and
// returns (DESIGN.md § Publish policy -- bounded retry, then drop and count;
// a dropped Add is healed later by reconciliation, never a correctness
// loss).
func (p *Publisher) PublishAdd(ctx context.Context, blockID backend.UUID, tenantID string, start, end time.Time, traceIDs [][]byte) {
	if !p.Enabled() {
		return
	}

	// Checked before any chunking work, and all-or-nothing: a rate-limited
	// Add must publish NO chunks for this block. A partial chunk set can
	// never complete on the consumer side (events.go's completion tracking
	// needs every chunk index 0..chunkCount-1 to arrive) and is strictly
	// worse than a clean drop, since reconciliation only ever heals a
	// wholly-missing block, never a stuck partial one. No per-event log
	// here -- a rate-limited hot loop must not log-spam; the counter below
	// is the signal.
	if !p.limiter.allow(tenantID) {
		p.metrics.publishesTotal.WithLabelValues(resultRateLimited).Inc()
		return
	}

	valid, invalid := filterValidTraceIDs(traceIDs)
	if invalid > 0 {
		p.metrics.invalidTraceIDsTotal.Add(float64(invalid))
	}

	deduped := dedupeTraceIDs(valid)
	if len(deduped) == 0 {
		return
	}

	chunks := chunkTraceIDs(deduped, p.cfg.ChunkSize)
	// chunkCount is final and identical across every chunk of this block --
	// the consumer's completion tracking depends on it (events.go's
	// recordChunk groups by (block, chunkCount)).
	chunkCount := uint32(len(chunks))
	partition := partitionForBlock(blockID, p.numPartitions)

	records := make([]*kgo.Record, len(chunks))
	for i, chunk := range chunks {
		value, err := (&tempopb.BloomGatewayEvent{
			Version: producerEventVersion,
			Type:    tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK,
			AddChunk: &tempopb.BloomGatewayAddChunk{
				BlockId:           blockID.String(),
				TenantId:          tenantID,
				StartTimeUnixNano: start.UnixNano(),
				EndTimeUnixNano:   end.UnixNano(),
				ChunkIndex:        uint32(i),
				ChunkCount:        chunkCount,
				TraceIds:          chunk,
			},
		}).Marshal()
		if err != nil {
			// Every field above is a primitive, string, or [][]byte --
			// Marshal cannot realistically fail. If it somehow does, treat
			// it exactly like any other publish failure (all-or-nothing per
			// block) rather than panicking or silently dropping only this
			// chunk.
			p.dropped(blockID, len(chunks), err)
			return
		}
		// The client is configured with ManualPartitioner
		// (ingest.NewWriterClient) -- a Key alone would route nothing, so
		// Partition must be set explicitly on every record.
		records[i] = &kgo.Record{Value: value, Partition: partition}
	}

	p.produce(ctx, blockID, records)
}

// PublishDelete publishes a Delete event for blockID. Same no-error,
// drop-and-count semantics as PublishAdd, and routes to the same partition a
// block's Adds used (partitionForBlock is a pure function of blockID).
// tenantID is used only for rate limiting (DESIGN.md § Multi-tenant cells
// limits Add/Delete alike) -- BloomGatewayDelete carries no tenant field on
// the wire, so it is never sent.
func (p *Publisher) PublishDelete(ctx context.Context, blockID backend.UUID, tenantID string) {
	if !p.Enabled() {
		return
	}

	if !p.limiter.allow(tenantID) {
		p.metrics.publishesTotal.WithLabelValues(resultRateLimited).Inc()
		return
	}

	value, err := (&tempopb.BloomGatewayEvent{
		Version: producerEventVersion,
		Type:    tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_DELETE,
		Delete:  &tempopb.BloomGatewayDelete{BlockId: blockID.String()},
	}).Marshal()
	if err != nil {
		p.dropped(blockID, 1, err)
		return
	}

	partition := partitionForBlock(blockID, p.numPartitions)
	p.produce(ctx, blockID, []*kgo.Record{{Value: value, Partition: partition}})
}

// Close idempotently flushes and closes the underlying Kafka client. Safe to
// call when disabled (nothing to close) and safe to call more than once.
func (p *Publisher) Close() {
	if p.client == nil {
		return
	}
	p.closeOnce.Do(func() {
		// Every publish already goes through the synchronous ProduceSync,
		// so nothing should be lingering in the client's buffers by the
		// time Close is called -- Flush is defensive, not load-bearing.
		if err := p.client.Flush(context.Background()); err != nil {
			level.Warn(p.logger).Log("msg", "bloomgatewayevents: flush before close failed", "err", err)
		}
		p.client.Close()
	})
}

// produce sends records for one block's event and records the outcome
// exactly once, regardless of how many records were sent: DESIGN.md's
// producer-side "ok"/"dropped" counters are per publish call (i.e. per
// block), never per record, so a partially-failed multi-chunk Add is one
// dropped block, not some ok and some dropped.
func (p *Publisher) produce(ctx context.Context, blockID backend.UUID, records []*kgo.Record) {
	ctx, cancel := p.withDeadlineCeiling(ctx)
	defer cancel()

	start := time.Now()
	results := p.client.ProduceSync(ctx, records...)
	p.metrics.publishDurationSeconds.Observe(time.Since(start).Seconds())

	if err := results.FirstErr(); err != nil {
		p.dropped(blockID, len(records), err)
		return
	}

	p.metrics.publishesTotal.WithLabelValues(resultOK).Inc()
}

// dropped records one dropped publish and logs a single bounded warning --
// block ID and chunk count only, NEVER trace ID bytes (arbitrary,
// tenant-controlled content that must never end up in logs).
func (p *Publisher) dropped(blockID backend.UUID, numChunks int, err error) {
	p.metrics.publishesTotal.WithLabelValues(resultDropped).Inc()
	level.Warn(p.logger).Log("msg", "bloomgatewayevents: dropping bloom-gateway event publish", "block", blockID.String(), "chunks", numChunks, "err", err)
}

// withDeadlineCeiling returns ctx as-is if it already carries a deadline --
// RecordDeliveryTimeout (set from cfg.Kafka.WriteTimeout by
// ingest.NewWriterClient) is already the primary bound on how long a
// produce can take. If ctx has no deadline, this applies a defensive
// ceiling of twice that timeout, purely so a caller that forgets to bound
// its own context can never hang indefinitely. The returned cancel must be
// called once the produce is done either way.
func (p *Publisher) withDeadlineCeiling(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 2*p.cfg.Kafka.WriteTimeout)
}

// filterValidTraceIDs returns the subset of ids whose length is in
// [minTraceIDLen, maxTraceIDLen], preserving order, plus a count of how many
// were dropped. The dropped IDs themselves are never returned -- only ever
// a count -- so a malformed input can't leak trace ID bytes through this
// path.
func filterValidTraceIDs(ids [][]byte) (valid [][]byte, invalid int) {
	valid = make([][]byte, 0, len(ids))
	for _, id := range ids {
		if len(id) < minTraceIDLen || len(id) > maxTraceIDLen {
			invalid++
			continue
		}
		valid = append(valid, id)
	}
	return valid, invalid
}
