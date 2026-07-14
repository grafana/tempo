// This file implements the Kafka consumer half of the write path
// (DESIGN.md § Write path "Consumers", § Event processing, § Backpressure
// and memory pressure). worker.go implements the other half: the
// fixed-size pool that drains Records() into events.go's Applier.
//
// Split of responsibility, restated because it drives every design choice
// below: Consumer owns everything Kafka-shaped (the bare kgo client,
// manual all-partitions assignment, the byte-bounded admission queue, and
// the narrow PositionRewinder surface WP18's reconstruction rewind needs).
// WorkerPool (worker.go) owns everything apply-shaped (decoding, calling
// Applier, and the per-partition contiguous "applied" watermark that is
// the snapshot's actual resume authority). Consumer's own fetch position
// (CurrentFetchOffsets) is deliberately a DIFFERENT, generally larger,
// number than WorkerPool's AppliedOffsets: records can sit in the queue,
// or be mid-apply on a worker, after being fetched but before being
// applied.
package bloomgateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/semaphore"

	"github.com/grafana/tempo/pkg/ingest"
)

// recordsChannelBuffer is the output channel's own item-count capacity. The
// real backpressure bound is Consumer.sem (byte-weighted, sized at
// queueMaxBytes) -- this buffer only needs to be generous enough that the
// Go channel's slot count never becomes an accidental SECONDARY,
// item-count bottleneck below the intended byte bound.
const recordsChannelBuffer = 4096

// Record is one not-yet-applied Kafka record, handed from Consumer's fetch
// loop to WorkerPool (worker.go). Value is the raw, still-encoded
// BloomGatewayEvent payload -- decoding happens in the worker, off the
// single fetch-loop goroutine, so a slow decode/apply never itself blocks
// PollFetches (only Consumer's byte semaphore does that, deliberately, via
// release below).
type Record struct {
	Partition int32
	Offset    int64
	Value     []byte

	// release frees this record's share of Consumer's byte-bounded queue
	// (golang.org/x/sync/semaphore.Weighted). Nil only for zero-value/
	// test-constructed Records; the worker calls it exactly once,
	// immediately after receiving the record and before doing any
	// decode/apply work, matching DESIGN.md's "worker pool ... drains a
	// bounded in-memory queue" -- a record leaves the queue (and releases
	// its budget) the instant a worker picks it up, not when that worker
	// finishes processing it. Unexported: purely an internal wiring
	// detail between consumer.go and worker.go, never part of any other
	// WP's contract.
	release func()
}

// PositionRewinder is the narrow surface WP18's reconstruction rewind
// needs (DESIGN.md § Reconstruction step 2: "note the consumer's current
// position ... and rewind it to (oldest index build time - margin)") --
// deliberately not the whole *Consumer, so WP18's tests can inject a fake
// implementation instead of standing up a real kgo/kadm client (AMENDMENT
// A6).
type PositionRewinder interface {
	// CurrentFetchOffsets returns, per partition, the next offset this
	// consumer will fetch -- i.e. one past the last record actually
	// delivered out of Records() for that partition. A partition absent
	// from the map has delivered nothing yet this session (true cold
	// start, or simply not reached yet).
	CurrentFetchOffsets() map[int32]int64

	// OffsetsAtOrBefore resolves wall-clock time t to the latest
	// per-partition offset at or before it, via kadm's timestamp-based
	// ListOffsets. Spiked directly against kfake (TestConsumer_
	// OffsetsAtOrBefore): kfake implements real timestamp-ordered binary
	// search server-side (kfake/02_list_offsets.go's findBatchMeta), not
	// just the -1/-2 sentinel offsets, so the production kadm path is
	// exercised as-is here -- no fake/injected rewinder was needed for
	// this method specifically (AMENDMENT A6 turned out to be a
	// non-issue; see the WP14 final report for what was spiked).
	OffsetsAtOrBefore(ctx context.Context, t time.Time) (map[int32]int64, error)

	// Rewind reassigns the given partitions to resume from the given
	// offsets, re-seeking any already-assigned partition -- the same
	// mechanism modules/blockbuilder relies on (consumePartition's "we
	// always rewind ... by reassigning the partition to the client").
	Rewind(offsets map[int32]int64) error
}

var _ PositionRewinder = (*Consumer)(nil)

