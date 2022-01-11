package ingester

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/search"
	"github.com/grafana/tempo/tempodb/wal"
)

type traceTooLargeError struct {
	traceID           common.ID
	maxBytes, reqSize int
}

func newTraceTooLargeError(traceID common.ID, maxBytes, reqSize int) *traceTooLargeError {
	return &traceTooLargeError{
		traceID:  traceID,
		maxBytes: maxBytes,
		reqSize:  reqSize,
	}
}

func (e traceTooLargeError) Error() string {
	return fmt.Sprintf(
		"%s max size of trace (%d) exceeded while adding %d bytes to trace %s",
		overrides.ErrorPrefixTraceTooLarge, e.maxBytes, e.reqSize, hex.EncodeToString(e.traceID))
}

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

const (
	traceDataType  = "trace"
	searchDataType = "search"
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricLiveTraces = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "ingester_live_traces",
		Help:      "The current number of lives traces per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
	metricBytesReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_bytes_received_total",
		Help:      "The total bytes received per tenant.",
	}, []string{"tenant", "data_type"})
)

type instance struct {
	tracesMtx   sync.Mutex
	traces      map[uint32]*liveTrace
	largeTraces map[uint32]int // maxBytes that trace exceeded
	traceCount  atomic.Int32

	blocksMtx        sync.RWMutex
	headBlock        *wal.AppendBlock
	completingBlocks []*wal.AppendBlock
	completeBlocks   []*wal.LocalBlock

	searchHeadBlock      *searchStreamingBlockEntry
	searchAppendBlocks   map[*wal.AppendBlock]*searchStreamingBlockEntry
	searchCompleteBlocks map[*wal.LocalBlock]*searchLocalBlockEntry

	lastBlockCut time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	bytesReceivedTotal *prometheus.CounterVec
	limiter            *Limiter
	writer             tempodb.Writer

	local       *local.Backend
	localReader backend.Reader
	localWriter backend.Writer

	hash hash.Hash32
}

type searchStreamingBlockEntry struct {
	b   *search.StreamingSearchBlock
	mtx sync.RWMutex
}

type searchLocalBlockEntry struct {
	b   *search.BackendSearchBlock
	mtx sync.RWMutex
}

func newInstance(instanceID string, limiter *Limiter, writer tempodb.Writer, l *local.Backend) (*instance, error) {
	i := &instance{
		traces:               map[uint32]*liveTrace{},
		largeTraces:          map[uint32]int{},
		searchAppendBlocks:   map[*wal.AppendBlock]*searchStreamingBlockEntry{},
		searchCompleteBlocks: map[*wal.LocalBlock]*searchLocalBlockEntry{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		bytesReceivedTotal: metricBytesReceivedTotal,
		limiter:            limiter,
		writer:             writer,

		local:       l,
		localReader: backend.NewReader(l),
		localWriter: backend.NewWriter(l),

		hash: fnv.New32(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) PushBytesRequest(ctx context.Context, req *tempopb.PushBytesRequest) error {
	for j := range req.Traces {

		// Search data is optional.
		var searchData []byte
		if len(req.SearchData) > j && len(req.SearchData[j].Slice) > 0 {
			searchData = req.SearchData[j].Slice
		}

		err := i.PushBytes(ctx, req.Ids[j].Slice, req.Traces[j].Slice, searchData)
		if err != nil {
			return err
		}
	}
	return nil
}

// PushBytes is used to push an unmarshalled tempopb.Trace to the instance
func (i *instance) PushBytes(ctx context.Context, id []byte, traceBytes []byte, searchData []byte) error {
	i.measureReceivedBytes(traceBytes, searchData)

	if !validation.ValidTraceID(id) {
		return status.Errorf(codes.InvalidArgument, "%s is not a valid traceid", hex.EncodeToString(id))
	}

	// check for max traces before grabbing the lock to better load shed
	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.traceCount.Load()))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "%s max live traces exceeded for tenant %s: %v", overrides.ErrorPrefixLiveTracesExceeded, i.instanceID, err)
	}

	return i.push(ctx, id, traceBytes, searchData)
}

func (i *instance) push(ctx context.Context, id, traceBytes, searchData []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	tkn := i.tokenForTraceID(id)

	if maxBytes, ok := i.largeTraces[tkn]; ok {
		return status.Errorf(codes.FailedPrecondition, (newTraceTooLargeError(id, maxBytes, len(traceBytes)).Error()))
	}

	trace := i.getOrCreateTrace(id)
	err := trace.Push(ctx, i.instanceID, traceBytes, searchData)
	if err != nil {
		if e, ok := err.(*traceTooLargeError); ok {
			i.largeTraces[tkn] = trace.maxBytes
			return status.Errorf(codes.FailedPrecondition, e.Error())
		}
	}

	return err
}

