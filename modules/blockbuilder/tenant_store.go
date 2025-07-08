package blockbuilder

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/blockbuilder/util"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/dataquality"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var metricBlockBuilderFlushedBlocks = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "flushed_blocks",
	}, []string{"tenant"},
)

const (
	reasonTraceTooLarge = "trace_too_large"
)

type tenantStore struct {
	tenantID      string
	idGenerator   util.IDGenerator
	cfg           BlockConfig
	startTime     time.Time
	cycleDuration time.Duration
	slackDuration time.Duration
	logger        log.Logger
	overrides     Overrides
	enc           encoding.VersionedEncoding
	wal           *wal.WAL

	liveTraces *livetraces.LiveTraces[[]byte]
}

func newTenantStore(tenantID string, partitionID, startOffset uint64, startTime time.Time, cycleDuration, slackDuration time.Duration, cfg BlockConfig, logger log.Logger, wal *wal.WAL, enc encoding.VersionedEncoding, o Overrides) (*tenantStore, error) {
	s := &tenantStore{
		tenantID:      tenantID,
		idGenerator:   util.NewDeterministicIDGenerator(tenantID, partitionID, startOffset),
		startTime:     startTime,
		cycleDuration: cycleDuration,
		slackDuration: slackDuration,
		cfg:           cfg,
		logger:        logger,
		overrides:     o,
		wal:           wal,
		enc:           enc,
		liveTraces:    livetraces.New(func(b []byte) uint64 { return uint64(len(b)) }, 0, 0), // passing 0s for max idle and live time b/c block builder doesn't cut idle traces
	}

	return s, nil
}

func (s *tenantStore) AppendTrace(traceID []byte, tr []byte, ts time.Time) error {
	maxSz := s.overrides.MaxBytesPerTrace(s.tenantID)

	if !s.liveTraces.PushWithTimestampAndLimits(ts, traceID, tr, 0, uint64(maxSz)) {
		// Record dropped spans due to trace too large
		// We have to unmarhal to count the number of spans.
		// TODO - There might be a better way
		t := &tempopb.Trace{}
		if err := t.Unmarshal(tr); err != nil {
			return err
		}
		count := 0
		for _, b := range t.ResourceSpans {
			for _, ss := range b.ScopeSpans {
				count += len(ss.Spans)
			}
		}
		overrides.RecordDiscardedSpans(count, reasonTraceTooLarge, s.tenantID)
	}

	return nil
}

func (s *tenantStore) Flush(ctx context.Context, r tempodb.Reader, w tempodb.Writer, c tempodb.Compactor) error {
	ctx, span := tracer.Start(ctx, "tenantStore.Flush", trace.WithAttributes(attribute.String("tenant", s.tenantID)))
	defer span.End()

	if s.liveTraces.Len() == 0 {
		span.AddEvent("no live_traces to flush")
		// This can happen if the tenant instance was created but
		// no live traces were successfully pushed. i.e. all exceeded max trace size.
		return nil
	}

	blockID, existingBlocksToBeCompacted, err := s.determineBlockIDs(ctx, r)
	if err != nil {
		return err
	}

	// TODO - Check if the existing block can be reused and exit early

	if len(existingBlocksToBeCompacted) > 0 {
		for _, blockID := range existingBlocksToBeCompacted {
			level.Warn(s.logger).Log(
				"msg", "Marking existing block compacted",
				"tenant", s.tenantID,
				"blockid", blockID,
			)
			span.AddEvent("marking existing block compacted", trace.WithAttributes(attribute.String("block_id", blockID.String())))
			if err := c.MarkBlockCompacted(s.tenantID, blockID); err != nil {
				return err
			}
		}
	}

	// Initial meta for creating the block
	meta := backend.NewBlockMeta(s.tenantID, (uuid.UUID)(blockID), s.enc.Version(), backend.EncNone, "")
	meta.DedicatedColumns = s.overrides.DedicatedColumns(s.tenantID)
	meta.ReplicationFactor = 1
	meta.TotalObjects = int64(s.liveTraces.Len())

	var (
		st     = time.Now()
		l      = s.wal.LocalBackend()
		reader = backend.NewReader(l)
		writer = backend.NewWriter(l)
		iter   = newLiveTracesIter(s.liveTraces)
	)

	level.Info(s.logger).Log(
		"msg", "Flushing block",
		"tenant", s.tenantID,
		"blockid", meta.BlockID,
		"meta", meta,
	)

	newMeta, err := s.enc.CreateBlock(ctx, &s.cfg.BlockCfg, meta, iter, reader, writer)
	if err != nil {
		return err
	}

	// Update meta timestamps which couldn't be known until we unmarshaled
	// all of the traces.
	start, end := iter.MinMaxTimestamps()
	newMeta.StartTime, newMeta.EndTime = s.adjustTimeRangeForSlack(time.Unix(0, int64(start)), time.Unix(0, int64(end)))

	newBlock, err := s.enc.OpenBlock(newMeta, reader)
	if err != nil {
		return err
	}

	if err := w.WriteBlock(ctx, NewWriteableBlock(newBlock, reader, writer)); err != nil {
		span.RecordError(err)
		return err
	}
	span.AddEvent("wrote block to backend", trace.WithAttributes(attribute.String("block_id", newMeta.BlockID.String())))

	metricBlockBuilderFlushedBlocks.WithLabelValues(s.tenantID).Inc()

	if err := s.wal.LocalBackend().ClearBlock((uuid.UUID)(newMeta.BlockID), s.tenantID); err != nil {
		return err
	}
	span.AddEvent("cleared block from wal", trace.WithAttributes(attribute.String("block_id", newMeta.BlockID.String())))

	level.Info(s.logger).Log(
		"msg", "Flushed block",
		"tenant", s.tenantID,
		"blockid", newMeta.BlockID,
		"elapsed", time.Since(st),
		"meta", newMeta,
	)

	return nil
}

// Adjust the time range based on when the record was added to the partition, factoring in slack and cycle duration.
// Any span with a start or end time outside this range will be constrained to the valid limits.
func (s *tenantStore) adjustTimeRangeForSlack(start, end time.Time) (time.Time, time.Time) {
	startOfRange := s.startTime.Add(-s.slackDuration)
	endOfRange := s.startTime.Add(s.slackDuration + s.cycleDuration)

	warn := false
	if start.Before(startOfRange) {
		warn = true
		start = startOfRange
	}
	// clock skew, missconfiguration or simply data tampering
	if start.After(endOfRange) {
		warn = true
		start = endOfRange
	}
	// this can happen with old spans added to Tempo
	// setting it to start to not jump forward unexpectedly
	if end.Before(start) {
		warn = true
		end = start
	}

	if end.After(endOfRange) {
		warn = true
		end = endOfRange
	}

	if warn {
		dataquality.WarnBlockBuilderOutsideIngestionSlack(s.tenantID)
	}

	return start, end
}

func (s *tenantStore) determineBlockIDs(ctx context.Context, r tempodb.Reader) (nextID backend.UUID, existingBlocksToBeCompacted []backend.UUID, err error) {
	for {
		nextID = s.idGenerator.NewID()

		meta, compactedMeta, err := r.BlockMeta(ctx, s.tenantID, nextID)
		if err != nil {
			return backend.UUID{}, nil, err
		}

		if compactedMeta != nil {
			// This ID is already in use but does not need to be compacted again
			continue
		}

		if meta != nil {
			// This ID is already in use and needs to be compacted
			existingBlocksToBeCompacted = append(existingBlocksToBeCompacted, nextID)
			continue
		}

		// This block is available
		return nextID, existingBlocksToBeCompacted, nil
	}
}