// Consumer is a bare (no consumer-group) Kafka reader, assigned ALL of the
// gateway's topic partitions manually and simultaneously, exactly once
// (DESIGN.md § Write path "Consumers": "every gateway instance is an
// independent consumer of all K partitions"). It NEVER reads its own
// consumer-group commits back to decide where to resume -- resume offsets
// always come from the caller (local snapshot / reconstruction
// bookkeeping, via Start's offsets parameter); the per-instance group id
// it commits under exists solely to power external lag-observability
// tooling (kadm), per §0's Kafka-plumbing decision.
type Consumer struct {
	cfg        ingest.KafkaConfig
	instanceID string
	logger     log.Logger

	client *kgo.Client
	adm    *kadm.Client
	sem    *semaphore.Weighted

	// metaAdm is a SEPARATE kadm.Client, backed by its own dedicated
	// *kgo.Client, used only for the read-only metadata/offset lookups that
	// go through kgo's RequestCachedMetadata path (discoverPartitions'
	// ListTopics, OffsetsAtOrBefore's ListOffsetsAfterMilli/
	// ListStartOffsets) -- see its doc comment on OffsetsAtOrBefore for why
	// this exists: a confirmed, reproducible data race (found by WP20's own
	// integration testing, not by this file's own unit tests, which never
	// exercise a long-lived Consumer this way) between kgo's own
	// autonomous background metadata-refresh loop and RequestCachedMetadata
	// reads issued on the SAME actively-fetching *kgo.Client. commitOffsets
	// (a write, not on this call path) is deliberately left on the
	// original shared adm/client -- only the two confirmed-at-risk read
	// paths move.
	metaAdm *kadm.Client

	queueMaxBytes int64
	records       chan Record

	// mu guards fetchOffsets only.
	mu           sync.Mutex
	fetchOffsets map[int32]int64

	wg        sync.WaitGroup
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// NewConsumer builds a Consumer against cfg's topic. instanceID (an added
// parameter beyond the plan's original sketch -- see the WP14 final
// report) is used only to derive the per-instance consumer-group id for
// lag-observability commits (cfg.GetConsumerGroup), never for resume.
func NewConsumer(cfg ingest.KafkaConfig, instanceID string, queueMaxBytes int64, logger log.Logger, reg prometheus.Registerer) (*Consumer, error) {
	kpromMetrics := ingest.NewReaderClientMetrics("bloom-gateway", reg)
	client, err := ingest.NewReaderClient(cfg, kpromMetrics, logger)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: new kafka reader client: %w", err)
	}

	// A second, dedicated client for read-only metadata/offset lookups --
	// see metaAdm's own field doc comment. A distinct metrics "component"
	// label ("bloom-gateway-meta" vs "bloom-gateway") avoids a duplicate-
	// registration panic against the same reg (NewReaderClientMetrics
	// registers a fresh collector set per call).
	metaKpromMetrics := ingest.NewReaderClientMetrics("bloom-gateway-meta", reg)
	metaClient, err := ingest.NewReaderClient(cfg, metaKpromMetrics, logger)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: new kafka metadata client: %w", err)
	}

	return &Consumer{
		cfg:           cfg,
		instanceID:    instanceID,
		logger:        logger,
		metaAdm:       kadm.NewClient(metaClient),
		client:        client,
		adm:           kadm.NewClient(client),
		sem:           semaphore.NewWeighted(queueMaxBytes),
		queueMaxBytes: queueMaxBytes,
		records:       make(chan Record, recordsChannelBuffer),
		fetchOffsets:  make(map[int32]int64),
	}, nil
}