func (i *instance) measureReceivedBytes(traceBytes []byte, searchData []byte) {
	// measure received bytes as sum of slice lengths
	// type byte is guaranteed to be 1 byte in size
	// ref: https://golang.org/ref/spec#Size_and_alignment_guarantees
	i.bytesReceivedTotal.WithLabelValues(i.instanceID, traceDataType).Add(float64(len(traceBytes)))
	i.bytesReceivedTotal.WithLabelValues(i.instanceID, searchDataType).Add(float64(len(searchData)))
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	tracesToCut := i.tracesToCut(cutoff, immediate)

	for _, t := range tracesToCut {
		trace.SortTraceBytes(t.traceBytes)

		out, err := proto.Marshal(t.traceBytes)
		if err != nil {
			return err
		}

		err = i.writeTraceToHeadBlock(t.traceID, out, t.searchData)
		if err != nil {
			return err
		}

		// return trace byte slices to be reused by proto marshalling
		//  WARNING: can't reuse traceid's b/c the appender takes ownership of byte slices that are passed to it
		tempopb.ReuseTraceBytes(t.traceBytes)
	}

	return nil
}

// CutBlockIfReady cuts a completingBlock from the HeadBlock if ready
// Returns a bool indicating if a block was cut along with the error (if any).
func (i *instance) CutBlockIfReady(maxBlockLifetime time.Duration, maxBlockBytes uint64, immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	now := time.Now()
	if i.lastBlockCut.Add(maxBlockLifetime).Before(now) || i.headBlock.DataLength() >= maxBlockBytes || immediate {
		completingBlock := i.headBlock

		i.completingBlocks = append(i.completingBlocks, completingBlock)

		err := i.resetHeadBlock()
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resetHeadBlock: %w", err)
		}

		return completingBlock.BlockID(), nil
	}

	return uuid.Nil, nil
}

// CompleteBlock() moves a completingBlock to a completeBlock. The new completeBlock has the same ID
func (i *instance) CompleteBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()

	var completingBlock *wal.AppendBlock
	for _, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			break
		}
	}
	i.blocksMtx.Unlock()

	if completingBlock == nil {
		return fmt.Errorf("error finding completingBlock")
	}

	ctx := context.Background()

	backendBlock, err := i.writer.CompleteBlockWithBackend(ctx, completingBlock, model.ObjectCombiner, i.localReader, i.localWriter)
	if err != nil {
		return errors.Wrap(err, "error completing wal block with local backend")
	}

	ingesterBlock, err := wal.NewLocalBlock(ctx, backendBlock, i.local)
	if err != nil {
		return errors.Wrap(err, "error creating ingester block")
	}

	// Search data (optional)
	i.blocksMtx.RLock()
	oldSearch := i.searchAppendBlocks[completingBlock]
	i.blocksMtx.RUnlock()

	var newSearch *search.BackendSearchBlock
	if oldSearch != nil {
		newSearch, err = i.writer.CompleteSearchBlockWithBackend(oldSearch.b, backendBlock.BlockMeta().BlockID, backendBlock.BlockMeta().TenantID, i.localReader, i.localWriter)
		if err != nil {
			return err
		}
	}

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if newSearch != nil {
		i.searchCompleteBlocks[ingesterBlock] = &searchLocalBlockEntry{
			b: newSearch,
		}
	}
	i.completeBlocks = append(i.completeBlocks, ingesterBlock)

	return nil
}

func (i *instance) ClearCompletingBlock(blockID uuid.UUID) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	var completingBlock *wal.AppendBlock
	for j, iterBlock := range i.completingBlocks {
		if iterBlock.BlockID() == blockID {
			completingBlock = iterBlock
			i.completingBlocks = append(i.completingBlocks[:j], i.completingBlocks[j+1:]...)
			break
		}
	}

	if completingBlock != nil {
		entry := i.searchAppendBlocks[completingBlock]
		if entry != nil {
			// Take write lock to ensure no searches are reading.
			entry.mtx.Lock()
			defer entry.mtx.Unlock()
			_ = entry.b.Clear()
			delete(i.searchAppendBlocks, completingBlock)
		}

		return completingBlock.Clear()
	}

	return errors.New("Error finding wal completingBlock to clear")
}

