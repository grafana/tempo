package blockbuilder

import (
	"bytes"
	"context"
	"slices"
	"sync"
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
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	flushConcurrency    = 4
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
		liveTraces:    livetraces.New[[]byte](func(b []byte) uint64 { return uint64(len(b)) }),
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

func (s *tenantStore) Flush(ctx context.Context, store tempodb.Writer) error {
	if s.liveTraces.Len() == 0 {
		// This can happen if the tenant instance was created but
		// no live traces were successfully pushed. i.e. all exceeded max trace size.
		return nil
	}

	// Initial meta for creating the block
	meta := backend.NewBlockMeta(s.tenantID, uuid.UUID(s.idGenerator.NewID()), s.enc.Version(), backend.EncNone, "")
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

	if err := store.WriteBlock(ctx, NewWriteableBlock(newBlock, reader, writer)); err != nil {
		return err
	}

	metricBlockBuilderFlushedBlocks.WithLabelValues(s.tenantID).Inc()

	if err := s.wal.LocalBackend().ClearBlock((uuid.UUID)(newMeta.BlockID), s.tenantID); err != nil {
		return err
	}

	level.Info(s.logger).Log(
		"msg", "Flushed block",
		"tenant", s.tenantID,
		"blockid", newMeta.BlockID,
		"elapsed", time.Since(st),
		"meta", newMeta,
	)

	return nil
}

func (s *tenantStore) adjustTimeRangeForSlack(start, end time.Time) (time.Time, time.Time) {
	startOfRange := s.startTime.Add(-s.slackDuration)
	endOfRange := s.startTime.Add(s.slackDuration + s.cycleDuration)

	warn := false
	if start.Before(startOfRange) {
		warn = true
		start = s.startTime
	}
	if end.After(endOfRange) || end.Before(start) {
		warn = true
		end = s.startTime
	}

	if warn {
		dataquality.WarnBlockBuilderOutsideIngestionSlack(s.tenantID)
	}

	return start, end
}

type entry struct {
	id   common.ID
	hash uint64
}

type chEntry struct {
	id  common.ID
	tr  *tempopb.Trace
	err error
}

type liveTracesIter struct {
	mtx        sync.Mutex
	liveTraces *livetraces.LiveTraces[[]byte]
	ch         chan []chEntry
	chBuf      []chEntry
	cancel     func()
	start, end uint64
}

func newLiveTracesIter(liveTraces *livetraces.LiveTraces[[]byte]) *liveTracesIter {
	ctx, cancel := context.WithCancel(context.Background())

	l := &liveTracesIter{
		liveTraces: liveTraces,
		ch:         make(chan []chEntry, 1),
		cancel:     cancel,
	}

	go l.iter(ctx)

	return l
}

func (i *liveTracesIter) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	if len(i.chBuf) == 0 {
		select {
		case entries, ok := <-i.ch:
			if !ok {
				return nil, nil, nil
			}
			i.chBuf = entries
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	// Pop next entry
	if len(i.chBuf) > 0 {
		entry := i.chBuf[0]
		i.chBuf = i.chBuf[1:]
		return entry.id, entry.tr, entry.err
	}

	// Channel is open but buffer is empty?
	return nil, nil, nil
}

func (i *liveTracesIter) iter(ctx context.Context) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	defer close(i.ch)

	// Get the list of all traces sorted by ID
	entries := make([]entry, 0, len(i.liveTraces.Traces))
	for hash, t := range i.liveTraces.Traces {
		entries = append(entries, entry{t.ID, hash})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return bytes.Compare(a.id, b.id)
	})

	// Begin sending to channel in chunks to reduce channel overhead.
	seq := slices.Chunk(entries, 10)
	for entries := range seq {
		output := make([]chEntry, 0, len(entries))

		for _, e := range entries {

			entry := i.liveTraces.Traces[e.hash]

			tr := new(tempopb.Trace)

			for _, b := range entry.Batches {
				// This unmarshal appends the batches onto the existing tempopb.Trace
				// so we don't need to allocate another container temporarily
				err := tr.Unmarshal(b)
				if err != nil {
					i.ch <- []chEntry{{err: err}}
					return
				}
			}

			// Update block timestamp bounds
			for _, b := range tr.ResourceSpans {
				for _, ss := range b.ScopeSpans {
					for _, s := range ss.Spans {
						if i.start == 0 || s.StartTimeUnixNano < i.start {
							i.start = s.StartTimeUnixNano
						}
						if s.EndTimeUnixNano > i.end {
							i.end = s.EndTimeUnixNano
						}
					}
				}
			}

			tempopb.ReuseByteSlices(entry.Batches)
			delete(i.liveTraces.Traces, e.hash)

			output = append(output, chEntry{
				id:  entry.ID,
				tr:  tr,
				err: nil,
			})
		}

		select {
		case i.ch <- output:
		case <-ctx.Done():
			return
		}
	}
}

// MinMaxTimestamps returns the earliest start, and latest end span timestamps,
// which can't be known until all contents are unmarshaled. The iterated must
// be exhausted before this can be accessed.
func (i *liveTracesIter) MinMaxTimestamps() (uint64, uint64) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	return i.start, i.end
}

func (i *liveTracesIter) Close() {
	i.cancel()
}

var _ common.Iterator = (*liveTracesIter)(nil)