// Start assigns every partition of cfg.Topic to this client in one
// AddConsumePartitions call (DESIGN.md § Write path "Consumers": "manual
// partition assignment"), then begins the background fetch loop. offsets
// gives the per-partition resume point for every partition the caller
// already has bookkeeping for (local snapshot / reconstruction); any
// partition NOT present in offsets -- including one this cell has never
// seen before, e.g. after a topic resize -- starts at AtStart() (true cold
// start, §0 Kafka-plumbing decision). offsets is NEVER derived from this
// consumer's own broker-side commits (§ Consumers: "broker-side commits
// exist only to power lag metrics").
func (c *Consumer) Start(ctx context.Context, offsets map[int32]int64) error {
	if err := ingest.WaitForKafkaBroker(ctx, c.client, c.logger); err != nil {
		return fmt.Errorf("bloomgateway: wait for kafka broker: %w", err)
	}

	partitions, err := c.discoverPartitions(ctx)
	if err != nil {
		return fmt.Errorf("bloomgateway: discover partitions for topic %q: %w", c.cfg.Topic, err)
	}

	assign := make(map[int32]kgo.Offset, len(partitions))
	c.mu.Lock()
	for _, p := range partitions {
		if at, ok := offsets[p]; ok {
			assign[p] = kgo.NewOffset().At(at)
			c.fetchOffsets[p] = at
		} else {
			assign[p] = kgo.NewOffset().AtStart()
			// fetchOffsets deliberately left unset: CurrentFetchOffsets
			// only reports a partition once its actual starting offset is
			// known from a delivered record (see enqueue) -- the true
			// log-start offset under retention may be above 0, so
			// guessing a number here could under-report progress.
		}
	}
	c.mu.Unlock()

	level.Info(c.logger).Log("msg", "bloomgateway: starting kafka consumer", "topic", c.cfg.Topic, "partitions", len(partitions), "resume_offsets_known", len(offsets))
	c.client.AddConsumePartitions(map[string]map[int32]kgo.Offset{c.cfg.Topic: assign})

	runCtx, cancel := context.WithCancel(ctx) //nolint:gosec // G118 false positive: cancel is stored on c.cancel and invoked in Stop.
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run(runCtx)
	}()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.commitLoop(runCtx)
	}()

	return nil
}

// discoverPartitions returns every partition currently configured for
// cfg.Topic. Manual assignment (unlike a consumer group) has no built-in
// notion of "all partitions", so this queries broker metadata once at
// Start -- letting K change (a resize) be picked up on the next restart
// without any code change (DESIGN.md § Write path: "resize is
// correctness-neutral"). Uses metaAdm (not adm): ListTopics goes through
// the same RequestCachedMetadata path as OffsetsAtOrBefore -- see that
// method's own doc comment for why this must not share a client with
// active fetching.
func (c *Consumer) discoverPartitions(ctx context.Context) ([]int32, error) {
	details, err := c.metaAdm.ListTopics(ctx, c.cfg.Topic)
	if err != nil {
		return nil, err
	}
	detail, ok := details[c.cfg.Topic]
	if !ok {
		return nil, fmt.Errorf("topic %q not found", c.cfg.Topic)
	}
	if detail.Err != nil {
		return nil, detail.Err
	}
	return detail.Partitions.Numbers(), nil
}

// Records is the channel WorkerPool ranges over (worker.go).
func (c *Consumer) Records() <-chan Record {
	return c.records
}

// run is the single fetch-loop goroutine: one PollFetches loop across
// every assigned partition (DESIGN.md § Concurrency: "the partition
// consumer is single-threaded per partition"). It never drops a record
// silently -- enqueue blocks on the byte semaphore instead (§ Backpressure
// and memory pressure: "pauses fetching and lag grows").
func (c *Consumer) run(ctx context.Context) {
	defer close(c.records)
	for ctx.Err() == nil {
		fetches := c.client.PollFetches(ctx)
		for _, fe := range fetches.Errors() {
			if errors.Is(fe.Err, context.Canceled) || errors.Is(fe.Err, context.DeadlineExceeded) {
				continue
			}
			level.Error(c.logger).Log("msg", "bloomgateway: kafka fetch error", "topic", fe.Topic, "partition", fe.Partition, "err", fe.Err)
			ingest.HandleKafkaError(fe.Err, c.client.ForceMetadataRefresh)
		}

		stopped := false
		fetches.EachPartition(func(part kgo.FetchTopicPartition) {
			if stopped {
				return
			}
			for _, rec := range part.Records {
				if !c.enqueue(ctx, rec) {
					stopped = true
					return
				}
			}
		})
		if stopped {
			return
		}
	}
}