// GetBlockToBeFlushed gets a list of blocks that can be flushed to the backend
func (i *instance) GetBlockToBeFlushed(blockID uuid.UUID) *wal.LocalBlock {
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, c := range i.completeBlocks {
		if c.BlockMeta().BlockID == blockID && c.FlushedTime().IsZero() {
			return c
		}
	}

	return nil
}

func (i *instance) ClearFlushedBlocks(completeBlockTimeout time.Duration) error {
	var err error

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for idx, b := range i.completeBlocks {
		flushedTime := b.FlushedTime()
		if flushedTime.IsZero() {
			continue
		}

		if flushedTime.Add(completeBlockTimeout).Before(time.Now()) {
			i.completeBlocks = append(i.completeBlocks[:idx], i.completeBlocks[idx+1:]...)

			searchEntry := i.searchCompleteBlocks[b]
			if searchEntry != nil {
				searchEntry.mtx.Lock()
				defer searchEntry.mtx.Unlock()
				delete(i.searchCompleteBlocks, b)
			}

			err = i.local.ClearBlock(b.BlockMeta().BlockID, i.instanceID)
			if err == nil {
				metricBlocksClearedTotal.Inc()
			}
			break
		}
	}

	return err
}

func (i *instance) FindTraceByID(ctx context.Context, id []byte) (*tempopb.Trace, error) {
	var err error
	var completeTrace *tempopb.Trace

	// live traces
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[i.tokenForTraceID(id)]; ok {
		allBytes, err := proto.Marshal(liveTrace.traceBytes)
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to marshal liveTrace: %w", err)
		}
		completeTrace, err = model.MustNewDecoder(model.CurrentEncoding).PrepareForRead(allBytes)
		if err != nil {
			i.tracesMtx.Unlock()
			return nil, fmt.Errorf("unable to unmarshal liveTrace: %w", err)
		}
	}
	i.tracesMtx.Unlock()

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// headBlock
	foundBytes, err := i.headBlock.Find(id, model.ObjectCombiner)
	if err != nil {
		return nil, fmt.Errorf("headBlock.Find failed: %w", err)
	}
	completeTrace, err = model.CombineForRead(foundBytes, i.headBlock.Meta().DataEncoding, completeTrace)
	if err != nil {
		return nil, fmt.Errorf("headblock unmarshal failed in FindTraceByID")
	}

	// completingBlock
	for _, c := range i.completingBlocks {
		foundBytes, err = c.Find(id, model.ObjectCombiner)
		if err != nil {
			return nil, fmt.Errorf("completingBlock.Find failed: %w", err)
		}
		completeTrace, err = model.CombineForRead(foundBytes, c.Meta().DataEncoding, completeTrace)
		if err != nil {
			return nil, fmt.Errorf("completingBlocks combine failed in FindTraceByID")
		}
	}

	// completeBlock
	for _, c := range i.completeBlocks {
		foundBytes, err = c.Find(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("completeBlock.Find failed: %w", err)
		}
		completeTrace, err = model.CombineForRead(foundBytes, c.BlockMeta().DataEncoding, completeTrace)
		if err != nil {
			return nil, fmt.Errorf("completeBlock combine failed in FindTraceByID")
		}
	}

	return completeTrace, nil
}

// AddCompletingBlock adds an AppendBlock directly to the slice of completing blocks.
// This is used during wal replay. It is expected that calling code will add the appropriate
// jobs to the queue to eventually flush these.
func (i *instance) AddCompletingBlock(b *wal.AppendBlock, s *search.StreamingSearchBlock) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	i.completingBlocks = append(i.completingBlocks, b)

	// search WAL
	if s == nil {
		return
	}
	i.searchAppendBlocks[b] = &searchStreamingBlockEntry{b: s}
}

// getOrCreateTrace will return a new trace object for the given request
//  It must be called under the i.tracesMtx lock
func (i *instance) getOrCreateTrace(traceID []byte) *liveTrace {
	fp := i.tokenForTraceID(traceID)
	trace, ok := i.traces[fp]
	if ok {
		return trace
	}

	maxBytes := i.limiter.limits.MaxBytesPerTrace(i.instanceID)
	maxSearchBytes := i.limiter.limits.MaxSearchBytesPerTrace(i.instanceID)
	trace = newTrace(traceID, maxBytes, maxSearchBytes)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()
	i.traceCount.Inc()

	return trace
}

// tokenForTraceID hash trace ID, should be called under lock
func (i *instance) tokenForTraceID(id []byte) uint32 {
	i.hash.Reset()
	_, _ = i.hash.Write(id)
	return i.hash.Sum32()
}

// resetHeadBlock() should be called under lock
func (i *instance) resetHeadBlock() error {

	// Clear large traces when cutting block
	i.tracesMtx.Lock()
	i.largeTraces = map[uint32]int{}
	i.tracesMtx.Unlock()

	oldHeadBlock := i.headBlock
	var err error
	newHeadBlock, err := i.writer.WAL().NewBlock(uuid.New(), i.instanceID, model.CurrentEncoding)
	if err != nil {
		return err
	}

	i.headBlock = newHeadBlock
	i.lastBlockCut = time.Now()

	// Create search data wal file
	f, version, enc, err := i.writer.WAL().NewFile(i.headBlock.BlockID(), i.instanceID, searchDir)
	if err != nil {
		return err
	}

	b, err := search.NewStreamingSearchBlockForFile(f, i.headBlock.BlockID(), version, enc)
	if err != nil {
		return err
	}
	if i.searchHeadBlock != nil {
		i.searchAppendBlocks[oldHeadBlock] = i.searchHeadBlock
	}
	i.searchHeadBlock = &searchStreamingBlockEntry{
		b: b,
	}
	return nil
}

func (i *instance) tracesToCut(cutoff time.Duration, immediate bool) []*liveTrace {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	// Set this before cutting to give a more accurate number.
	metricLiveTraces.WithLabelValues(i.instanceID).Set(float64(len(i.traces)))

	cutoffTime := time.Now().Add(cutoff)
	tracesToCut := make([]*liveTrace, 0, len(i.traces))

	for key, trace := range i.traces {
		if cutoffTime.After(trace.lastAppend) || immediate {
			tracesToCut = append(tracesToCut, trace)
			delete(i.traces, key)
		}
	}
	i.traceCount.Store(int32(len(i.traces)))

	return tracesToCut
}

func (i *instance) writeTraceToHeadBlock(id common.ID, b []byte, searchData [][]byte) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	err := i.headBlock.Append(id, b)
	if err != nil {
		return err
	}

	entry := i.searchHeadBlock
	if entry != nil {
		// Don't take a write lock on the block here. It is safe
		// for the appender to write to its file while a search
		// is reading it. This prevents stalling the write path
		// while a search is happening. There are mutexes internally
		// for the parts that aren't.
		err := entry.b.Append(context.TODO(), id, searchData)
		return err
	}

	return nil
}

func (i *instance) rediscoverLocalBlocks(ctx context.Context) ([]*wal.LocalBlock, error) {
	ids, err := i.localReader.Blocks(ctx, i.instanceID)
	if err != nil {
		return nil, err
	}

	hasWal := func(id uuid.UUID) bool {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()
		for _, b := range i.completingBlocks {
			if b.BlockID() == id {
				return true
			}
		}
		return false
	}

	var rediscoveredBlocks []*wal.LocalBlock

	for _, id := range ids {

		// Ignore blocks that have a matching wal. The wal will be replayed and the local block recreated.
		// NOTE - Wal replay must be done beforehand.
		if hasWal(id) {
			continue
		}

		// See if block is intact by checking for meta, which is written last.
		// If meta missing then block was not successfully written.
		meta, err := i.localReader.BlockMeta(ctx, id, i.instanceID)
		if err != nil {
			if err == backend.ErrDoesNotExist {
				// Partial/incomplete block found, remove, it will be recreated from data in the wal.
				level.Warn(log.Logger).Log("msg", "Unable to reload meta for local block. This indicates an incomplete block and will be deleted", "tenant", i.instanceID, "block", id.String())
				err = i.local.ClearBlock(id, i.instanceID)
				if err != nil {
					return nil, errors.Wrapf(err, "deleting bad local block tenant %v block %v", i.instanceID, id.String())
				}
				continue
			}

			return nil, err
		}

		b, err := encoding.NewBackendBlock(meta, i.localReader)
		if err != nil {
			return nil, err
		}

		ib, err := wal.NewLocalBlock(ctx, b, i.local)
		if err != nil {
			return nil, err
		}

		rediscoveredBlocks = append(rediscoveredBlocks, ib)

		sb := search.OpenBackendSearchBlock(b.BlockMeta().BlockID, b.BlockMeta().TenantID, i.localReader)

		i.blocksMtx.Lock()
		i.completeBlocks = append(i.completeBlocks, ib)
		i.searchCompleteBlocks[ib] = &searchLocalBlockEntry{b: sb}
		i.blocksMtx.Unlock()

		level.Info(log.Logger).Log("msg", "reloaded local block", "tenantID", i.instanceID, "block", id.String(), "flushed", ib.FlushedTime())
	}

	return rediscoveredBlocks, nil
}