// enqueue admits one fetched record into the byte-bounded queue, blocking
// the fetch loop on c.sem.Acquire when the queue is full -- the
// backpressure mechanism itself (never a silent drop, §0 Kafka-plumbing
// decision). It returns false only when ctx is done, signaling run to stop.
func (c *Consumer) enqueue(ctx context.Context, rec *kgo.Record) bool {
	sz := int64(len(rec.Value))
	acquireSz := sz
	if acquireSz > c.queueMaxBytes {
		// A single record heavier than the whole configured budget would
		// otherwise block Acquire forever: golang.org/x/sync/semaphore
		// never admits a request larger than the semaphore's total size
		// (semaphore.go's Acquire: "if n > s.size ... <-done"). Clamp so
		// it is instead admitted once the queue fully drains, and log so
		// an operator can see the misconfiguration/producer-size mismatch
		// rather than silently stalling.
		level.Warn(c.logger).Log("msg", "bloomgateway: record larger than queue byte budget; admitting once queue drains", "partition", rec.Partition, "offset", rec.Offset, "record_bytes", sz, "queue_max_bytes", c.queueMaxBytes)
		acquireSz = c.queueMaxBytes
	}
	if acquireSz < 0 {
		acquireSz = 0
	}

	if err := c.sem.Acquire(ctx, acquireSz); err != nil {
		return false
	}

	var once sync.Once
	release := func() { once.Do(func() { c.sem.Release(acquireSz) }) }

	c.mu.Lock()
	c.fetchOffsets[rec.Partition] = rec.Offset + 1
	c.mu.Unlock()

	select {
	case c.records <- Record{Partition: rec.Partition, Offset: rec.Offset, Value: rec.Value, release: release}:
		return true
	case <-ctx.Done():
		release()
		return false
	}
}

// CurrentFetchOffsets implements PositionRewinder.
func (c *Consumer) CurrentFetchOffsets() map[int32]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[int32]int64, len(c.fetchOffsets))
	for p, o := range c.fetchOffsets {
		out[p] = o
	}
	return out
}

// OffsetsAtOrBefore implements PositionRewinder.
//
// Kafka's ListOffsets protocol only exposes "first offset AT OR AFTER a
// timestamp" (kmsg.ListOffsetsRequest; see kadm.Client.ListOffsetsAfterMilli
// and kfake/02_list_offsets.go's default case). "At or before" is derived
// here by subtracting one from that result, floored at the partition's
// start offset (never negative). Rounding down this way is deliberately
// conservative: DESIGN.md's own reconstruction rewind explicitly tolerates
// over-replay ("over-replay is free: application is idempotent") but never
// under-replay, so when the exact boundary is ambiguous this resolves to an
// offset at least as old as necessary, never newer.
//
// If a partition has no offset at or after t (every record already
// predates t, e.g. a quiet period or t computed slightly into the future),
// kadm's ListOffsetsAfterMilli documents that it returns the partition's
// current end offset instead -- so the "-1" here correctly lands on the
// last available record, not on some offset past it.
//
// Deliberately calls c.metaAdm, NOT c.adm: this method's call chain
// (ListOffsetsAfterMilli -> listOffsets -> ListTopics -> Metadata ->
// kgo.Client.RequestCachedMetadata) reads the client's cached per-topic
// partition metadata, which races -- confirmed directly, via go test -race,
// by this WP's own full-lifecycle integration tests (never exercised by
// consumer_test.go's own narrower unit tests, which don't run a Consumer
// this long-lived/concurrently) -- against that SAME client's own
// autonomous background metadata-refresh loop
// (kgo/metadata.go:updateMetadataLoop -> fetchTopicMetadata, which sorts a
// slice that can share backing memory with what RequestCachedMetadata
// concurrently reads). c.client (used for active fetching, via
// AddConsumePartitions) runs that refresh loop continuously and busily;
// c.metaAdm's own dedicated, non-fetching client only runs it passively,
// which is what actually avoids the race in practice, not merely relocates
// it. Root-caused to vendor/github.com/twmb/franz-go, not this repo's own
// code; reported prominently in this WP's final report rather than
// patching the vendored dependency directly.
func (c *Consumer) OffsetsAtOrBefore(ctx context.Context, t time.Time) (map[int32]int64, error) {
	afterListed, err := c.metaAdm.ListOffsetsAfterMilli(ctx, t.UnixMilli(), c.cfg.Topic)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: list offsets after %s: %w", t, err)
	}
	if err := afterListed.Error(); err != nil {
		return nil, fmt.Errorf("bloomgateway: list offsets after %s: %w", t, err)
	}

	startListed, err := c.metaAdm.ListStartOffsets(ctx, c.cfg.Topic)
	if err != nil {
		return nil, fmt.Errorf("bloomgateway: list start offsets: %w", err)
	}

	result := make(map[int32]int64, len(afterListed[c.cfg.Topic]))
	for partition, lo := range afterListed[c.cfg.Topic] {
		offset := lo.Offset - 1
		if start, ok := startListed.Lookup(c.cfg.Topic, partition); ok && offset < start.Offset {
			offset = start.Offset
		}
		if offset < 0 {
			offset = 0
		}
		result[partition] = offset
	}
	return result, nil
}

// Rewind implements PositionRewinder. Re-seeks the given partitions to the
// given offsets via a RemoveConsumePartitions followed by
// AddConsumePartitions, mirroring modules/blockbuilder's own comment on the
// same pattern ("we always rewind the partition's offset ... by
// reassigning the partition to the client"). The Remove step is load-
// bearing, not cosmetic: kgo's direct consumer only re-seeks a partition
// during its next metadata-triggered assignment pass if that partition
// looks NEW (consumer.go's findNewAssignments) -- calling
// AddConsumePartitions alone on a partition that is already being actively
// fetched updates only the client's internal "requested offset" bookkeeping
// and does not disturb its live cursor, so a bare Add is silently a no-op
// for anything already assigned. Remove first invalidates the cursor
// (assignInvalidateMatching), making the partition look new again so the
// subsequent Add is actually honored.
func (c *Consumer) Rewind(offsets map[int32]int64) error {
	if len(offsets) == 0 {
		return nil
	}
	partitions := make([]int32, 0, len(offsets))
	assign := make(map[int32]kgo.Offset, len(offsets))
	c.mu.Lock()
	for p, at := range offsets {
		partitions = append(partitions, p)
		assign[p] = kgo.NewOffset().At(at)
		c.fetchOffsets[p] = at
	}
	c.mu.Unlock()
	c.client.RemoveConsumePartitions(map[string][]int32{c.cfg.Topic: partitions})
	c.client.AddConsumePartitions(map[string]map[int32]kgo.Offset{c.cfg.Topic: assign})
	return nil
}

// Close stops the fetch and commit loops and closes the underlying kgo
// client. Idempotent.
func (c *Consumer) Close() error {
	c.closeOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		c.wg.Wait()
		c.client.Close()
		c.metaAdm.Close() // closes metaAdm's own dedicated underlying *kgo.Client
	})
	return nil
}

// commitLoop periodically commits this consumer's own fetch position under
// its per-instance consumer-group id, purely for external lag-observability
// tooling (DESIGN.md § Write path "Consumers": "per-instance group id for
// offset observability"; § Consumers: "broker-side commits exist only to
// power lag metrics"). Nothing in this package ever reads these commits
// back -- resume always comes from the caller (Start's offsets parameter).
// Reuses cfg.ConsumerGroupOffsetCommitInterval (already a first-class
// ingest.KafkaConfig field, matching livestore's PartitionReader) rather
// than adding a new config knob; an interval of 0 disables the loop.
func (c *Consumer) commitLoop(ctx context.Context) {
	interval := c.cfg.ConsumerGroupOffsetCommitInterval
	if interval <= 0 {
		return
	}
	group := c.cfg.GetConsumerGroup(c.instanceID, 0)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.commitOffsets(ctx, group)
		}
	}
}

// commitOffsets commits one partition per CommitAllOffsets call,
// deliberately not a single batched multi-partition request: the in-repo
// kfake test harness (pkg/ingest/testkafka.Cluster's offsetCommit) hard-
// asserts exactly one partition per OffsetCommit request, and committing
// per-partition is valid against a real broker too (just N small requests
// instead of one batched request -- fine for this low-frequency,
// best-effort, observability-only path). A failed commit is logged and
// retried next tick; it is never propagated as a fetch-loop error, since
// DESIGN.md never depends on it succeeding.
func (c *Consumer) commitOffsets(ctx context.Context, group string) {
	for partition, at := range c.CurrentFetchOffsets() {
		offsets := make(kadm.Offsets, 1)
		offsets.Add(kadm.Offset{Topic: c.cfg.Topic, Partition: partition, At: at})
		if err := c.adm.CommitAllOffsets(ctx, group, offsets); err != nil {
			level.Warn(c.logger).Log("msg", "bloomgateway: failed to commit offset for lag observability; will retry next tick", "group", group, "partition", partition, "err", err)
		}
	}
}
